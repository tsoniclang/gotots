package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
	"github.com/tsoniclang/gotots/internal/target/tsgoprinter"
	"github.com/tsoniclang/gotots/internal/toolchain"
)

func main() {
	moduleDirectory := flag.String("module", "", "GoToTS module directory containing the pinned TS-Go tool")
	goBinary := flag.String("go", "", "selected Go executable")
	tsgoBinary := flag.String("tsgo", "", "selected pinned TS-Go executable")
	toolCache := flag.String("tool-cache", "", "selected .temp/cache root for sealed tools")
	workingDirectory := flag.String("cwd", "", "working directory supplied to TS-Go")
	preserveSourceNewlines := flag.Bool("preserve-source-newlines", false, "preserve source newlines")
	neverASCIIEscape := flag.Bool("never-ascii-escape", false, "never ASCII-escape output")
	terminateUnterminatedLiterals := flag.Bool("terminate-unterminated-literals", false, "terminate unterminated literals")
	flag.Parse()
	if *moduleDirectory == "" || *workingDirectory == "" {
		fmt.Fprintln(os.Stderr, "-module and -cwd are required")
		os.Exit(2)
	}
	cacheRoot := *toolCache
	if cacheRoot == "" {
		cacheRoot = filepath.Join(*moduleDirectory, ".temp", "cache", "toolchain")
	}
	selectedGo, err := toolchain.ResolveGo(*goBinary, cacheRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	selectedTSGo, err := tsgo.ResolveTool(selectedGo, *moduleDirectory, *tsgoBinary)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	err = tsgoprinter.Run(tsgoprinter.Config{
		Tool:             selectedTSGo,
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
