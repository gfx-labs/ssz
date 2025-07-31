// Example demonstrating vector field type usage with genssz
// This is a documentation file showing how to use generated code
// +build ignore

package main

import (
	"encoding/hex"
	"fmt"
)

// This file demonstrates how to use the generated code from vector_example.yaml
// To generate the code:
//   genssz -input vector_example.yaml -output vector_example_generated.go

func main() {
	// Example 1: Working with numeric vectors
	fmt.Println("=== Numeric Vectors Example ===")
	
	// Create a NumericVectors instance
	nv := NewNumericVectors()
	
	// Set vector values
	smallInts := [8]uint16{100, 200, 300, 400, 500, 600, 700, 800}
	nv.SetSmallInts(smallInts)
	
	mediumInts := [4]uint32{10000, 20000, 30000, 40000}
	nv.SetMediumInts(mediumInts)
	
	largeInts := [2]uint64{1000000000, 2000000000}
	nv.SetLargeInts(largeInts)
	
	// Read values back
	fmt.Printf("Small ints: %v\n", nv.SmallInts())
	fmt.Printf("Medium ints: %v\n", nv.MediumInts())
	fmt.Printf("Large ints: %v\n", nv.LargeInts())
	
	// Example of index-based access
	fmt.Println("\n=== Index-Based Access ===")
	fmt.Printf("Small int at index 2: %d\n", nv.SmallIntsAt(2))
	fmt.Printf("Medium int at index 1: %d\n", nv.MediumIntsAt(1))
	fmt.Printf("Large int at index 0: %d\n", nv.LargeIntsAt(0))
	
	// Set individual elements
	nv.SetSmallIntsAt(3, 999)
	fmt.Printf("Small int at index 3 after update: %d\n", nv.SmallIntsAt(3))
	
	// Example 2: Working with vectors of references
	fmt.Println("\n=== Vector of References Example ===")
	
	// Create coordinates
	coord1 := NewCoordinateWithValues(100, 200, 300)
	coord2 := NewCoordinateWithValues(400, 500, 600)
	coord3 := NewCoordinateWithValues(700, 800, 900)
	
	// Create a path with waypoints
	path := NewPath()
	waypoints := [10]Coordinate{}
	waypoints[0] = coord1
	waypoints[1] = coord2
	waypoints[2] = coord3
	// Other waypoints remain zero-initialized
	
	path.SetWaypoints(waypoints)
	
	// Read waypoints back
	readWaypoints := path.Waypoints()
	for i := 0; i < 3; i++ {
		wp := readWaypoints[i]
		fmt.Printf("Waypoint %d: x=%d, y=%d, z=%d\n", i, wp.X(), wp.Y(), wp.Z())
	}
	
	// Example 3: Complex container with mixed types
	fmt.Println("\n=== Mixed Container Example ===")
	
	// Create game state
	gameState := NewGameState()
	gameState.SetPlayerId(12345)
	
	// Set scores
	scores := [5]uint32{1000, 2000, 1500, 3000, 2500}
	gameState.SetScores(scores)
	
	// Set positions
	positions := [5]Coordinate{}
	for i := 0; i < 5; i++ {
		positions[i] = NewCoordinateWithValues(uint32(i*100), uint32(i*100+50), uint32(i*100+25))
	}
	gameState.SetPositions(positions)
	
	// Set checksum
	var checksum [32]byte
	copy(checksum[:], []byte("example checksum data"))
	gameState.SetChecksum(checksum)
	
	// Display game state
	fmt.Printf("Player ID: %d\n", gameState.PlayerId())
	fmt.Printf("Scores: %v\n", gameState.Scores())
	
	// Hash the game state
	hash, err := gameState.HashSSZ()
	if err != nil {
		fmt.Printf("Error hashing: %v\n", err)
	} else {
		fmt.Printf("Game state hash: %s\n", hex.EncodeToString(hash[:]))
	}
	
	// Demonstrate serialization
	fmt.Println("\n=== Serialization Example ===")
	serialized, err := gameState.MarshalSSZ()
	if err != nil {
		fmt.Printf("Error marshaling: %v\n", err)
	} else {
		fmt.Printf("Serialized size: %d bytes\n", len(serialized))
		fmt.Printf("Expected size: %d bytes\n", gameState.SizeSSZ())
	}
}