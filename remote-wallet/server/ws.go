package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	common "github.com/piotrnar/gocoin/remote-wallet/common"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

// WsWalletRemoteServer implements WalletRemoteServer interface.
type WsWalletRemoteServer struct {
	logf func(f string, v ...interface{})
    server *http.Server
    handler MessageHandler
}

func NewWSCommunicationServer() *WsWalletRemoteServer {
    ws := WsWalletRemoteServer{}
    ws.logf = log.Printf
    return &ws
}

func (s *WsWalletRemoteServer) Listen() error {
    l, err := net.Listen("tcp", os.Args[1])
	if err != nil {
		panic(err)
	}
	log.Printf("listening on ws://%v", l.Addr())
	server := &http.Server{
		Handler: s,
	}
    if err != nil {
        return err
    }

    errc := make(chan error, 1)
	go func() {
		errc <- server.Serve(l)
	}()

    sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	select {
	case err := <-errc:
		log.Printf("failed to serve: %v", err)
	case sig := <-sigs:
		log.Printf("terminating: %v", sig)
	}
    ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	return s.Shutdown(ctx)
}

func (s *WsWalletRemoteServer) SetHandler(msgHandler MessageHandler) error {
    s.handler = msgHandler
    return nil
}

func (s *WsWalletRemoteServer) Shutdown(ctx context.Context) error {
    return s.server.Shutdown(ctx)
}


type WSMessageWriter struct {
    conn *websocket.Conn
}

func (w WSMessageWriter) Write(msg common.Msg) error {
    ctx := context.Background()
    return wsjson.Write(ctx, w.conn, msg)
}

func (s *WsWalletRemoteServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		s.logf("%v", err)
		return  
	}
    ctx := context.Background()
    if(err != nil){
        panic(err)
    }
    mwriter := WSMessageWriter{conn: c}
    for {
        var msg common.Msg
        if err := wsjson.Read(ctx, c, &msg); err != nil{
            fmt.Println(err)
            break
        }
        s.handler.ReceiveMessage(msg, mwriter)
    }        
}
