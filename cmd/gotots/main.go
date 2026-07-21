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
	"github.com/tsoniclang/gotots/internal/language/analyze"
)

const usage = `usage: gotots <command> [args]

commands:
  inspect constructs <file>   report the catalog constructs in a Go source file`

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
	if len(args) != 2 || args[0] != "constructs" {
		return &UnsupportedCommandError{Command: strings.TrimSpace("inspect " + strings.Join(args, " "))}
	}
	inventory, err := compiler.InspectConstructs(args[1])
	if err != nil {
		return err
	}
	printInventory(stdout, inventory)
	return nil
}

func printInventory(stdout io.Writer, inventory analyze.Inventory) {
	fmt.Fprintf(stdout, "constructs in %s:\n", inventory.Path)
	for _, occurrence := range inventory.Occurrences {
		fmt.Fprintf(stdout, "  %-16s %-12s %d\n",
			occurrence.Kind, occurrence.Kind.Category(), occurrence.Count)
	}
}
