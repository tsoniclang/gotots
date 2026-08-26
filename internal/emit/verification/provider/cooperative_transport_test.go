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

func CallClosed(left, right string) int { return newClosedComparer().compare(left, right) }

func DeferClosed() {
	value := newClosedComparer()
	defer value.compare("left", "right")
}

func GoClosed() {
	value := newClosedComparer()
	go value.compare("left", "right")
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

func CallOpen(
	value *openComparer,
	left, right string,
) int {
	return value.compare(left, right)
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
		"CallClosed",
		"DeferClosed",
		"GoClosed",
		"SortMethod",
		"SortExported",
		"SortOpen",
		"CallOpen",
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
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	printed := artifacts.printed
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
		"export async function CallOpen(",
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
	if !strings.Contains(
		printed,
		"public compare: (($0: gostring, $1: gostring) => int) | undefined",
	) {
		t.Fatalf("closed callback field lacks synchronous transport:\n%s", printed)
	}
	if !strings.Contains(
		printed,
		"public compare: (($0: gostring, $1: gostring) => Awaitable<int>) | undefined",
	) {
		t.Fatalf("open callback field lost canonical transport:\n%s", printed)
	}
	for _, synchronous := range []string{"CallClosed", "GoClosed"} {
		if !strings.Contains(printed, "export function "+synchronous+"(") ||
			strings.Contains(printed, "export async function "+synchronous+"(") {
			t.Fatalf(
				"closed callback invocation retained cooperative transport in %s:\n%s",
				synchronous,
				printed,
			)
		}
	}
	if !strings.Contains(printed, "export async function DeferClosed(") {
		t.Fatalf("deferred recovery transport lost its canonical effect:\n%s", printed)
	}
	if !strings.Contains(
		printed,
		"=> int) | undefined = loadPointer<closedComparer>",
	) {
		t.Fatalf("deferred callback capture lost synchronous transport:\n%s", printed)
	}
	waveThreeTypecheck(t, workingDirectory, artifacts.paths)
}
