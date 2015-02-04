package debora

import (
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
	DeboraSrcPath       = path.Join(GoPath, "github.com", "ebuchman", "debora")
	DeboraCmdPath       = path.Join(DeboraSrcPath, "cmd", "debora")
	DeboraHost          = "localhost:56565" // local debora daemon
	DeveloperDeboraHost = "0.0.0.0:8009"    // developer's debora for this app
	DebMasterHost       = "localhost:56567" // developer's debora in process with app
)

// Debra interface from caller is two functions:
// 	Add(key []byte) starts a new debora process or add a key to an existing one
//	Call() calls the debora server and has her take down this process, update it, and restart it

// check if a debora already exists for this key
// record details of current process
// start a new process with http server
// tell the new process about ourselves
func Add(key string) error {
	if !isDeboraRunning() {
		// blocks until debora starts
		if err := startDebora(); err != nil {
			return err
		}

	}

	pid := os.Getpid()
	if !knownDeb(pid) {
		if err := deboraAdd(key, pid); err != nil {
			return err
		}
	}
	return nil
}

// request a signed nonce
// git pull and install the latest
// take down the process
// start it up again
func Call() error {
	if !isDeboraRunning() {
		return fmt.Errorf("Debora is not running on this machine")
	}
	pid := os.Getpid()
	if !knownDeb(pid) {
		return fmt.Errorf("Unknown key")
	}
	return callDebora(pid)
}

/*
	There are three debora servers:
	1. Client side daemon (stand alone process on client machine)
	2. Developer side in-process (wait for develoepr to trigger broadcast)
	3. Developer side call daemon (communicates with clients once they have begun the call sequence)
*/

// This function blocks. It's the main debora daemon
// It should be run by the new debora process
// and never by another application
func ListenAndServe() error {
	deb := &Debora{
		debKeys: make(map[int]string),
		debIds:  make(map[string]int),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", deb.ping)
	mux.HandleFunc("/add", deb.add)
	mux.HandleFunc("/call", deb.call)
	mux.HandleFunc("/known", deb.known)
	if err := http.ListenAndServe(DeboraHost, mux); err != nil {
		return err
	}
	return nil
}

// Spawn a new go routine to listen and serve http
// for this process. Responds to `debora -call` issued
// by developer
func Master(appName string, callFunc func(payload []byte)) {
	deb := &DebMaster{
		callFunc: callFunc,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/call", deb.call)
	go func() {
		if err := http.ListenAndServe(DebMasterHost, mux); err != nil {
			log.Println("Error on deb master listen:", err)
		}
	}()
}
