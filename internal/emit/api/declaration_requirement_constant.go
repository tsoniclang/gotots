package api

import "go/types"

// ConstantProjectionType resolves a validated concrete constant-capable basic
// representation. Raw types.BasicKind values are representable outside this
// package, so every public projection boundary uses this owner before indexing
// types.Typ. Untyped kinds and unsafe.Pointer are not projection types.
func ConstantProjectionType(
	projection types.BasicKind,
) (*types.Basic, bool) {
	index := int(projection)
	if projection == types.Invalid ||
		index < 0 ||
		index >= len(types.Typ) {
		return nil, false
	}
	selected := types.Typ[index]
	if selected == nil ||
		selected.Info()&types.IsUntyped != 0 ||
		selected.Info()&(types.IsBoolean|
			types.IsInteger|
			types.IsFloat|
			types.IsComplex|
			types.IsString) == 0 {
		return nil, false
	}
	return selected, true
}

// ConstantProjectionName is the exported name of an untyped constant's
// projection at one target basic representation. The `$` separator cannot occur
// in a Go source identifier, so a projection name never collides with a user
// declaration, and distinct (constant, representation) pairs never collide with
// each other. Both the declaration owner and every use site derive the name
// through this one function.
func ConstantProjectionName(
	base string,
	projection types.BasicKind,
) (string, error) {
	selected, ok := ConstantProjectionType(projection)
	if base == "" || !ok {
		return "", &NameError{
			Name:   base,
			Reason: "constant projection identity is invalid",
		}
	}
	return base + "$" + selected.Name(), nil
}

// NewConstantProjectionRequirement requires one untyped constant to be
// projected once at the given target basic representation.
func NewConstantProjectionRequirement(
	constant *types.Const,
	projection types.BasicKind,
) (DeclarationRequirement, error) {
	switch {
	case constant == nil:
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "constant projection constant is nil",
		}
	case !validConstantProjection(projection):
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "constant projection target representation is invalid",
		}
	}
	return DeclarationRequirement{
		owner:      MustSourceArtifactOwner(constant),
		kind:       DeclarationRequirementConstantProjection,
		projection: projection,
	}, nil
}

// NewLocalConstantProjectionRequirement requires one untyped constant declared
// inside a function to be projected once, at the given target basic
// representation, at its original lexical declaration. The enclosing function
// owns reconstruction because a function-local constant has no package
// declaration artifact; the dedup key is the
// (function, constant, representation) triple.
func NewLocalConstantProjectionRequirement(
	owner *types.Func,
	constant *types.Const,
	projection types.BasicKind,
) (DeclarationRequirement, error) {
	switch {
	case owner == nil:
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "local constant projection owner is nil",
		}
	case constant == nil:
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "local constant projection constant is nil",
		}
	case !validConstantProjection(projection):
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "local constant projection target representation is invalid",
		}
	}
	return DeclarationRequirement{
		owner:      MustSourceArtifactOwner(owner),
		kind:       DeclarationRequirementLocalConstantProjection,
		constant:   constant,
		projection: projection,
	}, nil
}

func validConstantProjection(projection types.BasicKind) bool {
	_, ok := ConstantProjectionType(projection)
	return ok
}
