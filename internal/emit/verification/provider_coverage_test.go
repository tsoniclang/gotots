package emit_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib/certify"
	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestLinkedProviderDoesNotReconstructPrivateGoRepresentation(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/providercoverage\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package providercoverage

import (
	"reflect"
	"slices"
	"encoding/binary"
	"io/fs"
	"sync"
	"sync/atomic"
)

type Cell struct { Value int }

func Copy(value reflect.Value) reflect.Value {
	return value
}

func Zero() reflect.Value {
	var value reflect.Value
	return value
}

func Grow(values []Cell) []Cell {
	return slices.Grow(values, 2)
}

func PutBig(buffer []byte, value uint32) {
	binary.BigEndian.PutUint32(buffer, value)
}

func AppendLittle(buffer []byte, value uint32) []byte {
	return binary.LittleEndian.AppendUint32(buffer, value)
}

func BigOrder() binary.ByteOrder {
	return binary.BigEndian
}

func PathFailure(failure error) error {
	return &fs.PathError{Op: "open", Path: "/tmp/input", Err: failure}
}

func DeferredRecovery(mutex *sync.Mutex, flag *atomic.Bool) {
	defer mutex.Unlock()
	defer flag.Store(false)
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: project,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	scope := program.Roots()[0].Types().Scope()
	copyRoot, err := emit.NewRoot(scope.Lookup("Copy"))
	if err != nil {
		t.Fatal(err)
	}
	zeroRoot, err := emit.NewRoot(scope.Lookup("Zero"))
	if err != nil {
		t.Fatal(err)
	}
	growRoot, err := emit.NewRoot(scope.Lookup("Grow"))
	if err != nil {
		t.Fatal(err)
	}
	putBigRoot, err := emit.NewRoot(scope.Lookup("PutBig"))
	if err != nil {
		t.Fatal(err)
	}
	appendLittleRoot, err := emit.NewRoot(scope.Lookup("AppendLittle"))
	if err != nil {
		t.Fatal(err)
	}
	bigOrderRoot, err := emit.NewRoot(scope.Lookup("BigOrder"))
	if err != nil {
		t.Fatal(err)
	}
	pathFailureRoot, err := emit.NewRoot(scope.Lookup("PathFailure"))
	if err != nil {
		t.Fatal(err)
	}
	deferredRecoveryRoot, err := emit.NewRoot(scope.Lookup("DeferredRecovery"))
	if err != nil {
		t.Fatal(err)
	}
	certificate := linkedProviderCertificate(t)
	options := emit.DefaultOptions()
	options.StandardLibrary = certificate
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{
			copyRoot,
			zeroRoot,
			growRoot,
			putBigRoot,
			appendLittleRoot,
			bigOrderRoot,
			pathFailureRoot,
			deferredRecoveryRoot,
		},
		options,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range emission.Files() {
		if file.Kind() == emit.TargetFileEnvironmentContract {
			t.Fatalf("linked provider retained ambient contract %q", file.OutputPath())
		}
	}
	assertProviderGrowCapabilityABI(t, emission)
	assertProviderRepresentationABI(t, emission)
}

func linkedProviderCertificate(t *testing.T) *certify.Certificate {
	t.Helper()
	repository := repositoryRoot()
	certificate, err := certify.Verify(certify.Config{
		RepositoryRoot:      repository,
		ProviderRoot:        filepath.Join(repository, "gostdlib"),
		ManifestPath:        filepath.Join(repository, "gostdlib", "contract", "manifest.json"),
		ModuleMapPath:       filepath.Join(repository, "gostdlib", "contract", "modules.json"),
		FacetMapPath:        filepath.Join(repository, "gostdlib", "contract", "facets.json"),
		RuntimeContractPath: filepath.Join(repository, "gostdlib", "contract", "runtime.json"),
		TSConfigPath:        filepath.Join(repository, "gostdlib", "tsconfig.json"),
		ScratchDirectory:    t.TempDir(),
		GoBinary:            "go",
		Backend:             "node",
		MinimumGoVersion:    "go1.26.4",
		MaximumGoVersion:    "go1.26.4",
	})
	if err != nil {
		t.Fatal(err)
	}
	return certificate
}

func assertProviderGrowCapabilityABI(
	t *testing.T,
	emission emit.ProgramEmission,
) {
	t.Helper()
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSource {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			function, ok := statement.(tsgo.FunctionDeclaration)
			if !ok || function.Name().Text() != "Grow" {
				continue
			}
			body, ok := function.Body().(tsgo.Block)
			if !ok || len(body.Statements()) != 1 {
				t.Fatalf("Grow body = %T", function.Body())
			}
			returned, ok := body.Statements()[0].(tsgo.ReturnStatement)
			if !ok {
				t.Fatalf("Grow statement = %T", body.Statements()[0])
			}
			call, ok := returned.Expression().(tsgo.CallExpression)
			if !ok || len(call.Arguments()) != 4 {
				t.Fatalf(
					"Grow return = %T with %d arguments",
					returned.Expression(),
					len(call.Arguments()),
				)
			}
			for index := range 2 {
				if _, ok := call.Arguments()[index].(tsgo.Identifier); !ok {
					t.Fatalf(
						"Grow capability %d = %T",
						index,
						call.Arguments()[index],
					)
				}
			}
			return
		}
	}
	t.Fatal("linked provider Grow call is absent")
}

