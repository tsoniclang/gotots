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

func BinarySearchDirect(source []string, target string) (int, bool) {
	return slices.BinarySearchFunc(source, target, strings.Compare)
}

func CompareDirect(left, right []string) int {
	return slices.CompareFunc(left, right, strings.Compare)
}

func ContainsDirect(source []string) bool {
	return slices.ContainsFunc(source, func(value string) bool {
		return len(value) > 2
	})
}

func EqualDirect(left, right []string) bool {
	return slices.EqualFunc(left, right, strings.EqualFold)
}

func IndexDirect(source []string) int {
	return slices.IndexFunc(source, func(value string) bool {
		return len(value) > 2
	})
}

func CompactDirect(source []string) []string {
	return slices.CompactFunc(source, strings.EqualFold)
}

func DeleteDirect(source []string) []string {
	return slices.DeleteFunc(source, func(value string) bool {
		return len(value) > 2
	})
}

func MapsEqualDirect(left, right map[string]string) bool {
	return maps.EqualFunc(left, right, strings.EqualFold)
}

func ContainsOpen(source []string, predicate func(string) bool) bool {
	return slices.ContainsFunc(source, predicate)
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
			mustProviderRoot(t, scope.Lookup("BinarySearchDirect")),
			mustProviderRoot(t, scope.Lookup("CompareDirect")),
			mustProviderRoot(t, scope.Lookup("ContainsDirect")),
			mustProviderRoot(t, scope.Lookup("EqualDirect")),
			mustProviderRoot(t, scope.Lookup("IndexDirect")),
			mustProviderRoot(t, scope.Lookup("CompactDirect")),
			mustProviderRoot(t, scope.Lookup("DeleteDirect")),
			mustProviderRoot(t, scope.Lookup("MapsEqualDirect")),
			mustProviderRoot(t, scope.Lookup("ContainsOpen")),
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
		"ErrorsAsTypeKernel<code__from_providerprojection>(($argument0:",
		"GenericAddress$kernel<T>",
		"SlicesSortFuncSynchronousKernel<RuntimeSlice<gostring>, gostring, gostring>(",
		"SlicesSortStableFuncSynchronousKernel<RuntimeSlice<gostring>, gostring, gostring>(",
		"SlicesBinarySearchFuncSynchronousKernel<",
		"SlicesCompareFuncSynchronousKernel<",
		"SlicesContainsFuncSynchronousKernel<",
		"SlicesEqualFuncSynchronousKernel<",
		"SlicesIndexFuncSynchronousKernel<",
		"SlicesCompactFuncSynchronousKernel<",
		"SlicesDeleteFuncSynchronousKernel<",
		"MapsEqualFuncSynchronousKernel<",
		"SlicesContainsFuncKernel<",
		"export function SortDirect(source: RuntimeSlice<gostring>): void",
		"export function SortStableDirect(source: RuntimeSlice<gostring>): void",
		"export function SortNamed(source: RuntimeSlice<gostring>): void",
		"export function SortLocal(source: RuntimeSlice<gostring>): void",
		"export function SortGenericNamed(source: RuntimeSlice<gostring>): void",
		"export function SortMethod(source: RuntimeSlice<gostring>): void",
		"export async function SortVariable(",
		"export async function SortOpen(",
		"export async function SortCooperative(",
		"export function SortRecovering(source: RuntimeSlice<gostring>): void",
		"export function BinarySearchDirect(",
		"export function CompareDirect(",
		"export function ContainsDirect(",
		"export function EqualDirect(",
		"export function IndexDirect(",
		"export function CompactDirect(",
		"export function DeleteDirect(",
		"export function MapsEqualDirect(",
		"export async function ContainsOpen(",
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
	for _, forbidden := range []string{
		"$goCapability_",
		"support/generics/capabilities/",
	} {
		if strings.Contains(printed, forbidden) {
			t.Fatalf("provider generic projection retains %q:\n%s", forbidden, printed)
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
		"export async function BinarySearchDirect(",
		"export async function CompareDirect(",
		"export async function ContainsDirect(",
		"export async function EqualDirect(",
		"export async function IndexDirect(",
		"export async function CompactDirect(",
		"export async function DeleteDirect(",
		"export async function MapsEqualDirect(",
		"SlicesBinarySearchFuncKernel<RuntimeSlice<gostring>",
		"SlicesCompareFuncKernel<RuntimeSlice<gostring>",
		"SlicesEqualFuncKernel<RuntimeSlice<gostring>",
		"SlicesIndexFuncKernel<RuntimeSlice<gostring>",
		"SlicesCompactFuncKernel<RuntimeSlice<gostring>",
		"SlicesDeleteFuncKernel<RuntimeSlice<gostring>",
		"MapsEqualFuncKernel<GoMapValue<gostring, gostring>",
	} {
		if strings.Contains(printed, superseded) {
			t.Fatalf("provider generic projection retained %q:\n%s", superseded, printed)
		}
	}
	waveThreeTypecheck(t, workingDirectory, artifacts.paths)
}

func TestGenericSynchronousCallbackSelectionUsesExactFieldEvidence(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/synchronousfield\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package synchronousfield

import "slices"

type closedComparer struct {
	compare func(string, string) int
}

func newClosedComparer() *closedComparer {
	value := &closedComparer{}
	value.compare = nil
	value.compare = value.compareMethod
	return value
}

func (*closedComparer) compareMethod(left, right string) int {
	if left < right { return -1 }
	if left > right { return 1 }
	return 0
}

func SortClosed(source []string) {
	value := newClosedComparer()
	slices.SortFunc(source, value.compare)
}

type directComparer struct{}

func (directComparer) compare(left, right string) int {
	if left < right { return -1 }
	if left > right { return 1 }
	return 0
}

func SortMethod(source []string) {
	slices.SortFunc(source, directComparer{}.compare)
}

type exportedComparer struct {
	Compare func(string, string) int
}

func SortExported(source []string, value *exportedComparer) {
	slices.SortFunc(source, value.Compare)
}

type openComparer struct {
	compare func(string, string) int
}

func SortOpen(source []string, compare func(string, string) int) {
	value := &openComparer{compare: compare}
	slices.SortFunc(source, value.compare)
}

type cooperativeComparer struct {
	compare func(string, string) int
}

func (value *cooperativeComparer) compareMethod(
	left, right string,
) int {
	ready := make(chan struct{})
	<-ready
	return 0
}

func SortCooperative(source []string, value *cooperativeComparer) {
	value.compare = value.compareMethod
	slices.SortFunc(source, value.compare)
}

type literalComparer struct {
	compare func(string, string) int
}

func newLiteralComparer() *literalComparer {
	value := &literalComparer{}
	value.compare = func(left, right string) int {
		ready := make(chan struct{})
		<-ready
		return 0
	}
	return value
}

func SortLiteral(source []string) {
	value := newLiteralComparer()
	slices.SortFunc(source, value.compare)
}

type addressedComparer struct {
	compare func(string, string) int
}

func address(value *addressedComparer) *func(string, string) int {
	return &value.compare
}

func SortAddressed(source []string, value *addressedComparer) {
	value.compare = directComparer{}.compare
	slices.SortFunc(source, value.compare)
}

type comparer interface {
	compare(string, string) int
}

type interfaceComparer struct {
	compare func(string, string) int
}

func SortInterface(source []string, provider comparer) {
	value := &interfaceComparer{compare: provider.compare}
	slices.SortFunc(source, value.compare)
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
	var roots []emit.Root
	for _, name := range []string{
		"SortClosed",
		"SortMethod",
		"SortExported",
		"SortOpen",
		"SortCooperative",
		"SortLiteral",
		"SortAddressed",
		"SortInterface",
	} {
		roots = append(roots, mustProviderRoot(t, scope.Lookup(name)))
	}
	emission, err := emit.CompileWithOptions(program, roots, options)
	if err != nil {
		t.Fatal(err)
	}
	printed := materializeArtifacts(t, emission, t.TempDir()).printed
	for _, synchronous := range []string{
		"export function SortClosed(",
		"export function SortMethod(",
	} {
		if !strings.Contains(printed, synchronous) {
			t.Fatalf("closed synchronous callback lacks %q:\n%s", synchronous, printed)
		}
	}
	for _, canonical := range []string{
		"export async function SortExported(",
		"export async function SortOpen(",
		"export async function SortCooperative(",
		"export async function SortLiteral(",
		"export async function SortAddressed(",
		"export async function SortInterface(",
	} {
		if !strings.Contains(printed, canonical) {
			t.Fatalf("open callback lacks %q:\n%s", canonical, printed)
		}
	}
	if !strings.Contains(printed, "SlicesSortFuncSynchronousKernel<") {
		t.Fatalf("closed callback lacks synchronous provider kernel:\n%s", printed)
	}
}
