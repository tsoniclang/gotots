package stagecheck

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/language/structure"
)

type checkerTypeParameterOwner struct {
	object     types.Object
	definition identity.DefinitionID
	role       semantic.TypeParameterRole
	ordinal    int
}

func (verifier *checkerTypeVerifier) indexTypeParameterOwners(
	index *structure.TransientIndex,
) {
	verifier.indexCheckerPackageTypeParameters(
		verifier.expected.loaded.Types(),
	)
	view := verifier.expected.loaded.CheckerView()
	if view == nil || index == nil {
		return
	}
	for definition := range verifier.expected.definitions {
		node, present := index.CheckedDefinitionNode(definition)
		if !present {
			node, present = index.DefinitionNode(definition)
		}
		declaration, callable := node.(*ast.FuncDecl)
		if !present || !callable ||
			(declaration.Name.Name != "_" &&
				declaration.Name.Name != "init") {
			continue
		}
		object, objectPresent := view.DefOf(declaration.Name)
		function, functionObject := object.(*types.Func)
		if !objectPresent || !functionObject {
			continue
		}
		signature, _ := function.Type().(*types.Signature)
		if signature == nil {
			continue
		}
		verifier.registerCheckerTypeParameters(
			signature.TypeParams(),
			checkerTypeParameterOwner{
				definition: definition,
				role:       semantic.TypeParameterCallable,
			},
		)
		verifier.registerCheckerTypeParameters(
			signature.RecvTypeParams(),
			checkerTypeParameterOwner{
				definition: definition,
				role:       semantic.TypeParameterReceiver,
			},
		)
	}
}

func (
	verifier *checkerTypeVerifier,
) indexCheckerDeclarationTypeParameters(object types.Object) {
	switch typed := object.(type) {
	case *types.TypeName:
		switch declared := typed.Type().(type) {
		case *types.Named:
			verifier.indexEncounteredNamedTypeParameters(declared)
		case *types.Alias:
			verifier.indexEncounteredAliasTypeParameters(declared)
		}
	case *types.Func:
		verifier.indexCheckerFunctionTypeParameters(typed.Origin())
	}
}

func (verifier *checkerTypeVerifier) indexCheckerPackageTypeParameters(
	pkg *types.Package,
) {
	if pkg == nil {
		return
	}
	scope := pkg.Scope()
	for _, name := range scope.Names() {
		object := scope.Lookup(name)
		switch typed := object.(type) {
		case *types.TypeName:
			switch declared := typed.Type().(type) {
			case *types.Named:
				origin := declared.Origin()
				verifier.registerCheckerTypeParameters(
					origin.TypeParams(),
					checkerTypeParameterOwner{
						object: typed,
						role:   semantic.TypeParameterDeclared,
					},
				)
				for ordinal := 0; ordinal < origin.NumMethods(); ordinal++ {
					verifier.indexCheckerFunctionTypeParameters(
						origin.Method(ordinal).Origin(),
					)
				}
			case *types.Alias:
				verifier.registerCheckerTypeParameters(
					declared.TypeParams(),
					checkerTypeParameterOwner{
						object: typed,
						role:   semantic.TypeParameterDeclared,
					},
				)
			}
		case *types.Func:
			verifier.indexCheckerFunctionTypeParameters(
				typed.Origin(),
			)
		}
	}
}

func (verifier *checkerTypeVerifier) indexEncounteredNamedTypeParameters(
	named *types.Named,
) {
	if named == nil || named.Obj() == nil {
		return
	}
	origin := named.Origin()
	verifier.registerCheckerTypeParameters(
		origin.TypeParams(),
		checkerTypeParameterOwner{
			object: origin.Obj(),
			role:   semantic.TypeParameterDeclared,
		},
	)
	for ordinal := 0; ordinal < origin.NumMethods(); ordinal++ {
		verifier.indexCheckerFunctionTypeParameters(
			origin.Method(ordinal).Origin(),
		)
	}
}

func (verifier *checkerTypeVerifier) indexEncounteredAliasTypeParameters(
	alias *types.Alias,
) {
	if alias == nil || alias.Obj() == nil {
		return
	}
	verifier.registerCheckerTypeParameters(
		alias.TypeParams(),
		checkerTypeParameterOwner{
			object: alias.Obj(),
			role:   semantic.TypeParameterDeclared,
		},
	)
}

func (verifier *checkerTypeVerifier) indexCheckerFunctionTypeParameters(
	function *types.Func,
) {
	if function == nil {
		return
	}
	signature, _ := function.Type().(*types.Signature)
	if signature == nil {
		return
	}
	verifier.registerCheckerTypeParameters(
		signature.TypeParams(),
		checkerTypeParameterOwner{
			object: function,
			role:   semantic.TypeParameterCallable,
		},
	)
	verifier.registerCheckerTypeParameters(
		signature.RecvTypeParams(),
		checkerTypeParameterOwner{
			object: function,
			role:   semantic.TypeParameterReceiver,
		},
	)
}

