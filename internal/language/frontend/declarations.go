package frontend

import (
	"fmt"
	"go/constant"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

type localDeclarationGroup struct {
	scope identity.OccurrenceID
	class identity.SemanticObjectClass
}

func (index *objectIndex) assignLocalDeclarationOrdinals() error {
	groups := map[localDeclarationGroup][]types.Object{}
	for object, source := range index.sourceByObject {
		class, local := localDeclarationClass(object)
		if !local || source.IsZero() {
			continue
		}
		scope, err := index.bindingScope(object, source)
		if err != nil {
			return err
		}
		key := localDeclarationGroup{
			scope: scope,
			class: class,
		}
		groups[key] = append(groups[key], object)
	}
	for _, objects := range groups {
		sort.Slice(objects, func(left, right int) bool {
			leftSource := index.sourceByObject[objects[left]]
			rightSource := index.sourceByObject[objects[right]]
			if leftSource.Span().Start() != rightSource.Span().Start() {
				return leftSource.Span().Start() <
					rightSource.Span().Start()
			}
			if leftSource.Span().End() != rightSource.Span().End() {
				return leftSource.Span().End() <
					rightSource.Span().End()
			}
			return objects[left].Name() < objects[right].Name()
		})
		for ordinal, object := range objects {
			index.localOrdinals[object] = ordinal
		}
	}
	return nil
}

func (index *objectIndex) declarationID(
	object types.Object,
) (identity.SemanticDeclarationID, error) {
	if object == nil {
		return identity.SemanticDeclarationID{}, fmt.Errorf(
			"semantic declaration requires checker object",
		)
	}
	object = index.canonicalDeclarationObject(object)
	if existing := index.declarationIDs[object]; !existing.IsZero() {
		return existing, nil
	}
	var (
		id  identity.SemanticDeclarationID
		err error
	)
	if predeclared := index.predeclared[object]; predeclared.Valid() {
		class, classErr := predeclaredSemanticClass(
			predeclared.Class(),
		)
		if classErr != nil {
			return identity.SemanticDeclarationID{}, classErr
		}
		id, err = identity.NewPredeclaredDeclarationID(
			uint16(predeclared), class,
		)
	} else {
		class, classErr := semanticObjectClass(object)
		if classErr != nil {
			return identity.SemanticDeclarationID{}, classErr
		}
		if builtin, special := object.(*types.Builtin); special {
			if err := validateBuiltinObject(builtin); err != nil {
				return identity.SemanticDeclarationID{}, err
			}
		}
		switch {
		case class == identity.SemanticObjectField ||
			class == identity.SemanticObjectMethod:
			id, err = index.memberDeclarationID(object, class)
		case packageObject(object):
			pkg, packageErr := index.packageID(object.Pkg())
			if packageErr != nil {
				return identity.SemanticDeclarationID{}, packageErr
			}
			id, err = identity.NewPackageDeclarationID(
				pkg, class, object.Name(),
			)
		default:
			id, err = index.localDeclarationID(object, class)
		}
	}
	if err != nil {
		return identity.SemanticDeclarationID{}, err
	}
	if existing := index.declarationByID[id]; existing != nil &&
		existing != object {
		return identity.SemanticDeclarationID{}, fmt.Errorf(
			"semantic declaration identity %s has checker objects %s and %s",
			id, existing.Name(), object.Name(),
		)
	}
	index.declarationIDs[object] = id
	index.declarationByID[id] = object
	if err := index.registerDeclarationTypeParameters(
		object, id,
	); err != nil {
		return identity.SemanticDeclarationID{}, err
	}
	return id, nil
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
	owner := index.memberOwners[field]
	for {
		pointer, pointerOwner := owner.(*types.Pointer)
		if !pointerOwner {
			break
		}
		owner = pointer.Elem()
	}
	named, namedOwner := owner.(*types.Named)
	if !namedOwner || named.Origin() == named {
		return object
	}
	ordinal, err := index.fieldOrdinal(field)
	if err != nil {
		return object
	}
	origin := named.Origin()
	structure, ok := origin.Underlying().(*types.Struct)
	if !ok || ordinal >= structure.NumFields() {
		return object
	}
	originField := structure.Field(ordinal)
	index.memberOwners[originField] = origin
	return originField
}

func (index *objectIndex) memberDeclarationID(
	object types.Object,
	class identity.SemanticObjectClass,
) (identity.SemanticDeclarationID, error) {
	owner, err := index.typeBuilder.memberOwnerID(object)
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
	ordinal := 0
	if class == identity.SemanticObjectField {
		ordinal, err = index.fieldOrdinal(object)
		if err != nil {
			return identity.SemanticDeclarationID{}, err
		}
	}
	return identity.NewMemberDeclarationID(
		owner, memberPackage, class, object.Name(), ordinal,
	)
}

func (index *objectIndex) fieldOrdinal(
	object types.Object,
) (int, error) {
	owner := index.memberOwners[object]
	for {
		pointer, ok := owner.(*types.Pointer)
		if !ok {
			break
		}
		owner = pointer.Elem()
	}
	if named, ok := owner.(*types.Named); ok {
		owner = named.Underlying()
	}
	structure, ok := owner.(*types.Struct)
	if !ok {
		return 0, fmt.Errorf(
			"field %s has non-struct owner %T", object.Name(), owner,
		)
	}
	for ordinal := 0; ordinal < structure.NumFields(); ordinal++ {
		if structure.Field(ordinal) == object {
			return ordinal, nil
		}
	}
	return 0, fmt.Errorf(
		"field %s is absent from its declaring struct", object.Name(),
	)
}

func (index *objectIndex) localDeclarationID(
	object types.Object,
	class identity.SemanticObjectClass,
) (identity.SemanticDeclarationID, error) {
	source := index.sourceByObject[object]
	if source.IsZero() {
		return identity.SemanticDeclarationID{}, fmt.Errorf(
			"non-package declaration %s (%T, package=%v, parent=%v, pos=%d) has no source occurrence",
			object.Name(), object, object.Pkg(), object.Parent(),
			object.Pos(),
		)
	}
	scope, err := index.bindingScope(object, source)
	if err != nil {
		return identity.SemanticDeclarationID{}, err
	}
	ordinal, present := index.localOrdinals[object]
	if !present {
		return identity.SemanticDeclarationID{}, fmt.Errorf(
			"local declaration %s has no canonical ordinal",
			object.Name(),
		)
	}
	return identity.NewOccurrenceDeclarationID(
		scope, source, class, object.Name(), ordinal,
	)
}

func semanticObjectClass(
	object types.Object,
) (identity.SemanticObjectClass, error) {
	switch typed := object.(type) {
	case *types.Const:
		return identity.SemanticObjectConstant, nil
	case *types.TypeName:
		if typed.IsAlias() {
			return identity.SemanticObjectAlias, nil
		}
		return identity.SemanticObjectType, nil
	case *types.Var:
		if typed.IsField() {
			return identity.SemanticObjectField, nil
		}
		return identity.SemanticObjectVariable, nil
	case *types.Func:
		signature, _ := typed.Type().(*types.Signature)
		if signature != nil && signature.Recv() != nil {
			return identity.SemanticObjectMethod, nil
		}
		return identity.SemanticObjectFunction, nil
	case *types.Builtin:
		return identity.SemanticObjectBuiltin, nil
	case *types.Nil:
		return identity.SemanticObjectNil, nil
	default:
		return identity.SemanticObjectInvalid, fmt.Errorf(
			"checker object %T is not a semantic declaration", object,
		)
	}
}

func localDeclarationClass(
	object types.Object,
) (identity.SemanticObjectClass, bool) {
	if packageObject(object) {
		return identity.SemanticObjectInvalid, false
	}
	switch typed := object.(type) {
	case *types.Const:
		return identity.SemanticObjectConstant, true
	case *types.TypeName:
		if _, parameter := typed.Type().(*types.TypeParam); parameter {
			return identity.SemanticObjectInvalid, false
		}
		if typed.IsAlias() {
			return identity.SemanticObjectAlias, true
		}
		return identity.SemanticObjectType, true
	default:
		return identity.SemanticObjectInvalid, false
	}
}

func packageObject(object types.Object) bool {
	return object.Pkg() != nil &&
		object.Parent() == object.Pkg().Scope() &&
		object.Pkg().Scope().Lookup(object.Name()) == object
}

func predeclaredSemanticClass(
	class catalog.PredeclaredClass,
) (identity.SemanticObjectClass, error) {
	switch class {
	case catalog.PredeclaredClassType:
		return identity.SemanticObjectType, nil
	case catalog.PredeclaredClassConstant:
		return identity.SemanticObjectConstant, nil
	case catalog.PredeclaredClassNil:
		return identity.SemanticObjectNil, nil
	case catalog.PredeclaredClassFunction:
		return identity.SemanticObjectBuiltin, nil
	default:
		return identity.SemanticObjectInvalid, fmt.Errorf(
			"unsupported predeclared class %s", class,
		)
	}
}

func (index *objectIndex) declarationRecords() ([]semantic.Declaration, error) {
	if err := index.discoverOwnedDeclarations(); err != nil {
		return nil, err
	}
	ids := make(
		[]identity.SemanticDeclarationID,
		0,
		len(index.declarationByID),
	)
	for id, object := range index.declarationByID {
		if index.ownsDeclaration(object, id) {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(left, right int) bool {
		return ids[left].String() < ids[right].String()
	})
	out := make([]semantic.Declaration, 0, len(ids))
	for _, id := range ids {
		record, err := index.declarationRecord(
			index.declarationByID[id], id,
		)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, nil
}

func (index *objectIndex) discoverOwnedDeclarations() error {
	if index.input.provenance == semantic.ProvenanceLanguagePseudo &&
		index.input.id.ImportPath() == "builtin" {
		for _, kind := range catalog.AllPredeclared() {
			object := types.Universe.Lookup(kind.Name())
			if object == nil {
				return fmt.Errorf(
					"predeclared catalog object %s is absent",
					kind,
				)
			}
			if _, err := index.declarationID(object); err != nil {
				return err
			}
		}
		for object := range index.memberOwners {
			if object.Pkg() == nil {
				if _, err := index.declarationID(object); err != nil {
					return err
				}
			}
		}
		return nil
	}
	scope := index.input.loaded.Types().Scope()
	for _, name := range scope.Names() {
		object := scope.Lookup(name)
		if object == nil || index.isBinding(object) {
			continue
		}
		if _, err := index.declarationID(object); err != nil {
			return err
		}
	}
	for object := range index.sourceByObject {
		if _, local := localDeclarationClass(object); !local {
			continue
		}
		if _, err := index.declarationID(object); err != nil {
			return err
		}
	}
	for object := range index.memberOwners {
		if object.Pkg() == nil {
			continue
		}
		pkg, err := index.packageID(object.Pkg())
		if err != nil {
			return err
		}
		if pkg != index.input.id {
			continue
		}
		if _, err := index.declarationID(object); err != nil {
			return err
		}
	}
	return nil
}

func (index *objectIndex) ownsDeclaration(
	object types.Object,
	id identity.SemanticDeclarationID,
) bool {
	if id.Form() == identity.SemanticDeclarationPredeclared {
		return index.input.provenance ==
			semantic.ProvenanceLanguagePseudo &&
			index.input.id.ImportPath() == "builtin"
	}
	if object.Pkg() == nil &&
		index.input.provenance == semantic.ProvenanceLanguagePseudo &&
		index.input.id.ImportPath() == "builtin" {
		return true
	}
	if object.Pkg() == nil {
		return false
	}
	pkg, err := index.packageID(object.Pkg())
	return err == nil && pkg == index.input.id
}

func (index *objectIndex) declarationRecord(
	object types.Object,
	id identity.SemanticDeclarationID,
) (semantic.Declaration, error) {
	class, err := semanticObjectClass(object)
	if predeclared := index.predeclared[object]; predeclared.Valid() {
		class, err = predeclaredSemanticClass(predeclared.Class())
	}
	if err != nil {
		return semantic.Declaration{}, err
	}
	pkg := index.input.id
	if id.Form() != identity.SemanticDeclarationPredeclared {
		if object.Pkg() != nil {
			pkg, err = index.packageID(object.Pkg())
			if err != nil {
				return semantic.Declaration{}, err
			}
		}
	}
	typeID := identity.SemanticTypeID{}
	if _, builtin := object.(*types.Builtin); !builtin {
		typeID, err = index.typeBuilder.build(object.Type())
		if err != nil {
			return semantic.Declaration{}, fmt.Errorf(
				"declaration %s (%s) type %s: %w",
				id, object.Name(), object.Type(), err,
			)
		}
	}
	constantValue, err := semanticConstant(object)
	if err != nil {
		return semantic.Declaration{}, err
	}
	source := index.sourceByObject[object]
	if source.IsZero() &&
		id.Form() != identity.SemanticDeclarationPredeclared &&
		!index.definitionByObject[object].SyntheticRole().Valid() &&
		object.Pos().IsValid() {
		source, err = index.input.index.IdentifierOccurrence(
			object.Pos(), object.Name(),
		)
		if err != nil {
			return semantic.Declaration{}, fmt.Errorf(
				"declaration %s source identity: %w", id, err,
			)
		}
	}
	return semantic.NewDeclaration(
		id,
		pkg,
		class,
		object.Name(),
		typeID,
		source,
		object.Exported(),
		constantValue,
		index.input.authority,
	)
}

func validateBuiltinObject(object *types.Builtin) error {
	if object == nil {
		return fmt.Errorf("builtin declaration requires checker object")
	}
	if object.Pkg() == nil {
		if types.Universe.Lookup(object.Name()) != object {
			return fmt.Errorf(
				"non-canonical predeclared builtin %s", object.Name(),
			)
		}
		return nil
	}
	member := catalog.UnsafeMemberByName(object.Name())
	if object.Pkg().Path() != "unsafe" ||
		!member.Valid() ||
		member.Class() != catalog.UnsafeMemberClassBuiltin {
		return fmt.Errorf(
			"checker builtin %s.%s is absent from the closed language catalog",
			object.Pkg().Path(),
			object.Name(),
		)
	}
	return nil
}

func semanticConstant(
	object types.Object,
) (semantic.Constant, error) {
	value, isConstant := object.(*types.Const)
	if !isConstant {
		return semantic.Constant{}, nil
	}
	var kind semantic.ConstantKind
	switch value.Val().Kind() {
	case constant.Bool:
		kind = semantic.ConstantBool
	case constant.String:
		kind = semantic.ConstantString
	case constant.Int:
		kind = semantic.ConstantInteger
	case constant.Float:
		kind = semantic.ConstantFloat
	case constant.Complex:
		kind = semantic.ConstantComplex
	default:
		return semantic.Constant{}, fmt.Errorf(
			"constant %s has unsupported exact kind %s",
			object.Name(), value.Val().Kind(),
		)
	}
	return semantic.NewConstant(kind, value.Val().ExactString())
}
