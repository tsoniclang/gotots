// Command gotots is the project-focused Go-to-TypeScript compiler for TSTS.
//
// Current subcommands:
//
//	census   verify the source pin and produce the typed source census
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/tsoniclang/gotots/internal/census"
	"github.com/tsoniclang/gotots/internal/profile"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: gotots census [flags]")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "census":
		if err := runCensus(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "gotots census:", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n", os.Args[1])
		os.Exit(2)
	}
}

func runCensus(args []string) error {
	flags := flag.NewFlagSet("census", flag.ExitOnError)
	profilePath := flags.String("profile", "profiles/tsts/project.json", "project profile path")
	sourceDir := flags.String("source", "", "path to the pinned source checkout (required)")
	buildProfile := flags.String("build-profile", "linux-amd64", "build profile name from the project profile")
	outPath := flags.String("out", "", "census report output path (required)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *sourceDir == "" || *outPath == "" {
		return fmt.Errorf("--source and --out are required")
	}

	prof, err := profile.Load(*profilePath)
	if err != nil {
		return err
	}
	report, err := census.Run(prof, *sourceDir, *buildProfile)
	if err != nil {
		return err
	}
	if err := census.Write(report, *outPath); err != nil {
		return err
	}
	printSummary(report)
	return nil
}

func printSummary(r *census.Report) {
	fmt.Printf("pin        %s @ %s (%s)\n", r.Pin.GoModule, r.Pin.Revision[:12], r.Source.ToolchainOutput)
	fmt.Printf("partition  owned=%d test-support=%d hard-excluded=%d unselected=%d external-std=%d external-module=%d\n",
		len(r.Partition.Owned), len(r.Partition.TestOnly), len(r.Partition.HardExcluded),
		len(r.Partition.Unselected), len(r.Partition.ExternalStd), len(r.Partition.ExternalMod))
	fmt.Printf("production packages=%d files=%d lines=%d\n", r.Production.Packages, r.Production.Files, r.Production.Lines)
	fmt.Printf("           funcs=%d methods=%d bodyless=%d bodies=%d statements=%d\n",
		r.Production.Declarations.Functions, r.Production.Declarations.Methods,
		r.Production.Declarations.BodylessFunctions, r.Production.Bodies, r.Production.Statements)
	fmt.Printf("           types=%d aliases=%d consts=%d vars=%d\n",
		r.Production.Declarations.NamedTypes, r.Production.Declarations.Aliases,
		r.Production.Declarations.Constants, r.Production.Declarations.Variables)
	fmt.Printf("test       packages=%d files=%d lines=%d bodies=%d statements=%d\n",
		r.Test.Packages, r.Test.Files, r.Test.Lines, r.Test.Bodies, r.Test.Statements)

	printMap := func(label string, m map[string]int) {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fmt.Printf("%s:\n", label)
		for _, k := range keys {
			fmt.Printf("  %-24s %d\n", k, m[k])
		}
	}
	printMap("production constructs", r.Production.Constructs)
	printMap("production builtins", r.Production.Builtins)
	printMap("production range operands", r.Production.RangeOperands)
	printMap("production index operands", r.Production.IndexOperands)
	printMap("production directives", r.Production.Directives)

	fmt.Printf("external packages: %d\n", len(r.External))
	fmt.Printf("contradiction edges: %d\n", len(r.Contradictions))
	for _, e := range r.Contradictions {
		category := e.Class
		if e.Category != "" {
			category += ":" + e.Category
		}
		fmt.Printf("  [%s] %s -> %s (%s)\n", e.Scope, e.From, e.To, category)
	}
}
