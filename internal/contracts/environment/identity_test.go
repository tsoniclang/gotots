package environment_test

import (
	"crypto/sha256"
	"encoding/hex"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/environment"
)

func TestToolchainKeyIsExactAndMachineIndependent(t *testing.T) {
	profile, err := environment.NewBuildProfileForToolchain(
		"go1.26.4",
		"linux",
		"amd64",
		false,
		[]string{"noasm"},
	)
	if err != nil {
		t.Fatal(err)
	}
	wantBytes := sha256.Sum256([]byte(
		"go1.26.4\x00linux\x00amd64\x000\x00noasm",
	))
	want := hex.EncodeToString(wantBytes[:])
	got, err := environment.ToolchainKey(profile)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("toolchain key = %q, want %q", got, want)
	}
	if _, err := environment.ToolchainKey(environment.BuildProfile{}); err == nil {
		t.Fatal("empty toolchain version was accepted")
	}
	withCgo, err := environment.NewBuildProfileForToolchain(
		"go1.26.4",
		"linux",
		"amd64",
		true,
		[]string{"noasm"},
	)
	if err != nil {
		t.Fatal(err)
	}
	withCgoKey, err := environment.ToolchainKey(withCgo)
	if err != nil {
		t.Fatal(err)
	}
	if withCgoKey == got {
		t.Fatal("CGO selection did not change the toolchain key")
	}
}

func TestObjectContractUsesPackageAndReceiverIdentity(t *testing.T) {
	checked := checkPackage(t, `package sample
type Box struct{}
func (receiver *Box) Read(buffer []byte) (count int, err error) { return 0, nil }
`)
	box := checked.Scope().Lookup("Box").Type().(*types.Named)
	method := box.Method(0)
	contract, err := environment.Describe(method)
	if err != nil {
		t.Fatal(err)
	}
	if contract.Kind() != environment.ObjectFunction ||
		contract.Receiver() != "*example.test/sample.Box" ||
		contract.Identity() != "example.test/sample|kind=4|receiver=*example.test/sample.Box|name=Read" ||
		contract.Signature() != "func(buffer []byte) (count int, err error)|params=buffer|results=count,err" {
		t.Fatalf("method contract = %#v", contract)
	}
}

func TestObjectContractPreservesDefinedAndUnderlyingTypes(t *testing.T) {
	checked := checkPackage(t, `package sample
type Count int64
const Limit Count = 4
`)
	count, err := environment.Describe(checked.Scope().Lookup("Count"))
	if err != nil {
		t.Fatal(err)
	}
	if count.Signature() != "defined=example.test/sample.Count|underlying=int64" {
		t.Fatalf("type signature = %q", count.Signature())
	}
	limit, err := environment.Describe(checked.Scope().Lookup("Limit"))
	if err != nil {
		t.Fatal(err)
	}
	if limit.Value() != "4" || limit.Kind() != environment.ObjectConstant {
		t.Fatalf("constant contract = %#v", limit)
	}
}

func TestObjectContractRejectsNonDeclarationObjects(t *testing.T) {
	checked := checkPackage(t, `package sample
func Use(value int) { _ = value }
`)
	function := checked.Scope().Lookup("Use").(*types.Func)
	parameter := function.Type().(*types.Signature).Params().At(0)
	if _, err := environment.Describe(parameter); err == nil {
		t.Fatal("parameter acquired an environment declaration identity")
	}
}

func TestObjectContractOwnsUnsafeBuiltinsWithoutInventingASignature(t *testing.T) {
	builtin, ok := types.Unsafe.Scope().Lookup("String").(*types.Builtin)
	if !ok {
		t.Fatal("unsafe.String is not a selected toolchain builtin")
	}
	contract, err := environment.Describe(builtin)
	if err != nil {
		t.Fatal(err)
	}
	if contract.Kind() != environment.ObjectBuiltin ||
		contract.Identity() != "unsafe|kind=5|receiver=|name=String" ||
		contract.Signature() != "builtin" {
		t.Fatalf("unsafe builtin contract = %#v", contract)
	}
}

func checkPackage(t *testing.T, source string) *types.Package {
	t.Helper()
	files := token.NewFileSet()
	parsed, err := parser.ParseFile(files, "sample.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	checked, err := (&types.Config{}).Check(
		"example.test/sample",
		files,
		[]*ast.File{parsed},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return checked
}
