package debora

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
// and for representing processes/apps
type RequestObj struct {
	Key  string   // hex encoded DER public key
	Pid  int      // process id
	Args []string // command line call that started the process
	App  string   // process name
	Src  string   // code path
	Host string   // bootstrap node (developer's ip:port)
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
