package main

import (
	"context"

	"github.com/piotrnar/gocoin/remote-wallet/common"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type WalletRemoteClient struct {}

func (wrc *WalletRemoteClient) Connect(addr string) (*websocket.Conn, error) {
    ctx := context.Background()
	c, _, err := websocket.Dial(ctx, addr, nil)
	if err != nil {
        return c, err
	}

    return c, err
}

func (wrc *WalletRemoteClient) CloseConnection(c *websocket.Conn) {
    c.Close(websocket.StatusNormalClosure, "")
}

func (wrc *WalletRemoteClient) SendMessage(ctx context.Context, c *websocket.Conn,msgType common.MsgType, payload interface{}) error {
    msg := common.Msg{Type: msgType, Payload: payload}
    err := wsjson.Write(ctx, c, msg)
    if err != nil {
        return err
    }
    return nil
}

func (wrc *WalletRemoteClient) ReadMessage(ctx context.Context, c *websocket.Conn) (common.Msg, error) {
    var msg common.Msg
    err := wsjson.Read(ctx, c, &msg)
    if err != nil {
        return msg, err
    }
    return msg, nil
}


