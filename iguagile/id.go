package iguagile

import (
	"math"

	"github.com/minami14/idgo"
)

// IDGenerator generates id.
type IDGenerator struct {
	generator *idgo.IDGenerator
}

// NewIDGenerator is IDGenerator constructed.
func NewIDGenerator() (*IDGenerator, error) {
	store, err := idgo.NewLocalStore(math.MaxInt16)
	if err != nil {
		return nil, err
	}

	generator, err := idgo.NewIDGenerator(store)
	if err != nil {
		return nil, err
	}

	return &IDGenerator{generator}, nil
}

// Generate generates a id.
func (g *IDGenerator) Generate() (int, error) {
	return g.generator.Generate()
}

// Free deallocates the id.
func (g *IDGenerator) Free(id int) error {
	return g.generator.Free(id)
}
