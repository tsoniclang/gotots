package sourcefact

import (
	"go/token"
	"go/types"
	"testing"
)

func TestMemberOriginSetRejectsInexactDenominators(t *testing.T) {
	packageType := types.NewPackage("example.com/sourcefact", "sourcefact")
	first := types.NewField(token.NoPos, packageType, "First", types.Typ[types.Int32], false)
	second := types.NewField(token.NoPos, packageType, "Second", types.Typ[types.Int32], false)
	firstOrigin := testDeclarationOrigin(t, "")
	secondOrigin := testDeclarationOrigin(t, "")

	for name, testCase := range map[string]struct {
		objects []types.Object
		origins []DeclarationOrigin
	}{
		"missing origin":   {[]types.Object{first}, nil},
		"duplicate object": {[]types.Object{first, first}, []DeclarationOrigin{firstOrigin, secondOrigin}},
		"nil object":       {[]types.Object{nil}, []DeclarationOrigin{firstOrigin}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewMemberOriginSet(testCase.objects, testCase.origins); err == nil {
				t.Fatal("inexact member origin denominator was admitted")
			}
		})
	}

	left := testDeclarationOrigin(t, "environment:left")
	right := testDeclarationOrigin(t, "environment:right")
	if _, err := NewMemberOriginSet(
		[]types.Object{first, second},
		[]DeclarationOrigin{left, right},
	); err == nil {
		t.Fatal("inconsistent environment member basis was admitted")
	}
}

func TestMemberOriginSetUsesOnlyExactFieldOrdinalFoils(t *testing.T) {
	packageType := types.NewPackage("example.com/sourcefact", "sourcefact")
	original := types.NewField(token.NoPos, packageType, "Value", types.Typ[types.Int32], false)
	origin := testDeclarationOrigin(t, "")
	set, err := NewMemberOriginSet(
		[]types.Object{original},
		[]DeclarationOrigin{origin},
	)
	if err != nil {
		t.Fatal(err)
	}
	equivalent := types.NewField(token.NoPos, packageType, "Value", types.Typ[types.Int32], false)
	if selected, ok := set.field(0, equivalent); !ok || selected != origin {
		t.Fatal("equivalent checked field did not exact-join by ordinal")
	}
	wrongName := types.NewField(token.NoPos, packageType, "Other", types.Typ[types.Int32], false)
	wrongType := types.NewField(token.NoPos, packageType, "Value", types.Typ[types.Uint32], false)
	if _, ok := set.field(0, wrongName); ok {
		t.Fatal("field-name mutation survived the exact ordinal join")
	}
	if _, ok := set.field(0, wrongType); ok {
		t.Fatal("field-type mutation survived the exact ordinal join")
	}
}

func testDeclarationOrigin(t *testing.T, basis string) DeclarationOrigin {
	t.Helper()
	origin, err := NewDeclarationOrigin(
		"example.com/sourcefact",
		"example.com/sourcefact",
		"",
		"workspace",
		"example.com/sourcefact@workspace",
		"modules/example.com/sourcefact/source.ts",
		"checked-syntax:source.go",
		"source-digest",
		"program-digest",
		1,
		2,
		"go1.26",
	)
	if err != nil {
		t.Fatal(err)
	}
	if basis != "" {
		origin, err = origin.WithEnvironmentBasis(basis)
		if err != nil {
			t.Fatal(err)
		}
	}
	return origin
}
