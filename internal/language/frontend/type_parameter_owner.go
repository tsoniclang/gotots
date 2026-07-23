package frontend

import (
	"fmt"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

type typeParameterLocation struct {
	packagePath string
	position    token.Pos
	index       int
}

func (index *objectIndex) indexPackageTypeParameterOwners(
	pkg *types.Package,
) error {
	if pkg == nil {
		return fmt.Errorf(
			"type-parameter owner indexing requires checker package",
		)
	}
	scope := pkg.Scope()
	for _, name := range scope.Names() {
		object := scope.Lookup(name)
		if object == nil {
			continue
		}
		if _, err := index.declarationID(object); err != nil {
			return err
		}
		typeName, declaredType := object.(*types.TypeName)
		if !declaredType {
			continue
		}
		named, namedType := typeName.Type().(*types.Named)
		if !namedType {
			continue
		}
		for methodIndex := 0; methodIndex < named.NumMethods(); methodIndex++ {
			if _, err := index.declarationID(
				named.Method(methodIndex),
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func (index *objectIndex) registerDeclarationTypeParameters(
	object types.Object,
	declaration identity.SemanticDeclarationID,
) error {
	switch typed := object.(type) {
	case *types.TypeName:
		switch declared := typed.Type().(type) {
		case *types.Named:
			return index.registerTypeParameterList(
				declared.Origin().TypeParams(),
				declaration,
				identity.DefinitionID{},
				semantic.TypeParameterDeclared,
			)
		case *types.Alias:
			return index.registerTypeParameterList(
				declared.TypeParams(),
				declaration,
				identity.DefinitionID{},
				semantic.TypeParameterDeclared,
			)
		}
	case *types.Func:
		signature, ok := typed.Type().(*types.Signature)
		if !ok {
			return fmt.Errorf(
				"generic declaration %s has non-signature type %T",
				declaration, typed.Type(),
			)
		}
		if err := index.registerTypeParameterList(
			signature.TypeParams(),
			declaration,
			identity.DefinitionID{},
			semantic.TypeParameterCallable,
		); err != nil {
			return err
		}
		return index.registerTypeParameterList(
			signature.RecvTypeParams(),
			declaration,
			identity.DefinitionID{},
			semantic.TypeParameterReceiver,
		)
	}
	return nil
}

func (index *objectIndex) registerDefinitionTypeParameters(
	signature *types.Signature,
	definition identity.DefinitionID,
) error {
	if signature == nil || definition.IsZero() {
		return fmt.Errorf(
			"definition type-parameter registration requires signature and definition",
		)
	}
	if err := index.registerTypeParameterList(
		signature.TypeParams(),
		identity.SemanticDeclarationID{},
		definition,
		semantic.TypeParameterCallable,
	); err != nil {
		return err
	}
	return index.registerTypeParameterList(
		signature.RecvTypeParams(),
		identity.SemanticDeclarationID{},
		definition,
		semantic.TypeParameterReceiver,
	)
}

func (index *objectIndex) registerTypeParameterList(
	parameters *types.TypeParamList,
	declaration identity.SemanticDeclarationID,
	definition identity.DefinitionID,
	role semantic.TypeParameterRole,
) error {
	if parameters == nil {
		return nil
	}
	for ordinal := 0; ordinal < parameters.Len(); ordinal++ {
		parameter := parameters.At(ordinal)
		owner, err := semantic.NewTypeParameterOwner(
			declaration, definition, role, ordinal,
		)
		if err != nil {
			return err
		}
		if err := index.registerTypeParameterOwner(
			parameter, owner,
		); err != nil {
			return err
		}
	}
	return nil
}

func (index *objectIndex) registerTypeParameterOwner(
	parameter *types.TypeParam,
	owner semantic.TypeParameterOwner,
) error {
	if parameter == nil || parameter.Obj() == nil || owner.IsZero() {
		return fmt.Errorf(
			"type-parameter registration requires checker parameter and canonical owner",
		)
	}
	if existing := index.typeParameterOwners[parameter]; !existing.IsZero() &&
		existing != owner {
		return fmt.Errorf(
			"type parameter %s has owners %s and %s",
			parameter.Obj().Name(), existing, owner,
		)
	}
	index.typeParameterOwners[parameter] = owner
	location, present := typeParameterObjectLocation(parameter.Obj())
	if !present {
		return nil
	}
	if existing := index.typeParameterByLocation[location]; !existing.IsZero() &&
		existing != owner {
		return fmt.Errorf(
			"type-parameter checker location has owners %s and %s",
			existing, owner,
		)
	}
	index.typeParameterByLocation[location] = owner
	return nil
}

func (index *objectIndex) typeParameterOwner(
	parameter *types.TypeParam,
) (semantic.TypeParameterOwner, bool) {
	if parameter == nil || parameter.Obj() == nil {
		return semantic.TypeParameterOwner{}, false
	}
	if owner := index.typeParameterOwners[parameter]; !owner.IsZero() {
		return owner, true
	}
	location, present := typeParameterObjectLocation(parameter.Obj())
	if !present {
		return semantic.TypeParameterOwner{}, false
	}
	owner := index.typeParameterByLocation[location]
	return owner, !owner.IsZero()
}

func typeParameterObjectLocation(
	object *types.TypeName,
) (typeParameterLocation, bool) {
	if object == nil ||
		object.Pkg() == nil ||
		!object.Pos().IsValid() {
		return typeParameterLocation{}, false
	}
	parameter, typeParameter := object.Type().(*types.TypeParam)
	if !typeParameter || parameter.Index() < 0 {
		return typeParameterLocation{}, false
	}
	return typeParameterLocation{
		packagePath: object.Pkg().Path(),
		position:    object.Pos(),
		index:       parameter.Index(),
	}, true
}
