package identity

import (
	"slices"
	"strings"
	"testing"
)

type canonicalOrder[Self any] interface {
	comparable
	Compare(Self) int
}

func assertCanonicalOrder[ID canonicalOrder[ID]](
	t *testing.T,
	values []ID,
) {
	t.Helper()
	for leftIndex, left := range values {
		if left.Compare(left) != 0 {
			t.Fatalf("identity %d does not compare equal to itself", leftIndex)
		}
		for rightIndex, right := range values {
			forward := left.Compare(right)
			reverse := right.Compare(left)
			if (forward == 0) != (left == right) {
				t.Fatalf(
					"identity comparison equality differs at %d/%d",
					leftIndex,
					rightIndex,
				)
			}
			if sign(forward) != -sign(reverse) {
				t.Fatalf(
					"identity comparison is not antisymmetric at %d/%d",
					leftIndex,
					rightIndex,
				)
			}
		}
	}
	ordered := append([]ID(nil), values...)
	slices.SortFunc(ordered, func(left, right ID) int {
		return left.Compare(right)
	})
	for index := 1; index < len(ordered); index++ {
		if ordered[index-1].Compare(ordered[index]) >= 0 {
			t.Fatalf("identity order is not strict at %d", index)
		}
	}
}

func sign(value int) int {
	switch {
	case value < 0:
		return -1
	case value > 0:
		return 1
	default:
		return 0
	}
}

func TestCanonicalIdentityOrderIsStructuralAndAllocationFree(
	t *testing.T,
) {
	moduleA := mustModule(t, "example.com/a", "")
	moduleAV1 := mustModule(t, "example.com/a", "v1.0.0")
	moduleB := mustModule(t, "example.com/b", "")
	ownerA, _ := NewModuleOwner(moduleA)
	ownerAV1, _ := NewModuleOwner(moduleAV1)
	packageA, _ := NewPackageID(ownerA, "example.com/a")
	packageChild, _ := NewPackageID(ownerA, "example.com/a/child")
	packageVersioned, _ := NewPackageID(ownerAV1, "example.com/a")
	fileA, _ := NewFileID(ownerA, "a.go")
	fileB, _ := NewFileID(ownerA, "b.go")
	spanTwo, _ := NewSpanID(fileA, 2, 3)
	spanTen, _ := NewSpanID(fileA, 10, 11)
	occurrenceTwo, _ := NewOccurrenceID(spanTwo, 2)
	occurrenceTen, _ := NewOccurrenceID(spanTen, 10)
	definitionTwo, _ := NewSourceDefinitionID(
		occurrenceTwo, DefinitionFuncDecl,
	)
	definitionTen, _ := NewSourceDefinitionID(
		occurrenceTen, DefinitionFuncLit,
	)
	implicit, _ := NewImplicitDefinitionID(
		packageA, ImplicitDefinitionPackageInit,
	)
	headerTwo, _ := NewHeaderRegionID(definitionTwo)
	headerTen, _ := NewHeaderRegionID(definitionTen)
	boundaryTwo, _ := NewExecutionBoundaryID(definitionTwo)
	boundaryTen, _ := NewExecutionBoundaryID(definitionTen)
	regionTwo, _ := NewExecutableRegionID(definitionTwo)
	regionTen, _ := NewExecutableRegionID(definitionTen)
	typeA, _ := NewSemanticTypeID(strings.Repeat("0a", 32))
	typeB, _ := NewSemanticTypeID(strings.Repeat("0b", 32))
	declarationA, _ := NewPackageDeclarationID(
		packageA, SemanticObjectFunction, "A",
	)
	declarationB, _ := NewMemberDeclarationID(
		typeA, PackageID{}, SemanticObjectMethod, "B", 0,
	)
	bindingTwo, _ := NewSemanticBindingID(
		occurrenceTwo,
		OccurrenceID{},
		SemanticBindingParameter,
		2,
	)
	bindingTen, _ := NewSemanticBindingID(
		occurrenceTwo,
		occurrenceTen,
		SemanticBindingParameter,
		10,
	)
	operationTwo, _ := NewOperationID(definitionTwo, occurrenceTwo)
	operationTen, _ := NewOperationID(definitionTen, occurrenceTen)
	unsupportedTwo, _ := NewUnsupportedID(
		definitionTwo, occurrenceTwo,
	)
	unsupportedTen, _ := NewUnsupportedID(
		definitionTen, occurrenceTen,
	)

	assertCanonicalOrder(t, []ModuleID{{}, moduleA, moduleAV1, moduleB})
	assertCanonicalOrder(t, []Owner{
		{}, ownerA, ownerAV1, StandardLibraryOwner(), ToolchainOwner(),
		LanguagePseudoOwner(),
	})
	assertCanonicalOrder(t, []PackageID{
		{}, packageA, packageChild, packageVersioned,
	})
	assertCanonicalOrder(t, []FileID{{}, fileA, fileB})
	assertCanonicalOrder(t, []SpanID{{}, spanTwo, spanTen})
	assertCanonicalOrder(t, []OccurrenceID{
		{}, occurrenceTwo, occurrenceTen,
	})
	assertCanonicalOrder(t, []DefinitionID{
		{}, definitionTwo, definitionTen, implicit,
	})
	assertCanonicalOrder(t, []HeaderRegionID{
		{}, headerTwo, headerTen,
	})
	assertCanonicalOrder(t, []ExecutionBoundaryID{
		{}, boundaryTwo, boundaryTen,
	})
	assertCanonicalOrder(t, []ExecutableRegionID{
		{}, regionTwo, regionTen,
	})
	assertCanonicalOrder(t, []SemanticTypeID{{}, typeA, typeB})
	assertCanonicalOrder(t, []SemanticDeclarationID{
		{}, declarationA, declarationB,
	})
	assertCanonicalOrder(t, []SemanticBindingID{
		{}, bindingTwo, bindingTen,
	})
	assertCanonicalOrder(t, []OperationID{
		{}, operationTwo, operationTen,
	})
	assertCanonicalOrder(t, []UnsupportedID{
		{}, unsupportedTwo, unsupportedTen,
	})

	if spanTwo.Compare(spanTen) >= 0 {
		t.Fatal("numeric span order was replaced by decimal-string order")
	}
	if allocations := testing.AllocsPerRun(1000, func() {
		_ = operationTwo.Compare(operationTen)
		_ = declarationA.Compare(declarationB)
		_ = bindingTwo.Compare(bindingTen)
	}); allocations != 0 {
		t.Fatalf("typed identity comparisons allocate: %.2f", allocations)
	}
}
