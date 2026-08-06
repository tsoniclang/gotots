package main

import (
	"context"
	"fmt"
	"os"

	"github.com/tsoniclang/gotots/internal/command"
)

func main() {
	workingDirectory, err := os.Getwd()
	if err == nil {
		err = command.Run(
			context.Background(),
			workingDirectory,
			os.Args[1:],
			os.Stdout,
			os.Stderr,
		)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
