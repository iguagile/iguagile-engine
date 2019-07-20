package iguagile

import (
	"net"
)

// ServeTCP handles tcp request from the peer.
func ServeTCP(room *Room, conn *net.TCPConn) {
	client, err := NewClient(room, &ConnTCP{conn: conn})
	if err != nil {
		room.log.Println(err)
		return
	}
	room.Register(client)
}
