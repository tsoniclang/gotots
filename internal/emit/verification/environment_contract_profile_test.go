package emit_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestEnvironmentContractsPrintAndStrictTypecheckWithoutImplementations(
	t *testing.T,
) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/environmentcontract\n\ngo 1.26.4\n",
	)
	writeProgramFile(
		t,
		filepath.Join(project, "source.go"),
		`package environmentcontract

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
	"unicode/utf8"
	"unsafe"
)

func Render(value context.Context) string {
	if err := value.Err(); err != nil {
		return fmt.Sprint(err)
	}
	return os.Args[0]
}

type signal struct {
	ready <-chan struct{}
}

func (*signal) Signal() {}

func (s *signal) String() string {
	<-s.ready
	return "ready"
}

func NewSignal(ready <-chan struct{}) os.Signal {
	return &signal{ready: ready}
}

func RunWaitGroup(ready <-chan struct{}) {
	var group sync.WaitGroup
	group.Go(func() {
		<-ready
	})
	group.Wait()
}

func ReadPool(pool *sync.Pool, values []sync.Pool) any {
	result := pool.Get()
	_ = (&values[0]).Get()
	return result
}

func StopTicker(ticker *time.Ticker) {
	defer ticker.Stop()
}

func StopTimer(timer *time.Timer) bool {
	return timer.Stop()
}

type LocalPool sync.Pool

func EmptyLocalPool() LocalPool {
	return LocalPool{}
}

func ImportLocalPool(pool sync.Pool) LocalPool {
	return LocalPool(pool)
}

func ExportLocalPool(pool LocalPool) sync.Pool {
	return sync.Pool(pool)
}

type UnsafePair struct {
	First uint32
	Last  byte
}

func UnsafeRuntime(
	bytes []byte,
	text string,
) (string, []byte, *byte, *byte) {
	var pointer *byte
	if len(bytes) != 0 {
		pointer = &bytes[0]
	}
	return unsafe.String(pointer, len(bytes)),
		unsafe.Slice(pointer, len(bytes)),
		unsafe.StringData(text),
		unsafe.SliceData(bytes)
}

func UnsafeConstants() (uintptr, uintptr, uintptr) {
	var value UnsafePair
	return unsafe.Sizeof(value),
		unsafe.Alignof(value),
		unsafe.Offsetof(value.Last)
}

func RuneBoundary(value byte) bool {
	return value < utf8.RuneSelf
}
`,
	)
	program, err := load.Load(context.Background(), load.Request{
		Directory: project,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	scope := program.Roots()[0].Types().Scope()
	rootNames := []string{
		"Render",
		"NewSignal",
		"RunWaitGroup",
		"ReadPool",
		"StopTicker",
		"StopTimer",
		"EmptyLocalPool",
		"ImportLocalPool",
		"ExportLocalPool",
		"UnsafeRuntime",
		"UnsafeConstants",
		"RuneBoundary",
	}
	roots := make([]emit.Root, 0, len(rootNames))
	for _, name := range rootNames {
		root, rootErr := emit.NewRoot(scope.Lookup(name))
		if rootErr != nil {
			t.Fatal(rootErr)
		}
		roots = append(roots, root)
	}
	options := emit.DefaultOptions()
	options.ConcurrencySemantics = emit.ConcurrencySemanticsCooperative
	emission, err := emit.CompileWithOptions(
		program,
		roots,
		options,
	)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	contracts := 0
	var tickerStop tsgo.FunctionDeclaration
	var timerStop tsgo.FunctionDeclaration
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileEnvironmentContract {
			continue
		}
		contracts++
		for _, statement := range file.SourceFile().Statements() {
			function, ok := statement.(tsgo.FunctionDeclaration)
			if ok && function.Body() != nil {
				t.Fatalf(
					"environment function %s carries an implementation",
					function.Name().Text(),
				)
			}
			if !ok {
				continue
			}
			switch function.Name().Text() {
			case "Ticker_Stop":
				tickerStop = function
			case "Timer_Stop":
				timerStop = function
			}
		}
	}
	assertEnvironmentRecoveryFacet(t, tickerStop, true)
	assertEnvironmentRecoveryFacet(t, timerStop, false)
	if contracts < 4 {
		t.Fatalf("environment contract files = %d, want context/fmt/os/sync", contracts)
	}
	for _, required := range []string{
		"export declare function Sprint",
		"export declare const $state",
		"export interface Context",
		"export declare const Context$contract",
		"export declare function Context$is",
		"export declare class Pool",
		"export declare function Pool_Get($receiver: GoPointer<Pool",
		"export declare function String(",
		"export declare function Slice(",
		"export declare function StringData(",
		"export declare function SliceData(",
		"export declare const RuneSelf$uint8",
		"static async String(",
		"async String(",
		"export interface Signal",
		"String(): Promise<gostring>;",
		"WaitGroup_Go__from_sync",
		"async function (): Promise<void>",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf(
				"environment artifacts lack %q:\n%s",
				required,
				artifacts.printed,
			)
		}
	}
	waitGroupGo := environmentDeclarationLine(
		t,
		artifacts.printed,
		"export declare function WaitGroup_Go(",
	)
	if !strings.Contains(waitGroupGo, "=> Promise<void>") {
		t.Fatalf(
			"cooperative environment callback stayed synchronous:\n%s",
			waitGroupGo,
		)
	}
	for _, forbidden := range []string{
		"export declare function Sizeof(",
		"export declare function Alignof(",
		"export declare function Offsetof(",
		"export declare const RuneSelf:",
	} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatalf(
				"checker-folded unsafe builtin acquired an ambient call %q:\n%s",
				forbidden,
				artifacts.printed,
			)
		}
	}
	for _, privateField := range []string{
		"localSize",
		"victimSize",
	} {
		if strings.Contains(artifacts.printed, privateField) {
			t.Fatalf(
				"environment artifacts expose provider-private field %q:\n%s",
				privateField,
				artifacts.printed,
			)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := tsgo.Compile(
		ctx,
		repositoryRoot(),
		workingDirectory,
		append(
			[]string{
				"--target", "es2022",
				"--module", "nodenext",
				"--moduleResolution", "nodenext",
				"--strict",
				"--noEmit",
			},
			artifacts.paths...,
		),
	); err != nil {
		t.Fatal(err)
	}
}

func assertEnvironmentRecoveryFacet(
	t *testing.T,
	function tsgo.FunctionDeclaration,
	want bool,
) {
	t.Helper()
	if function == nil {
		t.Fatal("environment Stop method is absent")
	}
	found := false
	for _, parameter := range function.Parameters() {
		name, ok := parameter.Name().(tsgo.Identifier)
		if !ok || name.Text() != "$go$recovery" {
			continue
		}
		found = true
		if parameter.QuestionToken() == nil {
			t.Fatalf(
				"environment function %s recovery authority is required",
				function.Name().Text(),
			)
		}
	}
	if found != want {
		t.Fatalf(
			"environment function %s recovery facet = %t, want %t",
			function.Name().Text(),
			found,
			want,
		)
	}
}

func TestEnvironmentGenericCallableProfileIsTypedAndBodyless(
	t *testing.T,
) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/environmentgeneric\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package environmentgeneric

import "slices"

func Sum(values []int32, input <-chan int32) int32 {
	var total int32
	for value := range slices.Values(values) {
		total += value + <-input
	}
	return total
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: project,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("Sum"),
	)
	if err != nil {
		t.Fatal(err)
	}
	options := emit.DefaultOptions()
	options.ConcurrencySemantics = emit.ConcurrencySemanticsCooperative
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{root},
		options,
	)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	baseCount := 0
	profileCount := 0
	profileName := ""
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileEnvironmentContract {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			function, ok := statement.(tsgo.FunctionDeclaration)
			if !ok || function.Name() == nil {
				continue
			}
			name := function.Name().Text()
			switch {
			case name == "Values":
				baseCount++
			case strings.HasPrefix(name, "Values$cooperative_"):
				profileCount++
				profileName = name
				if function.Body() != nil {
					t.Fatal("environment callable profile acquired a body")
				}
			}
		}
	}
	if baseCount != 1 || profileCount != 1 {
		t.Fatalf(
			"environment Values declarations base=%d profile=%d, want 1/1:\n%s",
			baseCount,
			profileCount,
			artifacts.printed,
		)
	}
	if !strings.Contains(artifacts.printed, profileName) {
		t.Fatalf(
			"environment callable profile is absent:\n%s",
			artifacts.printed,
		)
	}
	if !strings.Contains(artifacts.printed, "Promise<bool>") {
		t.Fatalf(
			"environment profile declaration lacks cooperative yield ABI:\n%s",
			artifacts.printed,
		)
	}
	base := environmentDeclarationLine(
		t,
		artifacts.printed,
		"export declare function Values<",
	)
	if strings.Contains(base, "Promise<") {
		t.Fatalf(
			"environment base declaration was widened:\n%s",
			base,
		)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := tsgo.Compile(
		ctx,
		repositoryRoot(),
		workingDirectory,
		append(
			[]string{
				"--target", "es2022",
				"--module", "nodenext",
				"--moduleResolution", "nodenext",
				"--strict",
				"--noEmit",
			},
			artifacts.paths...,
		),
	); err != nil {
		t.Fatal(err)
	}
}

