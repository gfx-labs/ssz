# ssz


# work in progress

dont use this thing please thank you


## gostruct

gostruct is a library for generating go structs from SSZ specifications. these can be used with dynamic-ssz, fastssz, etc.

## solidssz

The theory is that immutable SSZ data structures are more efficiently stored as contiguous byte slices.

Instead of unmarshaling ssz bytes into a struct, one can instead just store the ssz bytes, and access fields out of the byte array at call time.
What this means is that the cost to deliver a struct to the payload does not require anything other than writing the raw byte buffer to the wire.

Given an SSZ specification, it is possible to generate accessor classes for different objects, which can be used to access fields of the object.

This strategy is used by erigon/caplin and was found to greatly reduce memory usage, see examples [here](https://github.com/erigontech/erigon/tree/main/cl/cltypes/solid)

