package runtime

import (
	"context"
	"os"
	"os/exec"
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
		testScalarABI(t, api.IntegerRepresentationNumber),
		api.ConcurrencySemanticsDisabled,
		map[api.RuntimeSymbol]struct{}{
			api.RuntimeArray:         {},
			api.RuntimeDeferPop:      {},
			api.RuntimePointer:       {},
			api.RuntimeSlice:         {},
			api.RuntimeUnsafePointer: {},
		},
		nil,
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

func TestDenseIndexDistinguishesNilValueFromAbsentStorage(t *testing.T) {
	assembled, err := AssemblePackage(
		tsgo.NewFactory(),
		testScalarABI(t, api.IntegerRepresentationNumber),
		api.ConcurrencySemanticsDisabled,
		map[api.RuntimeSymbol]struct{}{
			api.RuntimeDenseIndex: {},
		},
		nil,
		nil,
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
	var sourceNames []string
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
		sourceNames = append(sourceNames, relative)
		if err := os.WriteFile(
			filepath.Join(directory, filepath.FromSlash(relative)),
			[]byte(printed),
			0o644,
		); err != nil {
			client.Close()
			t.Fatal(err)
		}
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	runner := `import { GoDenseIndex } from "./dense-index.js";

const presentNil = GoDenseIndex.get<undefined>([undefined], 0);
const presentValue = GoDenseIndex.get<number>([7], 0);
let absent = "missing";
try {
    GoDenseIndex.get<number>(new Array<number>(1), 0);
} catch {
    absent = "panic";
}
console.log(presentValue, presentNil === undefined, absent);
`
	if err := os.WriteFile(
		filepath.Join(directory, "main.ts"),
		[]byte(runner),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(directory, "package.json"),
		[]byte(`{"type":"module"}`),
		0o644,
	); err != nil {
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
		"--outDir", "dist",
		"main.ts",
	}
	arguments = append(arguments, sourceNames...)
	if err := tsgo.Compile(
		ctx,
		root,
		directory,
		arguments,
	); err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(
		ctx,
		"node",
		filepath.Join(directory, "dist", "main.js"),
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("execute dense index: %v\n%s", err, output)
	}
	if got := strings.TrimSpace(string(output)); got != "7 true panic" {
		t.Fatalf("dense index output = %q, want %q", got, "7 true panic")
	}
}
