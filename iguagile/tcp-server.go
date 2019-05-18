package iguagile

import (
	"net"
)

func ServeTCP(room *Room, conn *net.TCPConn) {
	client := NewClientTCP(room, conn)
	room.Register(client)
}
