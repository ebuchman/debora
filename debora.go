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
	"strconv"
)

var GoPath = os.Getenv("GOPATH")
var DeboraBin = path.Join(GoPath, "bin", "debora")
var DeboraSrcPath = path.Join(GoPath, "github.com", "ebuchman", "debora")
var DeboraCmdPath = path.Join(DeboraSrcPath, "cmd", "debora")
var DeboraHost = "localhost:56565"

func init() {
	debHost := os.Getenv("DEBORA_HOST")
	if debHost != "" {
		DeboraHost = debHost
	}
}

// Debra interface from caller is two functions:
// 	Add(key []byte) starts a new debora process or add a key to an existing one
//	Call(key []byte) calls the debora server and has her take down this process, update it, and restart it

// check if a debora already exists for this key
// record details of current process
// start a new process with http server
// tell the new process about ourselves
func Add(key []byte) error {
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

// check if the debora server is running
func isDeboraRunning() bool {
	_, err := requestResponse(DeboraHost, "ping", nil)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

// start the debrora server
// install if not present
func startDebora() error {
	// if debora is not installed, install her
	if _, err := os.Stat(DeboraCmdPath); err != nil {
		if err = installDebora(); err != nil {
			return err
		}
	}

	cmd := exec.Command(DeboraCmdPath)
	if err := cmd.Start(); err != nil {
		return err
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

func requestResponse(host, method string, body []byte) ([]byte, error) {
	req, err := http.NewRequest("POST", host+"/"+method, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Length", strconv.Itoa(len(body)))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode > 399 {
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, contents)
	}
	return contents, nil
}

func deboraAdd(key []byte, pid int) error {
	reqObj := RequestObj{key, pid}
	b, err := json.Marshal(reqObj)
	if err != nil {
		return err
	}
	_, err = requestResponse(DeboraHost, "add", b)
	return err
}

func callDebora(pid int) error {
	reqObj := RequestObj{nil, pid}
	b, err := json.Marshal(reqObj)
	if err != nil {
		return err
	}
	_, err = requestResponse(DeboraHost, "call", b)
	return err
}

func knownDeb(pid int) bool {
	reqObj := RequestObj{nil, pid}
	b, err := json.Marshal(reqObj)
	if err != nil {
		log.Println(err)
		return false
	}
	b, err = requestResponse(DeboraHost, "known", b)
	if err != nil {
		log.Println(err)
		return false
	}
	if b == nil {
		return false
	}
	return true
}
