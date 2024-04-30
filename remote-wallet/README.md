# Introduction

This prototype demonstrates the intended changes for the Gocoin project. The remote wallet functionality allows Gocoin users to control their wallet through the BTC Node WebUI. This is achieved by running a Remote Wallet Server on the machine where the wallet is located and using the WalletRemoteClient to connect to it and interact with the Gocoin Wallet.

# Note

The primary goal was to develop a prototype for demonstration purposes. I intentionally prioritised speed over potential improvements in code quality. Additionally, the prototype does not feature user prompt functionality. This was done just to demonstrate my capability to work through the codebase and deliver results. 

## Some Caveats

- It is inteded to run on an environment where both the wallet and the btc node are running on the same machine.
- The binding port of the WalletRemoteServer must be 3421.
- Some values such as the binding ports of the servers are hardcoded.

# Build and Run Instructions

1. Build the WalletRemoteServer: `cd server/; go build`
2. Build the WalletRemoteClient: `cd client/; go build`
3. Start the WalletRemoteServer on the machine where the Gocoin wallet binary is located: `cd server/; ./server localhost:3421 $walletbinarypath $walletfolderpath` 
    - It is mandatory to run the server on localhost:3421 since this value has been hardcoded since this was developed just for demonstration purposes
    - $walletbinarypath is the path of your Gocoin wallet binary
    - $walletfolderpath is the path of the folder where your Gocoin wallet information resides
4. Start the WalletRemoteClient proxy server on the machine where the Gocoin BTC Node is located: `cd client/; ./client`
5. Build and run the Gocoin BTC node.

# Demo

Here is a recorded demo of the prototype: https://www.youtube.com/watch?v=Sd2u9l2tWpA
