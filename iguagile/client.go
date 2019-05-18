package iguagile

type Client interface {
	Run()
	Send([]byte)
	SendToAllClients([]byte)
	SendToOtherClients([]byte)
	CloseConnection()
	AddBuffer(*[]byte)
	GetID() []byte
}