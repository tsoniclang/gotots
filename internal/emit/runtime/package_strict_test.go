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
	corefixture "github.com/tsoniclang/gotots/internal/testfixture/tsoniccore"
)

func TestCanonicalRuntimePackagePassesUncheckedIndexStrictness(t *testing.T) {
	assembled, err := AssemblePackage(
		tsgo.NewFactory(),
		testScalarABI(t, api.IntegerRepresentationNumber),
		api.ConcurrencySemanticsDisabled,
		map[api.RuntimeSymbol]struct{}{
			api.RuntimeArray:         {},
			api.RuntimeDeferPop:      {},
			api.RuntimePanicNilError: {},
			api.RuntimePanicNilValue: {},
			api.RuntimeSlice:         {},
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
	var printedPackage strings.Builder
	for _, file := range assembled.Files() {
		printed, printErr := client.PrintNode(
			file.SourceFile(),
			tsgo.PrintOptions{},
		)
		if printErr != nil {
			client.Close()
			t.Fatal(printErr)
		}
		printedPackage.WriteString(printed)
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
	if err := corefixture.InstallResolutionOnly(directory); err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"declare private readonly then?: never",
		"$go$value: Pointer<GoPanicNilError>",
		"allocatePointer(new GoPanicNilError)",
	} {
		if !strings.Contains(printedPackage.String(), required) {
			t.Fatalf("panic-nil runtime lacks %q:\n%s", required, printedPackage.String())
		}
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
	}
	arguments = append(arguments, paths...)
	if err := tsgo.Compile(ctx, root, directory, arguments); err != nil {
		t.Fatal(err)
	}
	for _, module := range []string{
		"array.js",
		"interface-value.js",
		"panic.js",
		"slice.js",
	} {
		javascript, readErr := os.ReadFile(
			filepath.Join(directory, "dist", module),
		)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if strings.Contains(string(javascript), "then") {
			t.Fatalf(
				"erased class Promise exclusion reached %s:\n%s",
				module,
				javascript,
			)
		}
	}
}

func TestCanonicalRuntimePackageManifestResolvesEveryBuildStage(t *testing.T) {
	assembled, err := AssemblePackage(
		tsgo.NewFactory(),
		testScalarABI(t, api.IntegerRepresentationNumber),
		api.ConcurrencySemanticsDisabled,
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
	packageRoot := filepath.Join(
		directory,
		"node_modules",
		"@gotots",
		"runtime",
	)
	if err := os.MkdirAll(packageRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	client, err := tsgo.StartClient(root, directory)
	if err != nil {
		t.Fatal(err)
	}
	var sourcePaths []string
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
		target := filepath.Join(packageRoot, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			client.Close()
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(printed), 0o644); err != nil {
			client.Close()
			t.Fatal(err)
		}
		sourcePaths = append(sourcePaths, target)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(packageRoot, "package.json"),
		assembled.Manifest(),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(directory, "main.ts"),
		[]byte("import type { int32 } from \"@gotots/runtime/scalars.js\";\nconst value: int32 = 1;\nvoid value;\n"),
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
	if err := tsgo.Compile(ctx, root, directory, []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--noEmit",
		"main.ts",
	}); err != nil {
		t.Fatal(err)
	}
	published := filepath.Join(directory, "published-runtime")
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--declaration",
		"--rootDir", packageRoot,
		"--outDir", published,
	}
	arguments = append(arguments, sourcePaths...)
	if err := tsgo.Compile(ctx, root, directory, arguments); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(published, "package.json"),
		assembled.Manifest(),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	consumer := filepath.Join(directory, "published-consumer")
	consumerPackageParent := filepath.Join(consumer, "node_modules", "@gotots")
	if err := os.MkdirAll(consumerPackageParent, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(
		published,
		filepath.Join(consumerPackageParent, "runtime"),
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(consumer, "main.ts"),
		[]byte("import type { int32 } from \"@gotots/runtime/scalars.js\";\nconst value: int32 = 1;\nvoid value;\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(consumer, "package.json"),
		[]byte(`{"type":"module"}`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	if err := tsgo.Compile(ctx, root, consumer, []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--noEmit",
		"main.ts",
	}); err != nil {
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
