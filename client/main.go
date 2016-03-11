package main

import (
	"os"
	"fmt"
	"time"
	"bytes"
	"unsafe"
	"runtime"
	"os/signal"
	"runtime/debug"
	"github.com/piotrnar/gocoin/lib/btc"
	"github.com/piotrnar/gocoin/lib/qdb"
	"github.com/piotrnar/gocoin/lib/chain"
	"github.com/piotrnar/gocoin/lib"
	"github.com/piotrnar/gocoin/lib/others/sys"
	"github.com/piotrnar/gocoin/client/common"
	"github.com/piotrnar/gocoin/client/wallet"
	"github.com/piotrnar/gocoin/client/network"
	"github.com/piotrnar/gocoin/client/usif"
	"github.com/piotrnar/gocoin/client/usif/textui"
	"github.com/piotrnar/gocoin/client/usif/webui"
	"github.com/piotrnar/gocoin/lib/others/peersdb"
)


var killchan chan os.Signal = make(chan os.Signal)
var retryCachedBlocks bool


func LocalAcceptBlock(newbl *network.BlockRcvd) (e error) {
	sta := time.Now()
	newbl.TmQueuing = sta.Sub(newbl.Time) - newbl.TmDownload
	bl := newbl.Block
	e = common.BlockChain.CommitBlock(bl, newbl.BlockTreeNode)
	if e == nil {
		// new block accepted
		newbl.TmAccept = time.Now().Sub(sta)

		common.RecalcAverageBlockSize(false)

		for i:=1; i<len(bl.Txs); i++ {
			network.TxMined(bl.Txs[i])
		}

		if int64(bl.BlockTime()) > time.Now().Add(-10*time.Minute).Unix() {
			// Freshly mined block - do the inv and beeps...
			common.Busy("NetRouteInv")
			network.NetRouteInv(2, bl.Hash, newbl.Conn)

			if common.CFG.Beeps.NewBlock {
				fmt.Println("\007Received block", common.BlockChain.BlockTreeEnd.Height)
				textui.ShowPrompt()
			}

			if common.CFG.Beeps.MinerID!="" {
				//_, rawtxlen := btc.NewTx(bl[bl.TxOffset:])
				if bytes.Contains(bl.Txs[0].Serialize(), []byte(common.CFG.Beeps.MinerID)) {
					fmt.Println("\007Mined by '"+common.CFG.Beeps.MinerID+"':", bl.Hash)
					textui.ShowPrompt()
				}
			}

			if common.CFG.Beeps.ActiveFork && common.Last.Block == common.BlockChain.BlockTreeEnd {
				// Last block has not changed, so it must have been an orphaned block
				bln := common.BlockChain.BlockIndex[bl.Hash.BIdx()]
				commonNode := common.Last.Block.FirstCommonParent(bln)
				forkDepth := bln.Height - commonNode.Height
				fmt.Println("Orphaned block:", bln.Height, bl.Hash.String(), bln.BlockSize>>10, "KB")
				if forkDepth > 1 {
					fmt.Println("\007\007\007WARNING: the fork is", forkDepth, "blocks deep")
				}
				textui.ShowPrompt()
			}

			if wallet.BalanceChanged && common.CFG.Beeps.NewBalance{
				fmt.Print("\007")
			}
		}

		common.Last.Mutex.Lock()
		common.Last.Time = time.Now()
		common.Last.Block = common.BlockChain.BlockTreeEnd
		common.Last.Mutex.Unlock()

		if wallet.BalanceChanged {
			wallet.BalanceChanged = false
			fmt.Println("Your balance has just changed")
			fmt.Print(wallet.DumpBalance(wallet.MyBalance, nil, false, true))
			textui.ShowPrompt()
		}
	} else {
		fmt.Println("Warning: AcceptBlock failed. If the block was valid, you may need to rebuild the unspent DB (-r)")
	}
	return
}


