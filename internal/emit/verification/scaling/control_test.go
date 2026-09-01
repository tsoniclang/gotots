package emit_test

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestWaveFourRangeAndSwitchScaleWithSourceSyntax(t *testing.T) {
	counts := []int{4, 8, 16}
	sourceBytes := make([]int, len(counts))
	targetBytes := make([]int, len(counts))
	targetNodes := make([]int, len(counts))
	for index, count := range counts {
		source, target, nodes := compileWaveFourScaling(t, count)
		sourceBytes[index] = len(source)
		targetBytes[index] = len(target)
		targetNodes[index] = nodes
		if loops := strings.Count(
			target,
			"for (let rangeIndex",
		); loops != 1 {
			t.Fatalf("range loops at %d cases = %d, want one", count, loops)
		}
		if checks := strings.Count(
			target,
			"let switchMatch",
		); checks != count {
			t.Fatalf(
				"switch checks at %d cases = %d, want %d",
				count,
				checks,
				count,
			)
		}
	}
	assertWaveFourLinearDoubling(t, "source bytes", sourceBytes)
	assertWaveFourLinearDoubling(t, "target bytes", targetBytes)
	assertWaveFourLinearDoubling(t, "target AST nodes", targetNodes)
	t.Logf(
		"Wave 4 scaling cases=%v source=%v target=%v nodes=%v",
		counts,
		sourceBytes,
		targetBytes,
		targetNodes,
	)
}

