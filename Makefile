.PHONY: all build examples clean consensus zoo progressive variable-test

# Build genssz binary
build:
	@echo "Building genssz..."
	@go build -o genssz ./cmd/genssz

# Generate all examples
examples: build consensus zoo progressive variable-test
	@echo "✓ All examples generated"

# Generate consensus example (gostruct, solidssz, and pyssz)
consensus: build
	@echo "Generating consensus gostruct..."
	@./genssz gostruct -o examples/output/consensus/gostruct/generated.go examples/specs/consensus/schema.yaml
	@echo "Generating consensus solidssz..."
	@./genssz solidssz -o examples/output/consensus/solidssz examples/specs/consensus/schema.yaml
	@echo "Generating consensus pyssz..."
	@./genssz pyssz -o examples/output/consensus/pyssz/generated.py examples/specs/consensus/schema.yaml

# Generate zoo example (gostruct, solidssz, and pyssz)
zoo: build
	@echo "Generating zoo gostruct..."
	@./genssz gostruct -o examples/output/zoo/gostruct/generated.go examples/specs/zoo/schema.yaml
	@echo "Generating zoo solidssz..."
	@./genssz solidssz -o examples/output/zoo/solidssz examples/specs/zoo/schema.yaml
	@echo "Generating zoo pyssz..."
	@./genssz pyssz -o examples/output/zoo/pyssz/generated.py examples/specs/zoo/schema.yaml

# Generate progressive example (gostruct, solidssz, and pyssz)
progressive: build
	@echo "Generating progressive gostruct..."
	@./genssz gostruct -o examples/output/progressive/gostruct/generated.go examples/specs/progressive/schema.yaml
	@echo "Generating progressive solidssz..."
	@./genssz solidssz -o examples/output/progressive/solidssz/ examples/specs/progressive/schema.yaml
	@if [ $$? -eq 0 ]; then echo "✓ Successfully generated progressive solidssz"; fi
	@echo "Generating progressive pyssz..."
	@./genssz pyssz -o examples/output/progressive/pyssz/generated.py examples/specs/progressive/schema.yaml

# Generate variable-test example (both gostruct and solidssz)
variable-test: build
	@echo "Generating variable-test gostruct..."
	@./genssz gostruct -o examples/output/variable_test/gostruct/generated.go examples/specs/variable_test/schema.yaml
	@echo "Generating variable-test solidssz..."
	@./genssz solidssz -o examples/output/variable_test_solidssz examples/specs/variable_test/schema.yaml

# Clean generated files
clean:
	@echo "Cleaning generated files..."
	@rm -f genssz
	@rm -f examples/output/*/gostruct/generated.go
	@rm -f examples/output/*/solidssz/*_generated.go
	@rm -f examples/output/*_solidssz/*_generated.go
	@rm -f examples/output/*/pyssz/generated.py
	@echo "✓ Cleaned"

# Build and run all (default target)
all: examples
