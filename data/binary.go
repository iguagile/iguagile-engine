package data

import (
	"fmt"
)

const lengthUUID = 16
const lengthMessageType = 1
const lengthSubType = 1

// Message types
const (
	SystemMessage = iota
	UserData
)

// Traffic
const (
	Input = iota
	Output
)

// BinaryData is client and server data transfer format.
type BinaryData struct {
	UUID        []byte
	Target      byte
	MessageType byte
	SubType     byte
	Payload     []byte
}

// NewBinaryData return a BinaryData struct parsed and formatted binary.
func NewBinaryData(b []byte) (BinaryData, error) {
	p := BinaryData{}
	p.UUID = b[:lengthUUID]
	p.MessageType = b[lengthUUID : lengthUUID+lengthMessageType][0]

	switch p.MessageType {
	case SystemMessage:
		sub := b[lengthUUID+lengthMessageType : lengthUUID+lengthMessageType+lengthSubType]
		p.SubType = sub[0]
		return p, nil

	case UserData:
		p.Payload = b[lengthUUID+lengthMessageType+lengthSubType:]
		return p, nil

	default:

		return p, fmt.Errorf("unknown MessageType %v", p.MessageType)
	}
}
