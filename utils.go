package debora

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"path"
	"strconv"
)

// get the user's home directory
func homeDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return usr.HomeDir
}

// initalize the debora library by getting the root directory
// and the daemon's host address
func init() {
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
	configFile := path.Join(DeboraRoot, "config.json")
	if _, err := os.Stat(configFile); err != nil {
		b, err := json.Marshal(GlobalConfig)
		if err != nil {
			log.Fatal("Error marshalling global config", err)
		}
		err = ioutil.WriteFile(configFile, b, 0600)
		if err != nil {
			log.Fatal("Error writing configuration to file", err)
		}
	} else {
		b, err := ioutil.ReadFile(configFile)
		if err != nil {
			log.Fatal("Couldn't read config file:", err)
		}
		err = json.Unmarshal(b, &GlobalConfig)
		if err != nil {
			log.Fatal("Error unmarshalling config:", err)
		}
	}
}

// http json request and response
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
