package iguagile

import (
	"net"
)

// ServeTCP handles tcp request from the peer
func ServeTCP(room *Room, conn *net.TCPConn) {
	client := NewClientTCP(room, conn)
	room.Register(client)
}
