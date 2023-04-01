package iguagile

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/lucas-clemente/quic-go"

	"github.com/google/uuid"
	pb "github.com/iguagile/iguagile-room-proto/room"
	"google.golang.org/grpc"
)

const (
	userStreamPrefix = "U"
)

// Engine is engine manages rooms.
type Engine struct {
	serverID             int
	rooms                *sync.Map
	factory              RoomServiceFactory
	store                Store
	idGenerator          *idGenerator
	logger               *log.Logger
	serverProto          *pb.Server
	room_update_duration   time.Duration
	ServerUpdateDuration time.Duration
}

// New is a constructor of Engine.
func New(factory RoomServiceFactory, store Store) *Engine {
	return &Engine{
		rooms:                &sync.Map{},
		factory:              factory,
		store:                store,
		logger:               log.New(os.Stdout, "iguagile-engine ", log.Lshortfile),
		room_update_duration:   time.Minute * 3,
		ServerUpdateDuration: time.Minute * 3,
	}
}

// Start api and room engine.
func (e *Engine) Start(ctx context.Context, address, apiAddress string, tlsConf *tls.Config) error {
	listener, err := quic.ListenAddr(address, tlsConf, nil)
	if err != nil {
		return err
	}

	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return err
	}

	if port < 0 && port > 65535 {
		return fmt.Errorf("port number is out of valid range %v", port)
	}

	_, apiPortStr, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}

	apiPort, err := strconv.Atoi(apiPortStr)
	if err != nil {
		return err
	}

	if apiPort < 0 && apiPort > 65535 {
		return fmt.Errorf("api port number is out of valid range %v", apiPort)
	}

	serverID, err := e.store.GenerateServerID()
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	apiListener, err := net.Listen("tcp", apiAddress)
	if err != nil {
		return err
	}

	pb.RegisterRoomServiceServer(grpcServer, e)
	go func() {
		_ = grpcServer.Serve(apiListener)
	}()

	token := uuid.New()

	e.serverProto = &pb.Server{
		Host:     host,
		Port:     int32(port),
		ServerId: int32(serverID),
		ApiPort:  int32(apiPort),
		Token:    token[:],
	}

	e.idGenerator, err = newIDGenerator()
	if err != nil {
		return err
	}

	if err := e.store.RegisterServer(e.serverProto); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func(ctx context.Context) {
		serverTicker := time.NewTicker(e.ServerUpdateDuration)
		roomTicker := time.NewTicker(e.room_update_duration)
		defer serverTicker.Stop()
		defer roomTicker.Stop()
		for {
			select {
			case <-serverTicker.C:
				if err := e.store.RegisterServer(e.serverProto); err != nil {
					e.logger.Println(err)
				}
			case <-roomTicker.C:
				e.rooms.Range(func(_, value interface{}) bool {
					room, ok := value.(*Room)
					if !ok {
						return true
					}
					if !room.creatorConnected {
						return true
					}
					if err := e.store.RegisterRoom(room.roomProto); err != nil {
						e.logger.Println(err)
					}
					return true
				})
			case <-ctx.Done():
				return
			}
		}
	}(ctx)

	for {
		sess, err := listener.Accept(ctx)
		if err != nil {
			e.logger.Println(err)
			continue
		}

		conn := &quicConn{sess: sess}
		go func() {
			if err := e.serve(ctx, conn); err != nil {
				e.logger.Println(err)
			}
		}()
	}
}

func (e *Engine) serve(ctx context.Context, conn *quicConn) error {
	stream, err := conn.AcceptStream()
	if err != nil {
		return err
	}

	buf := make([]byte, maxMessageSize)
	n, err := stream.Read(buf)
	if err != nil {
		return err
	}

	if n != 4 {
		return fmt.Errorf("invalid id length %v", buf[:n])
	}

	roomID := int(binary.LittleEndian.Uint32(buf[:4]))
	r, ok := e.rooms.Load(roomID)
	if !ok {
		return fmt.Errorf("the room does not exist %v", roomID)
	}

	room, ok := r.(*Room)
	if !ok {
		return fmt.Errorf("invalid type %T", r)
	}

	if room.clientManager.count >= room.config.MaxUser {
		return fmt.Errorf("connected clients exceed room capacity %v %v", room.config.MaxUser, room.clientManager.count)
	}

	n, err = stream.Read(buf)
	if err != nil {
		return err
	}

	applicationName := string(buf[:n])
	if applicationName != room.config.ApplicationName {
		return fmt.Errorf("invalid application name %v %v", applicationName, room.config.ApplicationName)
	}

	n, err = stream.Read(buf)
	if err != nil {
		return err
	}

	version := string(buf[:n])
	if version != room.config.Version {
		return fmt.Errorf("invalid version %v %v", version, room.config.Version)
	}

	n, err = stream.Read(buf)
	if err != nil {
		return err
	}

	password := string(buf[:n])
	if room.config.Password != "" && password != room.config.Password {
		return fmt.Errorf("invalid password")
	}

	if !room.creatorConnected {
		n, err = stream.Read(buf)
		if err != nil {
			return err
		}

		if !bytes.Equal(buf[:n], room.config.Token) {
			return fmt.Errorf("invalid token %v %v", buf[:n], room.config.Token)
		}

		room.roomProto.ConnectedUser = 1

		if err := e.store.RegisterRoom(room.roomProto); err != nil {
			return err
		}

		room.creatorConnected = true
	}

	return room.serve(ctx, conn)
}

var errInvalidToken = fmt.Errorf("invalid room engine api token")

// CreateRoom creates new room.
func (e *Engine) CreateRoom(ctx context.Context, request *pb.CreateRoomRequest) (*pb.CreateRoomResponse, error) {
	if !bytes.Equal(request.ServerToken, e.serverProto.Token) {
		return nil, errInvalidToken
	}

	roomID, err := e.idGenerator.generate()
	if err != nil {
		return nil, err
	}

	roomID |= e.serverID

	config := &RoomConfig{
		RoomID:          roomID,
		ApplicationName: request.ApplicationName,
		Version:         request.Version,
		Password:        request.Password,
		MaxUser:         int(request.MaxUser),
		Token:           request.RoomToken,
		Info:            request.Information,
	}

	r, err := newRoom(e, config)
	if err != nil {
		return nil, err
	}

	service, err := e.factory.Create(r)
	if err != nil {
		return nil, err
	}
	r.service = service

	e.rooms.Store(roomID, r)

	r.roomProto = &pb.Room{
		RoomId:          int32(roomID),
		RequirePassword: request.Password != "",
		MaxUser:         request.MaxUser,
		ConnectedUser:   0,
		Server:          e.serverProto,
		ApplicationName: request.ApplicationName,
		Version:         request.Version,
		Information:     request.Information,
	}

	return &pb.CreateRoomResponse{Room: r.roomProto}, nil
}
