// Command gotots is the single GoToTS binary. It parses flags and invokes the
// compiler; there is one compilation route. Unrecognized commands fail closed
// with a typed error and a non-zero exit — never a partial or best-effort run.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tsoniclang/gotots/internal/compiler"
	"github.com/tsoniclang/gotots/internal/source"
)

const usage = `usage: gotots <command> [args]

commands:
  inspect constructs [-dir <dir>] [pattern ...]
      report canonical construct occurrences, roles, variants, implicit
      operations, directives, and exact denominators for the selected
      workspace packages (default pattern ./...)`

// UnsupportedCommandError reports a command line the binary does not implement.
// It is the CLI-level typed error; its string is rendered only here.
type UnsupportedCommandError struct {
	Command string
}

func (e *UnsupportedCommandError) Error() string {
	return fmt.Sprintf("GOTOTS_UNSUPPORTED_COMMAND: %q is not a supported command\n\n%s", e.Command, usage)
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run dispatches one command. It is separated from main for testability and
// returns a typed error rather than exiting.
func run(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return &UnsupportedCommandError{Command: ""}
	}
	switch args[0] {
	case "inspect":
		return runInspect(args[1:], stdout)
	default:
		return &UnsupportedCommandError{Command: strings.Join(args, " ")}
	}
}

func runInspect(args []string, stdout io.Writer) error {
	if len(args) == 0 || args[0] != "constructs" {
		return &UnsupportedCommandError{Command: strings.TrimSpace("inspect " + strings.Join(args, " "))}
	}
	request := source.Request{Dir: "."}
	rest := args[1:]
	var patterns []string
	for i := 0; i < len(rest); i++ {
		switch {
		case rest[i] == "-dir":
			if i+1 >= len(rest) {
				return &UnsupportedCommandError{Command: "inspect constructs -dir (missing directory)"}
			}
			i++
			request.Dir = rest[i]
		case strings.HasPrefix(rest[i], "-"):
			return &UnsupportedCommandError{Command: "inspect constructs " + rest[i]}
		default:
			patterns = append(patterns, rest[i])
		}
	}
	request.Patterns = patterns
	inspection, err := compiler.InspectConstructs(request)
	if err != nil {
		return err
	}
	return printInspection(stdout, inspection)
}

// printInspection renders the resolved universe, the canonical occurrence
// records, and the exact denominators. Write errors propagate; a failing
// writer fails the command.
func printInspection(stdout io.Writer, inspection *compiler.Inspection) error {
	p := func(format string, args ...any) error {
		_, err := fmt.Fprintf(stdout, format, args...)
		return err
	}
	ws := inspection.Workspace()
	if err := p("toolchain %s (%s)\n", ws.Toolchain().Version(), ws.Toolchain().Binary()); err != nil {
		return err
	}
	ownerCounts := map[string]int{}
	for _, pkg := range ws.Packages() {
		ownerCounts[pkg.ID().Owner().Class().String()]++
		if err := p("universe %s provenance=%s acquisition=%s disposition=%s moduleGo=%s selected=%v\n",
			pkg.ID(), pkg.Provenance(), pkg.Acquisition(), pkg.Disposition(), pkg.ModuleGoVersion(), pkg.Selected()); err != nil {
			return err
		}
	}
	inventory := inspection.Inventory()
	for _, pkg := range inventory.Packages() {
		if err := p("package %s\n", pkg.ID()); err != nil {
			return err
		}
		for _, file := range pkg.Files() {
			if err := p("  file %s goVersion=%s\n", file.File(), file.EffectiveGoVersion()); err != nil {
				return err
			}
			for _, occurrence := range file.Occurrences() {
				line := fmt.Sprintf("    %s kind=%s", occurrence.ID(), occurrence.Kind())
				if occurrence.Edge().Valid() {
					line += fmt.Sprintf(" edge=%s role=%s", occurrence.Edge(), occurrence.Role())
				}
				if occurrence.Token().Valid() {
					line += " token=" + occurrence.Token().String()
				}
				if occurrence.Variant() != 0 {
					line += " variant=" + occurrence.Variant().String()
				}
				for _, op := range occurrence.Implicit() {
					line += " implicit=" + op.String()
				}
				if err := p("%s\n", line); err != nil {
					return err
				}
			}
			for _, directive := range file.Directives() {
				if err := p("    directive %s tool=%s name=%s disposition=%s\n",
					directive.Kind(), directive.Tool(), directive.Name(), directive.Kind().Disposition()); err != nil {
					return err
				}
			}
		}
	}
	d := inventory.Denominators()
	if err := p("universe: closurePackages=%d moduleOwned=%d std=%d toolchain=%d languagePseudo=%d\n",
		len(ws.Packages()), ownerCounts["module"], ownerCounts["standard-library"],
		ownerCounts["toolchain"], ownerCounts["language-pseudo"]); err != nil {
		return err
	}
	return p("denominators: selectedPackages=%d files=%d occurrences=%d directives=%d variantBearing=%d implicitOps=%d unknownConstructs=0 unknownDirectives=0\n",
		d.Packages, d.Files, d.Occurrences, d.Directives, d.VariantBearing, d.ImplicitOps)
}
