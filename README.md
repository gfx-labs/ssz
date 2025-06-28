# ssz


# work in progress

dont use this thing please thank you


## info

[Read the SSZ spec here](https://github.com/ethereum/consensus-specs/blob/dev/ssz/simple-serialize.md)

This is an SSZ library, mostly based off my original work and research into ssz [here](https://gfx.cafe/open/slowssz)

the goal library actually provides two distinct ssz implementations. I dub these two implementations, "flexszz" and "solidssz"


## solidssz

The theory is that immutable SSZ data structures are more efficiently stored as contiguous byte slices.

Instead of unmarshaling ssz bytes into a struct, one can instead just store the ssz bytes, and access fields out of the byte array at call time.
What this means is that the cost to deliver a struct to the payload does not require anything other than writing the raw byte buffer to the wire.

Given an SSZ specification, it is possible to generate accessor classes for different objects, which can be used to access fields of the object.

This strategy is used by erigon/caplin and was found to greatly reduce memory usage, see examples [here](https://github.com/erigontech/erigon/tree/main/cl/cltypes/solid)


## flexssz

The flex implementation at its core has a

1. decoder to do an in-code single pass decode of ssz.
2. builder which builds an ssz object, and then can write the built ssz to a byte array.

using these, and some struct reflection, we can do struct marshalling and unmarshalling via tags.

there are some restrictions to this method, and it's not really suitable for any sort of critical or complex use cases, but it is useful for testing/labbing things out.

