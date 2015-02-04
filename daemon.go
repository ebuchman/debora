package debora

import (
	"crypto/rand"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"time"
)

/*
	Client side functions for sending requests to the local daemon
*/

// check if the debora server is running
func isDeboraRunning() bool {
	_, err := RequestResponse(DeboraHost, "ping", nil)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

// start the debrora server
// install if not present
// block until she starts
func startDebora() error {
	// if debora is not installed, install her
	if _, err := os.Stat(DeboraCmdPath); err != nil {
		if err = installDebora(); err != nil {
			return err
		}
	}

	cmd := exec.Command(DeboraCmdPath, "run")
	if err := cmd.Start(); err != nil {
		return err
	}
	for {
		time.Sleep(time.Second)
		if isDeboraRunning() {
			break
		}
	}
	return nil
}

// install the debora binary (server)
func installDebora() error {
	cur, _ := os.Getwd()
	os.Chdir(DeboraCmdPath)
	cmd := exec.Command("go", "get", "-d")
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("go", "install")
	if err := cmd.Run(); err != nil {
		return err
	}

	os.Chdir(cur)
	return nil
}

// add a process to debora
func deboraAdd(key, name string, pid int, args []string) error {
	reqObj := RequestObj{
		Key:  key,
		Pid:  pid,
		Args: args,
		App:  name,
		//Host: host,
	}
	b, err := json.Marshal(reqObj)
	if err != nil {
		return err
	}
	_, err = RequestResponse(DeboraHost, "add", b)
	return err
}

// initiate the debora call
func callDebora(pid int, host string) error {
	reqObj := RequestObj{
		Pid:  pid,
		Host: host,
	}
	b, err := json.Marshal(reqObj)
	if err != nil {
		return err
	}
	_, err = RequestResponse(DeboraHost, "call", b)
	return err
}

// check if a process is known to debora
func knownDeb(pid int) bool {
	reqObj := RequestObj{Pid: pid}
	b, err := json.Marshal(reqObj)
	if err != nil {
		log.Println(err)
		return false
	}
	b, err = RequestResponse(DeboraHost, "known", b)
	if err != nil {
		log.Println(err)
		return false
	}
	if b == nil {
		return false
	}
	return true
}

/*
	Functions run by the daemon to authenticate the developer and manage the processes
*/

// create random nonce, encrypt with public key
// send to developer, validate hmac response
func handshake(key, host string) (bool, error) {
	// generate nonce
	nonce := make([]byte, 32)
	_, err := rand.Read(nonce)
	if err != nil {
		return false, err
	}

	// encrypt nonce with developers public key
	cipherText, err := Encrypt(key, nonce)
	if err != nil {
		return false, err
	}

	// send encrypted nonce to developer
	response, err := RequestResponse(host, "handshake", cipherText)
	if err != nil {
		return false, err
	}

	// the mac is simply done on the nonce itself
	// using the nonce as key and message
	ok := CheckMAC(nonce, response, nonce)
	return ok, nil
}
