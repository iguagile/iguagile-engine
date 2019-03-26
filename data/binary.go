package data

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

	switch t {
	case Inbound: // ref https://github.com/iguagile/iguagile-engine/wiki/protocol#inbound
		p.Target = b[0:1][0]
		p.MessageType = b[1:2][0]
		p.SubType = b[2:3][0]
		p.Payload = b[2:]
	case Outbound: // ref: https://github.com/iguagile/iguagile-engine/wiki/protocol#outbound
		p.UUID = b[:16]
		p.MessageType = b[16:17][0]
		p.SubType = b[16:17][0]
		p.Payload = b[18:]
	}

	return p, nil
}
