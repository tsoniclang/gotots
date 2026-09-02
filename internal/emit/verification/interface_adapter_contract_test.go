package emit_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/output"
)

func TestInterfaceAdaptersContainOnlyDemandedContracts(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: interfaceContractDemandDirectory(),
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
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	valueAdapter := interfaceContractDemandAdapter(
		t,
		artifacts.paths,
		"Value__from_interfacecontractdemand",
	)
	for _, method := range []string{"First", "Shared", "Second"} {
		if count := strings.Count(valueAdapter, "\n    "+method+"("); count != 1 {
			t.Fatalf(
				"Value adapter method %s count = %d, want one:\n%s",
				method,
				count,
				valueAdapter,
			)
		}
	}
	for _, forbidden := range []string{
		"Value_Unused",
		"Value_privateUnused",
		"implements GoInterfaceValue",
	} {
		if strings.Contains(valueAdapter, forbidden) {
			t.Fatalf("Value adapter contains unrelated %s:\n%s", forbidden, valueAdapter)
		}
	}
	for _, required := range []string{
		"extends GoInterfaceValue",
		"super();",
	} {
		if !strings.Contains(valueAdapter, required) {
			t.Fatalf(
				"Value adapter lacks canonical interface inheritance %q:\n%s",
				required,
				valueAdapter,
			)
		}
	}
	classStart := strings.Index(
		valueAdapter,
		"class $goInterfaceAdapter$Named_interfacecontractdemand$Value ",
	)
	if classStart < 0 {
		t.Fatalf("Value adapter class header is absent:\n%s", valueAdapter)
	}
	classEnd := strings.Index(valueAdapter[classStart:], " {")
	if classEnd < 0 {
		t.Fatalf("Value adapter class header is unterminated:\n%s", valueAdapter)
	}
	classHeader := valueAdapter[classStart : classStart+classEnd]
	for _, contract := range []string{
		"First__from_interfacecontractdemand",
		"FirstTwin__from_interfacecontractdemand",
		"Second__from_interfacecontractdemand",
	} {
		if !strings.Contains(classHeader, contract) {
			t.Fatalf(
				"Value adapter class header lacks exact contract %q: %s",
				contract,
				classHeader,
			)
		}
	}
	otherAdapter := interfaceContractDemandAdapter(
		t,
		artifacts.paths,
		"Other__from_interfacecontractdemand",
	)
	const otherName = "$goInterfaceAdapter$Named_interfacecontractdemand$Other"
	for _, forbidden := range []string{
		"Other_Second",
		"Other_Shared",
		"Other_Unused",
		"export class " + otherName,
		"const " + otherName + "$methods",
	} {
		if strings.Contains(otherAdapter, forbidden) {
			t.Fatalf(
				"any-only Other adapter contains unrelated %s:\n%s",
				forbidden,
				otherAdapter,
			)
		}
	}
	if !strings.Contains(otherAdapter, "export const "+otherName+": {") ||
		!strings.Contains(
			otherAdapter,
			"} = createGoInterfaceAdapter<"+
				"Other__from_interfacecontractdemand>",
		) {
		t.Fatalf(
			"any-only Other adapter did not use the canonical typed factory:\n%s",
			otherAdapter,
		)
	}
	if count := strings.Count(
		artifacts.printed,
		"function createGoInterfaceAdapter<T>(",
	); count != 1 {
		t.Fatalf(
			"interface adapter factory count = %d, want one:\n%s",
			count,
			artifacts.printed,
		)
	}
	runner := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runner, `import "./program.js";
import { Audit } from "`+artifacts.sourceModule+`";

const values = Audit();
console.log(Array.from({ length: values.length }, (_, index) =>
    String(values.get(index))).join(" "));
`)
	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	waveThreeTypecheck(t, workingDirectory, append(artifacts.paths, runner))
	targetOutput := runProgram(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "runner.js"),
	)
	goOutput := executeInterfaceContractDemandGo(t, workingDirectory)
	if targetOutput != goOutput {
		t.Fatalf(
			"interface-contract demand output differs\nTypeScript: %q\nGo: %q",
			targetOutput,
			goOutput,
		)
	}
}

