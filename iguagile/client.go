package iguagile

// Client is a middleman between the connection and the room.
type Client interface {
	Run()
	Send([]byte)
	GetID() int
	GetIDByte() []byte
	Close() error
}
