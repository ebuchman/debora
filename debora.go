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
	HomeDir             = homeDir()
	GoPath              = os.Getenv("GOPATH")
	DeboraRoot          = path.Join(HomeDir, ".debora")
	DeboraBin           = path.Join(GoPath, "bin", "debora")
	DeboraSrcPath       = path.Join(GoPath, "src", "github.com", "ebuchman", "debora")
	DeboraCmdPath       = path.Join(DeboraSrcPath, "cmd", "debora")
	DeboraHost          = "localhost:56565" // local debora daemon
	DeveloperDeboraHost = "0.0.0.0:8009"    // developer's debora for this app
	DebMasterHost       = "localhost:56567" // developer's debora in process with app
)

// Debra interface from caller is two functions:
// 	Add(key []byte) starts a new debora process or add a key to an existing one
//	Call() calls the debora server and has her take down this process, update it, and restart it

// Add the current process to debora's control table
// The only thing provided by the calling app is the developers public key
// If debora is not running, start her.
func Add(key string) error {
	if !isDeboraRunning() {
		// blocks until debora starts
		if err := startDebora(); err != nil {
			return err
		}
	}

	pid := os.Getpid()
	if !knownDeb(pid) {
		if err := deboraAdd(key, ARGS[0], pid, ARGS); err != nil {
			return err
		}
	}
	return nil
}

// Initiate sequence to upgrade and restart the current process
func Call(payload []byte) error {
	if !isDeboraRunning() {
		return fmt.Errorf("Debora is not running on this machine")
	}
	pid := os.Getpid()
	if !knownDeb(pid) {
		return fmt.Errorf("Unknown key")
	}

	var reqObj = RequestObj{}
	if err := json.Unmarshal(payload, &reqObj); err != nil {
		return err
	}
	host := reqObj.Host
	return callDebora(pid, host)
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
func DeboraListenAndServe() error {
	deb := &Debora{
		debKeys: make(map[int]string),
		debIds:  make(map[string]int),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", deb.ping)
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
func DeveloperListenAndServe(host string) error {
	deb := DeveloperDebora{}

	mux := http.NewServeMux()
	mux.HandleFunc("/handshake", deb.handshake)
	log.Println("Developer debora listening on", host)
	if err := http.ListenAndServe(host, mux); err != nil {
		return err
	}
	return nil

}
