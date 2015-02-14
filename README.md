Debora
------

A simple tool for allowing developers to push updates to the network running their p2p application.

# High level overview

A developer has some p2p application with a bunch of peers running the software over some testnet.
The developer updates the source code and wants to push the changes to everyone on the network.
The developer runs `debora call <appname>` which pings a server running in his instance of the application.
The server has a hook into the peer list of the p2p protocol, and broadcasts a special message to all the peers.
The peers receive the message and challenge the sender to authenticate via public key.
The peers upgrade and restart the software.

# Interface 

Debora requires three additions to an application's source code:

1. `DebListenAndServe(appName string, port int, callFunc func(payload []byte))` must be called in the process running on the developer's machine only
2. `Add(key, src, app string)` must be called towards the start of the program on every peer's machine
3. `Call(remote string, payload []byte)` must be called when the peer receives a special message over the wire from the developer

Logging can be turned on by running `Logging(true)`. Log output is prefaced with the process id and app name.

# Low level overview

To integrate Debora, add a new `MsgDeboraTy` to your P2P protocol which is presumably sent by the developer
and initiates the upgrade sequence by calling `debora.Call(remote string, payload []byte)`, where remote is
the address of the peer sending `MsgDeboraTy` and `payload` is the payload that came in the msg.
This payload is originally crafted when the developer runs `debora call <appname>`, and should be left unmodified.
The end result of `Call` is to kill the running process, upgrade it, and restart it.

When a client receives the message and runs `debora.Call(remote string, payload []byte)`, it will send a request to a local debora instance which will generate
a random nonce, encrypt it with the developers public key, and send it to the developer's deborah (port is given in payload).
If the response includes an HMAC signed with the random nonce, then the developer has been authenticated, and deborah will
shutdown the appropriate process, upgrade it with a `git pull` and `go install`, and restart it.

Note we make every effort to avoid having the users open any extra ports, so we use existing, possibly outbound, connections from the p2p layer.

We do, however, require the developer to expose another port (for the authentication protocol, so as to not require more additions to the p2p protocol of the application)

As a developer, first create a new key pair for your app with `debora -keygen <appname>`. Take the public key and hardcode it into your application's souce code.

In the beginning of the program, `debora.Add(key, src, app string)` should be called, where `key` is the public key generated in the previous step,
`src` is the repository's path (eg. `github.com/ebuchman/debora`), and `app` is the name of your app (used as a reference later - must not be empty).
`Add` will start the debora process on the client machine if it is not already running,
and add the application's process id and provided key to the debora's table of processes. 
Debora will now use this key to negotiate a shared secret for an HMAC, and will only accept messages 
regarding this process if they are signed with the appropriate hmac key.
This negotiation occurs every time `Call(remote string, payload []byte)` is called (ie. every time the special message is recieved in the p2p protocol)

Now, include a `debora-dev` flag in your apps cli which when provided calls `debora.DebListenAndServe(appName string, port int, callFunc func(payload []byte))`, 
where `callFunc` is responsible for broadcasting a `MsgDeboraTy` message containing the payload to all peers. Do not modify the payload.
`DebListenAndServe` will start a little in-process http server which can be called with `debora -call <appname>`, 
triggering the `DeboraMsgTy` broadcast and hence the upgrade protocol in all connected peers. 
If a client attempts this, it will fail as they (presumably) do not have the appropriate private key to pass the authentication (hmac) step.

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
The two nodes will now ping eachother back and forth using a dead simple http protocol (our simulated p2p protocol)

Now, to initiate the upgrade procedure, open a new window (again, would be on the developer's machine), and run

```
debora call --remote-port 56565 example
```

This will ping the in-process debora server running with the app on the developer's machine, triggering it to broadcast the upgrade message to all connected peers.
You should see the original node receive the broadcast, initiate the handshake to authenticate the caller, and then upgrade, terminate, and restart.

And that's that!

Welcome to Debora.
