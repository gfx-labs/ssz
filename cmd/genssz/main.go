package main

import (
	"github.com/alecthomas/kong"
	"github.com/gfx-labs/ssz/pkg/gen/generators/gostructgen"
	"github.com/gfx-labs/ssz/pkg/gen/generators/solidsszgen"
)

var CLI struct {
	Solidssz SolidsszCmd `cmd:"" help:"Generate optimized fixed-size SSZ types from cuessz schema"`
	Gostruct GostructCmd `cmd:"" help:"Generate Go structs with SSZ annotations from schema"`
}

type SolidsszCmd struct {
	Output string   `short:"o" help:"Output file path" required:""`
	Input  []string `arg:"" help:"Input cuessz schema files (JSON or YAML)" required:""`
}

func (cmd *SolidsszCmd) Run() error {
	return solidsszgen.Generate(cmd.Output, cmd.Input)
}

type GostructCmd struct {
	Output string   `short:"o" help:"Output file path" required:""`
	Input  []string `arg:"" help:"Input schema files (JSON or YAML)" required:""`
}

func (cmd *GostructCmd) Run() error {
	return gostructgen.Generate(cmd.Output, cmd.Input)
}

func main() {
	ctx := kong.Parse(&CLI,
		kong.Name("genssz"),
		kong.Description("Generate Go code from cuessz SSZ schema definitions"),
		kong.UsageOnError(),
	)

	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
