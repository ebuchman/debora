package debora

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
)

var (
	// important paths
	HomeDir       = homeDir()
	GoPath        = os.Getenv("GOPATH")
	GoSrc         = path.Join(GoPath, "src")
	DeboraRoot    = path.Join(HomeDir, ".debora")
	DeboraApps    = path.Join(DeboraRoot, "apps")
	DeboraConfig  = path.Join(DeboraRoot, "config.json")
	DeboraBin     = path.Join(GoPath, "bin", "debora")
	DeboraSrcPath = path.Join(GoPath, "src", "github.com", "ebuchman", "debora")
	DeboraCmdPath = path.Join(DeboraSrcPath, "cmd", "debora")

	deboraHost string // host debora for this app process
)

// Debra interface from caller is two functions:
// 	Add(key, src, app, logfile string)
//		starts a new debora process or add a key to an existing one
//	Call(remote string, payload []byte)
//		calls the local debora server and has her take down this process, update it, and restart it

// Add the current process to debora's control table
// If the process was started by the user, no debora exists.
//  	Start one, and have it launch the app proper
// The calling app provides dev's public key, path to src, app name, and a directory for debora logs
// This function should be called as early as possible in the program
func Add(key, src, app, logfile string) error {
	host, err := ResolveHost(app)
	if err != nil {
		return err
	}

	logger.Printf("Resolve host for %s: %s\n", app, host)

	// if this is a new instance of the app
	// and there is no current debora,
	// start her and block forever.
	// debora will start a new instance of the app that doesn't block
	if host == "" {
		if err := startDebora(app, ARGS, -1); err != nil {
			return err
		}
		logger.Println("We started deb and she's running. Block forever")
		<-make(chan int)
		return nil
	}

	// if debora's not running,
	// a mistake was made, so cleanup and try again
	if !rpcIsDeboraRunning(host) {
		logger.Printf("Found bad host, cleaning file. %s: %s\n", app, host)
		if err := CleanHosts(app); err != nil {
			return err
		}
		return Add(key, src, app, logfile)
	}

	// set the global host variable for this process
	// so we can get it easily in Call
	deboraHost = host

	pid := os.Getpid()
	if rpcKnownDeb(host, pid) {
		return fmt.Errorf("The process has already been added to debora")
	}

	logger.Printf("The developers public key is %s", key)

	if err := rpcAdd(host, key, app, src, logfile, pid, ARGS); err != nil {
		return err
	}

	return nil
}

// Initiate sequence to upgrade and restart the current process
// Payload is json encoded ReqObj with Host field, which gives us the host's port
// but we need to use the knowledge of the p2p layer to get its ip address
// Call this function when the 'signal' is received from trusted developer
func Call(remoteHost string, payload []byte) error {
	localHost := deboraHost
	if !rpcIsDeboraRunning(localHost) {
		return fmt.Errorf("Debora is not running on this machine")
	}
	pid := os.Getpid()
	if !rpcKnownDeb(localHost, pid) {
		return fmt.Errorf("This process is not known to debora. Did you run Add first?")
	}

	var reqObj = RequestObj{}
	if err := json.Unmarshal(payload, &reqObj); err != nil {
		return err
	}

	// get port from address provided by developer
	_, port, err := net.SplitHostPort(reqObj.Host)
	if err != nil {
		return err
	}
	// get ip from address provided by caller
	ip, _, err := net.SplitHostPort(remoteHost)

	remoteHost = net.JoinHostPort(ip, port)

	return rpcCall(localHost, remoteHost, reqObj.Commit, pid)
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
// New port is granted by the operating system
// If a debora is already running for this application,
// this new debora will take over
func DeboraListenAndServe(app string) error {

	deb := new(Debora)

	mux := http.NewServeMux()
	mux.HandleFunc("/ping", deb.ping)
	mux.HandleFunc("/kill", deb.kill)
	mux.HandleFunc("/start", deb.start)
	mux.HandleFunc("/restart", deb.restart)
	mux.HandleFunc("/add", deb.add)
	mux.HandleFunc("/call", deb.call)
	mux.HandleFunc("/known", deb.known)

	// let the OS choose a port for us
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return err
	}
	// write port to file
	addr := ln.Addr().String()
	_, port, _ := net.SplitHostPort(addr)
	if err := WritePort(app, port); err != nil {
		return err
	}
	logger.Println("Debora listening on: ", addr)
	// Serve
	srv := &http.Server{Addr: addr, Handler: mux}
	return srv.Serve(ln)
}

// Spawn a new go routine to listen and serve http
// for this process. Responds to `debora call` issued
// by developer. This should run in-process with the app
// on the developer's machine
func DebListenAndServe(appName string, port int, callFunc func(payload []byte)) {
	deb := &DebMaster{
		callFunc: callFunc,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/call", deb.call)
	go func() {
		host := "localhost:" + strconv.Itoa(port)
		logger.Println("DebMaster listening on", host)
		if err := http.ListenAndServe(host, mux); err != nil {
			logger.Println("Error on deb master listen:", err)
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
	logger.Println("Developer debora listening on", host)
	if err := http.ListenAndServe(host, mux); err != nil {
		return err
	}
	return nil

}
