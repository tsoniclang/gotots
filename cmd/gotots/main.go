// Command gotots is the project-focused Go-to-TypeScript compiler for TSTS.
//
// Current subcommands:
//
//	census        verify the source pin and produce the typed source census
//	toolchain-id  print the resolved toolchain identity for pinning
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/tsoniclang/gotots/internal/census"
	"github.com/tsoniclang/gotots/internal/goenv"
	"github.com/tsoniclang/gotots/internal/pinning"
	"github.com/tsoniclang/gotots/internal/profile"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: gotots <census|toolchain-id> [flags]")
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "census":
		err = runCensus(os.Args[2:])
	case "toolchain-id":
		err = runToolchainID()
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n", os.Args[1])
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "gotots %s: %v\n", os.Args[1], err)
		os.Exit(1)
	}
}

// runToolchainID measures the local toolchain identity so its digests can be
// reviewed and recorded in a pin file. This is the bootstrap path: it runs
// before any pin exists and therefore executes the ambient go command
// unverified. Census verification never uses this path.
func runToolchainID() error {
	goExecutable, err := goenv.Locate()
	if err != nil {
		return err
	}
	resolved, err := goenv.Bootstrap(goExecutable)
	if err != nil {
		return err
	}
	identity, err := pinning.ToolchainIdentity(resolved)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(identity, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func runCensus(args []string) error {
	flags := flag.NewFlagSet("census", flag.ExitOnError)
	profilePath := flags.String("profile", "profiles/tsts/project.json", "project profile path")
	sourceDir := flags.String("source", "", "path to the pinned source checkout (required)")
	buildProfile := flags.String("build-profile", "linux-amd64", "build profile name from the project profile")
	outDir := flags.String("out", "", "report output directory (required)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *sourceDir == "" || *outDir == "" {
		return fmt.Errorf("--source and --out are required")
	}

	prof, err := profile.Load(*profilePath)
	if err != nil {
		return err
	}
	result, err := census.Run(prof, *sourceDir, *buildProfile)
	if err != nil {
		return err
	}
	if err := census.WriteReports(result, *outDir); err != nil {
		return err
	}
	printSummary(result.Report)
	return nil
}

func printSummary(r *census.Report) {
	fmt.Printf("pin        %s @ %s (%s)\n", r.Pin.GoModule, r.Pin.Revision[:12], r.Source.ToolchainVersion)
	fmt.Printf("clean      before-load=%v after-load=%v\n", r.Source.CleanBeforeLoad, r.Source.CleanAfterLoad)
	fmt.Printf("partition  owned=%d test-support=%d hard-excluded=%d unselected=%d external-std=%d external-module=%d\n",
		r.Partition.Owned, r.Partition.TestOnly, r.Partition.HardExcluded,
		r.Partition.Unselected, r.Partition.ExternalStd, r.Partition.ExternalMod)
	fmt.Printf("records    files=%d declarations=%d directives=%d rare-constructs=%d\n",
		len(r.Files), len(r.Declarations), len(r.Directives), len(r.RareConstructs))
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

	unknownDirectives := 0
	for _, d := range r.Directives {
		if !d.Known {
			unknownDirectives++
			fmt.Printf("  UNKNOWN directive %s at %s:%d\n", d.Directive, d.File, d.Line)
		}
	}
	fmt.Printf("unknown directives: %d\n", unknownDirectives)
	fmt.Printf("external packages (direct): %d\n", len(r.External))
	fmt.Printf("contradiction edges: %d\n", len(r.Contradictions))
	for _, e := range r.Contradictions {
		category := e.Class
		if e.Category != "" {
			category += ":" + e.Category
		}
		fmt.Printf("  [%s] %s -> %s (%s) at %s\n", e.Scope, e.From, e.To, category, e.File)
	}
}
