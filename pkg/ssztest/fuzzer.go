package ssztest

import (
	"fmt"
	"time"

	"github.com/gfx-labs/cuessz"
)

// FuzzConfig configures fuzzing behavior
type FuzzConfig struct {
	// NumIterations is the number of random values to generate
	NumIterations int

	// Seed is the random seed (0 = random)
	Seed int64

	// StopOnFirstMismatch stops fuzzing when a mismatch is found
	StopOnFirstMismatch bool

	// Verbose enables detailed logging
	Verbose bool

	// TypeFilter limits fuzzing to specific types (nil = all types)
	TypeFilter []string
}

// DefaultFuzzConfig returns default fuzzing configuration
func DefaultFuzzConfig() FuzzConfig {
	return FuzzConfig{
		NumIterations:       100,
		Seed:                time.Now().UnixNano(),
		StopOnFirstMismatch: true,
		Verbose:             false,
		TypeFilter:          nil,
	}
}

// Fuzzer performs fuzzing across multiple SSZ implementations
type Fuzzer struct {
	generator  *Generator
	comparator *Comparator
	config     FuzzConfig
}

// NewFuzzer creates a new fuzzer
func NewFuzzer(defs map[string]cuessz.Def, impls []SSZImplementation, config FuzzConfig) *Fuzzer {
	return &Fuzzer{
		generator:  NewGeneratorWithSeed(defs, config.Seed),
		comparator: NewComparator(defs, impls...),
		config:     config,
	}
}

// FuzzResult contains the results of a fuzzing run
type FuzzResult struct {
	Config         FuzzConfig
	TotalTests     int
	PassedTests    int
	FailedTests    int
	ErrorTests     int
	Duration       time.Duration
	Mismatches     []*ComparisonResult
	Errors         []*ComparisonResult
	TestedTypes    map[string]int // type name -> number of times tested
}

// Fuzz runs the fuzzer
func (f *Fuzzer) Fuzz() *FuzzResult {
	startTime := time.Now()

	result := &FuzzResult{
		Config:      f.config,
		Mismatches:  make([]*ComparisonResult, 0),
		Errors:      make([]*ComparisonResult, 0),
		TestedTypes: make(map[string]int),
	}

	// Determine which types to test
	typesToTest := f.getTypesToTest()

	if f.config.Verbose {
		fmt.Printf("Starting fuzz test with %d iterations, testing %d types\n",
			f.config.NumIterations, len(typesToTest))
		fmt.Printf("Seed: %d\n", f.config.Seed)
	}

	for i := 0; i < f.config.NumIterations; i++ {
		// Pick a random type
		typeIdx := int(f.generator.rng.Uint64N(uint64(len(typesToTest))))
		typeName := typesToTest[typeIdx]

		result.TestedTypes[typeName]++
		result.TotalTests++

		// Generate random value
		value, err := f.generator.GenerateNamed(typeName)
		if err != nil {
			if f.config.Verbose {
				fmt.Printf("[%d] Failed to generate %s: %v\n", i+1, typeName, err)
			}
			continue
		}

		// Compare across implementations
		compareResult, err := f.comparator.Compare(typeName, value)
		if err != nil {
			if f.config.Verbose {
				fmt.Printf("[%d] Comparison error for %s: %v\n", i+1, typeName, err)
			}
			continue
		}

		// Check results
		if compareResult.HasErrors() {
			result.ErrorTests++
			result.Errors = append(result.Errors, compareResult)

			if f.config.Verbose {
				fmt.Printf("[%d] Errors in %s:\n", i+1, typeName)
				for name, err := range compareResult.Errors {
					fmt.Printf("  %s: %v\n", name, err)
				}
			}
		} else if compareResult.HasMismatches() {
			result.FailedTests++
			result.Mismatches = append(result.Mismatches, compareResult)

			if f.config.Verbose || f.config.StopOnFirstMismatch {
				fmt.Printf("[%d] MISMATCH found in %s:\n", i+1, typeName)
				fmt.Print(compareResult.Report())
			}

			if f.config.StopOnFirstMismatch {
				break
			}
		} else {
			result.PassedTests++
			if f.config.Verbose {
				fmt.Printf("[%d] ✓ %s passed\n", i+1, typeName)
			}
		}
	}

	result.Duration = time.Since(startTime)
	return result
}

func (f *Fuzzer) getTypesToTest() []string {
	if f.config.TypeFilter != nil {
		return f.config.TypeFilter
	}

	types := make([]string, 0, len(f.generator.defs))
	for name := range f.generator.defs {
		types = append(types, name)
	}
	return types
}

// Report generates a summary report
func (r *FuzzResult) Report() string {
	report := "=== Fuzz Test Results ===\n\n"
	report += fmt.Sprintf("Total tests:  %d\n", r.TotalTests)
	report += fmt.Sprintf("Passed:       %d (%.1f%%)\n", r.PassedTests,
		float64(r.PassedTests)*100/float64(r.TotalTests))
	report += fmt.Sprintf("Failed:       %d (%.1f%%)\n", r.FailedTests,
		float64(r.FailedTests)*100/float64(r.TotalTests))
	report += fmt.Sprintf("Errors:       %d (%.1f%%)\n", r.ErrorTests,
		float64(r.ErrorTests)*100/float64(r.TotalTests))
	report += fmt.Sprintf("Duration:     %v\n", r.Duration)
	report += fmt.Sprintf("Tests/sec:    %.0f\n\n", float64(r.TotalTests)/r.Duration.Seconds())

	if len(r.TestedTypes) > 0 {
		report += "Types tested:\n"
		for typeName, count := range r.TestedTypes {
			report += fmt.Sprintf("  %s: %d\n", typeName, count)
		}
		report += "\n"
	}

	if len(r.Mismatches) > 0 {
		report += fmt.Sprintf("Found %d mismatches:\n", len(r.Mismatches))
		for i, m := range r.Mismatches {
			report += fmt.Sprintf("\nMismatch %d:\n", i+1)
			report += m.Report()
		}
	}

	if len(r.Errors) > 0 {
		report += fmt.Sprintf("\n%d tests had errors:\n", len(r.Errors))
		for i, e := range r.Errors {
			report += fmt.Sprintf("\nError %d (%s):\n", i+1, e.TypeName)
			for name, err := range e.Errors {
				report += fmt.Sprintf("  %s: %v\n", name, err)
			}
		}
	}

	return report
}

// Success returns true if all tests passed
func (r *FuzzResult) Success() bool {
	return r.FailedTests == 0 && r.ErrorTests == 0
}
