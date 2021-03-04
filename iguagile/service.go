package iguagile

// ReceiveFunc processes data send from the client to the engine.
type ReceiveFunc func(senderID int, data []byte) error

// RoomService implements the processing performed by the room
type RoomService interface {
	// ReceiveFunc returns function processes data sent from the client to the engine.
	// This method is called once every time the client connects.
	ReceiveFunc(streamName string) (ReceiveFunc, error)

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
func (s *RelayService) ReceiveFunc(streamName string) (ReceiveFunc, error) {
	return func(senderID int, data []byte) error {
		s.room.SendToAllClients(streamName, data)
		return nil
	}, nil
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
