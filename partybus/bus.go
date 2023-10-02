package partybus

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Peer struct {
	ID      string
	conn    *websocket.Conn
	wsMutex sync.Mutex
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
		logger.Errorw("cannot upgrade query to websocket", "error", err.Error())
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
			logger.Errorw("read", "error", err.Error(), "peer", peer.ID)
			break
		}

		msg, err := ParseBusMessage(json)
		if err != nil {
			logger.Errorw("parse", "error", err.Error(), "peer", peer.ID)
			break
		}

		if err = session.checkFromField(peer, msg); err != nil {
			logger.Errorw("check `from` field", "error", err.Error(), "peer", peer.ID)
			sendCloseMessage(peer.conn, websocket.ClosePolicyViolation)
			break
		}

		if err = session.handleMessage(peer, msg); err != nil {
			logger.Errorw("handling message", "error", err.Error(), "peer", peer.ID)
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
		logger.Errorw("write close message", "error", err)
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
			logger.Infow("peer id registered", "session-id", session.ID, "peer-id", peer.ID)
			status := NewStatusSessionMessage(session.ID, session.getPeerIds())
			session.broadcast(status)
		}
		return nil
	case PEER:
		peerMsg, ok := msg.(PeerMessage)
		if !ok {
			return fmt.Errorf("casting of message to PeerMessage failed")
		}

		if peerMsg.To == nil || len(peerMsg.To) == 0 {
			logger.Infow("broadcast peermsg", "from", peerMsg.From, "size", len(peerMsg.Msg))
			session.broadcast(msg)
		} else {
			logger.Infow("multicast peermsg", "from", peerMsg.From, "to", strings.Join(peerMsg.To, ", "), "size", len(peerMsg.Msg))
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
	peer.wsMutex.Lock()
	defer peer.wsMutex.Unlock()

	err := peer.conn.WriteJSON(msg)
	if err != nil {
		logger.Errorw("sendToPeer", "error", err.Error())
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

	logger.Infow("session created", "session-id", sessionID)
	return session
}

func deleteSession(sessionID string) {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	delete(sessions, sessionID)
	logger.Infow("session deleted", "session-id", sessionID)
}

func (session *Session) addPeer(conn *websocket.Conn) *Peer {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	peer := &Peer{
		conn: conn,
	}
	session.Peers[peer] = true
	logger.Infow("peer added", "session-id", session.ID)

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
	logger.Infow("peer removed", "session-id", session.ID, "peer-id", peer.ID)
}
