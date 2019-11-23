package iguagile

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"sync"

	"github.com/google/uuid"
	pb "github.com/iguagile/iguagile-room-proto/room"
	"google.golang.org/grpc"
)

// RoomServer is server manages rooms.
type RoomServer struct {
	serverID    int
	rooms       *sync.Map
	store       Store
	idGenerator IDGenerator
	logger      *log.Logger
	serverProto *pb.Server
}

// NewRoomServer is a constructor of RoomServer.
func NewRoomServer(store Store, address string) (*RoomServer, error) {
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, err
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

	return &RoomServer{
		serverID:    serverID,
		rooms:       &sync.Map{},
		store:       store,
		logger:      &log.Logger{},
		serverProto: server,
	}, nil
}

// Run starts api and room server.
func (s *RoomServer) Run(roomListener net.Listener, apiPort int) error {
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
	idByte, err := client.read()
	if err != nil {
		return err
	}

	if len(idByte) != 4 {
		return fmt.Errorf("invalid id length %v", idByte)
	}

	roomID := int(binary.LittleEndian.Uint32(idByte))
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

	applicationName, err := client.read()
	if err != nil {
		return err
	}

	if string(applicationName) != room.config.ApplicationName {
		return fmt.Errorf("invalid application name %v %v", applicationName, room.config.ApplicationName)
	}

	version, err := client.read()
	if err != nil {
		return err
	}

	if string(version) != room.config.Version {
		return fmt.Errorf("invalid version %v %v", version, room.config.Version)
	}

	password, err := client.read()
	if err != nil {
		return err
	}

	if room.config.Password != "" && string(password) != room.config.Password {
		return fmt.Errorf("invalid password %v %v", password, room.config.Password)
	}

	if !room.creatorConnected {
		token, err := client.read()
		if err != nil {
			return err
		}

		if !bytes.Equal(token, room.config.Token) {
			return fmt.Errorf("invalid token %v %v", token, room.config.Token)
		}

		room.roomProto.ConnectedUser = 1

		if err := s.store.RegisterRoom(room.roomProto); err != nil {
			return err
		}

		room.creatorConnected = true
	}

	room.Serve(conn)
	return nil
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

	config := &RoomConfig{
		RoomID:          roomID,
		ApplicationName: request.ApplicationName,
		Version:         request.Version,
		Password:        request.Password,
		MaxUser:         int(request.MaxUser),
		Token:           request.RoomToken,
	}

	r, err := NewRoom(s.store, config)
	if err != nil {
		return nil, err
	}
	s.rooms.Store(roomID, r)

	r.roomProto = &pb.Room{
		RoomId:          int32(roomID),
		RequirePassword: request.Password != "",
		MaxUser:         request.MaxUser,
		ConnectedUser:   0,
		Server:          s.serverProto,
		ApplicationName: request.ApplicationName,
		Version:         request.Version,
	}

	return &pb.CreateRoomResponse{Room: r.roomProto}, nil
}
