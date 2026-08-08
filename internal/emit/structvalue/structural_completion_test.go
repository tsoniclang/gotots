package structvalue_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestAnonymousStructValueFamilyCompiles(t *testing.T) {
	_, err := compileTemporaryStructProgram(t, `package boundary

func Anonymous(value int32) bool {
	left := struct {
		Value int32 `+"`json:\"value\"`"+`
		Ready bool
	}{Value: value, Ready: true}
	right := left
	right.Value = value + 1
	return left != right
}
`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAnonymousStructLocalComponentUsesExactLexicalPlacement(t *testing.T) {
	emission, err := compileTemporaryStructProgram(t, `package boundary

func Local(value int32) int32 {
	type Local int32
	item := struct{ Value Local }{Value: Local(value)}
	return int32(item.Value)
}
`)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range emission.Files() {
		if file.OutputPath() == output.AnonymousStructSupportPath {
			t.Fatal("local anonymous struct escaped into compilation support")
		}
	}
	source := structTargetSource(t, emission)
	var function tsgo.FunctionDeclaration
	for _, statement := range source.Statements() {
		candidate, ok := statement.(tsgo.FunctionDeclaration)
		if ok && candidate.Name().Text() == "Local" {
			function = candidate
			break
		}
	}
	if function == nil {
		t.Fatal("Local target function is absent")
	}
	body := function.Body().(tsgo.Block).Statements()
	if len(body) != 4 {
		t.Fatalf(
			"lexical function statements = %d, want enum, class, value, return",
			len(body),
		)
	}
	localEnum := body[0].(tsgo.EnumDeclaration)
	anonymousClass := body[1].(tsgo.ClassDeclaration)
	if !strings.HasPrefix(localEnum.Name().Text(), "Local") ||
		!strings.HasPrefix(anonymousClass.Name().Text(), "$goStruct_") {
		t.Fatalf(
			"lexical definition order = %q/%q",
			localEnum.Name().Text(),
			anonymousClass.Name().Text(),
		)
	}
}

func TestAnonymousStructLexicalPlacementIsIdentityAndScopeExact(t *testing.T) {
	emission, err := compileTemporaryStructProgram(t, `package boundary

func First(value int32) int32 {
	type Local int32
	item := struct{ Value Local }{Value: Local(value)}
	return int32(item.Value)
}

func Second(value int32) int32 {
	type Local int32
	item := struct{ Value Local }{Value: Local(value)}
	return int32(item.Value)
}

func Nested(enabled bool, value int32) int32 {
	if enabled {
		type Local int32
		item := struct{ Value Local }{Value: Local(value)}
		return int32(item.Value)
	}
	return 0
}
`)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range emission.Files() {
		if file.OutputPath() == output.AnonymousStructSupportPath {
			t.Fatal("lexical anonymous struct escaped into compilation support")
		}
	}
	source := structTargetSource(t, emission)
	anonymousNames := make(map[string]struct{})
	for _, name := range []string{"First", "Second"} {
		function := targetFunctionByName(t, source, name)
		body := function.Body().(tsgo.Block).Statements()
		anonymous := body[1].(tsgo.ClassDeclaration)
		anonymousNames[anonymous.Name().Text()] = struct{}{}
	}
	if len(anonymousNames) != 2 {
		t.Fatal("same-spelled local type declarations unified generated classes")
	}
	nested := targetFunctionByName(t, source, "Nested")
	nestedBody := nested.Body().(tsgo.Block).Statements()
	branch := nestedBody[0].(tsgo.IfStatement)
	thenBlock := branch.ThenStatement().(tsgo.Block).Statements()
	if len(thenBlock) != 4 {
		t.Fatalf("nested lexical statements = %d, want four", len(thenBlock))
	}
	if _, ok := thenBlock[0].(tsgo.EnumDeclaration); !ok {
		t.Fatalf("nested local type = %T, want enum", thenBlock[0])
	}
	if anonymous, ok := thenBlock[1].(tsgo.ClassDeclaration); !ok ||
		!strings.HasPrefix(anonymous.Name().Text(), "$goStruct_") {
		t.Fatalf("nested anonymous declaration = %T", thenBlock[1])
	}
}

func targetFunctionByName(
	t *testing.T,
	source tsgo.SourceFile,
	name string,
) tsgo.FunctionDeclaration {
	t.Helper()
	for _, statement := range source.Statements() {
		function, ok := statement.(tsgo.FunctionDeclaration)
		if ok && function.Name().Text() == name {
			return function
		}
	}
	t.Fatalf("target function %q is absent", name)
	return nil
}

func TestRecursiveTaggedAndBlankNamedStructsCompile(t *testing.T) {
	_, err := compileTemporaryStructProgram(t, `package boundary

type Link struct {
	Value int32 `+"`json:\"value\"`"+`
	_ int32
	Next *Link
}

type Tree struct {
	Value int32
	Children []Tree
}

func CopyLink(value Link) Link {
	return value
}

func EqualLink(left, right Link) bool {
	return left == right
}

func CopyTree(value Tree) Tree {
	return value
}
`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDefinedSliceAndPointerFamiliesCompile(t *testing.T) {
	_, err := compileTemporaryStructProgram(t, `package boundary

type Numbers []int32
type NumbersAlias = Numbers
type NumberPointer *int32
type NumberPointerAlias = NumberPointer

func SliceValue(value Numbers, index int32) int32 {
	copy := value
	copy[index] = copy[index] + 1
	window := copy[0:len(copy)]
	return window[index]
}

func SliceNil(value Numbers) bool {
	return value == nil
}

func PointerValue(value NumberPointer) int32 {
	copy := value
	if copy == nil {
		return 0
	}
	*copy = *copy + 1
	return *copy
}

func PointerConversion(value *int32) NumberPointer {
	return NumberPointer(value)
}
`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStructuralExtensionsStrictDifferential(t *testing.T) {
	projectDirectory := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(projectDirectory, "go.mod"),
		"module example.com/structural\n\ngo 1.26.4\n",
	)
	writeProgramFile(
		t,
		filepath.Join(projectDirectory, "source.go"),
		`package structural

type Numbers []int32
type NumberPointer *int32
type NumbersAlias = Numbers
type NumberPointerAlias = NumberPointer
type Record struct{ Value int32 }
type RecordPointer *Record
type Pair [2]int32
type PairPointer *Pair
type BlankValue struct {
	_ int32
	Value int32 `+"`json:\"value\"`"+`
}

type Link struct {
	Value int32 `+"`json:\"value\"`"+`
	_ int32
	Next *Link
}

type Tree struct {
	Value int32
	Children []Tree
}

func SliceResult() int32 {
	var zero Numbers
	if zero != nil {
		return -1
	}
	value := make(Numbers, 2)
	value[0] = 3
	alias := value
	alias[0] = 5
	window := value[0:len(value)]
	return window[0] + int32(len(alias))
}

func SliceConversionResult() int32 {
	base := []int32{2, 4}
	named := Numbers(base)
	plain := []int32(named)
	return plain[0] + plain[1]
}

func SliceLiteralAndAddressResult() int32 {
	value := Numbers{2, 3}
	pointer := &value[1]
	*pointer = 7
	return value[0] + value[1]
}

func AliasResult() int32 {
	values := NumbersAlias{3}
	value := int32(4)
	pointer := NumberPointerAlias(&value)
	return values[0] + *pointer
}

func PointerResult() int32 {
	value := int32(3)
	var zero NumberPointer
	if zero != nil {
		return -1
	}
	pointer := NumberPointer(&value)
	alias := pointer
	*alias = *alias + 4
	return *pointer
}

func PointerAddressResult() int32 {
	value := int32(3)
	pointer := NumberPointer(&value)
	plain := &*pointer
	*plain = 8
	return *pointer
}

func PointerEqualityResult() bool {
	value := int32(3)
	first := NumberPointer(&value)
	same := first
	plain := (*int32)(first)
	converted := NumberPointer(plain)
	return first == same && first == converted
}

func PointerFieldResult() int32 {
	value := Record{Value: 4}
	pointer := RecordPointer(&value)
	pointer.Value = 9
	return pointer.Value
}

func PointerArrayResult() int32 {
	value := Pair{1, 2}
	pointer := PairPointer(&value)
	element := &pointer[1]
	*element = 6
	return pointer[0] + pointer[1]
}

func BlankFieldResult() bool {
	left := BlankValue{9, 4}
	right := left
	return left == right && left.Value == 4
}

func RecursivePointerResult() int32 {
	tail := Link{Value: 2}
	head := Link{Value: 1, Next: &tail}
	copy := head
	copy.Value = 9
	copy.Next.Value = 5
	return head.Value*100 + copy.Value*10 + tail.Value
}

func RecursiveSliceResult() int32 {
	root := Tree{
		Value: 1,
		Children: []Tree{{Value: 2}},
	}
	copy := root
	copy.Value = 9
	return root.Value*100 + copy.Value*10 + copy.Children[0].Value
}

func AnonymousResult(value int32) bool {
	left := struct {
		Value int32 `+"`json:\"value\"`"+`
		Ready bool
	}{Value: value, Ready: true}
	right := left
	right.Value = value + 1
	return left.Value == value && left != right
}

func NestedAnonymousResult() int32 {
	value := struct {
		Inner struct{ Value int32 }
	}{
		Inner: struct{ Value int32 }{Value: 5},
	}
	copy := value
	copy.Inner.Value = 9
	return value.Inner.Value*10 + copy.Inner.Value
}

func LocalAnonymousResult(value int32) bool {
	type Local int32
	var zero struct{ Value Local }
	item := struct{ Value Local }{Value: Local(value)}
	copy := item
	return zero.Value == 0 && copy == item
}
`,
	)
	program, err := load.Load(context.Background(), load.Request{
		Directory: projectDirectory,
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
	workingDirectory := t.TempDir()
	targetPaths, module := materializeStructProgramWithGolden(
		t,
		workingDirectory,
		emission,
		false,
	)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runnerPath, fmt.Sprintf(`import {
	AnonymousResult,
	BlankFieldResult,
	LocalAnonymousResult,
	NestedAnonymousResult,
	RecursiveSliceResult,
	SliceConversionResult,
	SliceResult,
} from %q;

console.log(SliceResult());
console.log(SliceConversionResult());
console.log(RecursiveSliceResult());
console.log(AnonymousResult(8));
console.log(BlankFieldResult());
console.log(LocalAnonymousResult(7));
console.log(NestedAnonymousResult());
`, module))
	targetPaths = append(targetPaths, runnerPath)
	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	compileStructTypeScript(t, workingDirectory, targetPaths)
	targetOutput := runProgram(
		t,
		workingDirectory,
		"node",
		filepath.Join("out", "runner.js"),
	)

	goRunner := filepath.Join(workingDirectory, "go-runner")
	writeProgramFile(t, filepath.Join(goRunner, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/structural v0.0.0

replace example.com/structural => %s
`, filepath.ToSlash(projectDirectory)))
	writeProgramFile(t, filepath.Join(goRunner, "main.go"), `package main

import (
	"fmt"
	structural "example.com/structural"
)

func main() {
	fmt.Println(structural.SliceResult())
	fmt.Println(structural.SliceConversionResult())
	fmt.Println(structural.RecursiveSliceResult())
	fmt.Println(structural.AnonymousResult(8))
	fmt.Println(structural.BlankFieldResult())
	fmt.Println(structural.LocalAnonymousResult(7))
	fmt.Println(structural.NestedAnonymousResult())
}
`)
	goOutput := runProgram(
		t,
		goRunner,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
	if targetOutput != goOutput {
		t.Fatalf(
			"TypeScript output = %q, Go output = %q",
			targetOutput,
			goOutput,
		)
	}
}
