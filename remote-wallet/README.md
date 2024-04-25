# Introduction

This prototype demonstrates the intended changes for the Gocoin project. The remote wallet functionality allows Gocoin users to control their wallet through the BTC Node WebUI. This is achieved by running a Remote Wallet Server on the machine where the wallet is located and using the WalletRemoteClient to connect to it and interact with the Gocoin Wallet.

# Build and Run Instructions

1. Build the WalletRemoteServer: `cd server/; go build`
2. Build the WalletRemoteClient: `cd client/; go build`
3. Start the WalletRemoteServer on the machine where the Gocoin wallet binary is located: `cd server/; ./server localhost:3421`
4. Start the WalletRemoteClient proxy server on the machine where the Gocoin BTC Node is located: `cd client/; ./client`
5. Build and run the Gocoin BTC node.
