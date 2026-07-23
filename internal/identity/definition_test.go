package identity

import "testing"

func TestDefinitionKindIDsArePinned(t *testing.T) {
	want := map[DefinitionKind]uint8{
		DefinitionFuncDecl:           1,
		DefinitionFuncLit:            2,
		DefinitionPackageInitializer: 3,
		DefinitionBodylessDecl:       4,
		DefinitionImplicit:           5,
	}
	for kind, id := range want {
		if !kind.Valid() || uint8(kind) != id {
			t.Errorf("definition kind %s = %d, want %d", kind, kind, id)
		}
	}
	if DefinitionKind(6).Valid() {
		t.Fatal("unpinned definition kind was admitted")
	}
	if ImplicitDefinitionPackageInit != 1 ||
		SyntheticDefinitionAdapter != 1 ||
		SyntheticDefinitionType != 2 ||
		SyntheticDefinitionData != 3 {
		t.Fatal("implicit or synthetic identity IDs changed")
	}
}

func TestDefinitionIdentityFormsRoundTrip(t *testing.T) {
	owner := mustModuleOwner(t, "example.com/definitions", "v1.2.3")
	pkg, err := NewPackageID(owner, "example.com/definitions/pkg")
	if err != nil {
		t.Fatal(err)
	}
	file, err := NewFileID(owner, "pkg/definitions.go")
	if err != nil {
		t.Fatal(err)
	}
	span, err := NewSpanID(file, 10, 42)
	if err != nil {
		t.Fatal(err)
	}
	root, err := NewOccurrenceID(span, 47)
	if err != nil {
		t.Fatal(err)
	}

	var definitions []DefinitionID
	for kind := DefinitionKind(1); kind.Valid(); kind++ {
		if !kind.Source() {
			continue
		}
		definition, err := NewSourceDefinitionID(root, kind)
		if err != nil {
			t.Fatal(err)
		}
		definitions = append(definitions, definition)
	}
	implicit, err := NewImplicitDefinitionID(
		pkg, ImplicitDefinitionPackageInit,
	)
	if err != nil {
		t.Fatal(err)
	}
	definitions = append(definitions, implicit)
	for role := SyntheticDefinitionRole(1); role.Valid(); role++ {
		definition, err := NewSyntheticDefinitionID(
			pkg, role, "generatedName",
		)
		if err != nil {
			t.Fatal(err)
		}
		definitions = append(definitions, definition)
	}

	for _, definition := range definitions {
		parsed, err := ParseDefinitionID(definition.String())
		if err != nil {
			t.Fatalf("ParseDefinitionID(%q): %v", definition, err)
		}
		if parsed != definition {
			t.Errorf("definition round trip = %s, want %s", parsed, definition)
		}
		header, err := NewHeaderRegionID(definition)
		if err != nil || header.Definition() != definition ||
			header.String() != definition.String()+"#header" {
			t.Errorf("header identity for %s = %s, %v", definition, header, err)
		}
		boundary, err := NewExecutionBoundaryID(definition)
		if err != nil || boundary.Definition() != definition ||
			boundary.String() != definition.String()+"#execution" {
			t.Errorf(
				"execution boundary for %s = %s, %v",
				definition,
				boundary,
				err,
			)
		}
		region, err := NewExecutableRegionID(definition)
		if err != nil || region.Definition() != definition ||
			region.String() != definition.String()+"#executable" {
			t.Errorf(
				"executable region for %s = %s, %v",
				definition,
				region,
				err,
			)
		}
	}
}

func TestDefinitionIdentitiesFailClosed(t *testing.T) {
	owner := mustModuleOwner(t, "example.com/definitions", "")
	pkg, err := NewPackageID(owner, "example.com/definitions")
	if err != nil {
		t.Fatal(err)
	}
	file, err := NewFileID(owner, "definitions.go")
	if err != nil {
		t.Fatal(err)
	}
	span, err := NewSpanID(file, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	root, err := NewOccurrenceID(span, 47)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := NewSourceDefinitionID(
		OccurrenceID{}, DefinitionFuncDecl,
	); err == nil {
		t.Fatal("source definition accepted a zero occurrence")
	}
	if _, err := NewSourceDefinitionID(
		root, DefinitionImplicit,
	); err == nil {
		t.Fatal("source definition accepted an implicit kind")
	}
	if _, err := NewImplicitDefinitionID(
		PackageID{}, ImplicitDefinitionPackageInit,
	); err == nil {
		t.Fatal("implicit definition accepted a zero package")
	}
	if _, err := NewSyntheticDefinitionID(
		pkg, SyntheticDefinitionInvalid, "name",
	); err == nil {
		t.Fatal("synthetic definition accepted an invalid role")
	}
	if _, err := NewSyntheticDefinitionID(
		pkg, SyntheticDefinitionAdapter, "bad#name",
	); err == nil {
		t.Fatal("synthetic definition accepted a reserved name")
	}
	for _, malformed := range []string{
		"",
		root.String() + "/D0",
		root.String() + "/D6",
		pkg.String() + "#definition/unknown",
		pkg.String() + "#synthetic/unknown/name",
		pkg.String() + "#synthetic/adapter/",
	} {
		if _, err := ParseDefinitionID(malformed); err == nil {
			t.Errorf("ParseDefinitionID(%q) accepted malformed input", malformed)
		}
	}
	for _, constructor := range []func() error{
		func() error {
			_, err := NewHeaderRegionID(DefinitionID{})
			return err
		},
		func() error {
			_, err := NewExecutionBoundaryID(DefinitionID{})
			return err
		},
		func() error {
			_, err := NewExecutableRegionID(DefinitionID{})
			return err
		},
	} {
		if err := constructor(); err == nil {
			t.Fatal("derived region accepted a zero definition")
		}
	}
}

func TestCanonicalIdentityParsersRejectAlternateSpellings(t *testing.T) {
	owner := mustModuleOwner(t, "example.com/parsing", "v1.0.0")
	pkg, err := NewPackageID(owner, "example.com/parsing/pkg")
	if err != nil {
		t.Fatal(err)
	}
	file, err := NewFileID(owner, "pkg/file.go")
	if err != nil {
		t.Fatal(err)
	}
	span, err := NewSpanID(file, 7, 19)
	if err != nil {
		t.Fatal(err)
	}
	occurrence, err := NewOccurrenceID(span, 47)
	if err != nil {
		t.Fatal(err)
	}

	if parsed, err := ParseOwner(owner.String()); err != nil || parsed != owner {
		t.Fatalf("owner round trip = %s, %v", parsed, err)
	}
	if parsed, err := ParsePackageID(pkg.String()); err != nil || parsed != pkg {
		t.Fatalf("package round trip = %s, %v", parsed, err)
	}
	if parsed, err := ParseFileID(file.String()); err != nil || parsed != file {
		t.Fatalf("file round trip = %s, %v", parsed, err)
	}
	if parsed, err := ParseSpanID(span.String()); err != nil || parsed != span {
		t.Fatalf("span round trip = %s, %v", parsed, err)
	}
	if parsed, err := ParseOccurrenceID(
		occurrence.String(),
	); err != nil || parsed != occurrence {
		t.Fatalf("occurrence round trip = %s, %v", parsed, err)
	}

	for _, malformed := range []string{
		"mod=example.com/parsing@",
		"module=example.com/parsing",
		"STD",
		"std::fmt",
	} {
		if _, err := ParseOwner(malformed); err == nil {
			t.Errorf("ParseOwner(%q) accepted alternate spelling", malformed)
		}
	}
}