func assertProviderRepresentationABI(
	t *testing.T,
	emission emit.ProgramEmission,
) {
	t.Helper()
	wanted := map[string]struct {
		state  string
		member string
	}{
		"PutBig":       {state: "BigEndian", member: "PutUint32"},
		"AppendLittle": {state: "LittleEndian", member: "AppendUint32"},
	}
	imports := make(map[string]bool)
	for _, file := range emission.Files() {
		for _, statement := range file.SourceFile().Statements() {
			if declaration, ok := statement.(tsgo.ImportDeclaration); ok {
				module, ok := declaration.ModuleSpecifier().(tsgo.StringLiteral)
				if ok {
					imports[module.Text()] = true
				}
				continue
			}
			function, ok := statement.(tsgo.FunctionDeclaration)
			if !ok {
				continue
			}
			expectation, selected := wanted[function.Name().Text()]
			if !selected {
				continue
			}
			call := singleFunctionCall(t, function)
			member, ok := call.Expression().(tsgo.PropertyAccessExpression)
			if !ok || memberNameText(t, member.Name()) != expectation.member {
				t.Fatalf(
					"%s call target = %T %v",
					function.Name().Text(),
					call.Expression(),
					call.Expression(),
				)
			}
			if providerStateRead(t, member.Expression()) != expectation.state {
				t.Fatalf(
					"%s receiver = %T %v",
					function.Name().Text(),
					member.Expression(),
					member.Expression(),
				)
			}
			delete(wanted, function.Name().Text())
		}
	}
	if len(wanted) != 0 {
		t.Fatalf("provider representation calls are absent: %v", wanted)
	}
	for _, module := range []string{
		"@gotots/gostdlib/encoding/binary.js",
		"@gotots/gostdlib/internal/facets/named-encoding-binary.js",
		"@gotots/gostdlib/internal/facets/named-io-fs.js",
		"@gotots/gostdlib/internal/facets/recovery-sync.js",
	} {
		if !imports[module] {
			t.Fatalf("provider representation import %q is absent", module)
		}
	}
}

func providerStateRead(t *testing.T, source tsgo.Expression) string {
	t.Helper()
	switch selected := source.(type) {
	case tsgo.PropertyAccessExpression:
		return memberNameText(t, selected.Name())
	case tsgo.CallExpression:
		callee, ok := selected.Expression().(tsgo.PropertyAccessExpression)
		if !ok {
			t.Fatalf(
				"provider copy target = %T %v",
				selected.Expression(),
				selected.Expression(),
			)
		}
		operation := memberNameText(t, callee.Name())
		if operation != "$copy" && operation != "$fromStorage" ||
			len(selected.Arguments()) != 1 {
			t.Fatalf(
				"provider projection target member = %q, arguments = %d",
				operation,
				len(selected.Arguments()),
			)
		}
		return providerStateRead(t, selected.Arguments()[0])
	default:
		t.Fatalf("provider state receiver = %T %v", source, source)
	}
	return ""
}

func memberNameText(t *testing.T, name tsgo.MemberName) string {
	t.Helper()
	identifier, ok := name.(tsgo.Identifier)
	if !ok {
		t.Fatalf("member name = %T", name)
	}
	return identifier.Text()
}

func singleFunctionCall(
	t *testing.T,
	function tsgo.FunctionDeclaration,
) tsgo.CallExpression {
	t.Helper()
	body, ok := function.Body().(tsgo.Block)
	if !ok || len(body.Statements()) != 1 {
		t.Fatalf("%s body = %T", function.Name().Text(), function.Body())
	}
	var expression tsgo.Expression
	switch statement := body.Statements()[0].(type) {
	case tsgo.ExpressionStatement:
		expression = statement.Expression()
	case tsgo.ReturnStatement:
		expression = statement.Expression()
	default:
		t.Fatalf("%s statement = %T", function.Name().Text(), statement)
	}
	call, ok := expression.(tsgo.CallExpression)
	if !ok {
		t.Fatalf("%s expression = %T", function.Name().Text(), expression)
	}
	return call
}
