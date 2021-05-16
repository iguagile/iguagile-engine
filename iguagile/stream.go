package iguagile

type Stream struct {
	r       *Room
	streams map[int]quicStream
}

func (s *Stream) SendToAllClients(message []byte) {
	for _, stream := range s.streams {
		if _, err := stream.Write(message); err != nil {
			s.r.log.Println(err)
		}
	}
}

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

func (s *Stream) SendToClient(targetID int, message []byte) {
	stream, ok := s.streams[targetID]
	if !ok {
		return
	}

	if _, err := stream.Write(message); err != nil {
		s.r.log.Println(err)
	}
}
