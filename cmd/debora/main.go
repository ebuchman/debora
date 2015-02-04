package main

import (
	"github.com/codegangsta/cli"
	"github.com/ebuchman/debora"
	"log"
)

func main() {
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
			Flags:  []cli.Flag{},
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

func cliRun(c *cli.Context) {
	err := debora.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func cliCall(c *cli.Context) {
}

func cliKeygen(c *cli.Context) {
}
