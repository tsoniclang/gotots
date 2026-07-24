package compiler

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/source"
)

const stage2GenericOwnerFixture = `package generics

import (
	"reflect"
	"sync/atomic"
)

type Local[Left, Right any] struct {
	left  Left
	right Right
}

func (value Local[Left, Right]) First() Left {
	return value.left
}

func (value *Local[Left, Right]) init() {}

func LoadOr[T any](pointer *atomic.Pointer[T], fallback T) T {
	value := pointer.Load()
	if value != nil {
		return *value
	}
	return fallback
}

func Reflected[T any]() reflect.Type {
	return reflect.TypeFor[T]()
}

var Integer atomic.Pointer[int]
`

func TestStage2TypeParametersUseCanonicalGenericOwners(t *testing.T) {
	directory := t.TempDir()
	writeCompilerFile(
		t,
		directory,
		"go.mod",
		"module example.com/generics\n\ngo 1.26.0\n",
	)
	writeCompilerFile(
		t, directory, "generics.go", stage2GenericOwnerFixture,
	)
	request := source.Request{
		Dir: directory, Patterns: []string{"."},
		ProviderContract: contract.DefaultID,
	}
	artifactDirectory := t.TempDir()
	structurePath := filepath.Join(
		artifactDirectory, "provider.structure.gotots",
	)
	semanticPath := filepath.Join(
		artifactDirectory, "provider.semantic.gotots",
	)
	provider, err := AuditCatalog(
		request, structurePath, semanticPath,
	)
	if err != nil {
		t.Fatal(err)
	}
	request.ProviderStructureArtifact = structurePath
	request.ProviderStructureDigest = provider.Structure.Digest
	request.ProviderSemanticArtifact = semanticPath
	request.ProviderSemanticDigest = provider.Semantic.Digest
	inspection, err := inspectConstructsForTest(t, request)
	if err != nil {
		t.Fatal(err)
	}

	application := semanticPackageByImportPath(
		t, inspection.Semantic(), "example.com/generics",
	)
	applicationRoles := semanticTypeParameterRoles(application)
	methodInitDefinitions := 0
	for _, definition := range semanticDefinitions(application) {
		spec := definition.Spec()
		if spec.Name != "init" {
			continue
		}
		if len(spec.Declarations) != 1 ||
			spec.Receiver.IsZero() {
			t.Fatalf(
				"method init declaration/receiver=%d/%s, want 1/nonzero",
				len(spec.Declarations),
				spec.Receiver,
			)
		}
		methodInitDefinitions++
	}
	if applicationRoles[semantic.TypeParameterDeclared] < 2 ||
		applicationRoles[semantic.TypeParameterCallable] < 1 ||
		applicationRoles[semantic.TypeParameterReceiver] < 4 ||
		methodInitDefinitions != 1 {
		t.Fatalf(
			"application type-parameter roles=%v method-init=%d, want declared>=2 callable>=1 receiver>=4 method-init=1",
			applicationRoles,
			methodInitDefinitions,
		)
	}
	atomicPackage := semanticPackageByImportPath(
		t, inspection.Semantic(), "sync/atomic",
	)
	atomicRoles := semanticTypeParameterRoles(atomicPackage)
	if atomicRoles[semantic.TypeParameterDeclared] == 0 ||
		atomicRoles[semantic.TypeParameterReceiver] == 0 {
		t.Fatalf(
			"export-data type-parameter roles=%v, want declared and receiver",
			atomicRoles,
		)
	}
	if ReflectedForTest[int]() != reflect.TypeFor[int]() {
		t.Fatal("generic reflection fixture control differs")
	}
}

func ReflectedForTest[T any]() reflect.Type {
	return reflect.TypeFor[T]()
}

func semanticTypeParameterRoles(
	pkg semantic.Package,
) map[semantic.TypeParameterRole]int {
	roles := map[semantic.TypeParameterRole]int{}
	for _, record := range semanticTypes(pkg) {
		if record.Kind() == semantic.TypeParameter {
			roles[record.Spec().Parameter.Role()]++
		}
	}
	return roles
}
