package iguagile

import (
	"math"

	"github.com/minami14/idgo"
)

// idGenerator generates id.
type idGenerator struct {
	generator *idgo.IDGenerator
}

// newIDGenerator is idGenerator constructed.
func newIDGenerator() (*idGenerator, error) {
	store, err := idgo.NewLocalStore(math.MaxInt16)
	if err != nil {
		return nil, err
	}

	generator, err := idgo.NewIDGenerator(store)
	if err != nil {
		return nil, err
	}

	return &idGenerator{generator}, nil
}

// generate generates a id.
func (g *idGenerator) generate() (int, error) {
	return g.generator.Generate()
}

// free deallocates the id.
func (g *idGenerator) free(id int) error {
	return g.generator.Free(id)
}
