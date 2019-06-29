package id

import (
	"errors"
	"sync"
)

// Generator generate a id.
type Generator struct {
	mutex *sync.Mutex

	// allocatedID is slice to judgement if id is allocated.
	allocatedID []bool

	// nextTryID is the number to try next when allocate id.
	nextTryID int

	// allocatedIDCount is amount of currently allocated id.
	allocatedIDCount int

	// maxSize is the maximum value of allocatable id.
	maxSize int
}

// NewGenerator is Generator constructed.
func NewGenerator(maxSize int) (*Generator, error) {
	if maxSize <= 0 {
		return nil, errors.New("argument is negative number")
	}

	return &Generator{
		mutex: &sync.Mutex{},
		// time complexity is small because there is no need to search because id is an index
		allocatedID: make([]bool, maxSize),
		maxSize:     maxSize,
	}, nil
}

// Generate a new id
func (g *Generator) Generate() (int, error) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	for {
		if g.allocatedIDCount >= g.maxSize {
			return 0, errors.New("id is exhausted")
		}

		// Set nextTryID to 0 when nextTryID is greater than maxSize
		if g.nextTryID >= g.maxSize {
			g.nextTryID = 0
		}

		// Allocate and return nextTryID when nextTryID is not yet allocated
		if !g.allocatedID[g.nextTryID] {
			id := g.nextTryID
			g.allocatedID[id] = true
			g.nextTryID++
			g.allocatedIDCount++
			return int(id), nil
		}

		// When nextTryID was assigned, increment nextTryID and try allocate id again.
		g.nextTryID++
	}
}

// Free a used id
func (g *Generator) Free(id int) {
	g.mutex.Lock()
	if g.allocatedID[id] {
		g.allocatedID[id] = false
		g.allocatedIDCount--
	}
	g.mutex.Unlock()
}
