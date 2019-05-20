package iguagile

// Client is a middleman between the connection and the room.
type Client interface {
	Run()
	Send([]byte)
	SendToAllClients([]byte)
	SendToOtherClients([]byte)
	CloseConnection()
	GetID() []byte
}
