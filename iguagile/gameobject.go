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
	gameObject, ok := m.gameObjects[objectID]
	if !ok {
		return nil, fmt.Errorf("object not exists %v", objectID)
	}

	return gameObject, nil
}

// Add GameObject.
func (m *GameObjectManager) Add(gameObject *GameObject) error {
	if _, ok := m.gameObjects[gameObject.id]; ok {
		return fmt.Errorf("object exist %v", gameObject.id)
	}

	m.gameObjects[gameObject.id] = gameObject
	return nil
}

// Remove GameObject.
func (m *GameObjectManager) Remove(objectID int) {
	if _, ok := m.gameObjects[objectID]; !ok {
		return
	}

	delete(m.gameObjects, objectID)
}

// Exist checks the GameObject exists.
func (m *GameObjectManager) Exist(objectID int) bool {
	_, ok := m.gameObjects[objectID]
	return ok
}

// GetAllGameObjects returns all GameObjects.
func (m *GameObjectManager) GetAllGameObjects() map[int]*GameObject {
	return m.gameObjects
}

// Clear all GameObjects.
func (m *GameObjectManager) Clear() {
	m.gameObjects = make(map[int]*GameObject)
}
