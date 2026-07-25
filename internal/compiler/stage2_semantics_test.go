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

import (
	"example.com/stage2/left"
	other "example.com/stage2/right"
	"unsafe"
)

type Number int
type Alias = Number
type Sequence func(func(int) bool)

type Universal interface{}
type Empty interface {
	int
	string
}

type Recursive interface {
	Next() interface {
		Parent() Recursive
	}
}

type Fixed [1 + 1]int

type Box[T any] struct {
	Value T
}

func (box Box[T]) Get() T { return box.Value }

type Embedded struct {
	Nested int
}

type EmbeddedBox struct {
	Embedded
}

func Identity[T any](value T) T { return value }

func MixedBindings(named string) string {
	_ = other.Value
	return named
}

func UnnamedBindings(int, string) {}

var First, _ = pair()
var _, _ = pair()
var Left, Right = pair()
var AliasValue Alias
var External = left.Product{SKU: "external"}

func init() { First++ }
func _() { First++ }

func pair() (int, int) { return 1, 2 }

func ExternalCode() string { return External.Code() }

func Matrix(
	values map[string]int,
	input any,
	channel <-chan int,
	box Box[int],
	embedded EmbeddedBox,
) (int, bool) {
	inferred := [...]int{1, 2}
	value, mapOK := values["key"]
	if _, exists := values["missing"]; exists {
		value++
	}
	received, channelOK := <-channel
	asserted, assertOK := input.(int)
	converted := Number(value)
	aliasConverted := Alias(value)
	field := box.Value
	promoted := embedded.Nested
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
	for key, item := range values {
		value += len(key) + item
		break
	}
	var key string
	var item int
	_, item = pair()
	for key, item = range values {
		value += len(key) + item
		break
	}
	for _, item := range values {
		value += item
		break
	}
	for key, _ := range values {
		value += len(key)
		break
	}
	switch asserted {
	case 1:
		value++
	default:
		value--
	}
	return int(converted) + field + methodValue() +
		methodExpression(box) + received + pointer.Value +
		int(aliasConverted) + promoted + inferred[0] +
		int(unsafe.Sizeof(structValue)),
		mapOK && channelOK && assertOK
}

func Shadow(value int) int {
	result := value
	if result > 0 {
		value := result + 1
		result += value
	}
	return result + value
}

func FileScoped(value int) uintptr {
	return unsafe.Sizeof(value)
}

