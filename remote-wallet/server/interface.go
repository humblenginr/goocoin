package main

import (
	"context"

	"github.com/piotrnar/gocoin/remote-wallet/common"
)

// MessageWriter is similar to io.Writer but for writing common.Msg. 
type MessageWriter interface {
    Write(common.Msg) error
}

type MessageHandler interface {
    // ReceiveMessage is the function that will be called whenever a message is received by the WalletRemoteServer. WalletRemoteServer first parses the message, and sends it to the MessageHandler. 
    ReceiveMessage(common.Msg, MessageWriter)
}

// WalletRemoteServer is the abstraction that allows us to use different CommunicationServer protocols like websocket, bluetooth, NFC, HTTP etc. WalletRemoteServer abstracts the message parsing and sending details and allows the user to handle the messages easily.
type WalletRemoteServer interface {
    // Listen starts listening for connections and gets ready to process requests. Make sure you have set the message handler using SetHandler before starting to listen.
    Listen() error
    // Shutdown shuts the server down cleanly depending on the protocol used.
    Shutdown(context.Context) error
    // SetHandler sets the MessageHandler to be used after receiving and parsing the message
    SetHandler(MessageHandler) 
}
