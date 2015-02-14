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

func (deb *Debora) start(w http.ResponseWriter, r *http.Request) {
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

	if len(reqObj.Args) == 0 {
		http.Error(w, "Bad Request", http.StatusInternalServerError)
	}
	args := reqObj.Args
	prgm := args[0]
	if len(args) > 1 {
		args = args[1:]
	} else {
		args = []string{}
	}

	cmd := exec.Command(prgm, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

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

	deb.deb = reqObj
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
	if deb.deb.Pid == reqObj.Pid {
		// this need only be not nil or len 0
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

	// check if process is real
	pid := reqObj.Pid
	var proc *os.Process
	if proc, err = CheckValidProcess(pid); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	obj := deb.deb
	if obj.Pid != pid {
		http.Error(w, fmt.Sprintf("Unknown process id %d", pid), http.StatusInternalServerError)
		return
	}
	key := obj.Key

	// handshake with developer
	host := reqObj.Host
	logger.Println("ready to handshake with", host)
	ok, err := handshake(key, host)
	logger.Println("handshake:", ok, err)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		// TODO: respond signal from invalid dev
		logger.Println("Signal from invalid developer")
		return
	}

	logger.Println("the signal is authentic!")
	logger.Println("upgrading the binary")

	// upgrade the binary
	cur, _ := os.Getwd()
	srcDir := path.Join(GoPath, "src", obj.Src)
	if err := os.Chdir(srcDir); err != nil {
		logger.Println("bad dir", err)
		http.Error(w, fmt.Sprintf("bad directory %s", srcDir), http.StatusInternalServerError)
		return
	}
	if err := upgradeRepo(obj.Src); err != nil {
		logger.Println("err on upgrade", err)
		http.Error(w, fmt.Sprintf("error on upgrade %s", err.Error()), http.StatusInternalServerError)
		return
	}
	if err := installRepo(obj.Src); err != nil {
		logger.Println("install err", err)
		http.Error(w, fmt.Sprintf("error on repo install %s", err.Error()), http.StatusInternalServerError)
		return
	}

	os.Chdir(cur)

	logger.Println("terminating the process")
	// terminate the process
	err = proc.Signal(os.Interrupt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//<-ch

	logger.Println("restarting process")
	// restart process
	prgm := obj.Args[0]
	var args []string
	if len(obj.Args) < 2 {
		args = []string{}
	} else {
		args = obj.Args[1:]
	}
	logger.Println("Program:", prgm)
	logger.Println("args:", args)
	cmd := exec.Command(prgm, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		logger.Println("err on start:", err)
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
	logger.Println("Received call request on DebMaster")
	// read the request, unmarshal json
	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logger.Println("Issuing broadcast")
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
	logger.Println("Received handshake request from", r.RemoteAddr)
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
