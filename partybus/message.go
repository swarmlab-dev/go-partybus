package partybus

import (
	"encoding/json"
	"fmt"
)

// type definition

type BusMessageType = int

const (
	PEER   BusMessageType = 0
	HELLO  BusMessageType = 1
	LEAVE  BusMessageType = 2
	STATUS BusMessageType = 3
)

type BaseMessage struct {
	Type BusMessageType `json:"type"`
	From string         `json:"from"`
}

type HelloMessage = BaseMessage
type LeaveMessage = BaseMessage

type StatusMessage struct {
	Type  BusMessageType `json:"type"`
	From  string         `json:"from"`
	Peers []string       `json:"peers"`
}

type PeerMessage struct {
	Type BusMessageType `json:"type"`
	From string         `json:"from"`
	To   []string       `json:"to"`
	Msg  []byte         `json:"msg"`
}

// interface and implementations

type BusMessage interface {
	GetType() BusMessageType
	GetFrom() string
}

func (m BaseMessage) GetType() BusMessageType {
	return m.Type
}

func (m BaseMessage) GetFrom() string {
	return m.From
}

func (m StatusMessage) GetType() BusMessageType {
	return m.Type
}

func (m StatusMessage) GetFrom() string {
	return m.From
}

func (m PeerMessage) GetType() BusMessageType {
	return m.Type
}

func (m PeerMessage) GetFrom() string {
	return m.From
}

// factories

func NewBroadcastMessage(from string, msg []byte) PeerMessage {
	return PeerMessage{
		Type: PEER,
		From: from,
		Msg:  msg,
	}
}

func NewMulticastMessage(from string, to []string, msg []byte) PeerMessage {
	return PeerMessage{
		Type: PEER,
		From: from,
		To:   to,
		Msg:  msg,
	}
}

func NewHelloMessage(from string) HelloMessage {
	return HelloMessage{
		Type: HELLO,
		From: from,
	}
}

func NewLeaveSessionMessage(from string) LeaveMessage {
	return LeaveMessage{
		Type: LEAVE,
		From: from,
	}
}

func NewStatusSessionMessage(session string, peers []string) BusMessage {
	return StatusMessage{
		Type:  STATUS,
		From:  session,
		Peers: peers,
	}
}

// parser

func ParseBusMessage(jsonData []byte) (BusMessage, error) {
	var baseMessage BaseMessage
	err := json.Unmarshal(jsonData, &baseMessage)
	if err != nil {
		return nil, err
	}

	msgType := baseMessage.GetType()
	switch msgType {
	case HELLO:
		return baseMessage, nil
	case LEAVE:
		return baseMessage, nil
	case STATUS:
		var statusMessage StatusMessage
		err := json.Unmarshal(jsonData, &statusMessage)
		if err != nil {
			return nil, err
		}
		return statusMessage, nil
	case PEER:
		var peerMessage PeerMessage
		err := json.Unmarshal(jsonData, &peerMessage)
		if err != nil {
			return nil, err
		}
		return peerMessage, nil
	default:
		return nil, fmt.Errorf("type %i is unknown", msgType)
	}
}
