package debora

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
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

func (deb *Debora) restart(w http.ResponseWriter, r *http.Request) {
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

	if len(reqObj.Args) == 0 {
		http.Error(w, "Bad Request", http.StatusInternalServerError)
		return
	}

	fmt.Printf(".%d\n", reqObj.Pid)
	_, err = CheckValidProcess(reqObj.Pid)
	if err != nil {
		http.Error(w, fmt.Sprintf("Cannot find process id %d", reqObj.Pid), http.StatusInternalServerError)
	}

	// spin up a goroutine to watch the pid.
	// when the proc dies, restart it.
	go func() {
		// watch the process until it dies
		fmt.Println("Watching process", reqObj.Pid)
		for {
			time.Sleep(10 * time.Millisecond)
			_, err := CheckValidProcess(reqObj.Pid)
			if err != nil {
				break
			}
		}

		// restart process
		prgm := reqObj.Args[0]
		var args []string
		if len(reqObj.Args) < 2 {
			args = []string{}
		} else {
			args = reqObj.Args[1:]
		}
		deb.Logf(fmt.Sprintln("Restarting process:", prgm, args))
		cmd := exec.Command(prgm, args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			deb.Logf(fmt.Sprintln("Restart error:", err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		deb.Logln("Process successfully restarted")
	}()

	// this function returning without error
	// indicates to the parent that we're watching
	// the process, she can kill it
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

	// create log file if doesn't exist
	if _, err := os.Stat(deb.LogFile()); err != nil {
		os.Create(deb.LogFile())
	}
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

	// our local debora info
	obj := deb.deb

	// check if process is real
	var proc *os.Process
	if proc, err = CheckValidProcess(reqObj.Pid); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	pid := reqObj.Pid
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

	// anything after this point until the restart ought to
	// be logged to file
	commitHash := reqObj.Commit
	deb.Logf(fmt.Sprintf("The signal from %s is authentic\n", "DEV"))
	deb.Logf(fmt.Sprintf("Upgrading the binary to commit %s\n", commitHash))

	// fetch and checkout the updates
	objSrc := path.Join(GoSrc, obj.Src)
	if err := deb.upgradeCall(objSrc, commitHash); err != nil {
		deb.Logf(fmt.Sprintln("Upgrade error:", err))
		http.Error(w, fmt.Sprintf("error on upgrade %s", err.Error()), http.StatusInternalServerError)
		return
	}
	// upgrade the binary
	if err := deb.installRepo(objSrc); err != nil {
		deb.Logf(fmt.Sprintln("Tnstall error:", err))
		http.Error(w, fmt.Sprintf("error on repo install %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// The app (and possibly debora herself) have been upgraded
	// Now we bring up a new debora, to take over from ourselves.
	// Once we confirm she is up, we let her terminate and restart the app.
	// Once the app is restarted, this process can safely die, and the
	// new debora takes over

	// start the new debora process
	// and give it the pid of the app that's being reset.
	// blocks until the new process is up
	fmt.Println("STARTING NEW DEBORA")
	if err := startDebora(obj.App, obj.Args, obj.Pid); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// if startDebora returned successfully,
	// the new debora is watching the pid.
	// let's kill it, and let new debora restart it.
	deb.Logln("Terminating the process")
	// terminate the process
	err = proc.Signal(os.Interrupt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: poll new debora

	logger.Println("This debora process has been replaced by a new one")
	logger.Println("Goodbye!")
	os.Exit(0)
}

// expects the full path to the source directory and a valid hash.
// change to that directory
// exits if the directory is dirty.
// git fetch -a origin
// git checkout hash
func (deb *Debora) upgradeRepo(src, hash string) error {
	cur, err := os.Getwd()
	if err != nil {
		return err // this one is annoying ..
	}
	if err := os.Chdir(src); err != nil {
		return fmt.Errorf("Bad directory: %s", src)
	}
	defer os.Chdir(cur)

	// if the directory is dirty, abort upgrade
	cmd := exec.Command("git", "diff-files", "--quiet")
	if err := cmd.Run(); err != nil {
		errStr := "Working tree is dirty. Aborting upgrade."
		deb.Logln(errStr)
		return fmt.Errorf(errStr)
	}

	// fetch all remote updates
	buf := new(bytes.Buffer)
	cmd = exec.Command("git", "fetch", "-a", "origin")
	cmd.Stdout = buf
	cmd.Stderr = buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Git fetch error: %s", err.Error())
	}
	deb.Logf(string(buf.Bytes()))

	// chceckout the provided hash
	buf = new(bytes.Buffer)
	cmd = exec.Command("git", "checkout", hash)
	cmd.Stdout = buf
	cmd.Stderr = buf
	if err := cmd.Run(); err != nil {
		deb.Logf(string(buf.Bytes()))
		return fmt.Errorf("Git checkout error: %s", err.Error())
	}
	return nil
}

// src should be the full path
// upgradeCall will either upgrade the dir at objSrc or
// upgrade and install debora, depending on `hash`
func (deb *Debora) upgradeCall(src, hash string) error {

	// the hash may contain more information
	// if its just a hash, go to src, fetch, and checkout hash
	// else we its "debora_upgrade:hash", and we should upgrade
	// debora
	spl := strings.Split(hash, ":")
	switch len(spl) {
	case 1:
		if !isHex(hash) {
			return fmt.Errorf("Provided hash is not valid hex: %s", hash)
		}
		// its just a hash, git fetch and checkout
		return deb.upgradeRepo(src, hash)
	case 2:
		// its a directive and a hash
		cmd := spl[0]
		hash := spl[1]
		if !isHex(hash) {
			return fmt.Errorf("Provided hash is not valid hex: %s", hash)
		}
		// for now the only other thing we do is upgrade debora
		// and rebuild the app
		_ = cmd
		err := deb.upgradeRepo(DeboraCmdPath, hash)
		if err != nil {
			return err
		}
		return deb.installRepo(DeboraCmdPath)
	default:
		return fmt.Errorf("Unknown upgrade directive: %s", hash)
	}
}

// src is a full path
// cd to src, `go install`
func (deb *Debora) installRepo(src string) error {
	cur, err := os.Getwd()
	if err != nil {
		return err // this one is annoying ..
	}
	if err := os.Chdir(src); err != nil {
		return fmt.Errorf("Bad directory: %s", src)
	}
	defer os.Chdir(cur)

	buf := new(bytes.Buffer)
	cmd := exec.Command("go", "install")
	cmd.Stdout = buf
	cmd.Stderr = buf
	if err := cmd.Run(); err != nil {
		return err
	}
	deb.Logf(string(buf.Bytes()))
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
