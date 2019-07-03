package data

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

// NewInBoundData return a BinaryData struct parsed and formatted binary.
func NewInBoundData(b []byte) (BinaryData, error) {
	return BinaryData{
		Traffic:     Inbound,
		Target:      b[0:1][0],
		MessageType: b[1:2][0],
		Payload:     b[2:],
	}, nil
}

// NewOutBoundData return a BinaryData struct parsed and formatted binary.
func NewOutBoundData(b []byte) (BinaryData, error) {
	return BinaryData{
		Traffic:     Outbound,
		ID:          b[:2],
		MessageType: b[2:3][0],
		Payload:     b[3:],
	}, nil
}
