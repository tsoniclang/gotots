package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestCanonicalRuntimePackagePassesUncheckedIndexStrictness(t *testing.T) {
	assembled, err := AssemblePackage(
		tsgo.NewFactory(),
		api.IntegerRepresentationNumber,
		map[api.RuntimeSymbol]struct{}{
			api.RuntimeArray:   {},
			api.RuntimePointer: {},
			api.RuntimeSlice:   {},
		},
		[]api.PrimitiveAlias{api.PrimitiveInt32},
	)
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	client, err := tsgo.StartClient(root, directory)
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, file := range assembled.Files() {
		printed, printErr := client.PrintNode(
			file.SourceFile(),
			tsgo.PrintOptions{},
		)
		if printErr != nil {
			client.Close()
			t.Fatal(printErr)
		}
		relative := strings.TrimPrefix(
			file.OutputPath(),
			assembled.RootPath()+"/",
		)
		path := filepath.Join(directory, filepath.FromSlash(relative))
		if err := os.WriteFile(path, []byte(printed), 0o644); err != nil {
			client.Close()
			t.Fatal(err)
		}
		paths = append(paths, path)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--noUncheckedIndexedAccess",
		"--noEmit",
	}
	arguments = append(arguments, paths...)
	if err := tsgo.Compile(ctx, root, directory, arguments); err != nil {
		t.Fatal(err)
	}
}
