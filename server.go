package debora

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type Debora struct {
	debs map[int][]byte // map pids to keys
}

// blocks
func ListenAndServe() error {
	deb := &Debora{
		debs: make(map[int][]byte),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", deb.ping)
	mux.HandleFunc("/add", deb.add)
	mux.HandleFunc("/call", deb.call)
	mux.HandleFunc("/known", deb.known)
	if err := http.ListenAndServe(DeboraHost, mux); err != nil {
		return err
	}
	return nil
}

type RequestObj struct {
	Key []byte
	Pid int
}

// Check is debora server is running
func (deb *Debora) ping(w http.ResponseWriter, r *http.Request) {
	// I'm awake!
}

// Add a new process to debora
func (deb *Debora) add(w http.ResponseWriter, r *http.Request) {
	// read the request, unmarshal json
	p, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	var reqObj = RequestObj{}
	err = json.Unmarshal(p, &reqObj)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	// TODO: check if pid corresponds to real process
	// TODO: check key is appropriate length
	deb.debs[reqObj.Pid] = reqObj.Key
}

// Find out if a process is known to debora
func (deb *Debora) known(w http.ResponseWriter, r *http.Request) {
	// read the request, unmarshal json
	p, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	var reqObj = RequestObj{}
	err = json.Unmarshal(p, &reqObj)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if _, ok := deb.debs[reqObj.Pid]; ok {
		w.Write([]byte("ok"))
	}
}

// Call debora to take down a process, upgrade it, and restart
func (deb *Debora) call(w http.ResponseWriter, r *http.Request) {
	// read the request, unmarshal json
	p, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	var reqObj = RequestObj{}
	err = json.Unmarshal(p, &reqObj)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	// TODO: check pid is real process=
	key, ok := deb.debs[reqObj.Pid]
	if !ok {

	}
	_ = key

}
