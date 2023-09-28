package partybus

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Peer struct {
	ID   string
	conn *websocket.Conn
}

type Session struct {
	ID    string
	Peers map[*Peer]bool
}

var sessions = make(map[string]*Session)
var sessionMutex sync.Mutex
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func HandleQuery(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionID"]

	session, exists := sessions[sessionID]
	if !exists {
		session = createSession(sessionID)
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	peer := session.addPeer(conn)
	session.handlePeer(peer)
}

func (session *Session) handlePeer(peer *Peer) {
	defer peer.conn.Close()

	for {
		_, json, err := peer.conn.ReadMessage()
		if err != nil {
			Logger().Error("read", "error", err.Error())
			break
		}

		msg, err := ParseBusMessage(json)
		if err != nil {
			Logger().Error("parse", "error", err.Error())
			break
		}

		if err = session.checkFromField(peer, msg); err != nil {
			Logger().Error("check `from` field", "error", err.Error())
			sendCloseMessage(peer.conn, websocket.ClosePolicyViolation)
			break
		}

		if err = session.handleMessage(peer, msg); err != nil {
			Logger().Error("handling message", "error", err.Error())
			break
		}
	}

	session.deletePeer(peer)
	if len(session.Peers) == 0 {
		deleteSession(session.ID)
	} else {
		status := NewStatusSessionMessage(session.ID, session.getPeerIds())
		session.broadcast(status)
	}
}

func sendCloseMessage(conn *websocket.Conn, reason int) {
	err := conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(reason, ""), time.Now().Add(time.Second))
	if err != nil {
		Logger().Error("write close message", "error", err)
		return
	}
}

func (session *Session) checkFromField(peer *Peer, msg BusMessage) error {
	if msg.GetFrom() == "" {
		return errors.New("`from` field must not be empty")
	}

	if msg.GetFrom() == session.ID {
		return errors.New("`from` field cannot be the same as session id")
	}

	if peer.ID != "" && msg.GetFrom() != peer.ID {
		return errors.New("`from` field cannot change during a session")
	}

	for other := range session.Peers {
		if other == peer {
			continue
		}
		if other.ID == msg.GetFrom() {
			return errors.New("`from` field impersonating another peer")
		}
	}

	return nil
}

func (session *Session) handleMessage(peer *Peer, msg BusMessage) error {
	switch msg.GetType() {
	case HELLO:
		if peer.ID == "" {
			peer.ID = msg.GetFrom()
			Logger().Info("peer id registered", "session-id", session.ID, "peer-id", peer.ID)
			status := NewStatusSessionMessage(session.ID, session.getPeerIds())
			session.broadcast(status)
		}
		return nil
	case PEER:
		peerMsg, ok := msg.(PeerMessage)
		if !ok {
			return fmt.Errorf("casting of message to PeerMessage failed")
		}
		if peerMsg.To == nil {
			session.broadcast(msg)
		} else {
			session.multicast(peerMsg.To, peerMsg)
		}
		return nil
	case LEAVE:
		return fmt.Errorf("user is leaving")
	}
	return nil
}

func (session *Session) broadcast(msg BusMessage) {
	for peer := range session.Peers {
		if peer.ID == msg.GetFrom() {
			continue
		}

		session.sendToPeer(peer, msg)
	}
}

func (session *Session) multicast(to []string, msg BusMessage) {
	for other := range session.Peers {
		if slices.Contains(to, other.ID) {
			session.sendToPeer(other, msg)
		}
	}
}

func (session *Session) sendToPeer(peer *Peer, msg BusMessage) {
	err := peer.conn.WriteJSON(msg)
	if err != nil {
		logger.Error("sendToPeer", "error", err.Error())
	}
}

// Shared State Helper Function

func createSession(sessionID string) *Session {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	session := &Session{
		ID:    sessionID,
		Peers: make(map[*Peer]bool),
	}
	sessions[sessionID] = session

	Logger().Info("session created", "session-id", sessionID)
	return session
}

func deleteSession(sessionID string) {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	delete(sessions, sessionID)
	Logger().Info("session deleted", "session-id", sessionID)
}

func (session *Session) addPeer(conn *websocket.Conn) *Peer {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	peer := &Peer{
		conn: conn,
	}
	session.Peers[peer] = true
	Logger().Info("peer added", "session-id", session.ID)

	return peer
}

func (session *Session) getPeerIds() []string {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	ret := make([]string, len(session.Peers))
	i := 0
	for peer := range session.Peers {
		ret[i] = peer.ID
		i++
	}

	return ret
}

func (session *Session) deletePeer(peer *Peer) {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	delete(session.Peers, peer)
	Logger().Info("peer removed", "session-id", session.ID, "peer-id", peer.ID)
}
