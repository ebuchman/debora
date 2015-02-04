package debora

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

/*
	There are three debora servers:
	1. Client side daemon (stand alone process on client machine)
	2. Developer side in-process with app (waits for develoepr to trigger broadcast)
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
	log.Println("In Call!")
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

	// check if process is real
	// by sending it the 0 signal
	pid := reqObj.Pid
	log.Println("process id:", pid)
	var proc *os.Process
	if proc, err = CheckValidProcess(pid); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	key, ok := deb.debKeys[pid]
	log.Println("Key:", key)
	log.Println(deb.debKeys)
	if !ok {
		// TODO: respond (debora) unknown process id!
		http.Error(w, fmt.Sprintf("Unknown process id %d", pid), http.StatusInternalServerError)
		return
	}

	// handshake with developer:
	host := reqObj.Host
	log.Println("ready to handshake with", host)
	ok, err = handshake(key, host)
	log.Println("handshake:", ok, err)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		// TODO: respond signal from invalid dev
		log.Println("Signal from invalid developer")
		return
	}

	// the signal has been authenticated
	log.Println("the signal is authentic!")

	// TODO: upgrade the binary

	// terminate the process
	err = proc.Signal(os.Interrupt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = proc.Wait()
	if err != nil {
		// TODO: the process may be done now but some other error has
		// occured. We need to bring it back up!
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: restart process

}

/*
	2. Developer side in-process routes:
	- call: broadcast the upgrade message to all peers
*/

func (deb *DebMaster) call(w http.ResponseWriter, r *http.Request) {
	log.Println("Received call request on DebMaster")
	// read the request, unmarshal json
	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Println("Issuing broadcast")
	// broadcast the upgrade message to all the peers
	// the payload is json encoded RequestObj
	// but probably only the Host field is filled in
	deb.callFunc(payload)
}

/*
	3. Developer side call daemon routes:
	- handshake: decrypt the nonce and produce hmac
*/

func (deb *DeveloperDebora) handshake(w http.ResponseWriter, r *http.Request) {
	log.Println("Received handshake request from", r.RemoteAddr)
	// read the request, unmarshal json
	cipherText, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	plainText, err := Decrypt(deb.priv, cipherText)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// the plainText is the nonce
	// we sign it with itself
	mac := SignMAC(plainText, plainText)
	w.Write(mac)
}
