package main

import (
	//	"encoding/json"
	"flag"
	"fmt"
	"github.com/ebuchman/debora"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	deboraDev = flag.Bool("debora-dev", false, "allow debora management (dev side)")
	deboraM   = flag.Bool("debora", false, "allow remote developer to manage the process")
)

var (
	AppName   = "example"
	SrcPath   = "github.com/ebuchman/example"
	bootstrap = "0.0.0.0:8009" // developer's ip and port
	me        = "0.0.0.0:8010" // my ip and port
	CallPort  = 56565

	peers = make(map[string]string) // map from connected addr to listen addr
)

// initialize the app with keys
// this is a convenience function that in practice
// is executed only by the developer on their machine
func init() {
	debora.Logging(true)
	app := debora.App{
		Name:       AppName,
		PublicKey:  PublicKey,
		PrivateKey: PrivateKey,
	}
	debora.GlobalConfig.Apps["example"] = app
	debora.WriteConfig(debora.DeboraConfig)
}

func main() {
	flag.Parse()

	var (
		local  string
		remote string
	)

	// Non-developer agent
	// Adds current process to debora
	if *deboraM {
		fmt.Printf("%d: Adding proc to debora\n", os.Getpid())
		err := debora.Add(PublicKey, SrcPath, AppName)
		ifExit(err)
		local = me
		remote = bootstrap
	}

	// Developer agent
	// Listens for `debora call` command and broadcasts upgrade msg to connected peers
	if *deboraDev {
		fmt.Printf("%d: Running debora-dev server (listen to call))\n", os.Getpid())
		a := new(App)
		debora.DebListenAndServe("example", CallPort, a.broadcast)
		local = bootstrap
		remote = me
	}

	// Connect to bootstrap node and run the protocol

	fmt.Printf("%d: Run the protcol. Local %s, Remote %s\n", os.Getpid(), local, remote)
	runProtocol(local, remote)

}

type Peer struct {
	local  string
	remote string
}

// The example is a dead simple http protocol:
// Client contacts the bootstrap node and tells it its listen address.
// They repeatedly send eachother pings
// In a proper p2p protocol the dual http listeners would be replaced by a single persistent tcp connection
func runProtocol(local, remote string) {
	fmt.Println("Local Address:", local)
	fmt.Println("Remote Address:", remote)

	peer := &Peer{
		local:  local,
		remote: remote,
	}

	// connect to bootstrap node
	go func() {
		for {
			fmt.Println("connecting to:", remote)
			// send our listen address
			_, err := debora.RequestResponse(remote, "", []byte(local))
			if err != nil {
				fmt.Println(err)
			}
			time.Sleep(time.Second)
		}
	}()

	// listen for messages
	mux := http.NewServeMux()
	mux.HandleFunc("/", peer.handler)
	mux.HandleFunc("/debora", peer.deboraHandler)
	fmt.Println("Example protocol listening on", local)
	if err := http.ListenAndServe(local, mux); err != nil {
		log.Fatal(err)
	}
}

// handle pings and add peer to table
func (peer *Peer) handler(w http.ResponseWriter, r *http.Request) {
	caller := r.RemoteAddr
	listenAddr, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Println("received hi from: ", caller)
	peers[caller] = string(listenAddr)
}

// handle debora call (initiate upgrade process)
// ie. MsgDeboraTy
func (peer *Peer) deboraHandler(w http.ResponseWriter, r *http.Request) {
	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Println("got the call")
	if err := debora.Call(peer.remote, payload); err != nil {
		fmt.Println(err)
	}

}

type App struct {
}

// called by the developer's DebMaster when triggered by `debora call`
func (a *App) broadcast(payload []byte) {
	fmt.Println("Broadcast!")
	// broadcast MsgDeboraTy message with payload to all peers
	for conAddr, listenAddr := range peers {
		/*reqObj := debora.RequestObj{
			Host: bootstrap,
		}*/
		//b, _ := json.Marshal(reqObj)
		fmt.Printf("attempting broadcast to %s at %s\n", conAddr, listenAddr)
		// send MsgDeboraTy
		b, err := debora.RequestResponse(listenAddr, "debora", payload)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println("call response:", string(b))
	}
}

func ifExit(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
