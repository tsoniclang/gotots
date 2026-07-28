package structvalue_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestDefinedReferenceFamiliesRetainNominalIdentity(t *testing.T) {
	emission, err := compileTemporaryStructProgram(t, `package boundary

type FirstSlice []int32
type SecondSlice []int32
type FirstPointer *int32
type SecondPointer *int32

func FirstSliceValue() FirstSlice { return make(FirstSlice, 1) }
func FirstPointerValue(value *int32) FirstPointer { return FirstPointer(value) }
`)
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	paths, module := materializeStructProgramWithGolden(
		t,
		directory,
		emission,
		false,
	)
	runner := filepath.Join(directory, "nominality.ts")
	writeProgramFile(t, runner, `import * as values from "`+module+`";
const firstSlice = values.FirstSliceValue();
const firstPointer = values.FirstPointerValue(undefined);
if (firstSlice === undefined || firstPointer === undefined) {
    throw new Error("fixture value is nil");
}
const secondSlice: values.SecondSlice = firstSlice;
const secondPointer: values.SecondPointer = firstPointer;
void secondSlice;
void secondPointer;
`)
	if err := typecheckStructuralFiles(directory, append(paths, runner)); err == nil {
		t.Fatal("distinct defined slice/pointer families became assignable")
	}

	replacements := 0
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		mutated := strings.ReplaceAll(
			string(content),
			"declare private readonly $goType: void;\n",
			"",
		)
		if mutated == string(content) {
			continue
		}
		replacements++
		writeProgramFile(t, path, mutated)
	}
	if replacements == 0 {
		t.Fatal("nominality mutation removed no generated brands")
	}
	if err := typecheckStructuralFiles(
		directory,
		append(paths, runner),
	); err != nil {
		t.Fatalf("brand-removal foil did not expose structural assignability: %v", err)
	}
}

func typecheckStructuralFiles(directory string, paths []string) error {
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
	return tsgo.Compile(
		ctx,
		repositoryRoot(),
		directory,
		arguments,
	)
}
