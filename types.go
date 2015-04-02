package debora

import (
	"os"
)

// Debora daemon's main object for tracking processes and their developer's keys
type Debora struct {
	deb RequestObj
}

// DebMaster is the debora client within the
// developer's instance of the application process
// It's responsible for broadcasting the upgrade message to all peers
// and should only run on the developer's machine
type DebMaster struct {
	callFunc func(payload []byte)
}

// DebeloperDebora runs the server that responds to
// authentication requests from clients. It is short lived, spawned by
// `debora call` on the developer's machine, and killed by the developer
type DeveloperDebora struct {
	priv string
}

// For communicating with the debora daemon
// The same object is used for all communication
// and for representing processes/apps.
// So most of it is usually empty.
type RequestObj struct {
	Key     string   `json:",omitempty"` // hex encoded DER public key
	Pid     int      `json:",omitempty"` // process id
	Args    []string `json:",omitempty"` // command line call that started the process
	App     string   `json:",omitempty"` // process name
	Src     string   `json:",omitempty"` // install dir (cd to this before running git fetch. run `go install` from here)
	Commit  string   `json:",omitempty"` // commit hash to fetch (this can also be other trigger words, eg. to update debora herself)
	Host    string   `json:",omitempty"` // bootstrap node (developer's ip:port)
	LogFile string   `json:",omitempty"` // directory to store upgrade logs
}

type Config struct {
	Apps map[string]App
}

type App struct {
	Name       string `json:"name"`
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}

var GlobalConfig = Config{
	Apps: make(map[string]App),
}

// Simple log to file interface

func (d *Debora) LogFile() string {
	return d.deb.LogFile
}

func appendFile(file, text string) error {
	f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err = f.WriteString(text); err != nil {
		return err
	}
	return nil
}

func (d *Debora) Logln(s string) error {
	logger.Println(s)
	return appendFile(d.LogFile(), s+"\n")
}

func (d *Debora) Logf(s string) error {
	logger.Printf(s)
	return appendFile(d.LogFile(), s)
}