func (verifier *checkerTypeVerifier) registerCheckerTypeParameters(
	parameters *types.TypeParamList,
	owner checkerTypeParameterOwner,
) {
	if parameters == nil {
		return
	}
	for ordinal := 0; ordinal < parameters.Len(); ordinal++ {
		parameter := parameters.At(ordinal)
		candidate := owner
		candidate.ordinal = ordinal
		if prior, present := verifier.parameterOwners[parameter]; present && !sameCheckerTypeParameterOwner(
			prior, candidate,
		) {
			verifier.parameterConflict = fmt.Sprintf(
				"checker type parameter %s has two owners",
				parameter.Obj().Name(),
			)
		}
		verifier.parameterOwners[parameter] = candidate
	}
}

func sameCheckerTypeParameterOwner(
	left checkerTypeParameterOwner,
	right checkerTypeParameterOwner,
) bool {
	return sameCheckerOwnerObject(left.object, right.object) &&
		left.definition == right.definition &&
		left.role == right.role &&
		left.ordinal == right.ordinal
}

func sameCheckerOwnerObject(
	left types.Object,
	right types.Object,
) bool {
	if left == right {
		return true
	}
	if left == nil || right == nil ||
		left.Name() != right.Name() ||
		left.Pkg() == nil ||
		right.Pkg() == nil ||
		left.Pkg().Path() != right.Pkg().Path() {
		return false
	}
	return types.Identical(left.Type(), right.Type())
}

func (verifier *checkerTypeVerifier) verifyTypeParameterOwner(
	record semantic.TypeParameterOwner,
	parameter *types.TypeParam,
) error {
	if verifier.parameterConflict != "" {
		return fmt.Errorf("%s", verifier.parameterConflict)
	}
	expected, present := verifier.parameterOwners[parameter]
	if !present {
		return fmt.Errorf(
			"type parameter %s has no independently derived owner",
			parameter.Obj().Name(),
		)
	}
	if record.Role() != expected.role ||
		record.Ordinal() != expected.ordinal ||
		record.Ordinal() != parameter.Index() {
		return fmt.Errorf(
			"type parameter %s owner role/ordinal differs",
			parameter.Obj().Name(),
		)
	}
	if !expected.definition.IsZero() {
		if record.Definition() != expected.definition ||
			!record.Declaration().IsZero() {
			return fmt.Errorf(
				"type parameter %s definition owner differs",
				parameter.Obj().Name(),
			)
		}
		return nil
	}
	if expected.object == nil ||
		record.Declaration().IsZero() ||
		!record.Definition().IsZero() {
		return fmt.Errorf(
			"type parameter %s declaration owner is absent",
			parameter.Obj().Name(),
		)
	}
	return verifier.verifyTypeParameterDeclaration(
		record.Declaration(), expected.object,
	)
}

func (verifier *checkerTypeVerifier) verifyTypeParameterDeclaration(
	id identity.SemanticDeclarationID,
	object types.Object,
) error {
	if function, ok := object.(*types.Func); ok {
		object = function.Origin()
	}
	if object.Pkg() == nil {
		return fmt.Errorf(
			"type-parameter owner %s has no package", object.Name(),
		)
	}
	pkg := verifier.packageByPath[object.Pkg().Path()]
	class := independentObjectClass(object)
	if pkg.IsZero() || !class.Valid() {
		return fmt.Errorf(
			"type-parameter owner %s has no package identity",
			object.Name(),
		)
	}
	if id.Form() == identity.SemanticDeclarationPackageObject {
		expected, err := identity.NewPackageDeclarationID(
			pkg, class, object.Name(),
		)
		if err != nil || id != expected {
			return fmt.Errorf(
				"type-parameter package owner differs for %s",
				object.Name(),
			)
		}
		return nil
	}
	function, method := object.(*types.Func)
	if !method ||
		id.Form() != identity.SemanticDeclarationMember {
		return fmt.Errorf(
			"type-parameter declaration %s has invalid owner form",
			id,
		)
	}
	signature, _ := function.Type().(*types.Signature)
	if signature == nil || signature.Recv() == nil {
		return fmt.Errorf(
			"method owner %s has no receiver", function.Name(),
		)
	}
	receiver := stripCheckerPointer(signature.Recv().Type())
	if named, ok := receiver.(*types.Named); ok {
		receiver = named.Origin()
	}
	if err := verifier.verify(id.OwnerType(), receiver); err != nil {
		return fmt.Errorf(
			"type-parameter method owner type: %w", err,
		)
	}
	var memberPackage identity.PackageID
	if !object.Exported() {
		memberPackage = pkg
	}
	expected, err := identity.NewMemberDeclarationID(
		id.OwnerType(),
		memberPackage,
		identity.SemanticObjectMethod,
		object.Name(),
		0,
	)
	if err != nil || expected != id {
		return fmt.Errorf(
			"type-parameter method owner differs for %s",
			object.Name(),
		)
	}
	return nil
}
