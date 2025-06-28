module github.com/gfx-labs/ssz

go 1.24.4

require (
	github.com/dave/jennifer v1.7.1
	github.com/erigontech/erigon v1.9.7-0.20250627051334-b48bd312b712
	github.com/holiman/uint256 v1.3.2
	github.com/prysmaticlabs/gohashtree v0.0.4-beta
	github.com/stretchr/testify v1.10.0
	sigs.k8s.io/yaml v1.5.0
)

require (
	github.com/c2h5oh/datasize v0.0.0-20231215233829-aa82cc1e6500 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/erigontech/erigon-lib v0.0.0-00010101000000-000000000000 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.8 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/thomaso-mirodin/intmath v0.0.0-20160323211736-5dc6d854e46e // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/erigontech/erigon-lib => github.com/erigontech/erigon/erigon-lib v0.0.0-20250627051334-b48bd312b712

replace github.com/erigontech/erigon-db => github.com/erigontech/erigon/erigon-db v0.0.0-20250627051334-b48bd312b712
