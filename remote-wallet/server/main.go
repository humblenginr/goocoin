package main

import "os"

var (
    TempWalletFolderPath = "/Users/humblenginr/code/gocoin/wallet"
    TempWalletBinaryPath = TempWalletFolderPath + "/wallet"
)

func main() {
    if(len(os.Args) == 4) {
        TempWalletBinaryPath = os.Args[2]
        TempWalletFolderPath = os.Args[3]
    } 
    websocketWS := NewWSCommunicationServer()
    handler := NewHandler(TempWalletFolderPath, TempWalletBinaryPath)
    websocketWS.SetHandler(&handler)
    err := websocketWS.Listen()
    if err != nil {
        panic(err)
    }
}
