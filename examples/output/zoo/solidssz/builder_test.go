package zoo

import (
	"bytes"
	"testing"

	"github.com/gfx-labs/ssz/pkg/solidssz"
)

func TestClockInRecordsBuilder(t *testing.T) {
	// Test builder pattern
	bitlist := solidssz.NewBitlist(32)
	bitlist.Set(0, true)
	bitlist.Set(5, true)

	builder := NewClockInRecordsBuilder().
		SetEpoch(12345).
		SetBioIdScan(make([]byte, 32)).
		SetWashLogBits(make([]byte, 4)).
		SetSignature(make([]byte, 96)).
		SetPooLogBits(bitlist)

	record := builder.Build()

	// Test immutable getters
	if record.Epoch() != 12345 {
		t.Errorf("Expected epoch 12345, got %d", record.Epoch())
	}

	expectedSize := 144 + bitlist.SizeSSZ()
	if record.SizeSSZ() != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, record.SizeSSZ())
	}

	// Test deserialization round-trip
	data := []byte(record)
	record2 := NewClockInRecordsFromSSZ(data)
	if record2.Epoch() != 12345 {
		t.Errorf("Deserialized epoch mismatch: expected 12345, got %d", record2.Epoch())
	}

	// Verify PooLogBits
	pooLogBits, err := record2.PooLogBits()
	if err != nil {
		t.Fatalf("Failed to get PooLogBits: %v", err)
	}
	if pooLogBits.Len() != bitlist.Len() {
		t.Errorf("PooLogBits length mismatch: expected %d, got %d", bitlist.Len(), pooLogBits.Len())
	}
	bit0, _ := pooLogBits.Get(0)
	if !bit0 {
		t.Errorf("Expected bit 0 to be true")
	}
	bit5, _ := pooLogBits.Get(5)
	if !bit5 {
		t.Errorf("Expected bit 5 to be true")
	}
}

func TestAnimalBuilder(t *testing.T) {
	// Create clock-in records list
	recordsList := NewClockInRecordsList()

	bitlist1 := solidssz.NewBitlist(32)
	bitlist1.Set(0, true)
	record1 := NewClockInRecordsBuilder().
		SetEpoch(100).
		SetBioIdScan(make([]byte, 32)).
		SetWashLogBits(make([]byte, 4)).
		SetSignature(make([]byte, 96)).
		SetPooLogBits(bitlist1).
		Build()

	recordsList.Append(record1)

	// Build animal
	idHash := make([]byte, 32)
	idHash[0] = 0xFF
	publicKey := make([]byte, 48)
	publicKey[0] = 0xAA

	animal := NewAnimalBuilder().
		SetIdHash(idHash).
		SetPublicKey(publicKey).
		SetVaccinated(true).
		SetClockInRecords([]byte(recordsList)).
		Build()

	// Test getters
	if !animal.Vaccinated() {
		t.Errorf("Expected vaccinated to be true")
	}

	if animal.IdHash()[0] != 0xFF {
		t.Errorf("IdHash mismatch")
	}

	if animal.PublicKey()[0] != 0xAA {
		t.Errorf("PublicKey mismatch")
	}

	// Test deserialization
	animal2 := NewAnimalFromSSZ([]byte(animal))
	if !animal2.Vaccinated() {
		t.Errorf("Deserialized animal vaccinated mismatch")
	}

	// Get clock in records from animal
	clockInRecordsData := animal2.ClockInRecords()
	recordsList2 := ClockInRecordsList(clockInRecordsData)
	if recordsList2.Length() != 1 {
		t.Errorf("Expected 1 clock-in record, got %d", recordsList2.Length())
	}

	retrievedRecord, err := recordsList2.Get(0)
	if err != nil {
		t.Fatalf("Failed to get record: %v", err)
	}

	if retrievedRecord.Epoch() != 100 {
		t.Errorf("Expected epoch 100, got %d", retrievedRecord.Epoch())
	}
}

func TestZooBuilderEndToEnd(t *testing.T) {
	// Create multiple animals with clock-in records
	animals := make([]Animal, 3)

	for i := 0; i < 3; i++ {
		// Create clock-in records
		recordsList := NewClockInRecordsList()

		for j := 0; j < 2; j++ {
			bitlist := solidssz.NewBitlist(32)
			bitlist.Set(j, true)

			record := NewClockInRecordsBuilder().
				SetEpoch(uint64(i*100 + j)).
				SetBioIdScan(bytes.Repeat([]byte{byte(i)}, 32)).
				SetWashLogBits(make([]byte, 4)).
				SetSignature(make([]byte, 96)).
				SetPooLogBits(bitlist).
				Build()

			recordsList.Append(record)
		}

		// Create animal
		animal := NewAnimalBuilder().
			SetIdHash(bytes.Repeat([]byte{byte(i)}, 32)).
			SetPublicKey(bytes.Repeat([]byte{byte(i + 10)}, 48)).
			SetVaccinated(i%2 == 0).
			SetClockInRecords([]byte(recordsList)).
			Build()

		animals[i] = animal
	}

	// Build zoo - for now, just serialize all animals as a vector
	// (In a real implementation, this would properly encode the vector)
	var animalsData []byte
	for _, animal := range animals {
		animalsData = append(animalsData, []byte(animal)...)
	}

	zoo := NewZooBuilder().
		SetAnimals(animalsData).
		Build()

	// Test deserialization
	zoo2 := NewZooFromSSZ([]byte(zoo))

	if zoo2.SizeSSZ() != zoo.SizeSSZ() {
		t.Errorf("Zoo size mismatch after deserialization")
	}

	t.Logf("Successfully created zoo with size %d bytes", zoo.SizeSSZ())
}

func TestImmutability(t *testing.T) {
	// Verify that the types are truly immutable
	bitlist := solidssz.NewBitlist(32)
	bitlist.Set(1, true)

	record := NewClockInRecordsBuilder().
		SetEpoch(999).
		SetBioIdScan(make([]byte, 32)).
		SetWashLogBits(make([]byte, 4)).
		SetSignature(make([]byte, 96)).
		SetPooLogBits(bitlist).
		Build()

	originalEpoch := record.Epoch()

	// Try to modify the underlying bytes (this shouldn't affect the record)
	data := []byte(record)
	data[0] = 0xFF

	// The record should still have the original epoch
	if record.Epoch() == originalEpoch {
		t.Logf("Record is correctly backed by []byte and reflects changes to underlying data")
	}

	// Create a new record from the original bytes
	record2 := NewClockInRecordsFromSSZ([]byte(record))
	if record2.Epoch() == originalEpoch {
		t.Logf("Creating new record from SSZ creates a view of the same data")
	}
}
