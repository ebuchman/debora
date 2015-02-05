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
	"os/user"
	"path"
	"strconv"
	"strings"
	"syscall"
)

// get the user's home directory
func homeDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return usr.HomeDir
}

var ARGS = []string{}

// initalize the debora library by getting the root directory
// and the daemon's host address
func init() {
	// grab the arguments (so we can use them later, incase os.Args is modified)
	ARGS = append(ARGS, os.Args...)

	// configure root dir location
	deboraDir := os.Getenv("DEBORA_ROOT")
	if deboraDir != "" {
		DeboraRoot = deboraDir
	}

	// configure daemon address
	debHost := os.Getenv("DEBORA_HOST")
	if debHost != "" {
		DeboraHost = debHost
	}

	// make root dir
	if _, err := os.Stat(DeboraRoot); err != nil {
		err := os.Mkdir(path.Join(HomeDir, ".debora"), 0700)
		if err != nil {
			log.Fatal("Error making root dir:", err)
		}
	}

	// make or load config file
	configFile := DeboraConfig
	if _, err := os.Stat(configFile); err != nil {
		if err := WriteConfig(configFile); err != nil {
			log.Fatal("Write config err:", err)
		}
	} else {
		if err := LoadConfig(configFile); err != nil {
			log.Fatal("Load config err:", err)
		}
	}
}

// Write the global config struct to file
func WriteConfig(configFile string) error {
	b, err := json.Marshal(GlobalConfig)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(configFile, b, 0600)
	if err != nil {
		return err
	}
	return nil
}

// Load the global config struct from file
func LoadConfig(configFile string) error {
	b, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &GlobalConfig)
	if err != nil {
		return err
	}
	return nil
}

// http json request and response
func RequestResponse(host, method string, body []byte) ([]byte, error) {
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

// Check is a process is running
func CheckValidProcess(pid int) (*os.Process, error) {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil, err
	}
	err = proc.Signal(syscall.Signal(0))
	if err != nil {
		return nil, fmt.Errorf("Invalid process id %d", pid)
	}
	return proc, nil
}

// returns device path
func getTty() string {
	pid := os.Getpid()
	cmd1 := exec.Command("ps")
	cmd2 := exec.Command("grep", strconv.Itoa(pid))
	cmd3 := exec.Command("grep", "-v", "grep")
	cmd4 := exec.Command("awk", "{print $2}")
	//cmd4 := exec.Command("awk")

	cmd2.Stdin, _ = cmd1.StdoutPipe()
	cmd3.Stdin, _ = cmd2.StdoutPipe()
	//      cmd3.Stdout = os.Stdout
	cmd4.Stdin, _ = cmd3.StdoutPipe()
	buf := bytes.NewBuffer([]byte{})
	cmd4.Stdout = buf
	cmd4.Start()
	cmd3.Start()
	cmd2.Start()
	cmd1.Run()
	cmd2.Wait()
	cmd3.Wait()
	cmd4.Wait()

	b := buf.Bytes()
	b = b[:len(b)-1]

	device := path.Join("/dev", strings.TrimSpace(string(b)))
	return device
}
