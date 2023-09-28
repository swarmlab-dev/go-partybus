package main

import (
	"os"

	"github.com/swarmlab-dev/go-partybus/partybus"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "partybus-cli"
	app.Usage = "A simple command line to start/join a partybus"
	app.Version = "1.0.0"

	app.Commands = []cli.Command{
		createPartyBus(),
		joinPartyBus(),
	}

	err := app.Run(os.Args)
	if err != nil {
		partybus.Logger().Error("run", "error", err)
	}
}
