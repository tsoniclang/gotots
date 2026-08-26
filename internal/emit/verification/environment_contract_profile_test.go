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
	runtimefixture "github.com/tsoniclang/gotots/internal/testfixture/gototsruntime"
	corefixture "github.com/tsoniclang/gotots/internal/testfixture/tsoniccore"
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
	"encoding/binary"
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

func UnsafeText(
	bytes []byte,
) string {
	if len(bytes) == 0 {
		return ""
	}
	return unsafe.String(&bytes[0], len(bytes))
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

func NativeUint16(value []byte) uint16 {
	return binary.NativeEndian.Uint16(value)
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
		"UnsafeText",
		"UnsafeConstants",
		"RuneBoundary",
		"NativeUint16",
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
	var tickerStopDeferred tsgo.FunctionDeclaration
	var timerStop tsgo.FunctionDeclaration
	var timerStopDeferred tsgo.FunctionDeclaration
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
			case "Ticker_Stop$deferred":
				tickerStopDeferred = function
			case "Timer_Stop":
				timerStop = function
			case "Timer_Stop$deferred":
				timerStopDeferred = function
			}
		}
	}
	assertEnvironmentRecoveryFacet(t, tickerStop, tickerStopDeferred, false)
	assertEnvironmentRecoveryFacet(t, timerStop, timerStopDeferred, false)
	if contracts < 4 {
		t.Fatalf("environment contract files = %d, want context/fmt/os/sync", contracts)
	}
	for _, required := range []string{
		"export declare function Sprint",
		"export declare const $state",
		"export interface Context",
		"Err():",
		"export declare const Context$contract",
		"export declare function Context$is",
		"export declare class Pool",
		"export declare function Pool_Get($receiver: Pointer<Pool>",
		"export function goUnsafeString<",
		"export declare const RuneSelf$uint8",
		"littleEndian: littleEndian;",
		".littleEndian",
		"static String(",
		"String(): gostring;",
		"export interface Signal",
		"WaitGroup_Go",
		"() => void",
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
	if !strings.Contains(waitGroupGo, "=> void") {
		t.Fatalf(
			"environment callback is not synchronous:\n%s",
			waitGroupGo,
		)
	}
	for _, forbidden := range []string{
		"registerMethod(",
		"resolveMethod(",
		"registerCooperativeMethod(",
		"resolveCooperativeMethod(",
		"async ",
		"await ",
		"Promise<",
		"Awaitable<",
		"export declare function WaitGroup_Go__from_sync(",
		"export declare function String(",
		"export declare function Slice(",
		"export declare function StringData(",
		"export declare function SliceData(",
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
	if err := corefixture.InstallResolutionOnly(workingDirectory); err != nil {
		t.Fatal(err)
	}
	if err := runtimefixture.InstallResolution(workingDirectory, filepath.Join(workingDirectory, "out")); err != nil {
		t.Fatal(err)
	}
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
	ordinary tsgo.FunctionDeclaration,
	deferred tsgo.FunctionDeclaration,
	want bool,
) {
	t.Helper()
	if ordinary == nil {
		t.Fatal("environment Stop method is absent")
	}
	for _, parameter := range ordinary.Parameters() {
		name, ok := parameter.Name().(tsgo.Identifier)
		if ok && name.Text() == "$go$recovery" {
			t.Fatalf(
				"ordinary environment function %s exposes recovery authority",
				ordinary.Name().Text(),
			)
		}
	}
	if (deferred != nil) != want {
		t.Fatalf(
			"environment function %s private deferred entry = %t, want %t",
			ordinary.Name().Text(),
			deferred != nil,
			want,
		)
	}
	if deferred == nil {
		return
	}
	if len(deferred.Parameters()) != len(ordinary.Parameters())+1 {
		t.Fatalf("private deferred entry parameter count drifted")
	}
	name, ok := deferred.Parameters()[0].Name().(tsgo.Identifier)
	if !ok || name.Text() != "$go$recovery" ||
		deferred.Parameters()[0].QuestionToken() != nil {
		t.Fatal("private deferred entry lacks required recovery authority")
	}
}
