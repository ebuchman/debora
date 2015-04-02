package debora

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"path"
	"strconv"
	"syscall"
)

var logger *Logger

// get the user's home directory
func homeDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return usr.HomeDir
}

var ARGS = []string{}

func Logging(on bool) {
	if on {
		logger.level = 1
	} else {
		logger.level = 0
	}
}

// initalize the debora library by getting the root directory
// and the daemon's host address
func init() {

	// grab the arguments (so we can use them later, incase os.Args is modified)
	ARGS = append(ARGS, os.Args...)

	// initialize the logger
	logger = NewLogger(1, path.Base(ARGS[0]))

	// configure root dir location
	deboraDir := os.Getenv("DEBORA_ROOT")
	if deboraDir != "" {
		DeboraRoot = deboraDir
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

// Get host from file
func ResolveHost(app string) (string, error) {
	filename := path.Join(DeboraApps, app)
	if _, err := os.Stat(filename); err != nil {
		return "", nil
	}

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}

	// the file should simply contain the port
	return "localhost:" + string(b), nil
}

// Delete a file
func CleanHosts(app string) error {
	filename := path.Join(DeboraApps, app)
	return os.Remove(filename)
}

// Read port from file
func ReadPort(app string) (string, error) {
	filename := path.Join(DeboraApps, app)
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	// TODO
	return string(b), nil
}

// Write port to file
func WritePort(app, port string) error {
	filename := path.Join(DeboraApps, app)
	p := []byte(port)
	return ioutil.WriteFile(filename, p, 0600)
}

// Dead simple stupid convenient logger
type Logger struct {
	level int
	pid   int
	s     string
}

func NewLogger(level int, s string) *Logger {
	return &Logger{
		level: level,
		pid:   os.Getpid(),
		s:     s,
	}
}

func (l *Logger) Printf(f string, s ...interface{}) {
	if l.level > 0 {
		fmt.Printf("[ %d %s ] %s", l.pid, l.s, fmt.Sprintf(f, s...))

	}
}

func (l *Logger) Println(s ...interface{}) {
	if l.level > 0 {
		f := fmt.Sprintf("[ %d %s ] %s", l.pid, l.s, fmt.Sprint(s...))
		fmt.Println(f)
	}
}

func isHex(str string) bool {
	_, err := hex.DecodeString(str)
	return err == nil
}