func environmentDeclarationLine(
	t *testing.T,
	printed string,
	prefix string,
) string {
	t.Helper()
	start := strings.Index(printed, prefix)
	if start < 0 {
		t.Fatalf("environment declaration lacks %q:\n%s", prefix, printed)
	}
	end := strings.IndexByte(printed[start:], '\n')
	if end < 0 {
		return printed[start:]
	}
	return printed[start : start+end]
}

func TestInterfaceMethodTokenIsolatedFromUnrelatedValueABI(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveNineConcurrencyDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	scope := program.Roots()[0].Types().Scope()
	direct, err := emit.NewRoot(scope.Lookup("DirectSynchronousInterface"))
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := emit.CompileWithOptions(
		program,
		[]emit.Root{direct},
		waveNineOptions(),
	)
	if err != nil {
		t.Fatal(err)
	}
	unrelated, err := emit.NewRoot(scope.Lookup("AggregateClosures"))
	if err != nil {
		t.Fatal(err)
	}
	expanded, err := emit.CompileWithOptions(
		program,
		[]emit.Root{direct, unrelated},
		waveNineOptions(),
	)
	if err != nil {
		t.Fatal(err)
	}
	baselineDirectory := t.TempDir()
	expandedDirectory := t.TempDir()
	baselineArtifacts := materializeArtifacts(
		t,
		baseline,
		baselineDirectory,
	)
	expandedArtifacts := materializeArtifacts(
		t,
		expanded,
		expandedDirectory,
	)
	waveThreeTypecheck(
		t,
		baselineDirectory,
		baselineArtifacts.paths,
	)
	waveThreeTypecheck(
		t,
		expandedDirectory,
		expandedArtifacts.paths,
	)
	baselineCall := strings.TrimSpace(waveNineFunctionText(
		t,
		baselineArtifacts.printed,
		"DirectSynchronousInterface",
	))
	expandedCall := strings.TrimSpace(waveNineFunctionText(
		t,
		expandedArtifacts.printed,
		"DirectSynchronousInterface",
	))
	if baselineCall != expandedCall {
		t.Fatalf(
			"unrelated func() int32 ABI changed direct interface dispatch\nbaseline:\n%s\nexpanded:\n%s",
			baselineCall,
			expandedCall,
		)
	}
	baselineContract := strings.TrimSpace(interfaceDeclarationText(
		t,
		baselineArtifacts.printed,
		"Reader",
	))
	expandedContract := strings.TrimSpace(interfaceDeclarationText(
		t,
		expandedArtifacts.printed,
		"Reader",
	))
	if baselineContract != expandedContract {
		t.Fatalf(
			"unrelated func() int32 ABI changed interface contract\nbaseline:\n%s\nexpanded:\n%s",
			baselineContract,
			expandedContract,
		)
	}
	for _, target := range []string{baselineCall, baselineContract} {
		for _, forbidden := range []string{"async", "Promise<", "await "} {
			if strings.Contains(target, forbidden) {
				t.Fatalf(
					"synchronous interface artifact acquired %q:\n%s",
					forbidden,
					target,
				)
			}
		}
	}
	if !strings.Contains(expandedArtifacts.printed, "Promise<int32>") {
		t.Fatal("unrelated cooperative func() int32 ABI was not selected")
	}
}

func interfaceDeclarationText(
	t *testing.T,
	printed string,
	name string,
) string {
	t.Helper()
	start := strings.Index(printed, "export interface "+name+" ")
	if start < 0 {
		t.Fatalf("Wave 9 artifacts lack interface %s", name)
	}
	end := strings.Index(printed[start:], "\n}")
	if end < 0 {
		t.Fatalf("Wave 9 interface %s is unterminated", name)
	}
	return printed[start : start+end+2]
}
