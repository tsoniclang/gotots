package api

import (
	"go/token"
	"go/types"
	"testing"
)

func TestDeclarationSelectionIsConstantForSharedHomogeneousGraph(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/selection", "selection")
	typeName := types.NewTypeName(token.Pos(1), sourcePackage, "Value", nil)
	requirement, err := NewNamedStructOperationRequirement(
		typeName,
		NamedStructOperationCopy,
	)
	if err != nil {
		t.Fatal(err)
	}
	request, err := NewDeclarationRequirementRequest(requirement)
	if err != nil {
		t.Fatal(err)
	}
	requests := []RootRequest{request}
	for range 24 {
		requests = CombineRequests(requests, requests)
	}
	selected, work, err := selectRootRequestsWithWork(
		requests,
		declarationRequestKindMask,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 1 || selected[0] != requests[0] {
		t.Fatal("selection rebuilt the homogeneous request graph")
	}
	if work != 1 {
		t.Fatalf("selection work = %d, want 1", work)
	}
}

func TestRootRequestSelectionRejectsEmptySequence(t *testing.T) {
	_, err := SelectDeclarationRequests([]RootRequest{{
		sequence: &rootRequestSequence{},
	}})
	if err == nil {
		t.Fatal("empty root request sequence was accepted")
	}
}
