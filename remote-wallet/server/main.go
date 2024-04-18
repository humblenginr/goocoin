package main

func main() {
    websocketWS := NewWSCommunicationServer()
    handler := NewHandler("/Users/humblenginr/code/gocoin/wallet", "/Users/humblenginr/code/gocoin/wallet/wallet")
    websocketWS.SetHandler(&handler)
    err := websocketWS.Listen()
    if err != nil {
        panic(err)
    }
}
