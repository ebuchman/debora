package debora

import ()

// Debora daemon's main object for tracking processes and their developer's keys
type Debora struct {
	debKeys map[int]string // map pids to hex encoded DER pub keys
	debIds  map[string]int // map app names to pids
}

// DebMaster is the debora client within the
// developer's instance of the application process
// It's responsible for broadcasting the upgrade message to all peers
// and should only run on the developer's machine
type DebMaster struct {
	callFunc func(payload []byte)
}

// For communicating with the debora daemon
// The same object is used for local communication
// and for communication with the developer
type RequestObj struct {
	Key string // hex encoded der public key
	Pid int
	App string // process name

	nonce []byte // random bytes
}

type Config struct {
	Apps []App
}

type App struct {
	Name       string `json:"name"`
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}

var GlobalConfig = Config{[]App{App{}}}