func TestInterfaceAdapterIgnoresUnrelatedReceiverMethods(t *testing.T) {
	withoutUnused := compileInterfaceAdapterFixture(t, "")
	withUnused := compileInterfaceAdapterFixture(
		t,
		"func (Value) Unused() int32 { return 2 }\n",
	)
	if withoutUnused != withUnused {
		t.Fatalf(
			"unrelated receiver method changed adapter bytes\nwithout:\n%s\nwith:\n%s",
			withoutUnused,
			withUnused,
		)
	}
}

func TestInterfaceContractDemandIsDiscoveryOrderIndependent(t *testing.T) {
	directory := writeInterfaceAdapterFixture(t, "")
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	scope := program.Roots()[0].Types().Scope()
	box, err := emit.NewRoot(scope.Lookup("Box"))
	if err != nil {
		t.Fatal(err)
	}
	adapt, err := emit.NewRoot(scope.Lookup("Adapt"))
	if err != nil {
		t.Fatal(err)
	}
	var adapters []string
	for _, roots := range [][]emit.Root{
		{box, adapt},
		{adapt, box},
	} {
		emission, compileErr := emit.Compile(program, roots)
		if compileErr != nil {
			t.Fatal(compileErr)
		}
		artifacts := materializeArtifacts(t, emission, t.TempDir())
		adapters = append(adapters, interfaceContractDemandAdapter(
			t,
			artifacts.paths,
			"Value__from_interfaceadapterstable",
		))
	}
	if adapters[0] != adapters[1] {
		t.Fatalf(
			"interface adapter depends on demand discovery order\nbox first:\n%s\nadapt first:\n%s",
			adapters[0],
			adapters[1],
		)
	}
	for _, method := range []string{"First", "Second"} {
		if count := strings.Count(adapters[0], "\n    "+method+"("); count != 1 {
			t.Fatalf(
				"order-independent adapter method %s count = %d, want one:\n%s",
				method,
				count,
				adapters[0],
			)
		}
	}
}

func compileInterfaceAdapterFixture(t *testing.T, extraMethod string) string {
	t.Helper()
	directory := writeInterfaceAdapterFixture(t, extraMethod)
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
	artifacts := materializeArtifacts(t, emission, t.TempDir())
	return interfaceContractDemandAdapter(
		t,
		artifacts.paths,
		"Value__from_interfaceadapterstable",
	)
}

func writeInterfaceAdapterFixture(t *testing.T, extraMethod string) string {
	t.Helper()
	directory := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/interfaceadapterstable\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(directory, "source.go"), `package interfaceadapterstable

type First interface {
	First() int32
}

type Second interface {
	Second() int32
}

type Value struct{}

func (Value) First() int32 { return 1 }
func (Value) Second() int32 { return 2 }
`+extraMethod+`
func Box() First {
	return Value{}
}

func Adapt(value First) Second {
	result, _ := value.(Second)
	return result
}
`)
	return directory
}

func interfaceContractDemandAdapter(
	t *testing.T,
	paths []string,
	payload string,
) string {
	t.Helper()
	for _, path := range paths {
		if !strings.Contains(
			filepath.ToSlash(path),
			"/"+output.InterfaceAdapterSupportRoot+"/",
		) {
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(content), payload) {
			return string(content)
		}
	}
	t.Fatalf("interface adapter for %s is absent", payload)
	return ""
}

func executeInterfaceContractDemandGo(
	t *testing.T,
	workingDirectory string,
) string {
	t.Helper()
	modulePath, err := filepath.Abs(interfaceContractDemandDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	writeProgramFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(
		`module example.com/runner

go 1.26.4

require example.com/interfacecontractdemand v0.0.0

replace example.com/interfacecontractdemand => %s
`,
		filepath.ToSlash(modulePath),
	))
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	values "example.com/interfacecontractdemand"
)

func main() {
	for index, value := range values.Audit() {
		if index != 0 {
			fmt.Print(" ")
		}
		fmt.Print(value)
	}
	fmt.Println()
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

func interfaceContractDemandDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"interface",
		"contract-demand",
	)
}