func compileWaveFourScaling(
	t *testing.T,
	count int,
) (string, string, int) {
	t.Helper()
	directory := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/wave4scaling\n\ngo 1.26.4\n",
	)
	var source strings.Builder
	fmt.Fprintf(
		&source,
		"package wave4scaling\n\nfunc Scale(value [2]int32) int32 {\n"+
			"\tvar total int32\n"+
			"\tfor index := range [%d]int32{} { total += int32(index) }\n"+
			"\tswitch value {\n",
		count,
	)
	for index := range count {
		fmt.Fprintf(
			&source,
			"\tcase [2]int32{%d, %d}: total += %d\n",
			index,
			index+1,
			index,
		)
	}
	source.WriteString("\tdefault: total--\n\t}\n\treturn total\n}\n")
	writeProgramFile(
		t,
		filepath.Join(directory, "source.go"),
		source.String(),
	)
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	client, err := tsgo.StartClient(repositoryRoot(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSource {
			continue
		}
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := tsgo.EncodeSourceFile(file.SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		return source.String(), printed, waveFourEncodedNodes(t, encoded)
	}
	t.Fatal("Wave 4 scaling source artifact is absent")
	return "", "", 0
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

func TestWaveFourLabelsUseCheckerIdentityNotSourceSpelling(t *testing.T) {
	program, sourcePackage := loadWaveFourProgram(t)
	function := waveFourFunction(t, sourcePackage, "integerRange")
	labeled, branches := waveFourLabeledRange(t, function)
	labeled.Label.Name = "forgedDeclarationSpelling"
	for _, branch := range branches {
		branch.Label.Name = "forgedUseSpelling"
	}

	emission := compileWaveFourProgram(t, program, sourcePackage)
	target := waveFourTargetFunction(t, emission, "integerRange")
	body := target.Body().(tsgo.Block).Statements()
	var targetLabel tsgo.LabeledStatement
	for _, statement := range body {
		if candidate, ok := statement.(tsgo.LabeledStatement); ok {
			targetLabel = candidate
		}
	}
	if targetLabel == nil || targetLabel.Label().Text() != "outer" {
		t.Fatalf("target label is absent or not exact checker-owned outer")
	}
	loop := targetLabel.Statement().(tsgo.ForStatement)
	loopBody := loop.Statement().(tsgo.Block).Statements()
	continueStatement := loopBody[2].(tsgo.IfStatement).
		ThenStatement().(tsgo.Block).
		Statements()[0].(tsgo.ContinueStatement)
	breakStatement := loopBody[4].(tsgo.IfStatement).
		ThenStatement().(tsgo.Block).
		Statements()[0].(tsgo.BreakStatement)
	if continueStatement.Label().Text() != "outer" ||
		breakStatement.Label().Text() != "outer" {
		t.Fatalf(
			"branch labels = %q/%q, want outer",
			continueStatement.Label().Text(),
			breakStatement.Label().Text(),
		)
	}
}

func TestWaveFourMissingLabelUseEvidenceFailsAtBranchOwner(t *testing.T) {
	program, sourcePackage := loadWaveFourProgram(t)
	function := waveFourFunction(t, sourcePackage, "integerRange")
	_, branches := waveFourLabeledRange(t, function)
	delete(sourcePackage.TypesInfo().Uses, branches[0].Label)

	_, err := emit.Compile(
		program,
		[]emit.Root{mustWaveFourRoot(t, sourcePackage, "integerRange")},
	)
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) ||
		unsupported.Construct != "*ast.BranchStmt" ||
		unsupported.Category != api.CategoryStatement {
		t.Fatalf("error = %#v, want branch UnsupportedError", err)
	}
}

func loadWaveFourProgram(t *testing.T) (*load.Program, *load.Package) {
	t.Helper()
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveFourStatementDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return program, program.Roots()[0]
}

func compileWaveFourProgram(
	t *testing.T,
	program *load.Program,
	sourcePackage *load.Package,
) emit.ProgramEmission {
	t.Helper()
	roots, err := emit.ExportedAPIRoots(sourcePackage)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	return emission
}

func mustWaveFourRoot(
	t *testing.T,
	sourcePackage *load.Package,
	name string,
) emit.Root {
	t.Helper()
	object := sourcePackage.Types().Scope().Lookup(name)
	root, err := emit.NewRoot(object)
	if err != nil {
		t.Fatal(err)
	}
	return root
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

func waveFourLabeledRange(
	t *testing.T,
	function *ast.FuncDecl,
) (*ast.LabeledStmt, []*ast.BranchStmt) {
	t.Helper()
	var labeled *ast.LabeledStmt
	var branches []*ast.BranchStmt
	ast.Inspect(function.Body, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.LabeledStmt:
			labeled = node
		case *ast.BranchStmt:
			if node.Label != nil {
				branches = append(branches, node)
			}
		}
		return true
	})
	if labeled == nil || len(branches) != 2 {
		t.Fatalf("labeled range = %v with %d branches", labeled != nil, len(branches))
	}
	return labeled, branches
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

type waveEightScale struct {
	sourceBytes int
	targetBytes int
	targetNodes int
	runtime     string
}

func TestWaveEightControlGrowthIsLinearAndRuntimeIsConstant(t *testing.T) {
	client, err := tsgo.StartClient(repositoryRoot(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	counts := []int{1, 2, 4}
	measurements := make([]waveEightScale, 0, len(counts))
	for _, count := range counts {
		measurement := measureWaveEightScale(t, client, count)
		measurements = append(measurements, measurement)
	}
	sourceBytes := waveEightScaleValues(
		measurements,
		func(value waveEightScale) int { return value.sourceBytes },
	)
	targetBytes := waveEightScaleValues(
		measurements,
		func(value waveEightScale) int { return value.targetBytes },
	)
	targetNodes := waveEightScaleValues(
		measurements,
		func(value waveEightScale) int { return value.targetNodes },
	)
	assertWaveFourLinearDoubling(t, "Wave 8 source bytes", sourceBytes)
	assertWaveFourLinearDoubling(t, "Wave 8 target bytes", targetBytes)
	assertWaveFourLinearDoubling(t, "Wave 8 target AST nodes", targetNodes)
	if measurements[0].runtime != measurements[1].runtime ||
		measurements[1].runtime != measurements[2].runtime {
		t.Fatal("Wave 8 runtime support grew with callable count")
	}
	t.Logf(
		"Wave 8 scaling callables=%v source=%v target=%v nodes=%v runtime=%d",
		counts,
		sourceBytes,
		targetBytes,
		targetNodes,
		len(measurements[0].runtime),
	)
}

func measureWaveEightScale(
	t *testing.T,
	client *tsgo.Client,
	count int,
) waveEightScale {
	t.Helper()
	source, emission := compileWaveEightScale(t, count)
	result := waveEightScale{sourceBytes: len(source)}
	var printed strings.Builder
	var runtime strings.Builder
	for _, file := range emission.Files() {
		target, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := tsgo.EncodeSourceFile(file.SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		result.targetBytes += len(target)
		result.targetNodes += waveFourEncodedNodes(t, encoded)
		printed.WriteString(target)
		printed.WriteByte('\n')
		if strings.HasPrefix(file.OutputPath(), "runtime/") {
			runtime.WriteString(file.OutputPath())
			runtime.WriteByte('\n')
			runtime.WriteString(target)
		}
	}
	target := printed.String()
	for name, got := range map[string]int{
		"defer registrations": strings.Count(target, ".push("),
		"goto states":         strings.Count(target, "let gotoState"),
		"source body stores":  strings.Count(target, "result += value;"),
	} {
		if got != count {
			t.Fatalf("%s at %d callables = %d, want %d", name, count, got, count)
		}
	}
	result.runtime = runtime.String()
	return result
}

func compileWaveEightScale(
	t *testing.T,
	count int,
) (string, emit.ProgramEmission) {
	t.Helper()
	directory := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/wave8scale\n\ngo 1.26.4\n",
	)
	var source strings.Builder
	source.WriteString("package wave8scale\n\n")
	for index := range count {
		fmt.Fprintf(
			&source,
			"func Scale%d(value int) (result int) {\n"+
				"\tdefer func(captured int) { result += captured }(value)\n"+
				"\tgoto check%d\n"+
				"body%d:\n"+
				"\tresult += value\n"+
				"\tvalue--\n"+
				"check%d:\n"+
				"\tif value > 0 { goto body%d }\n"+
				"\treturn\n"+
				"}\n\n",
			index,
			index,
			index,
			index,
			index,
		)
	}
	writeProgramFile(
		t,
		filepath.Join(directory, "source.go"),
		source.String(),
	)
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.CompileWithOptions(program, roots, emit.Options{
		IntegerRepresentation: emit.IntegerRepresentationNumber,
		EvaluationOrder:       emit.EvaluationOrderDirect,
	})
	if err != nil {
		t.Fatal(err)
	}
	return source.String(), emission
}

func waveEightScaleValues(
	values []waveEightScale,
	selectValue func(waveEightScale) int,
) []int {
	result := make([]int, 0, len(values))
	for _, value := range values {
		result = append(result, selectValue(value))
	}
	return result
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

func repositoryRoot() string {
	return filepath.Join("..", "..", "..", "..")
}

func waveFourStatementDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"statement",
		"wave4",
	)
}
