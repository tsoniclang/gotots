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

func TestEnvironmentGenericCallableContractIsCanonicalAndBodyless(
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
	contractCount := 0
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
			if name == "Values" {
				contractCount++
				if function.Body() != nil {
					t.Fatal("environment callable contract acquired a body")
				}
			}
		}
	}
	if contractCount != 1 {
		t.Fatalf(
			"environment Values declarations=%d, want 1:\n%s",
			contractCount,
			artifacts.printed,
		)
	}
	if strings.Contains(artifacts.printed, "Values$cooperative_") {
		t.Fatalf("environment callable profile survived:\n%s", artifacts.printed)
	}
	if !strings.Contains(artifacts.printed, "Awaitable<bool>") {
		t.Fatalf(
			"environment contract lacks canonical cooperative yield ABI:\n%s",
			artifacts.printed,
		)
	}
	contract := environmentDeclarationLine(
		t,
		artifacts.printed,
		"export declare function Values<",
	)
	if !strings.Contains(
		contract,
		"export declare function Values<$T0, $T1>($argument0: $T0): Seq__from_iter<$T1>;",
	) {
		t.Fatalf(
			"environment source contract is not source-shaped:\n%s",
			contract,
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

func TestInterfaceMethodAwaitableABIIsStableAcrossUnrelatedValues(t *testing.T) {
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
	for _, required := range []string{
		"export async function DirectSynchronousInterface(): Promise<int32>",
		"return await goInterfaceNonNil<Reader>",
	} {
		if !strings.Contains(baselineCall, required) {
			t.Fatalf("interface call lacks %q:\n%s", required, baselineCall)
		}
	}
	if !strings.Contains(baselineContract, "Next(): Awaitable<int32>;") {
		t.Fatalf("interface contract lacks canonical Awaitable ABI:\n%s", baselineContract)
	}
	if strings.Contains(baselineContract, "Next(): Promise<int32>;") {
		t.Fatal("interface contract retained a blocking-only method profile")
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
