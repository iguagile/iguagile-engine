package iguagile

// RoomService implements the processing performed by the room
type RoomService interface {
	// receive processes data sent from the client to the server.
	Receive(senderID int, data []byte) error

	// OnRegisterClient is called when the client connects to the room.
	OnRegisterClient(clientID int) error

	// OnUnregisterClient is called when the client disconnects from the room
	OnUnregisterClient(clientID int) error

	// OnChangeHost is called when the host changes.
	OnChangeHost(clientID int) error

	// Destroy is called when the room is destroyed.
	Destroy() error
}

// RoomServiceFactory creates RoomServices.
type RoomServiceFactory interface {
	// Create creates a RoomService.
	Create(room *Room) (RoomService, error)
}

// EmptyRoomService is empty service used for relay server.
type EmptyRoomService struct{}

// receive for implement RoomService.
func (s EmptyRoomService) Receive(_ int, _ []byte) error {
	return nil
}

// OnRegisterClient for implement RoomService.
func (s EmptyRoomService) OnRegisterClient(_ int) error {
	return nil
}

// OnUnregisterClient for implement RoomService.
func (s EmptyRoomService) OnUnregisterClient(_ int) error {
	return nil
}

// OnChangeHost for implement RoomService.
func (s EmptyRoomService) OnChangeHost(_ int) error {
	return nil
}

// Destroy for implement RoomService.
func (s EmptyRoomService) Destroy() error {
	return nil
}

// EmptyRoomServiceFactory creates EmptyRoomServices.
type EmptyRoomServiceFactory struct{}

// Create creates a EmptyRoomService.
func (f EmptyRoomServiceFactory) Create(_ *Room) (RoomService, error) {
	return EmptyRoomService{}, nil
}