func retry_cached_blocks() bool {
	var idx int
	common.CountSafe("RedoCachedBlks")
	for idx < len(network.CachedBlocks) {
		newbl := network.CachedBlocks[idx]
		if newbl.Block.Height==common.BlockChain.BlockTreeEnd.Height+1 {
			common.Busy("Cache.LocalAcceptBlock "+newbl.Block.Hash.String())
			e := LocalAcceptBlock(newbl)
			if e != nil {
				fmt.Println("AcceptBlock:", e.Error())
				newbl.Conn.DoS("LocalAcceptBl")
			}
			// remove it from cache
			network.CachedBlocks = append(network.CachedBlocks[:idx], network.CachedBlocks[idx+1:]...)
			return len(network.CachedBlocks)>0
		} else {
			idx++
		}
	}
	return false
}


// Called from the blockchain thread
func HandleNetBlock(newbl *network.BlockRcvd) {
	if int(newbl.Block.Height)-int(common.BlockChain.BlockTreeEnd.Height) > 1 {
		// it's not linking - keep it for later
		//println("try block", newbl.Block.Height, "later as now we are at", common.BlockChain.BlockTreeEnd.Height)
		//network.NetBlocks <- newbl
		network.CachedBlocks = append(network.CachedBlocks, newbl)
		common.CountSafe("BlockPostone")
		return
	}

	common.Busy("LocalAcceptBlock "+newbl.Hash.String())
	e := LocalAcceptBlock(newbl)
	if e != nil {
		fmt.Println("AcceptBlock:", e.Error())
		newbl.Conn.DoS("LocalAcceptBl")
	} else {
		//println("block", newbl.Block.Height, "accepted")
		retryCachedBlocks = retry_cached_blocks()
	}

}


func defrag_db() {
	if (usif.DefragBlocksDB&1) != 0 {
		qdb.SetDefragPercent(1)
		fmt.Print("Defragmenting UTXO database")
		for {
			if !common.BlockChain.Unspent.Idle() {
				break
			}
			fmt.Print(".")
		}
		fmt.Println("done")
	}

	if (usif.DefragBlocksDB&2) != 0 {
		fmt.Println("Creating empty database in", common.GocoinHomeDir+"defrag", "...")
		os.RemoveAll(common.GocoinHomeDir+"defrag")
		defragdb := chain.NewBlockDB(common.GocoinHomeDir+"defrag")
		fmt.Println("Defragmenting the database...")
		blk := common.BlockChain.BlockTreeRoot
		for {
			blk = blk.FindPathTo(common.BlockChain.BlockTreeEnd)
			if blk==nil {
				fmt.Println("Database defragmenting finished successfully")
				fmt.Println("To use the new DB, move the two new files to a parent directory and restart the client")
				break
			}
			if (blk.Height&0xff)==0 {
				fmt.Printf("%d / %d blocks written (%d%%)\r", blk.Height, common.BlockChain.BlockTreeEnd.Height,
					100 * blk.Height / common.BlockChain.BlockTreeEnd.Height)
			}
			bl, trusted, er := common.BlockChain.Blocks.BlockGet(blk.BlockHash)
			if er != nil {
				fmt.Println("FATAL ERROR during BlockGet:", er.Error())
				break
			}
			nbl, er := btc.NewBlock(bl)
			if er != nil {
				fmt.Println("FATAL ERROR during NewBlock:", er.Error())
				break
			}
			nbl.Trusted = trusted
			defragdb.BlockAdd(blk.Height, nbl)
		}
		defragdb.Sync()
		defragdb.Close()
	}
}


