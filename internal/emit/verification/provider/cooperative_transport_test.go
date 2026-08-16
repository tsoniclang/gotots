package provider_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestContextProviderProfileConstructsExactInterfaceValue(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/contextprofile\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package contextprofile

import (
	"context"
	"sync"
	"time"
)

type blockingFailure struct { mutex *sync.Mutex }

func (failure *blockingFailure) Error() string {
	failure.mutex.Lock()
	failure.mutex.Unlock()
	return "parent failed"
}

type fixedContext struct { failure error }

func (*fixedContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (*fixedContext) Done() <-chan struct{} { return nil }
func (source *fixedContext) Err() error { return source.failure }
func (*fixedContext) Value(any) any { return nil }

func Run(key string, value string) (string, bool, string) {
	background := context.Background()
	todo := context.TODO()
	parent := &fixedContext{failure: &blockingFailure{mutex: &sync.Mutex{}}}
	child := context.WithValue(parent, key, value)
	selected, ok := child.Value(key).(string)
	identities := ok && child.Value("missing") == nil &&
		background == context.Background() && todo == context.TODO() &&
		background != todo
	return selected, identities, child.Err().Error()
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
	options := emit.DefaultOptions()
	options.ConcurrencySemantics = emit.ConcurrencySemanticsCooperative
	options.StandardLibrary = linkedProviderCertificate(t)
	// The context formatting family is implemented with exact chain
	// spellings: the constructor chain compiles through the used-provider
	// closure and emits the certified profile artifacts.
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{mustProviderRoot(
			t,
			program.Roots()[0].Types().Scope().Lookup("Run"),
		)},
		options,
	)
	if err != nil {
		t.Fatalf("implemented context family failed the closure gate: %v", err)
	}
	if len(emission.Files()) == 0 {
		t.Fatal("context profile compilation emitted no target files")
	}
}

func TestCallableFieldTransportUsesOneDeclarationDecision(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/fieldtransport\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package fieldtransport

import (
	"reflect"
	"slices"
)

type closedComparer struct {
	compare func(string, string) int
}

func (*closedComparer) compareMethod(left, right string) int {
	if left < right { return -1 }
	if left > right { return 1 }
	return 0
}

func newClosedComparer() *closedComparer {
	value := &closedComparer{}
	value.compare = value.compareMethod
	return value
}

func SortClosed(source []string) {
	value := newClosedComparer()
	slices.SortFunc(source, value.compare)
}

func ReflectClosedField() bool {
	field := reflect.ValueOf(newClosedComparer()).Elem().Field(0)
	return !field.CanSet()
}

type genericComparer[T any] struct {
	compare func(T, T) int
}

func newGenericComparer[T any](
	compare func(T, T) int,
) *genericComparer[T] {
	return &genericComparer[T]{compare: compare}
}

func SortGenericOpen(
	source []string,
	compare func(string, string) int,
) {
	value := newGenericComparer(compare)
	slices.SortFunc(source, value.compare)
}

type namedCompare func(string, string) int

type namedComparer struct {
	compare namedCompare
}

func compareText(left, right string) int {
	if left < right { return -1 }
	if left > right { return 1 }
	return 0
}

func SortNamedField(source []string) {
	value := &namedComparer{compare: namedCompare(compareText)}
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
		"ReflectClosedField",
		"SortGenericOpen",
		"SortNamedField",
	} {
		roots = append(roots, mustProviderRoot(t, scope.Lookup(name)))
	}
	emission, err := emit.CompileWithOptions(program, roots, options)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	printed := artifacts.printed
	for _, required := range []string{
		"export function SortClosed(",
		"SlicesSortFuncSynchronousKernel<",
		"public compare: (($0: gostring, $1: gostring) => int) | undefined",
		"export async function SortGenericOpen(",
		"compare: (($0: T, $1: T) => Awaitable<int>) | undefined",
		"export async function SortNamedField(",
		"public compare: namedCompare",
		"export class namedCompare",
		"public readonly $value:",
		"settable: false",
		"reflect: Value.Set using unaddressable value",
	} {
		if !strings.Contains(printed, required) {
			t.Fatalf("callable-field transport lacks %q", required)
		}
	}
	if strings.Contains(printed, "instance.compare =") {
		t.Fatal("private reflected field retained an impossible setter")
	}
	waveThreeTypecheck(t, workingDirectory, artifacts.paths)
}
