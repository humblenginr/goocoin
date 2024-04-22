package main

import (
	"bytes"
	"io"
	"net/http"
	"archive/zip"
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"strings"

	common "github.com/piotrnar/gocoin/remote-wallet/common"
	"nhooyr.io/websocket"
)

var (
    ClientServerPort = 8090
    WalletRemoteServerPort = 3421
)


func signTransactionRequest(zipReader *zip.Reader) string {
    // connect with the wallet remote server
    wrc := WalletRemoteClient{}
    wrsUrl := fmt.Sprintf("ws://localhost:%d", WalletRemoteServerPort)
    c, err := wrc.Connect(wrsUrl)
	if err != nil {
		log.Fatal(err)
	}

    // gather required information from the zip reader
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

    // TODO: Make it so that it supports multiple balance files 
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
    c.Close(websocket.StatusNormalClosure, "")
    return msg.Payload.(string)
}

func SignTransactionHandler(w http.ResponseWriter, req *http.Request) {
    w.Header().Set("Content-Type", "application/zip")
    buff := bytes.NewBuffer([]byte{})
    size, err := io.Copy(buff, req.Body)
    if err != nil {
        panic(err)
    }
    reader := bytes.NewReader(buff.Bytes())
    zipReader, err := zip.NewReader(reader, size)
    if err != nil {
        panic(err)
    }
    rawHex := signTransactionRequest(zipReader)
    fmt.Fprintf(w, "%s\n", rawHex)
}

// StartServer starts the proxy server that is responsible for forwarding the request from the WebUI to the WalletRemoteServer.
func StartProxyServer() {
    http.HandleFunc("/sign-transaction", SignTransactionHandler)
    port := fmt.Sprintf(":%d", ClientServerPort)
    http.ListenAndServe(port, nil)
    fmt.Printf("Client proxy server is listening on port: %d", ClientServerPort)
}
