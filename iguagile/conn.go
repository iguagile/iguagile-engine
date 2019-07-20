package iguagile

// Conn is a wrapper of network connection of each protocol.
type Conn interface {
	Write([]byte) error
	Read() ([]byte, error)
	Close() error
}
