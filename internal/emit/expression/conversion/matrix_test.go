package conversion_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	runtimefixture "github.com/tsoniclang/gotots/internal/testfixture/gototsruntime"
	corefixture "github.com/tsoniclang/gotots/internal/testfixture/tsoniccore"
)

func TestEveryRuntimeNumericConversionCellPrintsAndStrictTypechecks(
	t *testing.T,
) {
	directory := t.TempDir()
	writeFile(t, filepath.Join(directory, "go.mod"), `module example.com/matrix

go 1.26.4
`)
	source, wantFunctions := conversionMatrixSource()
	writeFile(t, filepath.Join(directory, "source.go"), source)
	loaded, err := load.One(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) != wantFunctions {
		t.Fatalf("matrix roots = %d, want %d", len(roots), wantFunctions)
	}
	for _, testCase := range []struct {
		name    string
		options emit.Options
	}{
		{"number", emit.DefaultOptions()},
		{
			"bigint",
			emit.Options{
				IntegerRepresentation: emit.IntegerRepresentationBigInt,
				EvaluationOrder:       emit.EvaluationOrderDirect,
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			emission, err := emit.CompileWithOptions(
				loaded.Program(),
				roots,
				testCase.options,
			)
			if err != nil {
				t.Fatalf("matrix compile failed: %v", err)
			}
			printedBytes := strictTypecheckEmission(t, emission)
			if printedBytes > 100_000 {
				t.Fatalf(
					"matrix output = %d bytes for %d conversions",
					printedBytes,
					wantFunctions,
				)
			}
		})
	}
}

func conversionMatrixSource() (string, int) {
	integers := []string{
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
	}
	floats := []string{"float32", "float64"}
	complexes := []string{"complex64", "complex128"}
	var source strings.Builder
	source.WriteString("package matrix\n\n")
	count := 0
	add := func(from, to string) {
		fmt.Fprintf(
			&source,
			"func C%03d(value %s) %s { return %s(value) }\n",
			count,
			from,
			to,
			to,
		)
		count++
	}
	for _, from := range integers {
		for _, to := range integers {
			add(from, to)
		}
		for _, to := range floats {
			add(from, to)
		}
	}
	for _, from := range floats {
		for _, to := range integers {
			add(from, to)
		}
		for _, to := range floats {
			add(from, to)
		}
	}
	for _, from := range complexes {
		for _, to := range complexes {
			add(from, to)
		}
	}
	return source.String(), count
}

func strictTypecheckEmission(
	t *testing.T,
	emission emit.ProgramEmission,
) int {
	t.Helper()
	workingDirectory := t.TempDir()
	if err := corefixture.InstallResolutionOnly(workingDirectory); err != nil {
		t.Fatal(err)
	}
	client, err := tsgo.StartClient(repositoryRoot(), workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	var paths []string
	printedBytes := 0
	for _, file := range emission.Files() {
		printed, err := client.PrintNode(
			file.SourceFile(),
			tsgo.PrintOptions{},
		)
		if err != nil {
			t.Fatal(err)
		}
		printedBytes += len(printed)
		path := filepath.Join(
			workingDirectory,
			filepath.FromSlash(file.OutputPath()),
		)
		writeFile(t, path, printed)
		paths = append(paths, path)
	}
	writeFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--noEmit",
	}
	arguments = append(arguments, paths...)
	if err := runtimefixture.InstallResolution(workingDirectory, filepath.Join(workingDirectory, "out")); err != nil {
		t.Fatal(err)
	}
	if err := tsgo.Compile(
		ctx,
		repositoryRoot(),
		workingDirectory,
		arguments,
	); err != nil {
		t.Fatalf("matrix failed strict typecheck: %v", err)
	}
	return printedBytes
}
