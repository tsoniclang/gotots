package command

import (
	"context"
	"fmt"
	"os"
	"testing"
)

func TestMain(testingMain *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == compileWorkerCommand {
		if err := runCompileWorker(context.Background(), os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(testingMain.Run())
}
