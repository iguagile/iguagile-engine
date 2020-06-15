package iguagile

// RoomService implements the processing performed by the room
type RoomService interface {
	// Receive processes data sent from the client to the server.
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

// RelayService is a service relays data.
type RelayService struct {
	room *Room
}

// Receive receives data and sends to all clients.
func (s *RelayService) Receive(senderID int, data []byte) error {
	s.room.SendToAllClients(senderID, data)
	return nil
}

// OnRegisterClient for implement RoomService.
func (s *RelayService) OnRegisterClient(_ int) error {
	return nil
}

// OnUnregisterClient for implement RoomService.
func (s *RelayService) OnUnregisterClient(_ int) error {
	return nil
}

// OnChangeHost for implement RoomService.
func (s RelayService) OnChangeHost(_ int) error {
	return nil
}

// Destroy for implement RoomService.
func (s RelayService) Destroy() error {
	return nil
}

// RelayServiceFactory creates RelayService.
type RelayServiceFactory struct{}

// Create creates a EmptyRoomService.
func (f RelayServiceFactory) Create(room *Room) (RoomService, error) {
	return &RelayService{room: room}, nil
}
