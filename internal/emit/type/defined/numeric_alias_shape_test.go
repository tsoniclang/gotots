package defined_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestFiniteEnumMutationRejectsLegalOpenNumericValue(t *testing.T) {
	directory := t.TempDir()
	writeDefinedFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/definedbasic\n\ngo 1.26.4\n",
	)
	writeDefinedFile(t, filepath.Join(directory, "source.go"), `package definedbasic

type Flags uint32

const (
	FlagsOne Flags = 1
	FlagsTwo Flags = 2
)

func Continue(flags Flags, values []uint32) uint32 {
	switch flags {
	case FlagsOne:
		return 1
	case FlagsTwo:
		return 2
	}
	rangeValues := values
	return rangeValues[0]
}
`)
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
	emission, err := emit.Compile(loaded.Program(), roots)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := printDefined(t, workingDirectory, emission)
	runner := filepath.Join(workingDirectory, "runner.ts")
	writeDefinedFile(t, runner, "export {};\n")
	if err := typecheckDefined(workingDirectory, artifacts.paths, runner); err != nil {
		t.Fatalf("open scalar numeric continuation failed: %v", err)
	}

	const alias = "export type Flags = uint32;\n"
	const finiteEnum = "export enum Flags {\n" +
		"    $goType = 1\n" +
		"}\n"
	aliasReplacements := 0
	for _, path := range artifacts.paths {
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		mutated := string(content)
		if strings.Contains(mutated, alias) {
			mutated = strings.Replace(mutated, alias, finiteEnum, 1)
			aliasReplacements++
		}
		if mutated == string(content) {
			continue
		}
		writeDefinedFile(t, path, mutated)
	}
	if aliasReplacements != 1 {
		t.Fatalf("finite-enum mutation replaced %d aliases, want one", aliasReplacements)
	}
	if err := typecheckDefined(workingDirectory, artifacts.paths, runner); err == nil {
		t.Fatal("finite enum admitted a legal value outside its declared member set")
	} else if !strings.Contains(err.Error(), "Type '2' is not assignable to type 'Flags'") {
		t.Fatalf("finite enum failed outside the legal open value: %v", err)
	}
}

func assertDefinedNumericAlias(
	t *testing.T,
	alias tsgo.TypeAliasDeclaration,
	wantUnderlying string,
) {
	t.Helper()
	if len(alias.TypeParameters()) != 0 {
		t.Fatalf("numeric alias %s acquired type parameters", alias.Name().Text())
	}
	target, ok := alias.Type().(tsgo.TypeReferenceNode)
	if !ok {
		t.Fatalf(
			"numeric alias %s target = %#v, want %q",
			alias.Name().Text(),
			alias.Type(),
			wantUnderlying,
		)
	}
	name, ok := target.TypeName().(tsgo.Identifier)
	if !ok || name.Text() != wantUnderlying || len(target.TypeArguments()) != 0 {
		t.Fatalf("numeric alias %s target = %#v, want %s", alias.Name().Text(), alias.Type(), wantUnderlying)
	}
}
