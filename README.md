Debora
------

A simple tool for allowing developers to push updates to the network running their p2p application.

# How it works

Add a new `MsgDeboraTy` to your P2P protocol which receives a message (presumably from the developer)
and initiates the upgrade sequence by calling `debora.Call(payload []byte)`, where `payload` is the payload that came in from 
the developer in the `MsgDeboraTy` msg.
This will kill the running process, upgrade it, and restart it.

Note we make every effort to avoid having the users open any extra ports (so we use existing, possibly outbound, connections from the p2p layer).

As a developer, first create a new key pair for your app with `debora -keygen <appname>`.

In the beginning of the program, you should call `debora.Add(key []byte)`, where key is the public key generated in the previous step,
(it should be hardcoded into the application).
This will start the debora process on the client machine if it is not already running,
and add the process id and provided key to the debora's table of processes. 
Debora will now use this key to negotiate a shared secret for an HMAC, and will only accept messages 
regarding this process if they are signed with the appropriate hmac key.
This negotiation occurs every time `Call(payload []byte)` is called.

Now, include a `debora-dev` flag in your apps cli which when provided calls `debora.DebMasterListenAndServe(appName string, callFunc func(payload []byte))`, 
where `callFunc` is responsible for broadcasting a `MsgDeboraTy` message containing the payload to all peers. 
`DebMasterListenAndServe` will start a little in-process http server which can be called with `debora -call <appname>`, 
triggering the `DeboraMsgTy` broadcast and hence the upgrade protocol in all connected peers. 
If a client attempts this, it will fail as they (presumably) do not have the appropriate key. 
The developer's public key should be hard coded into the application's source code and provided in `debora.Add(key)`.

When a client receives the message and runs `debora.Call(payload []byte)`, it will send a request to a local debora instance which will generate
a random nonce (encrypted with the developers public key) and send it to the developer's deborah (the address is provided in the payload). 
If the response includes an HMAC signed with the random nonce, then the developer has been authenticated, and deborah will
shutdown the appropriate process, upgrade it with a `git pull` and `go install`, and restart it.

# Example

There is a full example program in `cmd/example`.

Install the example with

```
cd $GOPATH/src/github.com/ebuchman/debora/cmd/example
go get -d
go install
```

Run the client node with

```
example -debora
```

This will start the application and add the process to a local debora daemon (which will also be started).
This simulates a user running some application on a p2p network

Run the developer's debora in a new window with

```
example -debora-dev
```

In practice this will be on a different machine (the developer's), and will typically serve as the bootstrap node. 
The two nodes will now ping eachother back and forth using a dead simple http protocol.

Now, to initiate the upgrade procedure, open a new window (again, would be on the developer's machine), and run

```
debora call --remote-port 56566 example
```

This will ping the in-process debora server running with the app on the developer's machine, triggering it to broadcast the upgrade message to all connected peers.
You should see the original node receive the broadcast, initiate the handshake to authenticate the caller, and then upgrade, terminate, and restart.

And that's that!

Welcome to Debora.
