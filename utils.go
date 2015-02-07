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
		if err := os.Mkdir(DeboraRoot, 0700); err != nil {
			log.Fatal("Error making root dir:", err)
		}
	}
	// make apps dir
	if _, err := os.Stat(DeboraApps); err != nil {
		if err := os.Mkdir(DeboraApps, 0700); err != nil {
			log.Fatal("Error making apps dir:", err)
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
	host = "http://" + host
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

// Check if a process is running by sending it the 0 signal
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

// every debora writes its port to a file
// named after the app
func resolveHost(app string) (string, error) {
	var host string
	filename := path.Join(DeboraApps, app)
	if _, err := os.Stat(filename); err != nil {
		// if the file does not exist
		// we have to find an available port
		// and add the file
		if fs, err := ioutil.ReadDir(DeboraApps); err != nil {
			return "", err
		} else {
			portsTaken := make(map[string]bool)
			// make map of all ports
			for _, f := range fs {
				if b, err := ioutil.ReadFile(f.Name()); err != nil {
					return "", err
				} else {
					portsTaken[string(b)] = true
				}
			}
			// find unused port
			for i := 0; ; i++ {
				host = strconv.Itoa(StartPort + i)
				taken := portsTaken[host]
				if !taken {
					// we found an unused port
					if err := ioutil.WriteFile(filename, []byte(host), 0600); err != nil {
						return "", err
					}
					break
				}
			}
		}
	}

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}

	// the file should simply contain the port
	return "localhost:" + string(b), nil
}
