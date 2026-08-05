package emit_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestDerivedGenericStorageUsesCanonicalProjection(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveSevenGenericDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup(
			"DefinedInstantiatedGeneric",
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}
	artifacts := materializeArtifacts(t, emission, t.TempDir())
	concrete := artifactSection(
		t,
		artifacts.printed,
		"export class ConcreteBox",
		"export type DerivedBox$Storage",
	)
	for _, required := range []string{
		"return new ConcreteBox({",
		"Value: $field0",
		"return this.$storage.Value;",
		"this.$storage.Value = $value;",
	} {
		if !strings.Contains(concrete, required) {
			t.Fatalf("concrete derived storage lacks %q:\n%s", required, concrete)
		}
	}
	for _, forbidden := range []string{
		"const $basis",
		"Box.$fromStorage",
		"Box.$zero",
	} {
		if strings.Contains(concrete, forbidden) {
			t.Fatalf("concrete derived storage contains %q:\n%s", forbidden, concrete)
		}
	}
	generic := artifactSection(
		t,
		artifacts.printed,
		"export class DerivedBox<T>",
		"export function DefinedInstantiatedGeneric",
	)
	if !strings.Contains(
		generic,
		"public static $make<T>($field0: GoStorage<T>): DerivedBox<T>",
	) || strings.Contains(generic, "public get Value") {
		t.Fatalf("generic derived storage is not storage-shaped:\n%s", generic)
	}
	if !strings.Contains(
		artifacts.printed,
		"return value.Value + DerivedBox.$storageOf(generic).Value;",
	) {
		t.Fatalf(
			"generic derived selection bypasses canonical storage:\n%s",
			artifacts.printed,
		)
	}
}

func artifactSection(t *testing.T, text, start, end string) string {
	t.Helper()
	startIndex := strings.Index(text, start)
	if startIndex < 0 {
		t.Fatalf("artifact lacks start marker %q", start)
	}
	endIndex := strings.Index(text[startIndex:], end)
	if endIndex < 0 {
		t.Fatalf("artifact lacks end marker %q", end)
	}
	return text[startIndex : startIndex+endIndex]
}
