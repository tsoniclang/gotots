package verify

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	runtimefixture "github.com/tsoniclang/gotots/internal/testfixture/gototsruntime"
	tsoniccorefixture "github.com/tsoniclang/gotots/internal/testfixture/tsoniccore"
	"github.com/tsoniclang/gotots/internal/toolchain"
)

func closureDirectory(relative string) string {
	return filepath.Join(
		"..",
		"..",
		"testdata",
		"constructs",
		"closure",
		"wave10",
		relative,
	)
}

func closureTSGoTool(t *testing.T) tsgo.Tool {
	t.Helper()
	repository := repositoryRoot(t)
	selectedGo, err := toolchain.ResolveGo(
		"",
		filepath.Join(repository, ".temp", "cache", "toolchain-tests"),
	)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := tsgo.ResolveTool(selectedGo, repository, "")
	if err != nil {
		t.Fatal(err)
	}
	return selected
}

func loadClosurePackage(
	t *testing.T,
	relative string,
) (*load.Program, *load.Package) {
	t.Helper()
	program, err := load.Load(context.Background(), load.Request{
		Directory: closureDirectory(relative),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return program, program.Roots()[0]
}

func closureRoot(
	t *testing.T,
	sourcePackage *load.Package,
	name string,
) emit.Root {
	t.Helper()
	root, err := emit.NewRoot(sourcePackage.Types().Scope().Lookup(name))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

type closureArtifacts struct {
	files            int
	bytes            int
	nodes            int
	largest          int
	printed          string
	workingDirectory string
	targetPaths      []string
	tool             tsgo.Tool
}

func materializeClosure(
	t *testing.T,
	emission emit.ProgramEmission,
) closureArtifacts {
	return materializeClosureWithSetup(t, emission, nil)
}

func materializeClosureWithSetup(
	t *testing.T,
	emission emit.ProgramEmission,
	setup func(string, emit.ProgramEmission),
) closureArtifacts {
	t.Helper()
	workingDirectory := t.TempDir()
	selectedTool := closureTSGoTool(t)
	client, err := tsgo.StartClientWithTool(selectedTool, workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	result := closureArtifacts{}
	var targetPaths []string
	var printed strings.Builder
	for _, file := range emission.Files() {
		target, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := tsgo.EncodeSourceFile(file.SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{
			": any",
			": unknown",
			" as any",
			" as unknown",
			".call(",
			".apply(",
			".bind(",
			"import(",
		} {
			if strings.Contains(target, forbidden) {
				t.Fatalf("%s contains forbidden %q", file.OutputPath(), forbidden)
			}
		}
		targetPath := filepath.Join(
			workingDirectory,
			filepath.FromSlash(file.OutputPath()),
		)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(targetPath, []byte(target), 0o644); err != nil {
			t.Fatal(err)
		}
		targetPaths = append(targetPaths, targetPath)
		result.files++
		result.bytes += len(target)
		result.nodes += encodedClosureNodes(t, encoded)
		if len(target) > result.largest {
			result.largest = len(target)
		}
		printed.WriteString(target)
		printed.WriteByte('\n')
	}
	if result.files == 0 ||
		result.bytes > 350_000 ||
		result.nodes > 60_000 ||
		result.largest > 80_000 {
		t.Fatalf(
			"Wave 10 artifact bounds: files=%d bytes=%d nodes=%d largest=%d",
			result.files,
			result.bytes,
			result.nodes,
			result.largest,
		)
	}
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--noEmit",
	}
	arguments = append(arguments, targetPaths...)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := os.WriteFile(
		filepath.Join(workingDirectory, "package.json"),
		[]byte("{\"type\":\"module\"}\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	if setup != nil {
		setup(workingDirectory, emission)
	}
	if err := tsoniccorefixture.InstallResolutionOnly(workingDirectory); err != nil {
		t.Fatal(err)
	}
	if err := runtimefixture.InstallResolution(workingDirectory, workingDirectory); err != nil {
		t.Fatal(err)
	}
	if err := tsgo.CompileWithTool(
		ctx,
		selectedTool,
		workingDirectory,
		arguments,
	); err != nil {
		t.Fatal(err)
	}
	result.printed = printed.String()
	result.workingDirectory = workingDirectory
	result.targetPaths = slices.Clone(targetPaths)
	result.tool = selectedTool
	return result
}

func executeLinkedRun(
	t *testing.T,
	emission emit.ProgramEmission,
	artifacts closureArtifacts,
	wantLiteral string,
) {
	t.Helper()
	sourceModule := ""
	for _, file := range emission.Files() {
		if file.Kind() == emit.TargetFileSource &&
			file.PackageName() == "externallinked" {
			sourceModule = "./" + strings.TrimSuffix(
				filepath.ToSlash(file.OutputPath()),
				".ts",
			) + ".js"
			break
		}
	}
	if sourceModule == "" {
		t.Fatal("linked source module is absent")
	}
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
	}
	arguments = append(arguments, artifacts.targetPaths...)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := tsgo.CompileWithTool(
		ctx,
		artifacts.tool,
		artifacts.workingDirectory,
		arguments,
	); err != nil {
		t.Fatal(err)
	}
	runner := fmt.Sprintf(
		"import { Run } from %q;\nif (Run() !== %s) throw new Error('linked result');\n",
		sourceModule,
		wantLiteral,
	)
	runnerPath := filepath.Join(artifacts.workingDirectory, "linked-runner.mjs")
	if err := os.WriteFile(runnerPath, []byte(runner), 0o644); err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(ctx, "node", runnerPath)
	command.Dir = artifacts.workingDirectory
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("execute linked function: %v\n%s", err, output)
	}
}

func executeExternalClosure(
	t *testing.T,
	emission emit.ProgramEmission,
	artifacts closureArtifacts,
	identity string,
) {
	t.Helper()
	if artifacts.workingDirectory == "" ||
		len(artifacts.targetPaths) == 0 || identity == "" {
		t.Fatal("external closure execution lacks materialized evidence")
	}
	sourceModule := ""
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSource ||
			file.PackageName() != "external" {
			continue
		}
		if sourceModule != "" {
			t.Fatal("external closure has multiple source modules")
		}
		sourceModule = "./" + strings.TrimSuffix(
			filepath.ToSlash(file.OutputPath()),
			".ts",
		) + ".js"
	}
	if sourceModule == "" {
		t.Fatal("external closure source module is absent")
	}
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
	}
	arguments = append(arguments, artifacts.targetPaths...)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := tsgo.CompileWithTool(
		ctx,
		artifacts.tool,
		artifacts.workingDirectory,
		arguments,
	); err != nil {
		t.Fatal(err)
	}
	runner := fmt.Sprintf(`import { Read } from %q;
let observed;
try {
    Read(undefined);
} catch (failure) {
    observed = failure.value.message;
}
if (observed !== %q) {
    throw new Error("external failure mismatch: " + String(observed));
}
`, sourceModule, "unresolved external Go function "+identity)
	runnerPath := filepath.Join(artifacts.workingDirectory, "external-runner.mjs")
	if err := os.WriteFile(runnerPath, []byte(runner), 0o644); err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(ctx, "node", runnerPath)
	command.Dir = artifacts.workingDirectory
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("execute external placeholder: %v\n%s", err, output)
	}
}

func encodedClosureNodes(t *testing.T, encoded []byte) int {
	t.Helper()
	const (
		headerSize       = 44
		nodesOffsetField = 40
		nodeWidth        = 28
	)
	if len(encoded) < headerSize {
		t.Fatalf("encoded target is %d bytes, want protocol header", len(encoded))
	}
	nodesOffset := int(binary.LittleEndian.Uint32(
		encoded[nodesOffsetField:headerSize],
	))
	if nodesOffset < headerSize ||
		nodesOffset > len(encoded) ||
		(len(encoded)-nodesOffset)%nodeWidth != 0 {
		t.Fatalf("encoded target has invalid node offset %d", nodesOffset)
	}
	return (len(encoded) - nodesOffset) / nodeWidth
}
