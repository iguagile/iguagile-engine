package iguagile

import (
	"fmt"
	"sync"
)

// GameObject is object to be synchronized.
type GameObject struct {
	id           int
	owner        *Client
	lifetime     byte
	resourcePath []byte
}

// lifetime
const (
	roomExist = iota
	ownerExist
)

// GameObjectManager manages GameObjects.
type GameObjectManager struct {
	gameObjects map[int]*GameObject
	*sync.Mutex
}

// NewGameObjectManager is GameObjectManager constructed.
func NewGameObjectManager() *GameObjectManager {
	return &GameObjectManager{
		gameObjects: make(map[int]*GameObject),
		Mutex:       &sync.Mutex{},
	}
}

// Get GameObject.
func (m *GameObjectManager) Get(objectID int) (*GameObject, error) {
	m.Lock()
	gameObject, ok := m.gameObjects[objectID]
	m.Unlock()
	if !ok {
		return nil, fmt.Errorf("object not exists %v", objectID)
	}

	return gameObject, nil
}

// Add GameObject.
func (m *GameObjectManager) Add(objectID int, gameObject *GameObject) error {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.gameObjects[objectID]; ok {
		return fmt.Errorf("object exist %v", objectID)
	}

	m.gameObjects[objectID] = gameObject
	return nil
}

// Remove GameObject.
func (m *GameObjectManager) Remove(objectID int) {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.gameObjects[objectID]; !ok {
		return
	}

	delete(m.gameObjects, objectID)
}

// Exist checks the GameObject exists.
func (m *GameObjectManager) Exist(objectID int) bool {
	m.Lock()
	_, ok := m.gameObjects[objectID]
	m.Unlock()
	return ok
}

// GetGameObjectsMap returns all GameObjects.
func (m *GameObjectManager) GetGameObjectsMap() map[int]*GameObject {
	return m.gameObjects
}

// Clear all GameObjects.
func (m *GameObjectManager) Clear() {
	m.Lock()
	m.gameObjects = make(map[int]*GameObject)
	m.Unlock()
}
