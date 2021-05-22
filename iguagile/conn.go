package iguagile

import (
	"context"
	"encoding/binary"

	"github.com/lucas-clemente/quic-go"
)

type quicStream struct {
	stream quic.Stream
}

func (q *quicStream) Read(buf []byte) (int, error) {
	_, err := q.stream.Read(buf[:2])
	if err != nil {
		return 0, err
	}

	size := int(binary.LittleEndian.Uint16(buf))
	receivedSizeSum := 0
	for receivedSizeSum < size {
		receivedSize, err := q.stream.Read(buf[receivedSizeSum:size])
		if err != nil {
			return 0, err
		}

		receivedSizeSum += receivedSize
	}

	return size, nil
}

func (q *quicStream) Write(buf []byte) (int, error) {
	size := len(buf)
	sizeByte := make([]byte, 2, size+2)
	binary.LittleEndian.PutUint16(sizeByte, uint16(size))
	buf = append(sizeByte, buf...)
	if _, err := q.stream.Write(buf); err != nil {
		return 0, err
	}
	return size, nil
}

func (q *quicStream) Close() error {
	return q.stream.Close()
}

type quicConn struct {
	sess quic.Session
}

func (q *quicConn) AcceptStream() (*quicStream, error) {
	stream, err := q.sess.AcceptStream(context.Background())
	if err != nil {
		return nil, err
	}

	return &quicStream{stream}, nil
}

func (q *quicConn) OpenStream() (*quicStream, error) {
	stream, err := q.sess.OpenStream()
	if err != nil {
		return nil, err
	}

	return &quicStream{stream}, nil
}

func (q *quicConn) ReceiveMessage() ([]byte, error) {
	return q.sess.ReceiveMessage()
}

func (q *quicConn) SendMessage(message []byte) error {
	return q.sess.SendMessage(message)
}

func (q *quicConn) Close() error {
	return nil
}
