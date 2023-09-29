package main

import (
	"fmt"
	"net/http"

	"github.com/swarmlab-dev/go-partybus/partybus"

	"github.com/gorilla/mux"
	"github.com/urfave/cli"
)

func createPartyBus() cli.Command {
	return cli.Command{
		Name:    "create",
		Aliases: []string{"c"},
		Usage:   "start a partybus http server",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "port",
				Value: 8080,
				Usage: "port on which to listen to",
			},
		},
		Action: func(c *cli.Context) error {
			port := c.Int("port")
			return startHttpServer(port)
		},
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info("http request", "uri", r.RequestURI)
		next.ServeHTTP(w, r)
	})
}

func startHttpServer(port int) error {
	r := mux.NewRouter()
	r.HandleFunc("/", root)
	r.HandleFunc("/{sessionID:[0-9a-zA-Z-]+}", partybus.HandleQuery)
	r.Use(loggingMiddleware)

	logger.Info("Server is running on :8080")
	return http.ListenAndServe(fmt.Sprintf(":%d", port), r)
}

func root(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "go-partybus, a very simple party bus service")
}
