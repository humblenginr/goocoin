package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"

	common "github.com/piotrnar/gocoin/remote-wallet/common"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	c, _, err := websocket.Dial(ctx, "ws://127.0.0.1:3421", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer c.CloseNow()
    balanceFiledata, _ := os.ReadFile("/Users/humblenginr/Downloads/payment/balance/de9243343418f3abe3e66990190362e620b8dde864f61bdf9e11f4f3a59ebee9.tx")
    
    payload := common.SignTransactionRequestPayload{
        PayCmd: "wallet -a=false -useallinputs -send tb1qdv9putdak25c52euujv8ng7n8fp5mup53tl2w7=0.00000100 -seq -2 -fee 0.00000384 -change tb1qeldsjx482jxgyq696d2y8cktcc07lzetqgsxtt",
        Tx2Sign: "0100000001e9be9ea5f3f4119edf1bf664e8ddb820e66203199069e6e3abf31834344392de0800000000feffffff0264000000000000001600146b0a1e2dbdb2a98a2b3ce49879a3d33a434df0340402000000000000160014cfdb091aa7548c820345d35443e2cbc61fef8b2b00000000",
        Unspent: "de9243343418f3abe3e66990190362e620b8dde864f61bdf9e11f4f3a59ebee9-008 # 0.00001000 BTC @ tb1qeldsjx482jxgyq696d2y8cktcc07lzetqgsxtt, block 2586930",
        BalanceFileName: "de9243343418f3abe3e66990190362e620b8dde864f61bdf9e11f4f3a59ebee9.tx",
        BalanceFileContents:hex.EncodeToString(balanceFiledata),
    }
    err = sendMessage(ctx, c, common.SignTransaction, payload)
	if err != nil {
		log.Fatal(err)
	}

    err = readMessage(ctx, c)
	if err != nil {
		log.Fatal(err)
	}

    c.Close(websocket.StatusNormalClosure, "")
}

func sendMessage(ctx context.Context, c *websocket.Conn,msgType common.MsgType, payload interface{}) error {
    msg := common.Msg{Type: msgType, Payload: payload}
    err := wsjson.Write(ctx, c, msg)
    if err != nil {
        return err
    }
    return nil
}

func readMessage(ctx context.Context, c *websocket.Conn) error {
    msg := common.Msg{}
    err := wsjson.Read(ctx, c, &msg)
    if err != nil {
        return err
    }
    fmt.Println(msg.Payload)
    return nil
}