func TestEmbeddedInterfacePromotionUsesInterfaceDispatch(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		options emit.Options
	}{
		{name: "number", options: emit.DefaultOptions()},
		{
			name: "bigint",
			options: emit.Options{
				IntegerRepresentation: emit.IntegerRepresentationBigInt,
				EvaluationOrder:       emit.EvaluationOrderPreserveGo,
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			program, err := load.Load(context.Background(), load.Request{
				Directory: embeddedInterfaceDirectory(),
				Pattern:   ".",
			})
			if err != nil {
				t.Fatal(err)
			}
			root, err := emit.NewRoot(
				program.Roots()[0].Types().Scope().Lookup("Audit"),
			)
			if err != nil {
				t.Fatal(err)
			}
			emission, err := emit.CompileWithOptions(
				program,
				[]emit.Root{root},
				testCase.options,
			)
			if err != nil {
				t.Fatal(err)
			}
			workingDirectory := t.TempDir()
			artifacts := materializeArtifacts(t, emission, workingDirectory)
			adapter := embeddedInterfaceAdapter(t, artifacts.paths)
			for _, required := range []string{
				"Pointer<Holder__from_embeddedinterface> | undefined",
				"loadPointer<Holder__from_embeddedinterface>",
				"equalPointer<Holder__from_embeddedinterface>",
				"hashPointer<Holder__from_embeddedinterface>",
				").Reader).Read()",
				"goInterfaceNonNil",
				".Read()",
				".Read$deferred($go$recovery)",
			} {
				if !strings.Contains(adapter, required) {
					t.Fatalf(
						"embedded-interface adapter lacks %q:\n%s",
						required,
						adapter,
					)
				}
			}
			for _, forbidden := range []string{
				"GoPointer",
				"runtime/pointer",
				"Reader_Read",
				"$fromStorage(",
				"switch (",
				".Read($go$recovery)",
			} {
				if strings.Contains(adapter, forbidden) {
					t.Fatalf(
						"embedded-interface adapter contains %q:\n%s",
						forbidden,
						adapter,
					)
				}
			}
			writeProgramFile(
				t,
				filepath.Join(workingDirectory, "package.json"),
				"{\"type\":\"module\"}\n",
			)
			waveThreeTypecheck(
				t,
				workingDirectory,
				artifacts.paths,
			)
			if artifacts.bytes > 48_000 || artifacts.largest > 24_000 {
				t.Fatalf(
					"embedded-interface artifacts exceed bounds: total=%d largest=%d",
					artifacts.bytes,
					artifacts.largest,
				)
			}
		})
	}
}

func embeddedInterfaceAdapter(t *testing.T, paths []string) string {
	t.Helper()
	for _, path := range paths {
		if !strings.Contains(
			filepath.ToSlash(path),
			"/"+output.InterfaceAdapterSupportRoot+"/",
		) {
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		target := string(content)
		if strings.Contains(target, "Holder__from_embeddedinterface") {
			return target
		}
	}
	t.Fatal("embedded-interface Holder adapter is absent")
	return ""
}

func executeEmbeddedInterfaceGo(
	t *testing.T,
	workingDirectory string,
) string {
	t.Helper()
	modulePath, err := filepath.Abs(embeddedInterfaceDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(
		workingDirectory,
		"go-runner-embedded-interface",
	)
	writeProgramFile(
		t,
		filepath.Join(runnerDirectory, "go.mod"),
		fmt.Sprintf(
			`module example.com/runner

go 1.26.4

require example.com/embeddedinterface v0.0.0

replace example.com/embeddedinterface => %s
`,
			filepath.ToSlash(modulePath),
		),
	)
	writeProgramFile(
		t,
		filepath.Join(runnerDirectory, "main.go"),
		`package main

import (
	"fmt"

	values "example.com/embeddedinterface"
)

func main() {
	fmt.Println(values.Audit())
}
`,
	)
	return runProgram(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}

func embeddedInterfaceDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"interface",
		"embedded-interface",
	)
}
