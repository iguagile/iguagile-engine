package iguagile

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"io"
	"net"

	"github.com/lucas-clemente/quic-go"
)

// Stream provide a ordered byte-stream abstraction to an application.
type Stream io.ReadWriteCloser

// Conn is a network connection.
type Conn interface {
	AcceptStream() (Stream, error)
	OpenStream() (Stream, error)
	ReceiveMessage() ([]byte, error)
	SendMessage([]byte) error
	Close() error
}

// Listener is a network listener.
type Listener interface {
	Accept() (Conn, error)
	Addr() net.Addr
	Close() error
}

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

func (q *quicConn) AcceptStream() (Stream, error) {
	stream, err := q.sess.AcceptStream(context.Background())
	if err != nil {
		return nil, err
	}

	return &quicStream{stream}, nil
}

func (q *quicConn) OpenStream() (Stream, error) {
	stream, err := q.sess.OpenStream()
	if err != nil {
		return nil, err
	}

	return &quicStream{stream}, nil
}

func (q *quicConn) ReceiveMessage() ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (q *quicConn) SendMessage(buf []byte) error {
	return errors.New("not implemented")
}

func (q *quicConn) Close() error {
	return nil
}

type quicListener struct {
	listener quic.Listener
}

// ListenQuic creates a QUIC server listening on a given address.
func ListenQuic(address string, tlsConfig *tls.Config) (Listener, error) {
	l, err := quic.ListenAddr(address, tlsConfig, nil)
	if err != nil {
		return nil, err
	}

	return &quicListener{l}, nil
}

func (q *quicListener) Accept() (Conn, error) {
	sess, err := q.listener.Accept(context.Background())
	if err != nil {
		return nil, err
	}

	return &quicConn{sess}, nil
}

func (q *quicListener) Addr() net.Addr {
	return q.listener.Addr()
}

func (q *quicListener) Close() error {
	return q.listener.Close()
}
