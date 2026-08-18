package emit_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestInterfaceAdaptersDeclareCanonicalGeneratedContracts(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		module    string
		source    string
		payload   string
		className string
		contract  string
	}{
		{
			name:   "anonymous",
			module: "anonymousinterfaceheritage",
			source: `package anonymousinterfaceheritage

type Value struct{}

func (Value) Ping() int32 { return 7 }

func Box() interface{ Ping() int32 } { return Value{} }
`,
			payload:   "Value__from_anonymousinterfaceheritage",
			className: "$goInterfaceAdapter$Named_anonymousinterfaceheritage$Value",
			contract:  "$goInterface$Interface_Method_anonymousinterfaceheritage$Ping_void_to_int32",
		},
		{
			name:   "predeclared",
			module: "predeclaredinterfaceheritage",
			source: `package predeclaredinterfaceheritage

type Failure struct{}

func (Failure) Error() string { return "failure" }

func Box() error { return Failure{} }

func BoxAnonymous() interface{ Error() string } { return Failure{} }
`,
			payload:   "Failure__from_predeclaredinterfaceheritage",
			className: "$goInterfaceAdapter$Named_predeclaredinterfaceheritage$Failure",
			contract:  "$goInterface$Interface_Method_Error_void_to_string",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			directory := t.TempDir()
			writeProgramFile(
				t,
				filepath.Join(directory, "go.mod"),
				"module example.com/"+testCase.module+"\n\ngo 1.26.4\n",
			)
			writeProgramFile(
				t,
				filepath.Join(directory, "source.go"),
				testCase.source,
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
			workingDirectory := t.TempDir()
			artifacts := materializeArtifacts(t, emission, workingDirectory)
			adapter := interfaceContractDemandAdapter(
				t,
				artifacts.paths,
				testCase.payload,
			)
			classStart := strings.Index(
				adapter,
				"class "+testCase.className+" ",
			)
			if classStart < 0 {
				t.Fatalf("adapter class is absent:\n%s", adapter)
			}
			classEnd := strings.Index(adapter[classStart:], " {")
			if classEnd < 0 {
				t.Fatalf("adapter header is unterminated:\n%s", adapter)
			}
			classHeader := adapter[classStart : classStart+classEnd]
			implementsStart := strings.Index(classHeader, " implements ")
			heritage := ""
			if implementsStart >= 0 {
				heritage = classHeader[implementsStart+len(" implements "):]
			}
			if !strings.Contains(adapter, testCase.contract+" as GoInterface") ||
				heritage != "GoInterface" {
				t.Fatalf(
					"canonical generated contract is absent from adapter heritage:\n%s\n%s",
					classHeader,
					adapter,
				)
			}
			waveThreeTypecheck(t, workingDirectory, artifacts.paths)
		})
	}
}
