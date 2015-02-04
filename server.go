package debora

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

/*
	There are three debora servers:
	1. Client side daemon (stand alone process on client machine)
	2. Developer side in-process (wait for develoepr to trigger broadcast)
	3. Developer side call daemon (communicates with clients once they have begun the call sequence)
*/

/*
	1. Client side daemon routes:
	- ping: is the server up
	- add: add a new app/process to the local debora
	- call: take down, upgrade, and restart calling process
	- known: is this app known to debora
*/

// Check if debora server is running
func (deb *Debora) ping(w http.ResponseWriter, r *http.Request) {
	// I'm awake!
}

// Add a new process to debora
func (deb *Debora) add(w http.ResponseWriter, r *http.Request) {
	// read the request, unmarshal json
	p, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	var reqObj = RequestObj{}
	err = json.Unmarshal(p, &reqObj)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	// TODO: check if pid corresponds to real process
	// TODO: check key is appropriate length
	deb.debKeys[reqObj.Pid] = reqObj.Key
}

// Find out if a process is known to debora
func (deb *Debora) known(w http.ResponseWriter, r *http.Request) {
	// read the request, unmarshal json
	p, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	var reqObj = RequestObj{}
	err = json.Unmarshal(p, &reqObj)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if _, ok := deb.debKeys[reqObj.Pid]; ok {
		w.Write([]byte("ok"))
	}
}

// Call debora to take down a process, upgrade it, and restart
func (deb *Debora) call(w http.ResponseWriter, r *http.Request) {
	// read the request, unmarshal json
	p, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var reqObj = RequestObj{}
	err = json.Unmarshal(p, &reqObj)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// TODO: check pid is real process
	key, ok := deb.debKeys[reqObj.Pid]
	if !ok {
		// TODO: respond unknown process id!
	}

	// handshake with developer:
	var host string //todo
	ok, err = handshake(key, host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		log.Println("Signal from invalid developer")
		return
	}
	// the signal has been authenticated
	// terminate the process

	// kill process
	// git pull and go install
	// restart process

}

/*
	2. Developer side in-process routes:
	- call: broadcast the upgrade message to all peers
*/

func (deb *DebMaster) call(w http.ResponseWriter, r *http.Request) {
	// broadcast the upgrade message to all the peers
	payload := []byte("A") // TODO
	deb.callFunc(payload)
}

/*
	3. Developer side call daemon routes:
	-
*/
