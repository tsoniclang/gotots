package frontend

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/tsoniclang/gotots/internal/identity"
)

func memberOwnerDescription(owners []types.Type) string {
	if len(owners) == 0 {
		return "<none>"
	}
	names := make([]string, 0, len(owners))
	for _, owner := range owners {
		names = append(names, types.TypeString(owner, nil))
	}
	return strings.Join(names, ",")
}

func memberObject(object types.Object) bool {
	if function, ok := object.(*types.Func); ok {
		signature, _ := function.Type().(*types.Signature)
		return signature != nil && signature.Recv() != nil
	}
	field, ok := object.(*types.Var)
	return ok && field.IsField()
}

func (index *objectIndex) canonicalDeclarationObject(
	object types.Object,
) types.Object {
	if function, ok := object.(*types.Func); ok {
		return function.Origin()
	}
	field, ok := object.(*types.Var)
	if !ok || !field.IsField() {
		return object
	}
	owners := index.memberOwnerRelations[field]
	if len(owners) != 1 {
		return object
	}
	owner := owners[0]
	ordinal, err := memberOrdinal(field, owner)
	if err != nil {
		return object
	}
	canonical, err := canonicalMemberObject(
		field, owner, ordinal,
	)
	if err != nil {
		return object
	}
	index.admitMemberOwner(canonical, owner)
	return canonical
}

func (index *objectIndex) declarationIDForSelection(
	selection *types.Selection,
) (identity.SemanticDeclarationID, error) {
	if selection == nil || selection.Obj() == nil {
		return identity.SemanticDeclarationID{}, fmt.Errorf(
			"semantic selection declaration requires checker selection",
		)
	}
	owner, err := selectionMemberOwner(selection)
	if err != nil {
		return identity.SemanticDeclarationID{}, err
	}
	ordinal := 0
	if field, ok := selection.Obj().(*types.Var); ok && field.IsField() {
		path := selection.Index()
		if len(path) == 0 {
			return identity.SemanticDeclarationID{}, fmt.Errorf(
				"semantic field selection %s has no index",
				field.Name(),
			)
		}
		ordinal = path[len(path)-1]
	}
	return index.declarationIDForMemberOwner(
		selection.Obj(), owner, ordinal,
	)
}

func (index *objectIndex) declarationIDForMemberOwner(
	object types.Object,
	owner types.Type,
	ordinal int,
) (identity.SemanticDeclarationID, error) {
	class, err := semanticObjectClass(object)
	if err != nil {
		return identity.SemanticDeclarationID{}, err
	}
	if class != identity.SemanticObjectField &&
		class != identity.SemanticObjectMethod {
		return identity.SemanticDeclarationID{}, fmt.Errorf(
			"checker object %s is not a member", object.Name(),
		)
	}
	owner = originMemberOwner(owner)
	ownerID, err := index.typeBuilder.memberOwnerTypeID(owner)
	if err != nil {
		return identity.SemanticDeclarationID{}, err
	}
	memberPackage := identity.PackageID{}
	if !object.Exported() {
		memberPackage, err = index.packageID(object.Pkg())
		if err != nil {
			return identity.SemanticDeclarationID{}, err
		}
	}
	if class == identity.SemanticObjectMethod {
		ordinal = 0
	}
	id, err := identity.NewMemberDeclarationID(
		ownerID,
		memberPackage,
		class,
		object.Name(),
		ordinal,
	)
	if err != nil {
		return identity.SemanticDeclarationID{}, err
	}
	canonical, err := canonicalMemberObject(
		object, owner, ordinal,
	)
	if err != nil {
		return identity.SemanticDeclarationID{}, err
	}
	admitted, err := index.admitDeclarationID(
		canonical, id, false,
	)
	if err != nil {
		return identity.SemanticDeclarationID{}, err
	}
	ownerPackage, err := index.memberOwnerPackage(owner, object)
	if err != nil {
		return identity.SemanticDeclarationID{}, err
	}
	if existing, present :=
		index.declarationOwnerPackage[admitted]; present &&
		existing != ownerPackage {
		return identity.SemanticDeclarationID{}, fmt.Errorf(
			"semantic member declaration %s has owner packages %s and %s",
			admitted, existing, ownerPackage,
		)
	}
	index.declarationOwnerPackage[admitted] = ownerPackage
	return admitted, nil
}

