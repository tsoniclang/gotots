package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
	"github.com/tsoniclang/gotots/internal/target/tsgoprinter"
)

func main() {
	moduleDirectory := flag.String("module", "", "GoToTS module directory containing the pinned TS-Go tool")
	workingDirectory := flag.String("cwd", "", "working directory supplied to TS-Go")
	preserveSourceNewlines := flag.Bool("preserve-source-newlines", false, "preserve source newlines")
	neverASCIIEscape := flag.Bool("never-ascii-escape", false, "never ASCII-escape output")
	terminateUnterminatedLiterals := flag.Bool("terminate-unterminated-literals", false, "terminate unterminated literals")
	flag.Parse()
	if *moduleDirectory == "" || *workingDirectory == "" {
		fmt.Fprintln(os.Stderr, "-module and -cwd are required")
		os.Exit(2)
	}
	err := tsgoprinter.Run(tsgoprinter.Config{
		ModuleDirectory:  *moduleDirectory,
		WorkingDirectory: *workingDirectory,
		PrintOptions: tsgo.PrintOptions{
			PreserveSourceNewlines:        *preserveSourceNewlines,
			NeverASCIIEscape:              *neverASCIIEscape,
			TerminateUnterminatedLiterals: *terminateUnterminatedLiterals,
		},
	}, os.Stdin, os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
