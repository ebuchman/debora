package debora

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
)

/*
	There are three debora servers:
	1. Client side daemon (stand alone process on client machine. One per app)
	2. Developer side in-process with app (waits for develoepr to trigger broadcast)
	3. Developer side call daemon (communicates with clients once they have begun the call sequence)
*/

/*
	1. Client side daemon routes:
	- ping: is the server up
	- kill: kill the debora process
	- add: add an app process to the local debora
	- call: take down, upgrade, and restart calling process
	- known: is this app known to debora
*/

// Check if debora server is running
func (deb *Debora) ping(w http.ResponseWriter, r *http.Request) {
	// I'm awake!
}

// TODO: secure this!
func (deb *Debora) kill(w http.ResponseWriter, r *http.Request) {
	log.Fatal("Goodbye")
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

	// check if process is real
	pid := reqObj.Pid
	if _, err = CheckValidProcess(pid); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: validate key length

	deb.debs[pid] = reqObj
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
	if _, ok := deb.debs[reqObj.Pid]; ok {
		// this need only be not nil or len 0
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
	pid := reqObj.Pid
	log.Println("process id:", pid)
	var proc *os.Process
	if proc, err = CheckValidProcess(pid); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	obj, ok := deb.debs[pid]
	key := obj.Key
	if !ok {
		http.Error(w, fmt.Sprintf("Unknown process id %d", pid), http.StatusInternalServerError)
		return
	}

	// handshake with developer
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

	log.Println("the signal is authentic!")
	log.Println("upgrading the binary")

	// upgrade the binary
	cur, _ := os.Getwd()
	srcDir := path.Join(GoPath, "src", obj.Src)
	if err := os.Chdir(srcDir); err != nil {
		log.Println("bad dir", err)
		http.Error(w, fmt.Sprintf("bad directory %s", srcDir), http.StatusInternalServerError)
		return
	}
	if err := upgradeRepo(obj.Src); err != nil {
		log.Println("err on upgrade", err)
		http.Error(w, fmt.Sprintf("error on upgrade %s", err.Error()), http.StatusInternalServerError)
		return
	}
	if err := installRepo(obj.Src); err != nil {
		log.Println("install err", err)
		http.Error(w, fmt.Sprintf("error on repo install %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// TODO: don't use tty. just enforce one debora daemon per process and use the os.Stdout/in for both
	/*	tty := obj.Tty
		f, err := os.OpenFile(tty, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			//TODO: can't open terminal device but still need to restart process!
			log.Println("Error opening device:", err)
		}*/

	// TODO: track the dir the original program was run in and use that!
	// Also, can we get it's stdout?!
	os.Chdir(cur)

	/*var ch chan int
	go func() {
		log.Println("waiting for shutdown")
		if _, err = proc.Wait(); err != nil {
			log.Println("err on wait:", err)
			// TODO: the process may be done now but some other error has
			// occured. We need to bring it back up!
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ch <- 0

	}()*/

	log.Println("terminating the process")
	// terminate the process
	err = proc.Signal(os.Interrupt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//<-ch

	log.Println("restarting process")
	// restart process
	prgm := obj.Args[0]
	var args []string
	if len(obj.Args) < 2 {
		args = []string{}
	} else {
		args = obj.Args[1:]
	}
	log.Println("Program:", prgm)
	log.Println("args:", args)
	cmd := exec.Command(prgm, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Println("err on start:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func upgradeRepo(src string) error {
	cmd := exec.Command("git", "stash")
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("git", "pull", "origin", "master")
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func installRepo(src string) error {
	cmd := exec.Command("go", "install")
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
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
