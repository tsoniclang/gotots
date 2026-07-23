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

func TestIntrinsicContractAdmitsOnlyHeaderTypeStructure(t *testing.T) {
	if !AllowsIntrinsicContract(
		KindStarExpr,
		RoleTypeExpression,
		ResolutionDomainHeader,
	) {
		t.Fatal("intrinsic contract rejected pointer type syntax")
	}
	if AllowsIntrinsicContract(
		KindAssignStmt,
		RoleStatement,
		ResolutionDomainHeader,
	) {
		t.Fatal("intrinsic contract admitted statement syntax")
	}
	if AllowsIntrinsicContract(
		KindStarExpr,
		RoleTypeExpression,
		ResolutionDomainExecutable,
	) {
		t.Fatal("intrinsic contract admitted executable domain")
	}
}

func TestIdentifierResolutionUsesParentAssignedRole(t *testing.T) {
	if !AllowsResolution(
		KindIdent,
		RoleTypeExpression,
		VariantNone,
		ResolutionDomainHeader,
		ResolutionClassType,
	) {
		t.Fatal("type-expression identifier cannot resolve as a type")
	}
	if AllowsResolution(
		KindIdent,
		RoleTypeExpression,
		VariantNone,
		ResolutionDomainHeader,
		ResolutionClassOperation,
	) {
		t.Fatal("type-expression identifier can resolve as an operation")
	}
	if !AllowsResolution(
		KindIdent,
		RoleAssignedValue,
		VariantNone,
		ResolutionDomainExecutable,
		ResolutionClassOperation,
	) {
		t.Fatal("value identifier cannot resolve as an operation")
	}
	for _, class := range []ResolutionClass{
		ResolutionClassType,
		ResolutionClassOperation,
	} {
		if !AllowsResolution(
			KindIdent,
			RoleOperand,
			VariantNone,
			ResolutionDomainExecutable,
			class,
		) {
			t.Fatalf(
				"operand identifier rejects context-selected %v",
				class,
			)
		}
	}
}

func TestTypeSetUnionBinaryExpressionResolvesAsType(t *testing.T) {
	if !AllowsResolution(
		KindBinaryExpr,
		RoleTypeExpression,
		VariantNone,
		ResolutionDomainHeader,
		ResolutionClassType,
	) {
		t.Fatal("type-set union binary expression cannot resolve as a type")
	}
	if !AllowsResolution(
		KindUnaryExpr,
		RoleLeftOperand,
		VariantNone,
		ResolutionDomainHeader,
		ResolutionClassType,
	) {
		t.Fatal("type-set approximation unary expression cannot resolve as a type")
	}
	if !AllowsResolution(
		KindBinaryExpr,
		RoleOperand,
		VariantNone,
		ResolutionDomainExecutable,
		ResolutionClassOperation,
	) {
		t.Fatal("ordinary binary expression cannot resolve as an operation")
	}
	for _, role := range []Role{
		RoleLeftOperand,
		RoleRightOperand,
		RoleIndexedOperand,
	} {
		if !AllowsResolution(
			KindIdent,
			role,
			VariantNone,
			ResolutionDomainHeader,
			ResolutionClassType,
		) {
			t.Fatalf("type-set %s cannot resolve as a type", role)
		}
		if !AllowsResolution(
			KindIdent,
			role,
			VariantNone,
			ResolutionDomainExecutable,
			ResolutionClassOperation,
		) {
			t.Fatalf("ordinary %s cannot resolve as an operation", role)
		}
	}
	for _, class := range []ResolutionClass{
		ResolutionClassBinding,
		ResolutionClassType,
	} {
		if !AllowsResolution(
			KindIdent,
			RoleIndex,
			VariantNone,
			ResolutionDomainHeader,
			class,
		) {
			t.Fatalf("generic index rejects context-selected %v", class)
		}
	}
	if !AllowsResolution(
		KindIdent,
		RoleIndex,
		VariantNone,
		ResolutionDomainExecutable,
		ResolutionClassOperation,
	) {
		t.Fatal("executable index rejects context-selected operation")
	}
}
