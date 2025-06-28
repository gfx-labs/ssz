package ssz

type HashableSSZ interface {
	HashSSZ() ([32]byte, error)
}

type Prehash [32]byte

func (p *Prehash) HashSSZ() ([32]byte, error) {
	return *p, nil
}