func (index *objectIndex) memberOwnerPackage(
	owner types.Type,
	object types.Object,
) (identity.PackageID, error) {
	owner = stripMemberOwnerPointer(owner)
	if named, ok := owner.(*types.Named); ok &&
		named.Obj().Pkg() != nil {
		return index.packageID(named.Obj().Pkg())
	}
	if object.Pkg() != nil {
		return index.packageID(object.Pkg())
	}
	return identity.PackageID{}, nil
}

func canonicalMemberObject(
	object types.Object,
	owner types.Type,
	ordinal int,
) (types.Object, error) {
	if function, ok := object.(*types.Func); ok {
		return function.Origin(), nil
	}
	field, ok := object.(*types.Var)
	if !ok || !field.IsField() {
		return nil, fmt.Errorf(
			"checker object %s is not a field or method",
			object.Name(),
		)
	}
	structure, ok := types.Unalias(
		stripMemberOwnerPointer(owner),
	).Underlying().(*types.Struct)
	if !ok || ordinal < 0 || ordinal >= structure.NumFields() {
		return nil, fmt.Errorf(
			"field %s ordinal %d is outside owner %s",
			field.Name(),
			ordinal,
			types.TypeString(owner, nil),
		)
	}
	canonical := structure.Field(ordinal)
	if canonical.Name() != field.Name() {
		return nil, fmt.Errorf(
			"field %s ordinal %d resolves to %s in owner %s",
			field.Name(),
			ordinal,
			canonical.Name(),
			types.TypeString(owner, nil),
		)
	}
	return canonical, nil
}

func memberOrdinal(
	object types.Object,
	owner types.Type,
) (int, error) {
	field, ok := object.(*types.Var)
	if !ok || !field.IsField() {
		return 0, nil
	}
	structure, ok := types.Unalias(
		stripMemberOwnerPointer(owner),
	).Underlying().(*types.Struct)
	if !ok {
		return 0, fmt.Errorf(
			"field %s has non-struct owner %s",
			field.Name(),
			types.TypeString(owner, nil),
		)
	}
	for ordinal := 0; ordinal < structure.NumFields(); ordinal++ {
		candidate := structure.Field(ordinal)
		if candidate == object ||
			(candidate.Name() == object.Name() &&
				candidate.Exported() == object.Exported() &&
				candidate.Embedded() == field.Embedded()) {
			return ordinal, nil
		}
	}
	return 0, fmt.Errorf(
		"field %s is absent from owner %s",
		field.Name(),
		types.TypeString(owner, nil),
	)
}

func selectionMemberOwner(
	selection *types.Selection,
) (types.Type, error) {
	if function, ok := selection.Obj().(*types.Func); ok {
		signature, _ := function.Origin().Type().(*types.Signature)
		if signature == nil || signature.Recv() == nil {
			return nil, fmt.Errorf(
				"selected method %s has no origin receiver",
				function.Name(),
			)
		}
		return originMemberOwner(signature.Recv().Type()), nil
	}
	current := stripMemberOwnerPointer(selection.Recv())
	path := selection.Index()
	if len(path) == 0 {
		return nil, fmt.Errorf(
			"selected field %s has no index",
			selection.Obj().Name(),
		)
	}
	for _, part := range path[:len(path)-1] {
		structure, ok := types.Unalias(current).
			Underlying().(*types.Struct)
		if !ok || part < 0 || part >= structure.NumFields() {
			return nil, fmt.Errorf(
				"selection path for %s crosses non-struct %s",
				selection.Obj().Name(),
				types.TypeString(current, nil),
			)
		}
		current = stripMemberOwnerPointer(
			structure.Field(part).Type(),
		)
	}
	return originMemberOwner(current), nil
}
