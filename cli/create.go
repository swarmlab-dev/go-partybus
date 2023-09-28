package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

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
			cli.IntFlag{
				Name:  "l",
				Value: 4,
				Usage: "set the log level, 0=Disabled 1=Error, 2=Warn, 3=Info, 4=Debug",
			},
		},
		Action: func(c *cli.Context) error {
			var lout io.Writer
			var level = slog.LevelError
			switch c.Int("l") {
			case 0:
				lout = io.Discard
			case 1:
				lout = os.Stdout
				level = slog.LevelError
			case 2:
				lout = os.Stdout
				level = slog.LevelWarn
			case 3:
				lout = os.Stdout
				level = slog.LevelInfo
			case 4:
				lout = os.Stdout
				level = slog.LevelDebug
			default:
				lout = os.Stdout
				level = slog.LevelError
			}
			partybus.SetLogger(slog.New(slog.NewJSONHandler(lout, &slog.HandlerOptions{Level: level})))

			port := c.Int("port")
			return startHttpServer(port)
		},
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		partybus.Logger().Info("http request", "uri", r.RequestURI)
		next.ServeHTTP(w, r)
	})
}

func startHttpServer(port int) error {
	r := mux.NewRouter()
	r.HandleFunc("/", root)
	r.HandleFunc("/{sessionID:[0-9a-zA-Z-]+}", partybus.HandleQuery)
	r.Use(loggingMiddleware)

	partybus.Logger().Info("Server is running on :8080")
	return http.ListenAndServe(fmt.Sprintf(":%d", port), r)
}

func root(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "go-partybus, a very simple party bus service")
}