func main() {
	var ptr *byte
	if unsafe.Sizeof(ptr) < 8 {
		fmt.Println("WARNING: Gocoin client shall be build for 64-bit arch. It will likely crash now.")
	}

	fmt.Println("Gocoin client version", lib.Version)
	runtime.GOMAXPROCS(runtime.NumCPU()) // It seems that Go does not do it by default

	// Disable Ctrl+C
	signal.Notify(killchan, os.Interrupt, os.Kill)
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("pkg: %v", r)
			}
			fmt.Println("main panic recovered:", err.Error())
			fmt.Println(string(debug.Stack()))
			network.NetCloseAll()
			common.CloseBlockChain()
			peersdb.ClosePeerDB()
			sys.UnlockDatabaseDir()
			os.Exit(1)
		}
	}()

	common.InitConfig()

	if common.FLAG.VolatileUTXO {
		qdb.VolatileMode = true
		fmt.Println("WARNING! Using UTXO database in a volatile mode. Make sure to close the client properly (do not kill it!)")
	}

	host_init() // This will create the DB lock file and keep it open

	if common.FLAG.Rescan && common.FLAG.VolatileUTXO {

		fmt.Println("UTXO database rebuilt complete in the volatile mode, so flush DB to disk and exit...")

	} else {

		common.RecalcAverageBlockSize(true)

		peersTick := time.Tick(5*time.Minute)
		txPoolTick := time.Tick(time.Minute)
		netTick := time.Tick(time.Second)

		peersdb.Testnet = common.Testnet
		peersdb.ConnectOnly = common.CFG.ConnectOnly
		peersdb.Services = common.Services
		peersdb.InitPeers(common.GocoinHomeDir)

		common.Last.Block = common.BlockChain.BlockTreeEnd
		common.Last.Time = time.Unix(int64(common.Last.Block.Timestamp()), 0)
		if common.Last.Time.After(time.Now()) {
			common.Last.Time = time.Now()
		}

		for k, v := range common.BlockChain.BlockIndex {
			network.ReceivedBlocks[k] = &network.OneReceivedBlock{Time: time.Unix(int64(v.Timestamp()), 0)}
		}
		network.LastCommitedHeader = common.Last.Block

		if common.CFG.TextUI.Enabled {
			go textui.MainThread()
		}

		if common.CFG.WebUI.Interface!="" {
			fmt.Println("Starting WebUI at", common.CFG.WebUI.Interface, "...")
			go webui.ServerThread(common.CFG.WebUI.Interface)
		}

		for !usif.Exit_now {
			common.CountSafe("MainThreadLoops")
			for retryCachedBlocks {
				retryCachedBlocks = retry_cached_blocks()
				// We have done one per loop - now do something else if pending...
				if len(network.NetBlocks)>0 || len(usif.UiChannel)>0 {
					break
				}
			}

			common.Busy("")

			select {
				case s := <-killchan:
					fmt.Println("Got signal:", s)
					usif.Exit_now = true
					continue

				case newbl := <-network.NetBlocks:
					common.CountSafe("MainNetBlock")
					common.Busy("HandleNetBlock()")
					HandleNetBlock(newbl)

				case newtx := <-network.NetTxs:
					common.CountSafe("MainNetTx")
					common.Busy("network.HandleNetTx()")
					network.HandleNetTx(newtx, false)

				case newal := <-network.NetAlerts:
					common.CountSafe("MainNetAlert")
					fmt.Println("\007" + newal)
					textui.ShowPrompt()

				case <-netTick:
					common.CountSafe("MainNetTick")
					common.Busy("network.NetworkTick()")
					network.NetworkTick()

				case cmd := <-usif.UiChannel:
					common.CountSafe("MainUICmd")
					common.Busy("UI command")
					cmd.Handler(cmd.Param)
					cmd.Done.Done()
					continue

				case <-peersTick:
					common.Busy("peersdb.ExpirePeers()")
					peersdb.ExpirePeers()

				case <-txPoolTick:
					common.Busy("network.ExpireTxs()")
					network.ExpireTxs()

				case <-time.After(time.Second/2):
					common.CountSafe("MainThreadTouts")
					if !retryCachedBlocks {
						common.Busy("BlockChain.Idle()")
						if common.BlockChain.Idle() {
							common.CountSafe("ChainIdleUsed")
						}
					}
					continue
			}
		}

		network.NetCloseAll()

		if usif.DefragBlocksDB!=0 {
			defrag_db()
		}
	}

	common.CloseBlockChain()
	peersdb.ClosePeerDB()
	sys.UnlockDatabaseDir()
}
