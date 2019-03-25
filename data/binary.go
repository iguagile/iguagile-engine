package data

import (
	"fmt"
)

const lengthUUID = 16
const lengthMessageType = 1
const lengthSubType = 1
const lengthTarget = 1

// Message types
const (
	SystemMessage = iota
	UserData
)

// Traffic
const (
	Inbound = iota
	Outbound
)

// BinaryData is client and server data transfer format.
type BinaryData struct {
	Traffic     int
	UUID        []byte
	Target      byte
	MessageType byte
	SubType     byte
	Payload     []byte
}

// NewBinaryData return a BinaryData struct parsed and formatted binary.
func NewBinaryData(b []byte, t int) (BinaryData, error) {
	p := BinaryData{}
	if t == Inbound {
		p.Target = b[:lengthTarget][0]
		p.MessageType = b[lengthTarget : lengthTarget+lengthMessageType][0]
	}
	if t == Outbound {
		p.UUID = b[:lengthUUID]
		p.MessageType = b[lengthUUID : lengthUUID+lengthMessageType][0]
	}

	switch p.MessageType {
	case SystemMessage:
		leng := 0
		if t == Outbound {
			leng += lengthUUID
		} else {
			leng += lengthTarget
		}
		leng += lengthMessageType
		sub := b[leng : leng+lengthSubType]
		p.SubType = sub[0]
		return p, nil

	case UserData:
		leng := 0
		if t == Outbound {
			leng += lengthUUID
		} else {
			leng += lengthTarget
		}
		leng += lengthMessageType + lengthSubType
		p.Payload = b[leng:]
		return p, nil

	default:

		return p, fmt.Errorf("unknown MessageType %v", p.MessageType)
	}
}
