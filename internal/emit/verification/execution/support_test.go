package emit_test

import (
	"context"
	"encoding/binary"
	"fmt"
	"go/ast"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	corefixture "github.com/tsoniclang/gotots/internal/testfixture/tsoniccore"
)

func loadDemandProgram(t *testing.T) *load.Program {
	t.Helper()
	program, err := load.Load(context.Background(), load.Request{
		Directory: demandProgramDirectory(),
		Pattern:   "./api",
	})
	if err != nil {
		t.Fatal(err)
	}
	return program
}

func executeDemandTypeScript(
	t *testing.T,
	workingDirectory string,
	targetPaths []string,
	files []emit.TargetFile,
) string {
	t.Helper()
	writeProgramFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
	var apiFile emit.TargetFile
	for _, file := range files {
		if file.Kind() == emit.TargetFilePackageAssembly &&
			file.PackageName() == "api" {
			apiFile = file
			break
		}
	}
	if apiFile.SourceFile() == nil {
		t.Fatal("emitted api file is absent")
	}
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	module := "./" + strings.TrimSuffix(apiFile.OutputPath(), ".ts") + ".js"
	writeProgramFile(t, runnerPath, `import "./program.js";
import { Run } from "`+module+`";

console.log(Run(0));
console.log(Run(1));
	console.log(Run(4));
`)
	outputDirectory := filepath.Join(workingDirectory, "out")
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--outDir", outputDirectory,
	}
	arguments = append(arguments, targetPaths...)
	arguments = append(arguments, runnerPath)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := tsgo.Compile(
		ctx,
		repositoryRoot(),
		workingDirectory,
		arguments,
	); err != nil {
		t.Fatal(err)
	}
	return runProgram(t, workingDirectory, "node", filepath.Join(outputDirectory, "runner.js"))
}

func runProgram(t *testing.T, directory, name string, arguments ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, name, arguments...)
	command.Dir = directory
	command.Env = append(os.Environ(), "GOMEMLIMIT=1GiB")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(arguments, " "), err, output)
	}
	return string(output)
}

func requireNativeGoEvidence(t *testing.T, output string) {
	t.Helper()
	if output == "" {
		t.Fatal("native Go fixture produced no evidence")
	}
}

func writeProgramFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func demandProgramDirectory() string {
	return filepath.Join(repositoryRoot(), "testdata", "projects", "demand-program")
}

func repositoryRoot() string {
	return filepath.Join("..", "..", "..", "..")
}

func waveThreeTypecheck(
	t *testing.T,
	workingDirectory string,
	paths []string,
) {
	t.Helper()
	if err := corefixture.InstallResolutionOnly(workingDirectory); err != nil {
		t.Fatal(err)
	}
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--noFallthroughCasesInSwitch",
		"--noUncheckedIndexedAccess",
		"--outDir", filepath.Join(workingDirectory, "out"),
	}
	arguments = append(arguments, paths...)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := tsgo.Compile(
		ctx,
		repositoryRoot(),
		workingDirectory,
		arguments,
	); err != nil {
		t.Fatal(err)
	}
}

type waveFourArtifacts struct {
	paths         []string
	sourceModule  string
	bytes         int
	nodes         int
	largest       int
	sizes         []artifactSize
	printed       string
	printedByKind map[emit.TargetFileKind][]string
}

type artifactSize struct {
	path  string
	bytes int
	nodes int
}

