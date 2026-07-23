package compiler

import (
	"fmt"
	"sort"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/source"
)

const stage2ContextMatrix = `package matrix

import "unsafe"

type Number int
type Alias = Number

type Box[T any] struct {
	Value T
}

func (box Box[T]) Get() T { return box.Value }

func Identity[T any](value T) T { return value }

var First, _ = pair()
var _, _ = pair()

func init() { First++ }

func pair() (int, int) { return 1, 2 }

func Matrix(
	values map[string]int,
	input any,
	channel <-chan int,
	box Box[int],
) (int, bool) {
	value, mapOK := values["key"]
	received, channelOK := <-channel
	asserted, assertOK := input.(int)
	converted := Number(value)
	field := box.Value
	methodValue := box.Get
	methodExpression := Box[int].Get
	generic := Identity[int](value)
	structValue := struct{ Value int }{Value: generic}
	pointer := &structValue
	pointer.Value = pointer.Value + 1
	if value > 0 {
		value++
	} else {
		value--
	}
loop:
	for index := 0; index < 1; index++ {
		value += index
		if value > 10 {
			break loop
		}
	}
	switch asserted {
	case 1:
		value++
	default:
		value--
	}
	return int(converted) + field + methodValue() +
		methodExpression(box) + received + pointer.Value +
		int(unsafe.Sizeof(structValue)),
		mapOK && channelOK && assertOK
}

func Copies(
	left struct{ Value int },
	right struct{ Value int },
) int {
	return left.Value + right.Value
}

func CopyCall(value struct{ Value int }) int {
	return Copies(value, value)
}

func Closure(base int) func(int) int {
	return func(delta int) int { return base + delta }
}
`

func TestStage2ContextMatrixAndSemanticConservation(t *testing.T) {
	inspection := inspectStage2Fixture(t)
	pkg := semanticPackageByImportPath(
		t, inspection.Semantic(), "example.com/stage2",
	)

	if len(pkg.Unsupported()) != 0 {
		t.Fatalf("unsupported records=%d, want 0", len(pkg.Unsupported()))
	}
	operations := pkg.Operations()
	requireOperationVariant(
		t, operations, semantic.OperationMapLookup,
		catalog.VariantMapLookupCommaOk,
	)
	requireOperationVariant(
		t, operations, semantic.OperationReceive,
		catalog.VariantReceiveCommaOk,
	)
	requireOperationVariant(
		t, operations, semantic.OperationTypeAssert,
		catalog.VariantAssertCommaOk,
	)
	requireOperationKind(t, operations, semantic.OperationConvert)
	requireOperationKind(t, operations, semantic.OperationFieldSelect)
	requireOperationKind(t, operations, semantic.OperationMethodValue)
	requireOperationKind(t, operations, semantic.OperationMethodExpression)
	requireOperationKind(t, operations, semantic.OperationGenericInstantiate)
	requireBuiltinCatalogs(t, inspection.Semantic(), operations)

	var labeledBranch semantic.Operation
	for _, operation := range operations {
		spec := operation.Spec()
		if operation.Kind() == semantic.OperationBranch &&
			!spec.Label.IsZero() {
			labeledBranch = operation
			break
		}
	}
	if labeledBranch.ID().IsZero() ||
		labeledBranch.Spec().ControlTarget.IsZero() {
		t.Fatal("labeled break did not retain both label and control target")
	}
	if labeledBranch.Spec().ControlTarget.Definition() !=
		labeledBranch.Definition() {
		t.Fatal("labeled break escaped its owning definition")
	}

	copyEffects := map[identity.OccurrenceID]bool{}
	for _, operation := range operations {
		if operation.Kind() != semantic.OperationCall {
			continue
		}
		for _, effect := range operation.Spec().Implicit {
			if effect.Kind() == catalog.ImplicitValueCopy {
				copyEffects[effect.Site()] = true
			}
		}
	}
	if len(copyEffects) < 2 {
		t.Fatalf(
			"distinct per-operand value-copy effects=%d, want at least 2",
			len(copyEffects),
		)
	}

	var capturedBase bool
	for _, binding := range pkg.Bindings() {
		if binding.Name() == "base" && len(binding.CapturedBy()) == 1 {
			capturedBase = true
		}
	}
	if !capturedBase {
		t.Fatal("closure capture for base was not materialized")
	}

	typeKinds := map[semantic.TypeKind]int{}
	for _, record := range pkg.Types() {
		typeKinds[record.Kind()]++
	}
	if typeKinds[semantic.TypeAlias] == 0 ||
		typeKinds[semantic.TypeNamed] == 0 ||
		typeKinds[semantic.TypeParameter] == 0 {
		t.Fatalf("semantic type classes=%v", typeKinds)
	}

	var (
		initDefinitions          int
		initializerCardinalities []int
	)
	for _, definition := range pkg.Definitions() {
		spec := definition.Spec()
		if spec.Name == "init" {
			initDefinitions++
			if len(spec.Declarations) != 0 {
				t.Fatal("func init introduced a package declaration")
			}
		}
		if definition.Form() == semantic.DefinitionFormInitializer {
			initializerCardinalities = append(
				initializerCardinalities, len(spec.Declarations),
			)
		}
	}
	sort.Ints(initializerCardinalities)
	if initDefinitions != 1 ||
		fmt.Sprint(initializerCardinalities) != "[0 1]" {
		t.Fatalf(
			"init definitions=%d initializer declaration cardinalities=%v",
			initDefinitions, initializerCardinalities,
		)
	}

	work := inspection.SemanticWork()
	if work.ContextAssignments != work.OccurrenceResolutions ||
		work.ContextAssignments == 0 ||
		work.OperationConstructions !=
			len(operations)+len(pkg.Unsupported()) {
		t.Fatalf("semantic work is not conserved: %+v", work)
	}
}

