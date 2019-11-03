package client

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"io"
	"log"
	"net"

	"github.com/lucas-clemente/quic-go"
)

type (
	// Target is the target to send data to.
	Target byte

	// MessageType is information for identifying the message type.
	MessageType byte

	// ReceivedBinaryFunc is a function called when a binary message is received.
	ReceivedBinaryFunc func(senderID int, data []byte) error
)

// RPC target
const (
	AllClients = Target(iota)
	OtherClients
	AllClientsBuffered
	OtherClientsBuffered
	Host
	Server
)

const (
	newConnection = MessageType(iota)
	exitConnection
	_ // instantiate
	_ // destroy
	_ // requestObjectControlAuthority
	_ // transferObjectControlAuthority
	migrateHost
	_ // register
	_ // transform
	_ // rpc
	binaryMessage
)

// User has information about users connected to iguagile server
type User struct {
	clientID int
}

// IguagileClient is client for iguagile.
type IguagileClient struct {
	clientID       int
	isHost         bool
	conn           io.ReadWriteCloser
	binaryHandlers []ReceivedBinaryFunc
	writeChan      chan []byte
	users          map[int]*User
	logger         *log.Logger
}

// New creates an instance of IguagileClient.
func New() *IguagileClient {
	return &IguagileClient{
		binaryHandlers: []ReceivedBinaryFunc{},
		writeChan:      make(chan []byte),
		users:          map[int]*User{},
		logger:         &log.Logger{},
	}
}

// DialTCP connects to iguagile server using TCP.
func (c *IguagileClient) DialTCP(addr *net.TCPAddr) error {
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return err
	}

	c.conn = conn
	return nil
}

// DialQUIC connects to iguagile server using QUIC.
func (c *IguagileClient) DialQUIC(ctx context.Context, addr string, tlsConfig *tls.Config) error {
	session, err := quic.DialAddrContext(ctx, addr, tlsConfig, nil)
	if err != nil {
		return err
	}

	stream, err := session.AcceptStream(ctx)
	if err != nil {
		return err
	}

	c.conn = stream
	return nil
}

// SetConnection sets connection that implements io.ReadWriteCloser regardless of protocol.
func (c *IguagileClient) SetConnection(conn io.ReadWriteCloser) {
	c.conn = conn
}

// SetLogger is setter for logger.
func (c *IguagileClient) SetLogger(logger *log.Logger) {
	c.logger = logger
}

// AddReceivedBinaryFunc adds a function to be called when a binary message is received.
func (c *IguagileClient) AddReceivedBinaryFunc(f ReceivedBinaryFunc) {
	c.binaryHandlers = append(c.binaryHandlers, f)
}

// Run starts sending and receiving.
func (c *IguagileClient) Run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	go c.readStart(cancel)
	go c.writeStart(ctx)
}

// SendBinary sends a binary message with target specified.
func (c *IguagileClient) SendBinary(target Target, data []byte) {
	c.writeChan <- append([]byte{byte(target), byte(binaryMessage)}, data...)
}

// Close closes the connection.
func (c *IguagileClient) Close() error {
	return c.conn.Close()
}

func (c *IguagileClient) readStart(cancel context.CancelFunc) {
	for {
		message, err := c.read()
		if err != nil {
			c.logger.Println(err)
			cancel()
			return
		}

		if len(message) < 3 {
			c.logger.Printf("invalid message length %v\n", message)
			cancel()
		}

		clientID := int(binary.LittleEndian.Uint16(message))
		messageType := MessageType(message[2])
		switch messageType {
		case binaryMessage:
			c.receivedBinaryMessage(clientID, message[3:])
		case newConnection:
			c.addUser(clientID)
		case exitConnection:
			c.removeUser(clientID)
		case migrateHost:
			c.migrateHost(clientID)
		}
	}
}

func (c *IguagileClient) receivedBinaryMessage(clientID int, message []byte) {
	for _, f := range c.binaryHandlers {
		if err := f(clientID, message); err != nil {
			c.logger.Println(err)
		}
	}
}

func (c *IguagileClient) addUser(clientID int) {
	c.users[clientID] = &User{
		clientID: clientID,
	}
}

func (c *IguagileClient) removeUser(clientID int) {
	delete(c.users, clientID)
}

func (c *IguagileClient) migrateHost(clientID int) {
	if clientID == c.clientID {
		c.isHost = true
	}
}

func (c *IguagileClient) writeStart(ctx context.Context) {
	for {
		select {
		case message := <-c.writeChan:
			if err := c.write(message); err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *IguagileClient) read() ([]byte, error) {
	sizeBuf := make([]byte, 2)
	if _, err := c.conn.Read(sizeBuf); err != nil {
		return nil, err
	}

	size := int(binary.LittleEndian.Uint16(sizeBuf))

	buf := make([]byte, size)
	if _, err := c.conn.Read(buf); err != nil {
		return nil, err
	}

	return buf, nil
}

func (c *IguagileClient) write(message []byte) error {
	size := len(message)
	sizeBuf := make([]byte, 2)
	binary.LittleEndian.PutUint16(sizeBuf, uint16(size))
	data := append(sizeBuf, message...)
	if _, err := c.conn.Write(data); err != nil {
		return err
	}
	return nil
}