func materializeArtifacts(
	t *testing.T,
	emission emit.ProgramEmission,
	workingDirectory string,
) waveFourArtifacts {
	t.Helper()
	if err := corefixture.InstallResolutionOnly(workingDirectory); err != nil {
		t.Fatal(err)
	}
	client, err := tsgo.StartClient(repositoryRoot(), workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	result := waveFourArtifacts{
		printedByKind: make(map[emit.TargetFileKind][]string),
	}
	for _, file := range emission.Files() {
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := tsgo.EncodeSourceFile(file.SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		nodes := waveFourEncodedNodes(t, encoded)
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
			if strings.Contains(printed, forbidden) {
				t.Fatalf(
					"%s contains forbidden %q:\n%s",
					file.OutputPath(),
					forbidden,
					printed,
				)
			}
		}
		targetPath := filepath.Join(
			workingDirectory,
			filepath.FromSlash(file.OutputPath()),
		)
		writeProgramFile(t, targetPath, printed)
		result.paths = append(result.paths, targetPath)
		result.bytes += len(printed)
		result.nodes += nodes
		result.printed += "\n// " + file.OutputPath() + "\n" + printed
		result.printedByKind[file.Kind()] = append(
			result.printedByKind[file.Kind()],
			printed,
		)
		result.sizes = append(result.sizes, artifactSize{
			path:  file.OutputPath(),
			bytes: len(printed),
			nodes: nodes,
		})
		if file.Kind() == emit.TargetFileSource {
			result.sourceModule = "./" +
				strings.TrimSuffix(file.OutputPath(), ".ts") +
				".js"
		}
	}
	if result.sourceModule == "" || len(result.sizes) == 0 {
		t.Fatal("Wave 4 fixture emitted no source module")
	}
	sort.Slice(result.sizes, func(left, right int) bool {
		if result.sizes[left].bytes != result.sizes[right].bytes {
			return result.sizes[left].bytes > result.sizes[right].bytes
		}
		return result.sizes[left].path < result.sizes[right].path
	})
	result.largest = result.sizes[0].bytes
	return result
}

func targetFunctionText(t *testing.T, printed, name string) string {
	t.Helper()
	startMarker := "export function " + name + "("
	start := strings.Index(printed, startMarker)
	if start < 0 {
		t.Fatalf("Wave 4 artifacts lack function %s", name)
	}
	remainder := printed[start+len(startMarker):]
	end := strings.Index(remainder, "\nexport function ")
	artifactEnd := strings.Index(remainder, "\n\n// ")
	if end < 0 || artifactEnd >= 0 && artifactEnd < end {
		end = artifactEnd
	}
	if end < 0 {
		return printed[start:]
	}
	return printed[start : start+len(startMarker)+end]
}

func packageAssemblyExports(
	files []emit.TargetFile,
	packageName string,
	name string,
) bool {
	for _, file := range files {
		if file.Kind() != emit.TargetFilePackageAssembly ||
			file.PackageName() != packageName {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			declaration, ok := statement.(tsgo.ExportDeclaration)
			if !ok {
				continue
			}
			exports, ok := declaration.ExportClause().(tsgo.NamedExports)
			if !ok {
				continue
			}
			for _, specifier := range exports.Elements() {
				identifier, ok := specifier.Name().(tsgo.Identifier)
				if ok && identifier.Text() == name {
					return true
				}
			}
		}
	}
	return false
}

func waveFourEncodedNodes(t *testing.T, encoded []byte) int {
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

func assertWaveFourLinearDoubling(
	t *testing.T,
	name string,
	values []int,
) {
	t.Helper()
	first := values[1] - values[0]
	second := values[2] - values[1]
	if first <= 0 || second*10 < first*17 || second*10 > first*23 {
		t.Fatalf(
			"%s = %v; doubling deltas %d/%d are not linear",
			name,
			values,
			first,
			second,
		)
	}
}

func waveFourFunction(
	t *testing.T,
	sourcePackage *load.Package,
	name string,
) *ast.FuncDecl {
	t.Helper()
	for _, file := range sourcePackage.Files() {
		for _, declaration := range file.Syntax().Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if ok && function.Name.Name == name {
				return function
			}
		}
	}
	t.Fatalf("Go function %s is absent", name)
	return nil
}

func waveFourTargetFunction(
	t *testing.T,
	emission emit.ProgramEmission,
	name string,
) tsgo.FunctionDeclaration {
	t.Helper()
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSource {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			function, ok := statement.(tsgo.FunctionDeclaration)
			if ok && function.Name().Text() == name {
				return function
			}
		}
	}
	t.Fatalf("target function %s is absent", name)
	return nil
}

func executeWaveNineGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(waveNineConcurrencyDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner-wave9")
	writeProgramFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(
		`module example.com/runner

go 1.26.4

require example.com/wave9concurrency v0.0.0

replace example.com/wave9concurrency => %s
`,
		filepath.ToSlash(modulePath),
	))
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	values "example.com/wave9concurrency"
)

func main() {
	fmt.Println(values.Audit())
}
`)
	return runProgram(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}

func waveNineConcurrencyDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"concurrency",
		"wave9",
	)
}

func executePackageStateGo(
	t *testing.T,
	projectDirectory string,
	workingDirectory string,
) string {
	t.Helper()
	absoluteProject, err := filepath.Abs(projectDirectory)
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-package-state")
	writeProgramFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/package-state-runner

go 1.26.4

require example.com/package-state v0.0.0

replace example.com/package-state => %s
`, filepath.ToSlash(absoluteProject)))
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	"example.com/package-state/api"
)

func main() {
	fmt.Println(api.Run())
	fmt.Println(api.Run())
}
`)
	return runProgram(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}

func executePackageStateTypeScript(
	t *testing.T,
	workingDirectory string,
	targetPaths []string,
	assemblyPath string,
	stringify bool,
) string {
	t.Helper()
	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	modulePath := "./" + strings.TrimSuffix(assemblyPath, ".ts") + ".js"
	runCall := "Run()"
	if stringify {
		runCall = "Run().toString()"
	}
	writeProgramFile(t, runnerPath, `import "./program.js";
import { Run } from "`+modulePath+`";

console.log(`+runCall+`);
console.log(`+runCall+`);
`)
	outputDirectory := filepath.Join(workingDirectory, "out")
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--outDir", outputDirectory,
	}
	arguments = append(arguments, targetPaths...)
	arguments = append(arguments, runnerPath)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := tsgo.Compile(
		ctx,
		repositoryRoot(),
		workingDirectory,
		arguments,
	); err != nil {
		t.Fatal(err)
	}
	return runProgram(
		t,
		workingDirectory,
		"node",
		filepath.Join(outputDirectory, "runner.js"),
	)
}
