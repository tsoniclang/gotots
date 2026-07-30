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

func ReadPool(pool *sync.Pool) any {
	return pool.Get()
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
	stopTimerRoot, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("StopTimer"),
	)
	if err != nil {
		t.Fatal(err)
	}
	renderRoot, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("Render"),
	)
	if err != nil {
		t.Fatal(err)
	}
	poolRoot, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("ReadPool"),
	)
	if err != nil {
		t.Fatal(err)
	}
	stopTickerRoot, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("StopTicker"),
	)
	if err != nil {
		t.Fatal(err)
	}
	emptyLocalPoolRoot, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("EmptyLocalPool"),
	)
	if err != nil {
		t.Fatal(err)
	}
	importLocalPoolRoot, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("ImportLocalPool"),
	)
	if err != nil {
		t.Fatal(err)
	}
	exportLocalPoolRoot, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("ExportLocalPool"),
	)
	if err != nil {
		t.Fatal(err)
	}
	unsafeRuntimeRoot, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("UnsafeRuntime"),
	)
	if err != nil {
		t.Fatal(err)
	}
	unsafeConstantsRoot, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("UnsafeConstants"),
	)
	if err != nil {
		t.Fatal(err)
	}
	runeBoundaryRoot, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("RuneBoundary"),
	)
	if err != nil {
		t.Fatal(err)
	}
	options := emit.DefaultOptions()
	options.ConcurrencySemantics = emit.ConcurrencySemanticsCooperative
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{
			renderRoot,
			poolRoot,
			stopTickerRoot,
			stopTimerRoot,
			emptyLocalPoolRoot,
			importLocalPoolRoot,
			exportLocalPoolRoot,
			unsafeRuntimeRoot,
			unsafeConstantsRoot,
			runeBoundaryRoot,
		},
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
		"export declare function Pool_Get",
		"export declare function String(",
		"export declare function Slice(",
		"export declare function StringData(",
		"export declare function SliceData(",
		"export declare const RuneSelf$uint8",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf(
				"environment artifacts lack %q:\n%s",
				required,
				artifacts.printed,
			)
		}
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
