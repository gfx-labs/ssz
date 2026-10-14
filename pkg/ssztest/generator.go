package ssztest

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/gfx-labs/cuessz"
)

// Generator generates random SSZ values based on a schema definition
type Generator struct {
	defs map[string]cuessz.Def
	rng  *RandomSource
}

// RandomSource provides controlled randomness for reproducible tests
type RandomSource struct {
	seed int64
}

// NewGenerator creates a new test data generator
func NewGenerator(defs map[string]cuessz.Def) *Generator {
	return &Generator{
		defs: defs,
		rng:  NewRandomSource(0),
	}
}

// NewGeneratorWithSeed creates a generator with a specific seed for reproducibility
func NewGeneratorWithSeed(defs map[string]cuessz.Def, seed int64) *Generator {
	return &Generator{
		defs: defs,
		rng:  NewRandomSource(seed),
	}
}

// NewRandomSource creates a new random source with the given seed
func NewRandomSource(seed int64) *RandomSource {
	return &RandomSource{seed: seed}
}

// Uint64 returns a random uint64
func (r *RandomSource) Uint64() uint64 {
	max := new(big.Int).SetUint64(^uint64(0))
	n, _ := rand.Int(rand.Reader, max)
	return n.Uint64()
}

// Uint64N returns a random uint64 in [0, n)
func (r *RandomSource) Uint64N(n uint64) uint64 {
	if n == 0 {
		return 0
	}
	max := new(big.Int).SetUint64(n)
	result, _ := rand.Int(rand.Reader, max)
	return result.Uint64()
}

// Uint32 returns a random uint32
func (r *RandomSource) Uint32() uint32 {
	return uint32(r.Uint64N(uint64(^uint32(0))))
}

// Uint16 returns a random uint16
func (r *RandomSource) Uint16() uint16 {
	return uint16(r.Uint64N(uint64(^uint16(0))))
}

// Uint8 returns a random uint8
func (r *RandomSource) Uint8() uint8 {
	return uint8(r.Uint64N(256))
}

// Bool returns a random boolean
func (r *RandomSource) Bool() bool {
	return r.Uint64N(2) == 1
}

// Bytes generates n random bytes
func (r *RandomSource) Bytes(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)
	return b
}

// GenerateValue generates a random value for the given type definition
func (g *Generator) GenerateValue(def cuessz.Def) (interface{}, error) {
	switch def.Type {
	case cuessz.TypeBoolean:
		return g.rng.Bool(), nil

	case cuessz.TypeUint8:
		return g.rng.Uint8(), nil

	case cuessz.TypeUint16:
		return g.rng.Uint16(), nil

	case cuessz.TypeUint32:
		return g.rng.Uint32(), nil

	case cuessz.TypeUint64:
		return g.rng.Uint64(), nil

	case cuessz.TypeUint128:
		return g.rng.Bytes(16), nil

	case cuessz.TypeUint256:
		return g.rng.Bytes(32), nil

	case cuessz.TypeBitVector:
		numBytes := (def.Size + 7) / 8
		return g.rng.Bytes(int(numBytes)), nil

	case cuessz.TypeBitList:
		// Generate random length up to limit
		length := g.rng.Uint64N(def.Limit + 1)
		numBytes := (length + 7) / 8
		data := g.rng.Bytes(int(numBytes))

		// Add sentinel bit
		sentinelPos := length
		sentinelByte := sentinelPos / 8
		sentinelBit := sentinelPos % 8

		if int(sentinelByte) >= len(data) {
			newData := make([]byte, sentinelByte+1)
			copy(newData, data)
			data = newData
		}
		data[sentinelByte] |= (1 << sentinelBit)

		return data, nil

	case cuessz.TypeVector:
		if len(def.Children) == 0 {
			return nil, fmt.Errorf("vector has no element type")
		}

		elemDef := def.Children[0].Def
		vector := make([]interface{}, def.Size)

		for i := uint64(0); i < def.Size; i++ {
			elem, err := g.GenerateValue(elemDef)
			if err != nil {
				return nil, err
			}
			vector[i] = elem
		}
		return vector, nil

	case cuessz.TypeList:
		if len(def.Children) == 0 {
			return nil, fmt.Errorf("list has no element type")
		}

		// Generate random length up to limit
		length := g.rng.Uint64N(min(def.Limit, 100) + 1) // Cap at 100 for test performance
		elemDef := def.Children[0].Def
		list := make([]interface{}, length)

		for i := uint64(0); i < length; i++ {
			elem, err := g.GenerateValue(elemDef)
			if err != nil {
				return nil, err
			}
			list[i] = elem
		}
		return list, nil

	case cuessz.TypeContainer, cuessz.TypeProgressiveContainer:
		container := make(map[string]interface{})

		for i, field := range def.Children {
			// For progressive containers, check active_fields
			if def.Type == cuessz.TypeProgressiveContainer {
				if i >= len(def.ActiveFields) || def.ActiveFields[i] == 0 {
					continue // Skip inactive fields
				}
			}

			value, err := g.GenerateValue(field.Def)
			if err != nil {
				return nil, err
			}
			container[field.Name] = value
		}
		return container, nil

	case cuessz.TypeRef:
		if refDef, ok := g.defs[def.Ref]; ok {
			return g.GenerateValue(refDef)
		}
		return nil, fmt.Errorf("unknown ref: %s", def.Ref)

	default:
		return nil, fmt.Errorf("unsupported type: %s", def.Type)
	}
}

// GenerateNamed generates a random value for a named type in the schema
func (g *Generator) GenerateNamed(name string) (interface{}, error) {
	def, ok := g.defs[name]
	if !ok {
		return nil, fmt.Errorf("unknown type: %s", name)
	}
	return g.GenerateValue(def)
}

func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}
