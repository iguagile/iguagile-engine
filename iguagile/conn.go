package iguagile

import (
	"context"
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

type quicConn struct {
	sess quic.Session
}

func (q *quicConn) AcceptStream() (Stream, error) {
	return q.sess.AcceptStream(context.Background())
}

func (q *quicConn) OpenStream() (Stream, error) {
	return q.sess.OpenStream()
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
