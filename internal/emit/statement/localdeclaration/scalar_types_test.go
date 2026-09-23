package localdeclaration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestScalarBindingsRetainWidthAcrossDeclarationForms(t *testing.T) {
	directory := t.TempDir()
	for name, content := range map[string]string{
		"go.mod": "module example.com/scalar-bindings\n\ngo 1.26.4\n",
		"source.go": `package bindings
func Pair() (int64, uint64) { return -1, 9007199254740993 }
func Bindings(input uint64, narrow float32) uint64 {
    var signed8 int8 = -1
    var unsigned8 uint8 = 1
    var signed16 int16 = -1
    var unsigned16 uint16 = 1
    var signed32 int32 = -1
    var unsigned32 uint32 = 1
    var signed64 int64 = -1
    var unsigned64 uint64 = input
    inferred64 := input
    floatValue := narrow
    fromPairSigned, fromPairUnsigned := Pair()
    _, _, _, _, _, _ = signed8, unsigned8, signed16, unsigned16, signed32, unsigned32
    _, _, _, _ = signed64, inferred64, floatValue, fromPairSigned
    return unsigned64 + fromPairUnsigned
}
`,
	} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	loaded, err := load.One(context.Background(), load.Request{Directory: directory, Pattern: "."})
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.CompileFile(loaded, loaded.Files()[0].Syntax())
	if err != nil {
		t.Fatal(err)
	}
	wanted := map[string]string{
		"signed8": "int8", "unsigned8": "uint8",
		"signed16": "int16", "unsigned16": "uint16",
		"signed32": "int32", "unsigned32": "uint32",
		"signed64": "int64", "unsigned64": "uint64",
		"inferred64": "uint64", "floatValue": "float32",
		"fromPairSigned": "int64", "fromPairUnsigned": "uint64",
	}
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSource {
			continue
		}
		function := localDeclarationFunctionByName(t, file.SourceFile(), "Bindings")
		assertScalarBindings(t, function.Body().(tsgo.Block).Statements(), wanted)
	}
	if len(wanted) != 0 {
		t.Fatalf("unverified scalar bindings: %v", wanted)
	}
}

func assertScalarBindings(t *testing.T, statements []tsgo.Statement, wanted map[string]string) {
	t.Helper()
	for _, statement := range statements {
		if block, ok := statement.(tsgo.Block); ok {
			assertScalarBindings(t, block.Statements(), wanted)
			continue
		}
		variables, ok := statement.(tsgo.VariableStatement)
		if !ok {
			continue
		}
		for _, declaration := range variables.DeclarationList().Declarations() {
			name, ok := declaration.Name().(tsgo.Identifier)
			if !ok {
				continue
			}
			expected, selected := wanted[name.Text()]
			if !selected {
				continue
			}
			reference, ok := declaration.Type().(tsgo.TypeReferenceNode)
			if !ok {
				t.Fatalf("%s has no exact scalar annotation", name.Text())
			}
			typeName, ok := reference.TypeName().(tsgo.Identifier)
			if !ok || typeName.Text() != expected {
				t.Fatalf("%s does not retain %s", name.Text(), expected)
			}
			delete(wanted, name.Text())
		}
	}
}
