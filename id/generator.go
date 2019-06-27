package id

import (
	"errors"
	"sync"
)

// Generator generate a 16bit id.
type Generator struct {
	mutex *sync.Mutex
	used  []bool
	next  uint
	count uint
	size  uint
}

// NewGenerator is Generator constructed.
func NewGenerator(size uint) *Generator {
	return &Generator{
		mutex: &sync.Mutex{},
		used:  make([]bool, size),
		size:  size,
	}
}

// Generate a new id
func (g *Generator) Generate() (int, error) {
	g.mutex.Lock()
	for {
		if g.count >= g.size {
			return 0, errors.New("id is exhausted")
		}

		if g.next >= g.size {
			g.next = 0
		}

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
