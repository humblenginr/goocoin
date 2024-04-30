package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	common "github.com/piotrnar/gocoin/remote-wallet/common"
	"nhooyr.io/websocket"
)

var (
    ClientServerPort = 8090
    WalletRemoteServerPort = 3421
)


func signTransactionRequest(payload common.SignTransactionRequestPayload) string {
    // connect with the wallet remote server
    wrc := WalletRemoteClient{}
    wrsUrl := fmt.Sprintf("ws://localhost:%d", WalletRemoteServerPort)
    c, err := wrc.Connect(wrsUrl)
	if err != nil {
		log.Fatal(err)
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
    fmt.Printf("Received a request to sign transaction, forwarding it to the remote wallet server...\n")
    w.Header().Set("Content-Type", "application/json")
    var payload common.SignTransactionRequestPayload
    body, err := io.ReadAll(req.Body)
    if err != nil {
        panic(err)
    }
    err = json.Unmarshal(body, &payload)
    if err != nil {
        panic(err)
    }
    rawHex := signTransactionRequest(payload)
    fmt.Printf("Received the signed transaction raw hex from the wallet remote server: %s\n", rawHex)
    fmt.Fprintf(w, "%s\n", rawHex)
}

// StartServer starts the proxy server that is responsible for forwarding the request from the WebUI to the WalletRemoteServer.
func StartProxyServer() {
    http.HandleFunc("/sign-transaction", SignTransactionHandler)
    port := fmt.Sprintf(":%d", ClientServerPort)
    fmt.Printf("Client proxy server is listening on port: %d\n", ClientServerPort)
    http.ListenAndServe(port, nil)
}
