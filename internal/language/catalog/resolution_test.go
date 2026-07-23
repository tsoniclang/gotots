package catalog

import "testing"

func TestResolutionAndVariantTablesAreTotal(t *testing.T) {
	for _, kind := range AllKinds() {
		if resolutionByKind[kind] == 0 {
			t.Errorf("kind %s has no semantic resolution class", kind)
		}
		if VariantBearing(kind) {
			if VariantAllowed(kind, VariantNone) &&
				kind != KindUnaryExpr {
				t.Errorf(
					"variant-bearing kind %s accepts none", kind,
				)
			}
		} else if !VariantAllowed(kind, VariantNone) {
			t.Errorf(
				"non-variant kind %s rejects none", kind,
			)
		}
	}
	owners := map[Variant][]Kind{}
	for _, kind := range AllKinds() {
		for _, variant := range AllVariants() {
			if !VariantAllowed(kind, variant) ||
				variant == VariantNone {
				continue
			}
			owners[variant] = append(owners[variant], kind)
		}
	}
	for _, variant := range AllVariants() {
		if variant == VariantNone {
			continue
		}
		if len(owners[variant]) == 0 {
			t.Errorf("variant %s has no owning kind", variant)
		}
	}
	genericOwners := owners[VariantGenericInstantiation]
	if len(genericOwners) != 2 ||
		genericOwners[0] != KindIndexExpr ||
		genericOwners[1] != KindIndexListExpr {
		t.Fatalf(
			"generic-instantiation owners = %v, want IndexExpr and IndexListExpr",
			genericOwners,
		)
	}
	for variant, kinds := range owners {
		if variant != VariantGenericInstantiation &&
			len(kinds) != 1 {
			t.Errorf(
				"variant %s has unexpected owners %v",
				variant, kinds,
			)
		}
	}
}

func TestIdentifierResolutionUsesParentAssignedRole(t *testing.T) {
	if !AllowsResolution(
		KindIdent,
		RoleTypeExpression,
		VariantNone,
		ResolutionClassType,
	) {
		t.Fatal("type-expression identifier cannot resolve as a type")
	}
	if AllowsResolution(
		KindIdent,
		RoleTypeExpression,
		VariantNone,
		ResolutionClassOperation,
	) {
		t.Fatal("type-expression identifier can resolve as an operation")
	}
	if !AllowsResolution(
		KindIdent,
		RoleAssignedValue,
		VariantNone,
		ResolutionClassOperation,
	) {
		t.Fatal("value identifier cannot resolve as an operation")
	}
}
