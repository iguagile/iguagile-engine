package iguagile

import (
	"net"
)

// ServeTCP handles tcp request from the peer.
func ServeTCP(room *Room, conn *net.TCPConn) {
	client, err := NewClientTCP(room, conn)
	if err != nil {
		room.log.Println(err)
		return
	}
	room.Register(client)
}