func requireBuiltinCatalogs(
	t *testing.T,
	model *semantic.Model,
	operations []semantic.Operation,
) {
	t.Helper()
	builtin := semanticPackageByImportPath(t, model, "builtin")
	if len(builtin.Declarations()) != len(catalog.AllPredeclared()) {
		t.Fatalf(
			"predeclared declarations=%d, want %d",
			len(builtin.Declarations()),
			len(catalog.AllPredeclared()),
		)
	}
	unsafePackage := semanticPackageByImportPath(t, model, "unsafe")
	members := map[string]semantic.Declaration{}
	for _, declaration := range unsafePackage.Declarations() {
		members[declaration.Name()] = declaration
	}
	if len(members) != len(catalog.AllUnsafeMembers()) {
		t.Fatalf(
			"unsafe declarations=%d, want %d",
			len(members),
			len(catalog.AllUnsafeMembers()),
		)
	}
	for _, member := range catalog.AllUnsafeMembers() {
		declaration, present := members[member.Name()]
		if !present {
			t.Errorf("unsafe member %s is absent", member)
			continue
		}
		switch member.Class() {
		case catalog.UnsafeMemberClassType:
			if declaration.Class() != identity.SemanticObjectType ||
				declaration.Type().IsZero() {
				t.Errorf("unsafe type member %s has invalid semantics", member)
			}
		case catalog.UnsafeMemberClassBuiltin:
			if declaration.Class() != identity.SemanticObjectBuiltin ||
				!declaration.Type().IsZero() {
				t.Errorf(
					"unsafe builtin member %s has an ordinary type",
					member,
				)
			}
		}
	}
	sizeOf := members[catalog.UnsafeMemberSizeof.Name()]
	var foundCall bool
	for _, operation := range operations {
		spec := operation.Spec()
		if operation.Kind() != semantic.OperationBuiltinCall ||
			spec.Object.Kind() !=
				semantic.ObjectReferenceDeclaration ||
			spec.Object.Declaration() != sizeOf.ID() {
			continue
		}
		foundCall = true
		if spec.ResultType.IsZero() ||
			len(spec.Operands) != 2 {
			t.Fatalf(
				"unsafe.Sizeof call-site evidence result=%s operands=%v",
				spec.ResultType, spec.Operands,
			)
		}
	}
	if !foundCall {
		t.Fatal("unsafe.Sizeof call-site semantics are absent")
	}
}

func TestStage2SemanticIdentitiesSurviveWorkspaceRelocation(t *testing.T) {
	first := semanticIdentitySet(inspectStage2Fixture(t))
	second := semanticIdentitySet(inspectStage2Fixture(t))
	if fmt.Sprint(first) != fmt.Sprint(second) {
		t.Fatalf(
			"semantic identities changed after workspace relocation\nfirst=%v\nsecond=%v",
			first, second,
		)
	}
}

func inspectStage2Fixture(t *testing.T) *Inspection {
	t.Helper()
	directory := t.TempDir()
	writeCompilerFile(
		t,
		directory,
		"go.mod",
		"module example.com/stage2\n\ngo 1.26.0\n",
	)
	writeCompilerFile(t, directory, "matrix.go", stage2ContextMatrix)
	inspection, err := InspectConstructs(source.Request{
		Dir: directory, Patterns: []string{"."},
		ProviderContract: contract.DefaultID,
	})
	if err != nil {
		t.Fatal(err)
	}
	return inspection
}

func semanticPackageByImportPath(
	t *testing.T,
	model *semantic.Model,
	importPath string,
) semantic.Package {
	t.Helper()
	var found semantic.Package
	if err := model.VisitPackages(func(pkg semantic.Package) error {
		if pkg.ID().ImportPath() == importPath {
			found = pkg
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if found.ID().IsZero() {
		t.Fatalf("semantic package %s is absent", importPath)
	}
	return found
}

func requireOperationKind(
	t *testing.T,
	operations []semantic.Operation,
	kind semantic.OperationKind,
) {
	t.Helper()
	for _, operation := range operations {
		if operation.Kind() == kind {
			return
		}
	}
	t.Errorf("operation kind %s is absent", kind)
}

func requireOperationVariant(
	t *testing.T,
	operations []semantic.Operation,
	kind semantic.OperationKind,
	variant catalog.Variant,
) {
	t.Helper()
	for _, operation := range operations {
		if operation.Kind() == kind &&
			operation.Variant() == variant {
			return
		}
	}
	t.Errorf("operation %s/%s is absent", kind, variant)
}

func semanticIdentitySet(inspection *Inspection) []string {
	var out []string
	_ = inspection.Semantic().VisitPackages(
		func(pkg semantic.Package) error {
			for _, definition := range pkg.Definitions() {
				out = append(
					out, "definition:"+definition.Definition().String(),
				)
			}
			for _, declaration := range pkg.Declarations() {
				out = append(
					out, "declaration:"+declaration.ID().String(),
				)
			}
			for _, binding := range pkg.Bindings() {
				out = append(out, "binding:"+binding.ID().String())
			}
			for _, record := range pkg.Types() {
				out = append(out, "type:"+record.ID().String())
			}
			for _, operation := range pkg.Operations() {
				out = append(out, "operation:"+operation.ID().String())
			}
			return nil
		},
	)
	sort.Strings(out)
	return out
}
