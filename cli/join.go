package main

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/swarmlab-dev/go-partybus/partybus"

	"github.com/anandvarma/namegen"
	"github.com/urfave/cli"
)

var wg sync.WaitGroup

func joinPartyBus() cli.Command {
	return cli.Command{
		Name:    "join",
		Aliases: []string{"j"},
		Usage:   "join a partybus",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "host",
				Value: "http://127.0.0.1:8080",
				Usage: "partybus server to connect to",
			},
			cli.StringFlag{
				Name:  "s",
				Value: namegen.New().Get(),
				Usage: "session id to join (ignored if path provided in uri)",
			},
			cli.StringFlag{
				Name:  "id",
				Value: namegen.New().Get(),
				Usage: "id of this local party",
			},
		},
		Action: func(c *cli.Context) error {
			uri := c.String("host")
			url, err := url.ParseRequestURI(uri)
			if err != nil {
				panic(err)
			}

			session := c.String("s")
			if url.Path != "" {
				session = url.Path
			}

			id := c.String("id")

			return aboardThePartyBus(url.Host, session, id)
		},
	}
}

func aboardThePartyBus(host string, session string, id string) error {
	out := make(chan partybus.PeerMessage)

	wg.Add(3)
	go listenUserInput(id, out)
	in, sig, err := partybus.ConnectToPartyBus(host, session, id, out)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "connected to session=%s with id=%s\n", session, id)

	go listenInputChannel(in)
	go listenStatusChannel(sig)

	wg.Wait()
	return nil
}

func listenUserInput(id string, out chan partybus.PeerMessage) {
	defer wg.Done()

	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		out <- partybus.NewBroadcastMessage(id, []byte(line))
	}
}

func listenInputChannel(in chan partybus.PeerMessage) {
	defer wg.Done()

	for {
		msg := <-in
		fmt.Fprintf(os.Stdout, "%s", string(msg.Msg))
	}
}

func listenStatusChannel(in chan partybus.StatusMessage) {
	defer wg.Done()

	for {
		msg := <-in
		fmt.Fprintf(os.Stderr, "STATUS: [ %s ]\n", strings.Join(msg.Peers, ", "))
	}
}
