package main

import (
	"encoding/json"
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/ebuchman/debora"
	"log"
	"os"
	"strconv"
)

func main() {
	log.Printf("New Debora Process (PID: %d)\n", os.Getpid())
	app := cli.NewApp()
	app.Name = "debora"
	app.Usage = ""
	app.Version = "0.1.0"
	app.Author = "Ethan Buchman"
	app.Email = "ethan@erisindustries.com"

	app.Flags = []cli.Flag{}

	app.Commands = []cli.Command{
		cli.Command{
			Name:   "run",
			Usage:  "run the debora daemon",
			Action: cliRun,
			Flags:  []cli.Flag{},
		},
		cli.Command{
			Name:   "call",
			Usage:  "broadcast upgrade msg to all peers",
			Action: cliCall,
			Flags: []cli.Flag{
				listenHostFlag,
				listenPortFlag,
				remoteHostFlag,
				remotePortFlag,
			},
		},
		cli.Command{
			Name:   "keygen",
			Usage:  "generate a new key pair",
			Action: cliKeygen,
			Flags:  []cli.Flag{},
		},
	}

	app.Run(os.Args)

}

// run debora and block forever
func cliRun(c *cli.Context) {
	err := debora.DeboraListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

// trigger the upgrade broadcast
func cliCall(c *cli.Context) {
	remoteHost := c.String("remote-host")
	remotePort := c.Int("remote-port")
	listenHost := c.String("listen-host")
	listenPort := c.Int("listen-port")

	remote := remoteHost + ":" + strconv.Itoa(remotePort)
	listen := listenHost + ":" + strconv.Itoa(listenPort)

	// we want the clients to know our address (port, really)
	reqObj := debora.RequestObj{
		Host: listen,
	}
	b, err := json.Marshal(reqObj)
	ifExit(err)

	log.Println("Triggering broadcast with request to:", remote)
	// trigger the broadcast with an http request
	_, err = debora.RequestResponse("http://"+remote, "call", b)
	ifExit(err)

	// listen and serve
	// blocks forever
	err = debora.DeveloperListenAndServe(listen)
	ifExit(err)
}

func cliKeygen(c *cli.Context) {
	/*	args := c.Args()
		if len(args) == 0 {
			log.Fatal("Must provide at least one argument (the app's name)")
		}
		name := args[0]*/
	key, err := debora.GenerateKey()
	ifExit(err)
	priv, pub, err := debora.EncodeKey(key)
	ifExit(err)
	fmt.Println("Private Key:", priv)
	fmt.Println("Public Key:", pub)
}

var (
	listenHostFlag = cli.StringFlag{
		Name:  "listen-host, lh",
		Value: "0.0.0.0",
		Usage: "listen address for authentication requests from clients seeking to upgrade",
	}

	listenPortFlag = cli.IntFlag{
		Name:  "listen-port, lp",
		Value: 56567,
		Usage: "listen port for authentication requests from clients seeking to upgrade",
	}

	remoteHostFlag = cli.StringFlag{
		Name:  "remote-host, rh",
		Value: "localhost",
		Usage: "remote address to trigger broadcast of upgrade message",
	}

	remotePortFlag = cli.IntFlag{
		Name:  "remote-port",
		Value: 56566,
		Usage: "remote port to trigger broadcast of upgrade message",
	}
)

func ifExit(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
