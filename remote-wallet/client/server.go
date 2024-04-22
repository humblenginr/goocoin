package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
)

type SignTransactionReq struct {
    PayCmd string `json:"payCmd"`
    Tx2Sign string `json:"tx2Sign"`
    Unspent string `json:"unspent"`
    BalanceFileName string `json:"balanceFileName"`
    BalanceFileContents string `json:"balanceFileContents"`
}

func sign(w http.ResponseWriter, req *http.Request) {
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
    rawHex := SignTransactionRequest(zipReader)
    fmt.Println(rawHex)
    fmt.Fprintf(w, "%s\n", rawHex)
}

func startServer() {
    http.HandleFunc("/sign-transaction", sign)

    http.ListenAndServe(":8090", nil)
}
