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
	"errors"
	"maps"
	"slices"
	"strings"
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

func GenericGrow[T any](source []T, count int) []T {
	return slices.Grow(source, count)
}

func GenericGrowValue[T any]() func([]T, int) []T {
	return slices.Grow[[]T, T]
}

func DeferredGenericGrow[T any](source []T, count int) {
	defer slices.Grow(source, count)
}

func SortValues[T any](source []T, compare func(T, T) int) {
	slices.SortFunc(source, compare)
}

func SortDirect(source []string) {
	slices.SortFunc(source, func(left, right string) int {
		return strings.Compare(left, right)
	})
}

func SortStableDirect(source []string) {
	slices.SortStableFunc(source, func(left, right string) int {
		return strings.Compare(left, right)
	})
}

func SortNamed(source []string) {
	slices.SortFunc(source, strings.Compare)
}

func compareLocal(left, right string) int {
	return strings.Compare(left, right)
}

func SortLocal(source []string) {
	slices.SortFunc(source, compareLocal)
}

func SortGenericNamed(source []string) {
	slices.SortFunc(source, cmp.Compare[string])
}

type stringComparer struct{}

func (stringComparer) Compare(left, right string) int {
	return strings.Compare(left, right)
}

func SortMethod(source []string) {
	slices.SortFunc(source, stringComparer{}.Compare)
}

func SortVariable(source []string) {
	compare := strings.Compare
	slices.SortFunc(source, compare)
}

func SortOpen(source []string, compare func(string, string) int) {
	slices.SortFunc(source, compare)
}

func SortCooperative(source []string, ready <-chan struct{}) {
	slices.SortFunc(source, func(left, right string) int {
		<-ready
		return strings.Compare(left, right)
	})
}

func compareRecovering(left, right string) (result int) {
	defer func() {
		if recover() != nil {
			result = 0
		}
	}()
	if left == "panic" {
		panic("comparison")
	}
	return strings.Compare(left, right)
}

func SortRecovering(source []string) {
	slices.SortFunc(source, compareRecovering)
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

func PrintValues(source []string) int {
	total := 0
	for range slices.Values(source) {
		total++
	}
	return total
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
			mustProviderRoot(t, scope.Lookup("GenericGrow")),
			mustProviderRoot(t, scope.Lookup("GenericGrowValue")),
			mustProviderRoot(t, scope.Lookup("DeferredGenericGrow")),
			mustProviderRoot(t, scope.Lookup("SortValues")),
			mustProviderRoot(t, scope.Lookup("SortDirect")),
			mustProviderRoot(t, scope.Lookup("SortStableDirect")),
			mustProviderRoot(t, scope.Lookup("SortNamed")),
			mustProviderRoot(t, scope.Lookup("SortLocal")),
			mustProviderRoot(t, scope.Lookup("SortGenericNamed")),
			mustProviderRoot(t, scope.Lookup("SortMethod")),
			mustProviderRoot(t, scope.Lookup("SortVariable")),
			mustProviderRoot(t, scope.Lookup("SortOpen")),
			mustProviderRoot(t, scope.Lookup("SortCooperative")),
			mustProviderRoot(t, scope.Lookup("SortRecovering")),
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
		"MapsCloneKernel<GoMapValue<gostring, int>, gostring, int>(",
		"SlicesGrowKernel<RuntimeSlice<gostring>, gostring, gostring>(",
		"SlicesCloneKernel<GoContainerStorage<T>>(",
		"SlicesCloneKernel<GoContainerStorage<T>>($argument0)",
		"CmpCompareKernel<gostring>(",
		"SlicesValuesKernel<RuntimeSlice<gostring>, gostring, gostring>(",
		"ErrorsAsTypeKernel<code__from_providerprojection>($goCapability_",
		"GenericAddress$kernel<T>",
		"SlicesSortFuncSynchronousKernel<RuntimeSlice<gostring>, gostring, gostring>(",
		"SlicesSortStableFuncSynchronousKernel<RuntimeSlice<gostring>, gostring, gostring>(",
		"export function SortDirect(source: RuntimeSlice<gostring>): void",
		"export function SortStableDirect(source: RuntimeSlice<gostring>): void",
		"export function SortNamed(source: RuntimeSlice<gostring>): void",
		"export function SortLocal(source: RuntimeSlice<gostring>): void",
		"export function SortGenericNamed(source: RuntimeSlice<gostring>): void",
		"export async function SortVariable(",
		"export async function SortMethod(",
		"export async function SortOpen(",
		"export async function SortCooperative(",
		"export function SortRecovering(source: RuntimeSlice<gostring>): void",
		"Pointer<T>",
		"addressOf<GoArray<GoContainerStorage<T>, 1>>(values)",
		"BigInt.asIntN(64, goNumberToBigInt(count))",
		"BigInt.asIntN(64, goNumberToBigInt($argument1))",
		"BigInt.asIntN(64, goNumberToBigInt(__gotots_argument_1))",
	} {
		if !strings.Contains(printed, exact) {
			t.Fatalf("provider generic projection lacks %q:\n%s", exact, printed)
		}
	}
	for _, superseded := range []string{
		"Concat<RuntimeSlice<gostring>, gostring>(",
		"Clone<gostring, int64>(",
		"SlicesValuesCooperative<",
		"SlicesValuesFullyCooperative<",
		"export async function SortDirect(",
		"export async function SortStableDirect(",
		"export async function SortNamed(",
		"export async function SortLocal(",
		"export async function SortGenericNamed(",
		"export async function SortRecovering(",
	} {
		if strings.Contains(printed, superseded) {
			t.Fatalf("provider generic projection retained %q:\n%s", superseded, printed)
		}
	}
	waveThreeTypecheck(t, workingDirectory, artifacts.paths)
}
