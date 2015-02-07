package debora

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
)

var (
	// important paths
	HomeDir       = homeDir()
	GoPath        = os.Getenv("GOPATH")
	DeboraRoot    = path.Join(HomeDir, ".debora")
	DeboraApps    = path.Join(DeboraRoot, "apps")
	DeboraConfig  = path.Join(DeboraRoot, "config.json")
	DeboraBin     = path.Join(GoPath, "bin", "debora")
	DeboraSrcPath = path.Join(GoPath, "src", "github.com", "ebuchman", "debora")
	DeboraCmdPath = path.Join(DeboraSrcPath, "cmd", "debora")

	DeboraHost          = "localhost:56565" // local debora daemon
	DeveloperDeboraHost = "0.0.0.0:8009"    // developer's debora for this app
	DebMasterHost       = "localhost:56566" // developer's debora in process with app

	deboraHost string // host debora for this app process
	StartPort  = 56565
)

// Debra interface from caller is two functions:
// 	Add(key, src string) starts a new debora process or add a key to an existing one
//	Call(payload []byte) calls the debora server and has her take down this process, update it, and restart it

// Add the current process to debora's control table
// The only thing provided by the calling app is the developers public key
// If debora is not running, start her.
// Call this function early on in the program
func Add(key, src, app string) error {
	host, err := resolveHost(app)
	if err != nil {
		return err
	}

	if !rpcIsDeboraRunning(host) {
		// XXX: blocks until debora starts
		if err := startDebora(host, app); err != nil {
			return err
		}
	}

	// set the global variable host for this process
	// so we can get it easily in Call
	deboraHost = host

	pid := os.Getpid()
	if rpcKnownDeb(host, pid) {
		return fmt.Errorf("The process has already been added to debora")
	}

	tty := getTty()
	if err := rpcAdd(host, key, ARGS[0], src, tty, pid, ARGS); err != nil {
		return err
	}

	return nil
}

// Initiate sequence to upgrade and restart the current process
// Payload is json encoded ReqObj with Host field
// Call this function when the 'signal' is received from trusted developer
func Call(payload []byte) error {
	host := deboraHost
	if !rpcIsDeboraRunning(host) {
		return fmt.Errorf("Debora is not running on this machine")
	}
	pid := os.Getpid()
	if !rpcKnownDeb(host, pid) {
		return fmt.Errorf("This process is not known to debora. Did you run Add first?")
	}

	var reqObj = RequestObj{}
	if err := json.Unmarshal(payload, &reqObj); err != nil {
		return err
	}
	// developer's address
	remote := reqObj.Host
	return rpcCall(host, remote, pid)
}

/*
	There are three debora servers:
	1. Client side daemon (stand alone process on client machine)
	2. Developer side in-process (wait for develoepr to trigger broadcast)
	3. Developer side call daemon (communicates with clients once they have begun the call sequence)

	These are their ListenAndServes
*/

// Main debora daemon.
// It should be run by the new debora process
// and never by another application.
// This function blocks.
func DeboraListenAndServe(app string) error {
	deb := &Debora{
		debs:  make(map[int]RequestObj),
		names: make(map[string]int),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", deb.ping)
	mux.HandleFunc("/kill", deb.kill)
	mux.HandleFunc("/add", deb.add)
	mux.HandleFunc("/call", deb.call)
	mux.HandleFunc("/known", deb.known)
	log.Println("Debora listening on:", DeboraHost)
	if err := http.ListenAndServe(DeboraHost, mux); err != nil {
		return err
	}
	return nil
}

// Spawn a new go routine to listen and serve http
// for this process. Responds to `debora call` issued
// by developer. This should run in-process with the app
// on the developer's machine
func DebMasterListenAndServe(appName string, callFunc func(payload []byte)) {
	deb := &DebMaster{
		callFunc: callFunc,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/call", deb.call)
	go func() {
		log.Println("DebMaster listening on", DebMasterHost)
		if err := http.ListenAndServe(DebMasterHost, mux); err != nil {
			log.Println("Error on deb master listen:", err)
		}
	}()
}

// Run a temporary server on the developer's machine to respond to
// authentication requests from clients
// Started by `debora call`.
func DeveloperListenAndServe(host, priv string) error {
	deb := DeveloperDebora{
		priv: priv,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/handshake", deb.handshake)
	log.Println("Developer debora listening on", host)
	if err := http.ListenAndServe(host, mux); err != nil {
		return err
	}
	return nil

}
