package main

import (
	"archive/zip"
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"strings"

	common "github.com/piotrnar/gocoin/remote-wallet/common"
	"nhooyr.io/websocket"
)

func SignTransactionRequest(zipReader *zip.Reader) string {
    wrc := WalletRemoteClient{}
    c, err := wrc.Connect("ws://127.0.0.1:3421")
	if err != nil {
		log.Fatal(err)
	}

    var balanceFileData string
    var paycmd string
    var tx2sign string
    var unspent string
    var balanceFileName string

    for _, f := range zipReader.File {
		fmt.Printf("Contents of %s:\n", f.Name)
		rc, err := f.Open()
		if err != nil {
            panic(err)
		}
        if(f.Name == "unspent.txt") {
            var dummy []byte
            rc.Read(dummy)
            unspent = string(dummy)
        } else if(f.Name == "pay_cmd.txt"){
            var dummy []byte
            rc.Read(dummy)
            paycmd = string(dummy)
        }else if(f.Name == "tx2sign.txt"){
            var dummy []byte
            rc.Read(dummy)
            tx2sign = string(dummy)
        } else {
            // this should be the balance file contents
            var dummy []byte
            rc.Read(dummy)
            balanceFileData = hex.EncodeToString(dummy)
            s := strings.Split(f.Name, "/")
            balanceFileName = s[1]
        }
		rc.Close()
	}

    // balanceFiledata, _ := os.ReadFile("/Users/humblenginr/Downloads/payment/balance/de9243343418f3abe3e66990190362e620b8dde864f61bdf9e11f4f3a59ebee9.tx")
    payload := common.SignTransactionRequestPayload{
        PayCmd: paycmd,
        Tx2Sign: tx2sign,
        Unspent: unspent,
        BalanceFileName: balanceFileName,
        BalanceFileContents:balanceFileData,
    }
    ctx := context.Background()
    err = wrc.SendMessage(ctx, c, common.SignTransaction, payload)
	if err != nil {
		log.Fatal(err)
	}
    msg, err := wrc.ReadMessage(ctx, c)
	if err != nil {
		log.Fatal(err)
	}
    fmt.Printf("Msg: %v\n", msg)
    c.Close(websocket.StatusNormalClosure, "")
    return msg.Payload.(string)
}

func main() {
    startServer()
   // sendDummySignTransactionReq()
}

