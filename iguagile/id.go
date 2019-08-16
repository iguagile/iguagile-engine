package iguagile

import (
	"math"

	"github.com/minami14/idgo"
)

type IDGenerator struct {
	generator *idgo.IDGenerator
}

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

func (g *IDGenerator) Generate() (int, error) {
	return g.generator.Generate()
}

func (g *IDGenerator) Free(id int) error {
	return g.generator.Free(id)
}
