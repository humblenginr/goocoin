package webui

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/piotrnar/gocoin/client/common"
	"github.com/piotrnar/gocoin/client/usif"
	"github.com/piotrnar/gocoin/lib/btc"
	"github.com/piotrnar/gocoin/lib/utxo"
)


const (
	AvgSignatureSize = 73
	AvgPublicKeySize = 34 /*Assumine compressed key*/
)


type MultisigAddr struct {
	MultiAddress string
	ScriptPubKey string
	KeysRequired, KeysProvided uint
	RedeemScript string
	ListOfAddres []string
}

type SignTransactionRequestPayload struct {
    PayCmd string `json:"payCmd"`
    Tx2Sign string `json:"tx2Sign"`
    Unspent string `json:"unspent"`
    BalanceFileName string `json:"balanceFileName"`
    BalanceFileContents string `json:"balanceFileContents"`
}

func dl_payment(w http.ResponseWriter, r *http.Request) {
	if !ipchecker(r) || !common.GetBool(&common.WalletON)  {
		return
	}

	var err string

    // here rather than downloading the zip file, we have to send it as a json to the client server
	if len(r.Form["outcnt"])==1 {
		var thisbal utxo.AllUnspentTx
		var pay_cmd string
		var totalinput, spentsofar uint64
		var change_addr *btc.BtcAddr

		tx := new(btc.Tx)
		tx.Version = 1
		tx.Lock_time = 0

		seq, er := strconv.ParseInt(r.Form["tx_seq"][0], 10, 64)
		if er != nil || seq < -2 || seq > 0xffffffff {
			err = "Incorrect Sequence value: " + r.Form["tx_seq"][0]
			goto error
		}

		outcnt, _ := strconv.ParseUint(r.Form["outcnt"][0], 10, 32)

		lck := new(usif.OneLock)
		lck.In.Add(1)
		lck.Out.Add(1)
		usif.LocksChan <- lck
		lck.In.Wait()
		defer lck.Out.Done()

		for i:=1; i<=int(outcnt); i++ {
			is := fmt.Sprint(i)
			if len(r.Form["txout"+is])==1 && r.Form["txout"+is][0]=="on" {
				hash := btc.NewUint256FromString(r.Form["txid"+is][0])
				if hash!=nil {
					vout, er := strconv.ParseUint(r.Form["txvout"+is][0], 10, 32)
					if er==nil {
						var po = btc.TxPrevOut{Hash:hash.Hash, Vout:uint32(vout)}
						if res := common.BlockChain.Unspent.UnspentGet(&po); res != nil {
							addr := btc.NewAddrFromPkScript(res.Pk_script, common.Testnet)

							unsp := &utxo.OneUnspentTx{TxPrevOut:po, Value:res.Value,
								MinedAt:res.BlockHeight, Coinbase:res.WasCoinbase, BtcAddr:addr}

							thisbal = append(thisbal, unsp)

							// Add the input to our tx
							tin := new(btc.TxIn)
							tin.Input = po
							tin.Sequence = uint32(seq)
							tx.TxIn = append(tx.TxIn, tin)

							// Add the value to total input value
							totalinput += res.Value

							// If no change specified, use the first input addr as it
							if change_addr == nil {
								change_addr = addr
							}
						}
					}
				}
			}
		}

		if change_addr == nil {
			// There werte no inputs
			return
		}

		for i:=1; ; i++ {
			adridx := fmt.Sprint("adr", i)
			btcidx := fmt.Sprint("btc", i)

			if len(r.Form[adridx])!=1 || len(r.Form[btcidx])!=1 {
				break
			}

			if len(r.Form[adridx][0])>1 {
				addr, er := btc.NewAddrFromString(r.Form[adridx][0])
				if er == nil {
					am, er := btc.StringToSatoshis(r.Form[btcidx][0])
					if er==nil && am>0 {
						if pay_cmd=="" {
							pay_cmd = "wallet -a=false -useallinputs -send "
						} else {
							pay_cmd += ","
						}
						pay_cmd += addr.String() + "=" + btc.UintToBtc(am)

						outs, er := btc.NewSpendOutputs(addr, am, common.CFG.Testnet)
						if er != nil {
							err = er.Error()
							goto error
						}
						tx.TxOut = append(tx.TxOut, outs...)

						spentsofar += am
					} else {
						err = "Incorrect amount (" + r.Form[btcidx][0] + ") for Output #" + fmt.Sprint(i)
						goto error
					}
				} else {
					err = "Incorrect address (" + r.Form[adridx][0] + ") for Output #" + fmt.Sprint(i)
					goto error
				}
			}
		}

		if pay_cmd=="" {
			err = "No inputs selected"
			goto error
		}

		pay_cmd += fmt.Sprint(" -seq ", seq)

		am, er := btc.StringToSatoshis(r.Form["txfee"][0])
		if er != nil {
			err = "Incorrect fee value: " + r.Form["txfee"][0]
			goto error
		}

		pay_cmd += " -fee " + r.Form["txfee"][0]
		spentsofar += am

		if len(r.Form["change"][0])>1 {
			addr, er := btc.NewAddrFromString(r.Form["change"][0])
			if er != nil {
				err = "Incorrect change address: " + r.Form["change"][0]
				goto error
			}
			change_addr = addr
		}
		pay_cmd += " -change " + change_addr.String()

		if totalinput > spentsofar {
			// Add change output
			outs, er := btc.NewSpendOutputs(change_addr, totalinput - spentsofar, common.CFG.Testnet)
			if er != nil {
				err = er.Error()
				goto error
			}
			tx.TxOut = append(tx.TxOut, outs...)
		}

        st := SignTransactionRequestPayload{}


		was_tx := make(map [[32]byte] bool, len(thisbal))
		for i := range thisbal {
			if was_tx[thisbal[i].TxPrevOut.Hash] {
				continue
			}
			was_tx[thisbal[i].TxPrevOut.Hash] = true
			txid := btc.NewUint256(thisbal[i].TxPrevOut.Hash[:])
			if dat, er := common.GetRawTx(thisbal[i].MinedAt, txid); er == nil {
                st.BalanceFileName = txid.String() + ".tx"
                st.BalanceFileContents = hex.EncodeToString(dat)
			} else {
				println(er.Error())
			}
		}

        b := bytes.NewBuffer(make([]byte, 0))
		for i := range thisbal {
			fmt.Fprintln(b, thisbal[i].UnspentTextLine())
		}
        st.Unspent = string(b.Bytes())

		if pay_cmd!="" {
            st.PayCmd = pay_cmd
		}

		// Non-multisig transaction ...
        st.Tx2Sign = string(tx.Serialize())

        jsonValue, err := json.Marshal(st)
        if err != nil {
            fmt.Println(err)
        }


        client := &http.Client{}
        requestURL := fmt.Sprintf("http://localhost:%d/sign-transaction", 8090)
        req, _ := http.NewRequest(http.MethodPost,requestURL, bytes.NewBuffer(jsonValue))
        req.Header["Content-Type"] =  []string{"application/json"}
        res, err := client.Do(req)
        if err != nil {
            fmt.Printf("error making http request: %s\n", err)
        }

        bodyBytes, err := io.ReadAll(res.Body)
        if err != nil {
            fmt.Printf("error parsing http response: %s\n", err)
        }
        bodyString := string(bodyBytes)
        fmt.Printf("Res: %v", bodyString)

		w.Header()["Content-Type"] = []string{"text/plain"}
		w.Write([]byte(bodyString))
		return
	} else {
		err = "Bad request"
	}
error:
	s := load_template("send_error.html")
	write_html_head(w, r)
	s = strings.Replace(s, "<!--ERROR_MSG-->", err, 1)
	w.Write([]byte(s))
	write_html_tail(w)
}


func p_snd(w http.ResponseWriter, r *http.Request) {
	if !ipchecker(r) {
		return
	}

	if !common.GetBool(&common.WalletON) {
		p_wallet_is_off(w, r)
		return
	}

	s := load_template("send.html")

	write_html_head(w, r)
	w.Write([]byte(s))
	write_html_tail(w)
}