func DefinitionScoped() int {
	unsafe := 1
	return unsafe
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

func NamedSequence() Sequence {
	return func(yield func(int) bool) { yield(1) }
}

func GenericClosure[T any](value T) func() T {
	return func() T { return value }
}
`

func TestStage2ContextMatrixAndSemanticConservation(t *testing.T) {
	inspection := inspectStage2Fixture(t)
	pkg := semanticPackageByImportPath(
		t, inspection.Semantic(), "example.com/stage2",
	)

	if pkg.UnsupportedCount() != 0 {
		t.Fatalf(
			"unsupported records=%d, want 0",
			pkg.UnsupportedCount(),
		)
	}
	operations := semanticOperations(pkg)
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
	var namedFunctionReturnConversion bool
	for _, operation := range operations {
		if operation.Kind() != semantic.OperationReturn {
			continue
		}
		for _, effect := range operation.Spec().Implicit {
			if effect.Kind() ==
				catalog.ImplicitAssignmentConversion &&
				effect.Site().KindID() ==
					uint16(catalog.KindFuncLit) &&
				!effect.Source().IsZero() &&
				!effect.Target().IsZero() &&
				effect.Source() != effect.Target() {
				namedFunctionReturnConversion = true
			}
		}
	}
	if !namedFunctionReturnConversion {
		t.Fatal(
			"named-function return lacks its function-literal assignment conversion",
		)
	}

	var (
		capturedBase         bool
		capturedGenericValue bool
		typeParameterCount   int
	)
	for _, binding := range semanticBindings(pkg) {
		if binding.Name() == "_" {
			t.Fatal("blank identifier became a semantic binding")
		}
		if binding.Name() == "base" && len(binding.CapturedBy()) == 1 {
			capturedBase = true
		}
		if binding.Name() == "value" &&
			binding.Role() == identity.SemanticBindingParameter &&
			len(binding.CapturedBy()) == 1 {
			capturedGenericValue = true
		}
		if binding.Role() ==
			identity.SemanticBindingTypeParameter {
			typeParameterCount++
			if len(binding.CapturedBy()) != 0 {
				t.Fatalf(
					"type parameter %s has runtime captures %v",
					binding.ID(), binding.CapturedBy(),
				)
			}
		}
	}
	if !capturedBase {
		t.Fatal("closure capture for base was not materialized")
	}
	if !capturedGenericValue || typeParameterCount == 0 {
		t.Fatalf(
			"generic closure value capture=%t type parameters=%d",
			capturedGenericValue, typeParameterCount,
		)
	}
	requireMixedExplicitAndImplicitBindings(t, semanticBindings(pkg))
	var blankResolution bool
	for _, resolution := range semanticResolutions(pkg) {
		if resolution.Kind() == semantic.ResolutionStructuralOnly &&
			resolution.Structural().Disposition() ==
				semantic.StructuralBlankIdentifier {
			blankResolution = true
			break
		}
	}
	if !blankResolution {
		t.Fatal("blank identifier structural resolution is absent")
	}
	var inferredArrayEllipsis bool
	for _, resolution := range semanticResolutions(pkg) {
		if resolution.Syntax() == catalog.KindEllipsis &&
			resolution.Role() == catalog.RoleArrayLength &&
			resolution.Kind() == semantic.ResolutionType {
			inferredArrayEllipsis = true
			break
		}
	}
	if !inferredArrayEllipsis {
		t.Fatal("inferred array ellipsis type resolution is absent")
	}

	typeKinds := map[semantic.TypeKind]int{}
	for _, record := range semanticTypes(pkg) {
		typeKinds[record.Kind()]++
	}
	if typeKinds[semantic.TypeAlias] == 0 ||
		typeKinds[semantic.TypeNamed] == 0 ||
		typeKinds[semantic.TypeParameter] == 0 {
		t.Fatalf("semantic type classes=%v", typeKinds)
	}
	requireDeclarationCardinalityAndMethodDescriptors(t, pkg)

	var (
		initDefinitions          int
		initializerCardinalities []int
	)
	for _, definition := range semanticDefinitions(pkg) {
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
		fmt.Sprint(initializerCardinalities) != "[0 1 1 2]" {
		t.Fatalf(
			"init definitions=%d initializer declaration cardinalities=%v",
			initDefinitions, initializerCardinalities,
		)
	}

	work := inspection.SemanticWork()
	totalOperations := 0
	totalUnsupported := 0
	if err := inspection.Semantic().VisitPackages(
		func(candidate semantic.Package) error {
			totalOperations += candidate.OperationCount()
			totalUnsupported += candidate.UnsupportedCount()
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
	if work.InputOccurrences == 0 ||
		work.ContextAssignments != work.InputOccurrences ||
		work.ObjectOccurrenceVisits != work.InputOccurrences ||
		work.ImplicitBindingVisits != work.InputOccurrences ||
		work.CaptureOccurrenceVisits != work.InputOccurrences ||
		work.ResolutionVisits != work.InputOccurrences ||
		work.OccurrenceResolutions != work.InputOccurrences ||
		work.OperationConstructions !=
			totalOperations+totalUnsupported {
		t.Fatalf("semantic work is not conserved: %+v", work)
	}
	if work.DefinitionContainmentVisits !=
		2*work.DefinitionContainmentEntries ||
		work.DefinitionContainmentEdges >
			work.DefinitionContainmentEntries ||
		work.LinearOperations() == 0 {
		t.Fatalf("semantic containment work is not bounded: %+v", work)
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
	directory := writeStage2Project(t)
	inspection, err := inspectConstructsForTest(t, source.Request{
		Dir: directory, Patterns: []string{"."},
		ProviderContract: contract.DefaultID,
	})
	if err != nil {
		t.Fatal(err)
	}
	return inspection
}

func writeStage2Project(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	writeCompilerFile(
		t,
		directory,
		"go.mod",
		"module example.com/stage2\n\ngo 1.26.0\n",
	)
	writeCompilerFile(t, directory, "matrix.go", stage2ContextMatrix)
	writeCompilerFile(
		t,
		directory,
		"left/left.go",
		"package left\n\ntype Local interface { hidden() }\n\ntype Product struct { SKU string }\n\nfunc (product Product) Code() string { return product.SKU }\n",
	)
	writeCompilerFile(
		t,
		directory,
		"right/right.go",
		"package right\n\nconst Value = 1\n\ntype Local interface { hidden() }\n",
	)
	return directory
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
			for _, definition := range semanticDefinitions(pkg) {
				out = append(
					out, "definition:"+definition.Definition().String(),
				)
			}
			for _, declaration := range semanticDeclarations(pkg) {
				out = append(
					out, "declaration:"+declaration.ID().String(),
				)
			}
			for _, binding := range semanticBindings(pkg) {
				out = append(out, "binding:"+binding.ID().String())
			}
			for _, record := range semanticTypes(pkg) {
				out = append(out, "type:"+record.ID().String())
			}
			for _, operation := range semanticOperations(pkg) {
				out = append(out, "operation:"+operation.ID().String())
			}
			return nil
		},
	)
	sort.Strings(out)
	return out
}
