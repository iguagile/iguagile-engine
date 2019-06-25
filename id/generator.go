package id

import (
	"errors"
	"math"
	"sync"
)

// Generator generate a 16bit id.
type Generator struct {
	mutex *sync.Mutex
	used  [1 << 16]bool
	next  uint16
	count int
}

// NewGenerator is Generator constructed.
func NewGenerator() *Generator {
	return &Generator{
		mutex: &sync.Mutex{},
	}
}

// Generate a new id
func (g *Generator) Generate() (int, error) {
	if g.count >= math.MaxInt16 {
		return 0, errors.New("id is exhausted")
	}
	g.mutex.Lock()
	for {
		if !g.used[g.next] {
			id := g.next
			g.used[id] = true
			g.next++
			g.count++
			g.mutex.Unlock()
			return int(id), nil
		}
		g.next++
	}
}

// Free a used id
func (g *Generator) Free(id int) {
	g.mutex.Lock()
	if g.used[id] {
		g.used[id] = false
		g.count--
	}
	g.mutex.Unlock()
}
