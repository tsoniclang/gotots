package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/tsoniclang/gotots/internal/testfixture/tsoniccore"
)

func main() {
	root := flag.String("root", "", "root of the resolution-only fixture consumer")
	flag.Parse()
	if *root == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "one explicit -root is required")
		os.Exit(2)
	}
	if err := tsoniccore.InstallResolutionOnly(*root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
