package iguagile

// Stream is a collection of streams with the same name that all clients have.
type Stream struct {
	r       *Room
	streams map[int]*quicStream
}

// SendToAllClients sends message to all clients through the stream.
func (s *Stream) SendToAllClients(message []byte) {
	for _, stream := range s.streams {
		if _, err := stream.Write(message); err != nil {
			s.r.log.Println(err)
		}
	}
}

// SendToOtherClients sends message to all clients except the sender through the stream.
func (s *Stream) SendToOtherClients(senderID int, message []byte) {
	for id, stream := range s.streams {
		if id == senderID {
			continue
		}

		if _, err := stream.Write(message); err != nil {
			s.r.log.Println(err)
		}
	}
}

// SendToClient sends message to target client through the stream.
func (s *Stream) SendToClient(targetID int, message []byte) {
	stream, ok := s.streams[targetID]
	if !ok {
		return
	}

	if _, err := stream.Write(message); err != nil {
		s.r.log.Println(err)
	}
}
