package progressive

import (
	"testing"
)

func TestSimpleMessageBuilder(t *testing.T) {
	// Test progressive container with skipped merkle position
	text := make([]byte, 256)
	copy(text, []byte("Hello, progressive SSZ!"))

	msg := NewSimpleMessageBuilder().
		SetText(text).
		SetTimestamp(1234567890).
		Build()

	// Verify size
	expectedSize := 256 + 8 // text + timestamp (no reserved position data)
	if msg.SizeSSZ() != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, msg.SizeSSZ())
	}

	// Verify getters
	retrievedText := msg.Text()
	if string(retrievedText[:23]) != "Hello, progressive SSZ!" {
		t.Errorf("Text mismatch")
	}

	if msg.Timestamp() != 1234567890 {
		t.Errorf("Expected timestamp 1234567890, got %d", msg.Timestamp())
	}

	// Test deserialization
	msg2 := NewSimpleMessageFromSSZ([]byte(msg))
	if msg2.Timestamp() != 1234567890 {
		t.Errorf("Deserialized timestamp mismatch")
	}

	t.Logf("✓ SimpleMessage (active fields: [1,0,1]) works correctly")
}

func TestTransactionBuilder(t *testing.T) {
	// Test progressive container with multiple skipped positions
	from := make([]byte, 32)
	from[0] = 0xAB
	from[31] = 0xCD

	tx := NewTransactionBuilder().
		SetFrom(from).
		SetAmount(1000000).
		Build()

	// Verify size - only 2 active fields
	expectedSize := 32 + 8 // from + amount (positions 1 and 2 are reserved)
	if tx.SizeSSZ() != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, tx.SizeSSZ())
	}

	// Verify getters
	retrievedFrom := tx.From()
	if retrievedFrom[0] != 0xAB || retrievedFrom[31] != 0xCD {
		t.Errorf("From address mismatch")
	}

	if tx.Amount() != 1000000 {
		t.Errorf("Expected amount 1000000, got %d", tx.Amount())
	}

	// Test deserialization
	tx2 := NewTransactionFromSSZ([]byte(tx))
	if tx2.Amount() != 1000000 {
		t.Errorf("Deserialized amount mismatch")
	}

	t.Logf("✓ Transaction (active fields: [1,0,0,1]) works correctly")
}

func TestUserProfileBuilder(t *testing.T) {
	// Test progressive container with all fields active
	username := make([]byte, 32)
	copy(username, []byte("alice"))

	email := make([]byte, 64)
	copy(email, []byte("alice@example.com"))

	profile := NewUserProfileBuilder().
		SetId(42).
		SetUsername(username).
		SetEmail(email).
		SetVerified(true).
		Build()

	// Verify all fields
	if profile.Id() != 42 {
		t.Errorf("Expected ID 42, got %d", profile.Id())
	}

	if profile.Verified() != true {
		t.Errorf("Expected verified true")
	}

	retrievedUsername := profile.Username()
	if string(retrievedUsername[:5]) != "alice" {
		t.Errorf("Username mismatch")
	}

	// Test round-trip
	profile2 := NewUserProfileFromSSZ([]byte(profile))
	if profile2.Id() != 42 {
		t.Errorf("Deserialized ID mismatch")
	}

	t.Logf("✓ UserProfile (active fields: [1,1,1,1]) works correctly")
}

func TestConfigBuilder(t *testing.T) {
	// Test all-fixed-fields progressive container
	config := NewConfigBuilder().
		SetMaxSize(1024).
		SetTimeout(30).
		SetEnabled(true).
		Build()

	if config.MaxSize() != 1024 {
		t.Errorf("Expected max_size 1024, got %d", config.MaxSize())
	}

	if config.Timeout() != 30 {
		t.Errorf("Expected timeout 30, got %d", config.Timeout())
	}

	if config.Enabled() != true {
		t.Errorf("Expected enabled true")
	}

	// Test round-trip
	config2 := NewConfigFromSSZ([]byte(config))
	if config2.Enabled() != true {
		t.Errorf("Deserialized enabled mismatch")
	}

	t.Logf("✓ Config (active fields: [1,1,1]) works correctly")
}

func TestProgressiveContainerSizes(t *testing.T) {
	// Verify that only active fields are serialized
	tests := []struct {
		name          string
		activeFields  []int
		activeCount   int
		expectedSize  int
		buildFunc     func() []byte
	}{
		{
			name:         "SimpleMessage [1,0,1]",
			activeFields: []int{1, 0, 1},
			activeCount:  2,
			expectedSize: 256 + 8,
			buildFunc: func() []byte {
				return NewSimpleMessageBuilder().
					SetText(make([]byte, 256)).
					SetTimestamp(100).
					Build()
			},
		},
		{
			name:         "Transaction [1,0,0,1]",
			activeFields: []int{1, 0, 0, 1},
			activeCount:  2,
			expectedSize: 32 + 8,
			buildFunc: func() []byte {
				return NewTransactionBuilder().
					SetFrom(make([]byte, 32)).
					SetAmount(500).
					Build()
			},
		},
		{
			name:         "Config [1,1,1]",
			activeFields: []int{1, 1, 1},
			activeCount:  3,
			expectedSize: 8 + 8 + 1,
			buildFunc: func() []byte {
				return NewConfigBuilder().
					SetMaxSize(1000).
					SetTimeout(60).
					SetEnabled(false).
					Build()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.buildFunc()
			if len(data) != tt.expectedSize {
				t.Errorf("%s: expected size %d, got %d (should only serialize %d active fields)",
					tt.name, tt.expectedSize, len(data), tt.activeCount)
			}
			t.Logf("✓ %s correctly serializes only active fields (%d bytes)",
				tt.name, len(data))
		})
	}
}
