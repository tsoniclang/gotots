package provider_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestLinkedProviderUntypedConstantsKeepContextualProjections(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/providerconstants\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package providerconstants

import "math"

func MaximumInteger() int64 { return math.MaxInt64 }
func MaximumUnsigned() uint64 { return math.MaxUint64 }
func MaximumFloat() float64 { return math.MaxFloat64 }
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory:    project,
		Pattern:      ".",
		BuildProfile: linkedProviderBuildProfile(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	scope := program.Roots()[0].Types().Scope()
	var roots []emit.Root
	for _, name := range []string{"MaximumInteger", "MaximumUnsigned", "MaximumFloat"} {
		root, rootErr := emit.NewRoot(scope.Lookup(name))
		if rootErr != nil {
			t.Fatal(rootErr)
		}
		roots = append(roots, root)
	}
	options := emit.DefaultOptions()
	options.IntegerRepresentation = emit.IntegerRepresentationBigInt
	options.StandardLibrary = linkedProviderCertificate(t)
	emission, err := emit.CompileWithOptions(program, roots, options)
	if err != nil {
		t.Fatal(err)
	}

	projectionFiles := 0
	seenInteger := false
	seenUnsigned := false
	seenFloat := false
	for _, file := range emission.Files() {
		switch file.Kind() {
		case emit.TargetFileEnvironmentContract:
			t.Fatalf("linked constants retained ambient contract %q", file.OutputPath())
		case emit.TargetFileStandardLibraryConstantProjection:
			projectionFiles++
			for _, statement := range file.SourceFile().Statements() {
				variable, ok := statement.(tsgo.VariableStatement)
				if !ok {
					continue
				}
				for _, declaration := range variable.DeclarationList().Declarations() {
					name := declaration.Name().(tsgo.Identifier).Text()
					switch name {
					case "MaxInt64$int64":
						literal, ok := declaration.Initializer().(tsgo.BigIntLiteral)
						seenInteger = ok && literal.Text() == "9223372036854775807n"
					case "MaxUint64$uint64":
						literal, ok := declaration.Initializer().(tsgo.BigIntLiteral)
						seenUnsigned = ok && literal.Text() == "18446744073709551615n"
					case "MaxFloat64$float64":
						literal, ok := declaration.Initializer().(tsgo.NumericLiteral)
						seenFloat = ok && literal.Text() == "1.7976931348623157e+308"
					}
				}
			}
		}
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	if projectionFiles != 1 || !seenInteger || !seenUnsigned || !seenFloat {
		t.Fatalf(
			"provider projections = files %d, integer %t, unsigned %t, float %t:\n%s",
			projectionFiles,
			seenInteger,
			seenUnsigned,
			seenFloat,
			artifacts.printed,
		)
	}
	for _, forbidden := range []string{
		"export declare const Max",
		"@gotots/gostdlib/math.js",
	} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatalf("provider projection retained %q:\n%s", forbidden, artifacts.printed)
		}
	}
	waveThreeTypecheck(t, workingDirectory, artifacts.paths)
}
