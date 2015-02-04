package main

import (
	"encoding/json"
	"flag"
	"github.com/ebuchman/debora"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

var (
	deboraDev = flag.Bool("debora-dev", false, "allow debora management (dev side)")
	deboraM   = flag.Bool("debora", false, "allow remote developer to manage the process")
)

var (
	bootstrap = "0.0.0.0:8009" // developer's ip and port
	me        = "0.0.0.0:8010" // my ip and port

	peers = make(map[string]string) // map from connected addr to listen addr
)

func main() {
	flag.Parse()

	var (
		local  string
		remote string
	)

	// Non-developer agent
	// Adds current process to debora
	if *deboraM {
		err := debora.Add(PublicKey)
		ifExit(err)
		local = me
		remote = bootstrap
	}

	// Developer agent
	// Listens for `debora call` command and broadcasts upgrade msg to connected peers
	if *deboraDev {
		debora.DebMasterListenAndServe("example", broadcast)
		local = bootstrap
		remote = me
	}

	// Connect to bootstrap node and run the protocol
	runProtocol(local, remote)

}

// The example is a dead simple http protocol:
// Client contacts the bootstrap node and tells it its listen address.
// They repeatedly send eachother pings
// In a proper p2p protocol the dual http listeners would be replaced by a single persistent tcp connection
func runProtocol(local, remote string) {
	log.Println("Local Address:", local)
	log.Println("Remote Address:", remote)

	// connect to bootstrap node
	go func() {
		for {
			log.Println("connecting to:", remote)
			// send our listen address
			_, err := debora.RequestResponse("http://"+remote, "", []byte(local))
			if err != nil {
				log.Println(err)
			}
			time.Sleep(time.Second)
		}
	}()

	// listen for messages
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)
	mux.HandleFunc("/debora", deboraHandler)
	log.Println("Example protocol listening on", local)
	if err := http.ListenAndServe(local, mux); err != nil {
		log.Fatal(err)
	}
}

// handle pings and add peer to table
func handler(w http.ResponseWriter, r *http.Request) {
	caller := r.RemoteAddr
	listenAddr, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Println("received hi from: ", caller)
	peers[caller] = string(listenAddr)
}

// handle debora call (initiate upgrade process)
// ie. MsgDeboraTy
func deboraHandler(w http.ResponseWriter, r *http.Request) {
	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Println("got the call")
	debora.Call(payload)
}

// called by the developer's DebMaster when triggered by `debora call`
func broadcast(payload []byte) {
	// broadcast MsgDeboraTy message with payload to all peers
	for conAddr, listenAddr := range peers {
		reqObj := debora.RequestObj{
			Host: bootstrap,
		}
		b, _ := json.Marshal(reqObj)
		log.Printf("attempting broadcast to %s at %s\n", conAddr, listenAddr)
		// send MsgDeboraTy
		debora.RequestResponse("http://"+listenAddr, "debora", b)
	}
}

func ifExit(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
