package main

var (
    TempWalletFolderPath = "/Users/humblenginr/code/gocoin/wallet"
    TempWalletBinaryPath = TempWalletFolderPath + "/wallet"
)

func main() {
    websocketWS := NewWSCommunicationServer()
    handler := NewHandler(TempWalletFolderPath, TempWalletBinaryPath)
    websocketWS.SetHandler(&handler)
    err := websocketWS.Listen()
    if err != nil {
        panic(err)
    }
}
