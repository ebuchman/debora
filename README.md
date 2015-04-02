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

# Details

When an application is first started (call it PROC1), before there is an existing debora, the call to `Add(key, src, app)`
will use an `exec.Command` to create a debora process (PROC2) and then hang forever. The hung process (PROC1) is our window into
the process group within which all our stopping and restarting of processes will occur. The new debora (PROC2) will write her listen 
port to a file, and use `exec.Command` to start a new instance of the application (PROC3), using the same command that started PROC1.

PROC3 is now our application proper (some distributed protocol, like a bittorent or blockchain client), and PROC2 is its debora. 
PROC3 should function normally as if there were no debora about it, until it receives a special "upgrade" message from a peer.
When the special "upgrade" message arrives, it runs `Call(remote, payload)`, which calls PROC2 (its debora) 
and asks her to authenticate the payload. Debora does so interactively by sending the alleged developer a random nonce encrypted with his 
public key. The message is authenticated if the alleged developer decrypts the nonce and returns HMAC(nonce, nonce).

Once the signal is authenticated, debora switches to the appropriate repo (hard coded in the source) and runs `git fetch -a origin` and
then `git checkout <hash>`, where `<hash>` is given in the payload. If the directory is dirty or any commands fail, the upgrade is aborted.
Debora then runs `go install` to install the new binary.

It is also possible to upgrade debora herself, by sending a payload with `upgrade\_debora:<hash>`. In that case, we switch to the debora 
directory (also hard coded), fetch, checkout, install. Finally we run `go install` on the app too, so the new debora changes take effect.

Since we want to also be able to upgrade debora, a new debora is created every time an application is upgraded. So if the upgrade and install is
successful, a new debora is started (PROC4), and told to watch the process id of PROC3 (our application). Once PROC4 is up and watching PROC3,
the old debora (PROC2) can kill PROC3 (originally her child), at which point PROC4 (new debora) will start a new instance of the application (PROC5), and PROC2
(the old debora) will terminate herself. Finally, we are left with PROC1 (window), PROC4 (new debora), and PROC5 (new application), and the cycle repeats.


# Notes 

We make every effort to avoid having the users open any extra ports, so we use existing, possibly outbound, connections from the p2p layer.

We do, however, require the developer to expose another port (for the authentication protocol, so as to not require more additions to the p2p protocol of the application)

Soon, we will allow the developer to pass a more general authentication closure through Debora's api to allow for more complex authentication schemes.

# HowTo

As a developer, first create a new key pair for your app with `debora -keygen <appname>`. Take the public key and hardcode it into your application's souce code.

In the beginning of the program, `debora.Add(key, src, app string)` should be called, where `key` is the public key generated in the previous step,
`src` is the repository's path (eg. `github.com/ebuchman/debora`), and `app` is the name of your app (used as a reference later - must not be empty).

Add a new message to the p2p protocol which when received calls `Call(remote, payload)`, where `remote` is the address of the peer who delivered the message and `payload` is the payload of bytes.

Now, include a `debora-dev` flag in your apps cli which when provided calls `debora.DebListenAndServe(appName string, port int, callFunc func(payload []byte))`, 
where `callFunc` is responsible for broadcasting the special debora message (with payload) to all peers. Do not modify the payload.
`DebListenAndServe` will start a little in-process http server which can be called with `debora call --commit <hash> <appname>`, 
triggering the special debora message broadcast, and asking all peers to upgrade.

If a client attempts to broadcast this message, it will fail as they (presumably) do not have the appropriate private key to pass the authentication (hmac) step.

Furthermore, new code is only installed from a location that is hardcoded in the source.

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
debora call --remote-port 56565 --commit <hash> example
```

where `<hash>` is a valid commit from this repo.

This will ping the in-process debora server running with the developer's version of the app, triggering it to broadcast the upgrade message to all connected peers.
You should see the original node receive the broadcast, initiate the handshake to authenticate the caller, and then upgrade, terminate, and restart.

And that's that!

Cheers, to Debora.
