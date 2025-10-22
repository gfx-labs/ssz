package cuessz

import (
	"list"
)

// SSZ uses 4-byte (uint32) length prefixes for variable-length types
#MaxSSZSize: 4294967296 // 2^32

// SSZType defines all valid SSZ type names
#SSZType: "uint8" | "uint16" | "uint32" | "uint64" | "uint128" | "uint256" | "boolean" |
	"container" | "progressive_container" | "vector" | "list" |
	"bitvector" | "bitlist" | "union" | "ref"

// Def represents an SSZ type definition
#Def: {
	type: #SSZType

	// Optional documentation for this type
	description?: string

	// this is the size for vectors and bitvectors (max uint32 due to 4-byte SSZ prefix)
	if list.Contains(["vector", "bitvector"], type) {
		size: uint & >0 & <=#MaxSSZSize
	}

	// this is the max length for lists and bitlists (max uint32 due to 4-byte SSZ prefix)
	if list.Contains(["list", "bitlist"], type) {
		limit: uint & >0 & <=#MaxSSZSize
	}

	// container, progressive_container, vector, list, union have children
	if list.Contains(["container", "progressive_container", "union"], type) {
		children: [...#Field]
	} // NOTE: it would be great to have cue if-else semantics here...

	// but for vector and list, it contains children, and it must be must be exactly length 1
	if list.Contains(["vector", "list"], type) {
		children: [...#Field] & list.MinItems(1) & list.MaxItems(1)
	}

	// ref types must provide a string ref
	if type == "ref" {
		ref: string
	}

	if type == "progressive_container" {
		// Bitset indicating which merkle tree positions contain actual fields (0 = skip, 1 = field present)
		// - Length can be up to 256
		// - Count of 1s must equal number of children
		// - Last element must be 1 if present (cannot end in 0)
		active_fields: [...int]
	}
}

// Field represents a named field within a container (used in children arrays)
#Field: {
	name:         string
	def:          #Def
	description?: string // Optional documentation for this field
}

// Schema represents a collection of named type definitions
#Schema: {
	// Schema version for compatibility tracking
	version: string | *"1.0.0"

	// type definitions (the actual business)
	defs: {[string]: #Def}

	// Optional metadata
	metadata?: {
		namespace?:   string
		description?: string
		authors?: [...string]
	}
}
