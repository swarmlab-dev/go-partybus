package partybus

import (
	"net/url"

	"github.com/gorilla/websocket"
)

func ConnectToPartyBus(host string, session string, id string, out chan PeerMessage) (chan PeerMessage, chan StatusMessage, error) {
	u := url.URL{Scheme: "ws", Host: host, Path: session}
	ws, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	// say hello
	err = ws.WriteJSON(NewHelloMessage(id))
	if err != nil {
		return nil, nil, err
	}

	// routing received from the socket down to the `in` channel (PeerMessage) and `sig` channel (StatusMessage)
	done := make(chan struct{})
	in := make(chan PeerMessage)
	sig := make(chan StatusMessage)
	go func() {
		defer close(done)
		defer close(in)
		defer close(sig)
		defer ws.Close()

		for {
			_, json, err := ws.ReadMessage()
			if err != nil {
				logger.Errorw("read", "error", err.Error())
				break
			}

			msg, err := ParseBusMessage(json)
			if err != nil {
				logger.Errorw("parse error:%s", err.Error())
				break
			}

			logger.Debugw("recv", "msg", msg)
			switch msg.GetType() {
			case PEER:
				in <- msg.(PeerMessage)
			case STATUS:
				sig <- msg.(StatusMessage)
			}
		}
	}()

	// routing received PeerMessage from `out` channel up to the websocket
	go func() {
		for {
			select {
			case <-done:
				return
			case msg := <-out:
				err := ws.WriteJSON(msg)
				if err != nil {
					logger.Errorw("write", "error", err)
					return
				}
			}
		}
	}()

	return in, sig, nil
}
