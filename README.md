Debora
------

A simple tool for managing forced updates by a developer in p2p applications.

# How it works

The simplest approach is to add a new `DeboraMsgTy` to your P2P protocol which receives a message from the developer
and initiates the upgrade sequence by calling `debora.Call(host string)`, where host is the developer's ip address. 
This will kill the running process, upgrade it, and restart it.

Note we make every effort to avoid having the users open any extra ports (so we use existing, possibly outbound, connections from the p2p layer)

As a developer, first create a new key pair for your app with `debora -keygen <appname>`.

In the beginning of the program, you should call `debora.Add(key []byte)`, where key is the public key generated in the previous step.
This will start the debora process on the client machine if it does not already exist and add the process id and provided key. 
Debora will only accept messages for this process signed by the appropriate key.

Now, include a `debora-dev` flag in your apps cli which when provided calls `debora.Master(appName string, callFunc func())`, 
where `callFunc` is responsible for broadcasting an empty `DeboraMsgTy` message to all peers. This will start a little in-process http server 
which can be called with `debora -call <appname>`, triggering the upgrade protocol in all connected peers. If a client attempts this, it will fail as they
(presumably) do not have the appropriate key). The developer's public key should be hard coded into the application's source code and provided in `debora.Add(key)`.

When a client receives the message and runs `debora.Call(host string)`, it will send a request to a local debora instance which will generate
a random nonce and send it to the developer's deborah (at the provided host address) to be signed. If the signature checks out (matches the public key), deborah will
shutdown the appropriate process, upgrade it with a `git pull` and `go install`, and restart it.


