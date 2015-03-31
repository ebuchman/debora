package debora

import (
	"os"
)

// Debora daemon's main object for tracking processes and their developer's keys
type Debora struct {
	deb RequestObj
}

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
// and for representing processes/apps
type RequestObj struct {
	Key     string   // hex encoded DER public key
	Pid     int      // process id
	Args    []string // command line call that started the process
	App     string   // process name
	Src     string   // install dir (cd to this before running git fetch. run `go install` from here)
	Commit  string   // commit hash to fetch
	Host    string   // bootstrap node (developer's ip:port)
	LogFile string   // directory to store upgrade logs
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
