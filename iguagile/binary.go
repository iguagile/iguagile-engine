package iguagile

import "errors"

// Traffic
const (
	Inbound = iota
	Outbound
)

// Message types
const (
	NewConnect = iota
	ExitConnect
)

// BinaryData is client and server data transfer format.
type BinaryData struct {
	Traffic     int
	ID          []byte
	Target      byte
	MessageType byte
	Payload     []byte
}

// ErrInvalidDataFormat is when given unknown data.
var ErrInvalidDataFormat = errors.New("invalid data length")

// NewInBoundData return a BinaryData struct parsed and formatted binary.
func NewInBoundData(b []byte) (*BinaryData, error) {
	if len(b) < 2 {
		return nil, ErrInvalidDataFormat
	}

	return &BinaryData{
		Traffic:     Inbound,
		Target:      b[0],
		MessageType: b[1],
		Payload:     b[2:],
	}, nil
}

// NewOutBoundData return a BinaryData struct parsed and formatted binary.
func NewOutBoundData(b []byte) (*BinaryData, error) {
	if len(b) < 3 {
		return nil, ErrInvalidDataFormat
	}

	return &BinaryData{
		Traffic:     Outbound,
		ID:          b[:2],
		MessageType: b[2],
		Payload:     b[3:],
	}, nil
}
