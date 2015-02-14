package debora

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"time"
)

/*
	Client side functions for sending requests to the local daemon
*/

// check if the debora server is running
func rpcIsDeboraRunning(host string) bool {
	_, err := RequestResponse(host, "ping", nil)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

// start the debrora server
// install if not present
// block until she starts
// spawn the app
func startDebora(app string, args []string) error {
	// if debora is not installed, install her
	//if _, err := os.Stat(DeboraBin); err != nil {
	if err := installDebora(); err != nil {
		return err
	}
	//}

	cmd := exec.Command(DeboraBin, "run", app)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	for {
		time.Sleep(time.Millisecond * 10)
		p := path.Join(DeboraApps, app)
		if _, err := os.Stat(p); err != nil {
			// loop until the new deb process
			// finds a port and writes it to file
			continue
		}

		b, err := ioutil.ReadFile(p)
		if err != nil {
			return err
		}
		host := "localhost:" + string(b)

		if rpcIsDeboraRunning(host) {
			if err := rpcStartApp(host, app, args); err != nil {
				return err
			}
			break
		}
	}
	return nil
}

// install the debora binary (server)
func installDebora() error {
	fmt.Println("Installing debora ...")
	cur, _ := os.Getwd()
	if err := os.Chdir(DeboraCmdPath); err != nil {
		return err
	}
	/*cmd := exec.Command("go", "get", "-d")
	if err := cmd.Run(); err != nil {
		return err
	}*/
	cmd := exec.Command("go", "install", "-v")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	os.Chdir(cur)
	return nil
}

// tell debora to start a new instance of us to be the app process
func rpcStartApp(host, app string, args []string) error {
	reqObj := RequestObj{
		Args: args,
		App:  app,
	}
	b, err := json.Marshal(reqObj)
	if err != nil {
		return err
	}
	_, err = RequestResponse(host, "start", b)
	return err
}

// add a process to debora
func rpcAdd(host, key, name, src string, pid int, args []string) error {
	reqObj := RequestObj{
		Key:  key,
		Pid:  pid,
		Args: args,
		App:  name,
		Src:  src,
		Host: host,
	}
	b, err := json.Marshal(reqObj)
	if err != nil {
		return err
	}
	_, err = RequestResponse(host, "add", b)
	return err
}

// initiate the debora call
func rpcCall(host, remote string, pid int) error {
	reqObj := RequestObj{
		Pid:  pid,
		Host: remote,
	}
	b, err := json.Marshal(reqObj)
	if err != nil {
		return err
	}
	_, err = RequestResponse(host, "call", b)
	return err
}

// check if the process is known to debora
func rpcKnownDeb(host string, pid int) bool {
	reqObj := RequestObj{Pid: pid}
	b, err := json.Marshal(reqObj)
	if err != nil {
		fmt.Println(err)
		return false
	}
	b, err = RequestResponse(host, "known", b)
	if err != nil {
		fmt.Println(err)
		return false
	}
	if b == nil || len(b) == 0 {
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
	fmt.Println("sending nonce to dev:", host)
	response, err := RequestResponse(host, "handshake", cipherText)
	if err != nil {
		return false, err
	}

	// the mac is simply done on the nonce itself
	// using the nonce as key and message
	ok := CheckMAC(nonce, response, nonce)
	return ok, nil
}
