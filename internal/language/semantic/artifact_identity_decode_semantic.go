package semantic

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

func (decoder wireIdentityDecoder) decodeDefinition(
	record wireDefinitionIdentity,
) (identity.DefinitionID, error) {
	kind := identity.DefinitionKind(record.Kind)
	root, err := decoder.occurrence(record.Root)
	if err != nil {
		return identity.DefinitionID{}, err
	}
	pkg, err := decoder.packageID(record.Package)
	if err != nil {
		return identity.DefinitionID{}, err
	}
	implicit := identity.ImplicitDefinitionOp(record.Implicit)
	synthetic := identity.SyntheticDefinitionRole(record.Synthetic)
	switch {
	case kind.Source():
		if !pkg.IsZero() || implicit != 0 ||
			synthetic != 0 || record.Name != "" {
			return identity.DefinitionID{}, fmt.Errorf(
				"source definition identity carries inactive fields",
			)
		}
		return identity.NewSourceDefinitionID(root, kind)
	case kind == identity.DefinitionImplicit &&
		implicit.Valid():
		if !root.IsZero() || synthetic != 0 || record.Name != "" {
			return identity.DefinitionID{}, fmt.Errorf(
				"implicit definition identity carries inactive fields",
			)
		}
		return identity.NewImplicitDefinitionID(pkg, implicit)
	case kind == identity.DefinitionImplicit &&
		synthetic.Valid():
		if !root.IsZero() || implicit != 0 {
			return identity.DefinitionID{}, fmt.Errorf(
				"synthetic definition identity carries inactive fields",
			)
		}
		return identity.NewSyntheticDefinitionID(
			pkg,
			synthetic,
			record.Name,
		)
	default:
		return identity.DefinitionID{}, fmt.Errorf(
			"semantic wire definition identity is invalid",
		)
	}
}

func (decoder wireIdentityDecoder) decodeDeclaration(
	record wireDeclarationIdentity,
) (identity.SemanticDeclarationID, error) {
	form := identity.SemanticDeclarationForm(record.Form)
	pkg, err := decoder.packageID(record.Package)
	if err != nil {
		return identity.SemanticDeclarationID{}, err
	}
	ownerType, err := decoder.typeID(record.OwnerType)
	if err != nil {
		return identity.SemanticDeclarationID{}, err
	}
	memberPackage, err := decoder.packageID(record.MemberPkg)
	if err != nil {
		return identity.SemanticDeclarationID{}, err
	}
	owner, err := decoder.occurrence(record.Owner)
	if err != nil {
		return identity.SemanticDeclarationID{}, err
	}
	occurrence, err := decoder.occurrence(record.Occurrence)
	if err != nil {
		return identity.SemanticDeclarationID{}, err
	}
	class := identity.SemanticObjectClass(record.Class)
	switch form {
	case identity.SemanticDeclarationPackageObject:
		if !ownerType.IsZero() || !memberPackage.IsZero() ||
			record.Ordinal != 0 || record.Predeclared != 0 ||
			!owner.IsZero() || !occurrence.IsZero() {
			return identity.SemanticDeclarationID{}, fmt.Errorf(
				"package declaration identity carries inactive fields",
			)
		}
		return identity.NewPackageDeclarationID(
			pkg,
			class,
			record.Name,
		)
	case identity.SemanticDeclarationMember:
		if !pkg.IsZero() || record.Predeclared != 0 ||
			!owner.IsZero() || !occurrence.IsZero() {
			return identity.SemanticDeclarationID{}, fmt.Errorf(
				"member declaration identity carries inactive fields",
			)
		}
		return identity.NewMemberDeclarationID(
			ownerType,
			memberPackage,
			class,
			record.Name,
			record.Ordinal,
		)
	case identity.SemanticDeclarationPredeclared:
		if !pkg.IsZero() || !ownerType.IsZero() ||
			!memberPackage.IsZero() || record.Name != "" ||
			record.Ordinal != 0 || !owner.IsZero() ||
			!occurrence.IsZero() {
			return identity.SemanticDeclarationID{}, fmt.Errorf(
				"predeclared declaration identity carries inactive fields",
			)
		}
		return identity.NewPredeclaredDeclarationID(
			record.Predeclared,
			class,
		)
	case identity.SemanticDeclarationOccurrence:
		if !pkg.IsZero() || !ownerType.IsZero() ||
			!memberPackage.IsZero() || record.Predeclared != 0 {
			return identity.SemanticDeclarationID{}, fmt.Errorf(
				"occurrence declaration identity carries inactive fields",
			)
		}
		return identity.NewOccurrenceDeclarationID(
			owner,
			occurrence,
			class,
			record.Name,
			record.Ordinal,
		)
	default:
		return identity.SemanticDeclarationID{}, fmt.Errorf(
			"semantic wire declaration form %d is invalid",
			record.Form,
		)
	}
}

func (decoder wireIdentityDecoder) decodeBinding(
	record wireBindingIdentity,
) (identity.SemanticBindingID, error) {
	owner, err := decoder.occurrence(record.Owner)
	if err != nil {
		return identity.SemanticBindingID{}, err
	}
	declaration, err := decoder.occurrence(record.Declaration)
	if err != nil {
		return identity.SemanticBindingID{}, err
	}
	return identity.NewSemanticBindingID(
		owner,
		declaration,
		identity.SemanticBindingRole(record.Role),
		record.Ordinal,
	)
}

func (decoder wireIdentityDecoder) decodeOperation(
	record wireOperationIdentity,
) (identity.OperationID, error) {
	definition, err := decoder.definition(record.Definition)
	if err != nil {
		return identity.OperationID{}, err
	}
	occurrence, err := decoder.occurrence(record.Occurrence)
	if err != nil {
		return identity.OperationID{}, err
	}
	implicit := identity.ImplicitDefinitionOp(record.Implicit)
	if implicit.Valid() {
		if !occurrence.IsZero() {
			return identity.OperationID{}, fmt.Errorf(
				"implicit operation identity carries occurrence",
			)
		}
		return identity.NewImplicitOperationID(
			definition,
			implicit,
			record.Ordinal,
		)
	}
	if record.Implicit != 0 || record.Ordinal != 0 {
		return identity.OperationID{}, fmt.Errorf(
			"source operation identity carries implicit fields",
		)
	}
	return identity.NewOperationID(definition, occurrence)
}

func (decoder wireIdentityDecoder) decodeUnsupported(
	record wireUnsupportedIdentity,
) (identity.UnsupportedID, error) {
	definition, err := decoder.definition(record.Definition)
	if err != nil {
		return identity.UnsupportedID{}, err
	}
	occurrence, err := decoder.occurrence(record.Occurrence)
	if err != nil {
		return identity.UnsupportedID{}, err
	}
	return identity.NewUnsupportedID(definition, occurrence)
}
