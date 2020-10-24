package iguagile

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	pb "github.com/iguagile/iguagile-room-proto/room"
	"google.golang.org/grpc"
)

// RoomServer is server manages rooms.
type RoomServer struct {
	serverID             int
	rooms                *sync.Map
	factory              RoomServiceFactory
	store                Store
	idGenerator          *IDGenerator
	logger               *log.Logger
	serverProto          *pb.Server
	RoomUpdateDuration   time.Duration
	ServerUpdateDuration time.Duration
}

// ErrPortIsOutOfRange is invalid ports request.
var ErrPortIsOutOfRange = fmt.Errorf("port is out of range")

// NewRoomServer is a constructor of RoomServer.
func NewRoomServer(factory RoomServiceFactory, store Store, address string) (*RoomServer, error) {
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, err
	}
	if port > 65535 || port < 0 {
		return nil, ErrPortIsOutOfRange
	}

	serverID, err := store.GenerateServerID()
	if err != nil {
		return nil, err
	}

	token := uuid.New()

	server := &pb.Server{
		Host:     host,
		Port:     int32(port),
		ServerId: int32(serverID),
		Token:    token[:],
	}

	idGenerator, err := NewIDGenerator()
	if err != nil {
		return nil, err
	}

	return &RoomServer{
		serverID:             serverID,
		rooms:                &sync.Map{},
		factory:              factory,
		store:                store,
		logger:               log.New(os.Stdout, "iguagile-server ", log.Lshortfile),
		serverProto:          server,
		RoomUpdateDuration:   time.Minute * 3,
		ServerUpdateDuration: time.Minute * 3,
		idGenerator:          idGenerator,
	}, nil
}

// Run starts api and room server.
func (s *RoomServer) Run(roomListener net.Listener, apiPort int) error {
	if apiPort > 65535 || apiPort < 0 {
		return ErrPortIsOutOfRange
	}

	s.serverProto.ApiPort = int32(apiPort)
	server := grpc.NewServer()
	apiListener, err := net.Listen("tcp", fmt.Sprintf(":%v", apiPort))
	if err != nil {
		return err
	}

	pb.RegisterRoomServiceServer(server, s)
	go func() {
		_ = server.Serve(apiListener)
	}()

	if err := s.store.RegisterServer(s.serverProto); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func(ctx context.Context) {
		serverTicker := time.NewTicker(s.ServerUpdateDuration)
		roomTicker := time.NewTicker(s.RoomUpdateDuration)
		for {
			select {
			case <-serverTicker.C:
				if err := s.store.RegisterServer(s.serverProto); err != nil {
					s.logger.Println(err)
				}
			case <-roomTicker.C:
				s.rooms.Range(func(_, value interface{}) bool {
					room, ok := value.(*Room)
					if !ok {
						return true
					}
					if !room.creatorConnected {
						return true
					}
					if err := s.store.RegisterRoom(room.roomProto); err != nil {
						s.logger.Println(err)
					}
					return true
				})
			case <-ctx.Done():
				return
			}
		}
	}(ctx)

	for {
		conn, err := roomListener.Accept()
		if err != nil {
			s.logger.Println(err)
			continue
		}

		if err := s.Serve(conn); err != nil {
			s.logger.Println(err)
		}
	}
}

// Serve handles requests from the peer.
func (s *RoomServer) Serve(conn io.ReadWriteCloser) error {
	client := &Client{conn: conn}
	buf := make([]byte, maxMessageSize)
	n, err := client.read(buf)
	if err != nil {
		return err
	}

	if n != 4 {
		return fmt.Errorf("invalid id length %v", buf[:n])
	}

	roomID := int(binary.LittleEndian.Uint32(buf[:4]))
	r, ok := s.rooms.Load(roomID)
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

	n, err = client.read(buf)
	if err != nil {
		return err
	}

	applicationName := string(buf[:n])
	if applicationName != room.config.ApplicationName {
		return fmt.Errorf("invalid application name %v %v", applicationName, room.config.ApplicationName)
	}

	n, err = client.read(buf)
	if err != nil {
		return err
	}

	version := string(buf[:n])
	if version != room.config.Version {
		return fmt.Errorf("invalid version %v %v", version, room.config.Version)
	}

	n, err = client.read(buf)
	if err != nil {
		return err
	}

	password := string(buf[:n])
	if room.config.Password != "" && password != room.config.Password {
		return fmt.Errorf("invalid password %v %v", password, room.config.Password)
	}

	if !room.creatorConnected {
		n, err := client.read(buf)
		if err != nil {
			return err
		}

		if !bytes.Equal(buf[:n], room.config.Token) {
			return fmt.Errorf("invalid token %v %v", buf[:n], room.config.Token)
		}

		room.roomProto.ConnectedUser = 1

		if err := s.store.RegisterRoom(room.roomProto); err != nil {
			return err
		}

		room.creatorConnected = true
	}

	return room.serve(conn)
}

var errInvalidToken = fmt.Errorf("invalid room server api token")

// CreateRoom creates new room.
func (s *RoomServer) CreateRoom(ctx context.Context, request *pb.CreateRoomRequest) (*pb.CreateRoomResponse, error) {
	if !bytes.Equal(request.ServerToken, s.serverProto.Token) {
		return nil, errInvalidToken
	}

	roomID, err := s.idGenerator.Generate()
	if err != nil {
		return nil, err
	}

	roomID |= s.serverID

	config := &RoomConfig{
		RoomID:          roomID,
		ApplicationName: request.ApplicationName,
		Version:         request.Version,
		Password:        request.Password,
		MaxUser:         int(request.MaxUser),
		Token:           request.RoomToken,
		Info:            request.Information,
	}

	r, err := newRoom(s, config)
	if err != nil {
		return nil, err
	}

	service, err := s.factory.Create(r)
	if err != nil {
		return nil, err
	}
	r.service = service

	s.rooms.Store(roomID, r)

	r.roomProto = &pb.Room{
		RoomId:          int32(roomID),
		RequirePassword: request.Password != "",
		MaxUser:         request.MaxUser,
		ConnectedUser:   0,
		Server:          s.serverProto,
		ApplicationName: request.ApplicationName,
		Version:         request.Version,
		Information:     request.Information,
	}

	return &pb.CreateRoomResponse{Room: r.roomProto}, nil
}
