package penguin

import (
	"testing"
	"encoding/hex"
)

func TestPenguinHashSSZ(t *testing.T) {
	// Create a new penguin
	p := NewPenguin()
	
	// Set some values
	var name [32]byte
	copy(name[:], []byte("Emperor Penguin"))
	p.SetName(name)
	
	var species [2]byte
	species[0] = 0xFF // All 1s for first byte
	species[1] = 0x00 // All 0s for second byte
	p.SetSpecies(species)
	
	p.SetAwesomness(1000)
	p.SetCuteness(255)
	
	// Create and set identity
	identity := NewIdentity()
	identity.SetId(9876543210)
	var pubKey [48]byte
	copy(pubKey[:], []byte("test-public-key-for-penguin"))
	identity.SetPublicKey(pubKey)
	p.SetIdentity(identity)
	
	// Test MarshalSSZ
	data, err := p.MarshalSSZ()
	if err != nil {
		t.Fatalf("MarshalSSZ failed: %v", err)
	}
	if len(data) != 93 {
		t.Errorf("Expected 93 bytes, got %d", len(data))
	}
	
	// Test HashSSZ
	hash, err := p.HashSSZ()
	if err != nil {
		t.Fatalf("HashSSZ failed: %v", err)
	}
	
	// Print the hash for debugging
	t.Logf("Penguin hash: %s", hex.EncodeToString(hash[:]))
	
	// Verify hash is deterministic
	hash2, err := p.HashSSZ()
	if err != nil {
		t.Fatalf("Second HashSSZ failed: %v", err)
	}
	
	if hash != hash2 {
		t.Errorf("Hash not deterministic: %x != %x", hash, hash2)
	}
	
	// Test FillHashBuffer
	buf := make([]byte, 160) // Updated to 160 bytes as required by FillHashBuffer
	err = p.FillHashBuffer(buf)
	if err != nil {
		t.Fatalf("FillHashBuffer failed: %v", err)
	}
	
	// Verify buffer has been filled correctly
	// First 32 bytes should be the name field
	for i := 0; i < 32; i++ {
		if buf[i] != data[i] {
			t.Errorf("Buffer mismatch at position %d: expected %x, got %x", i, data[i], buf[i])
		}
	}
}

func TestPenguinGettersSetters(t *testing.T) {
	p := NewPenguin()
	
	// Test name
	var name [32]byte
	copy(name[:], []byte("Adelie Penguin"))
	p.SetName(name)
	
	gotName := p.Name()
	if gotName != name {
		t.Errorf("Name mismatch")
	}
	
	// Test species  
	species := [2]byte{0x12, 0x34}
	p.SetSpecies(species)
	
	gotSpecies := p.Species()
	if gotSpecies != species {
		t.Errorf("Species mismatch")
	}
	
	// Test awesomness
	p.SetAwesomness(42)
	if p.Awesomness() != 42 {
		t.Errorf("Awesomness mismatch: expected 42, got %d", p.Awesomness())
	}
	
	// Test cuteness
	p.SetCuteness(100)
	if p.Cuteness() != 100 {
		t.Errorf("Cuteness mismatch: expected 100, got %d", p.Cuteness())
	}
}

func TestNewPenguinWithValues(t *testing.T) {
	var name [32]byte
	copy(name[:], []byte("King Penguin"))
	species := [2]byte{0xAA, 0xBB}
	awesomness := uint16(9999)
	cuteness := uint8(200)
	
	// Create an identity
	identity := NewIdentity()
	identity.SetId(12345)
	var pubKey [48]byte
	copy(pubKey[:], []byte("test-public-key"))
	identity.SetPublicKey(pubKey)
	
	p := NewPenguinWithValues(name, species, awesomness, cuteness, identity)
	
	if p.Name() != name {
		t.Errorf("Name not set correctly")
	}
	if p.Species() != species {
		t.Errorf("Species not set correctly")
	}
	if p.Awesomness() != awesomness {
		t.Errorf("Awesomness not set correctly")
	}
	if p.Cuteness() != cuteness {
		t.Errorf("Cuteness not set correctly") 
	}
	// Verify identity
	if p.Identity().Id() != 12345 {
		t.Errorf("Identity ID not set correctly")
	}
}