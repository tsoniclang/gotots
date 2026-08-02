package provider_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestProviderGenericCallsUseCertifiedTargetProjection(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/providerprojection\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package providerprojection

import (
	"cmp"
	"context"
	"errors"
	"maps"
	"slices"
	"time"
)

type code int

func (value code) Error() string { return "code" }

func AsCode(failure error) (code, bool) {
	return errors.AsType[code](failure)
}

func Concat(left, right []string) []string {
	return slices.Concat(left, right)
}

func Clone(source map[string]int) map[string]int {
	return maps.Clone(source)
}

func Grow(source []string) []string {
	return slices.Grow(source, 1)
}

func GenericClone[T any](source []T) []T {
	return slices.Clone(source)
}

func GenericCloneValue[T any]() func([]T) []T {
	return slices.Clone[[]T, T]
}

func CompareValue() func(string, string) int {
	return cmp.Compare[string]
}

func DeferredCompare() {
	defer cmp.Compare("left", "right")
}

func PrintValues(source []string) {
	_, cancel := context.WithCancel(context.Background())
	for range slices.Values(source) {
		cancel()
	}
}

func GenericAddress[T any](first, second T) T {
	values := [1]T{first}
	pointer := &values[0]
	values[0] = second
	return *pointer
}

func TimeAddress() time.Time {
	return GenericAddress(time.Time{}, time.Time{})
}
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
	options := emit.DefaultOptions()
	options.ConcurrencySemantics = emit.ConcurrencySemanticsCooperative
	options.StandardLibrary = linkedProviderCertificate(t)
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{
			mustProviderRoot(t, scope.Lookup("Concat")),
			mustProviderRoot(t, scope.Lookup("Clone")),
			mustProviderRoot(t, scope.Lookup("GenericClone")),
			mustProviderRoot(t, scope.Lookup("GenericCloneValue")),
			mustProviderRoot(t, scope.Lookup("CompareValue")),
			mustProviderRoot(t, scope.Lookup("DeferredCompare")),
			mustProviderRoot(t, scope.Lookup("Grow")),
			mustProviderRoot(t, scope.Lookup("PrintValues")),
			mustProviderRoot(t, scope.Lookup("AsCode")),
			mustProviderRoot(t, scope.Lookup("TimeAddress")),
		},
		options,
	)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	printed := artifacts.printed
	for _, exact := range []string{
		"SlicesConcatKernel<gostring>(",
		"MapsCloneKernel<GoMapValue<gostring, int64>, gostring, int64>(",
		"SlicesGrowKernel<RuntimeSlice<gostring>, gostring, gostring>(",
		"SlicesCloneKernel<GoContainerStorage<T>>(",
		"SlicesCloneKernel<GoContainerStorage<T>>($argument0)",
		"CmpCompareKernel<gostring>(",
		"SlicesValuesCooperative<RuntimeSlice<gostring>, gostring, gostring>(",
		"ErrorsAsTypeKernel<code__from_providerprojection>($goCapability_",
		"GenericAddress$kernel<T>",
		"GoPointerType<T>",
	} {
		if !strings.Contains(printed, exact) {
			t.Fatalf("provider generic projection lacks %q:\n%s", exact, printed)
		}
	}
	for _, superseded := range []string{
		"Concat<RuntimeSlice<gostring>, gostring>(",
		"Clone<gostring, int64>(",
	} {
		if strings.Contains(printed, superseded) {
			t.Fatalf("provider generic projection retained %q:\n%s", superseded, printed)
		}
	}
	waveThreeTypecheck(t, workingDirectory, artifacts.paths)
}
