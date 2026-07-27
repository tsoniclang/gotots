package api

import "go/types"

type DeclarationRequirementKind uint8

const (
	DeclarationRequirementInvalid DeclarationRequirementKind = iota
	DeclarationRequirementNamedStructOperation
	DeclarationRequirementAddressableStorage
	DeclarationRequirementConstantProjection
	DeclarationRequirementLocalConstantProjection
)

func (k DeclarationRequirementKind) Valid() bool {
	return k == DeclarationRequirementNamedStructOperation ||
		k == DeclarationRequirementAddressableStorage ||
		k == DeclarationRequirementConstantProjection ||
		k == DeclarationRequirementLocalConstantProjection
}

type DeclarationRequirement struct {
	owner     types.Object
	kind      DeclarationRequirementKind
	operation NamedStructOperation
	variable  *types.Var
	// constant is the untyped constant a local projection materializes. A
	// package projection owns the constant directly (owner is the constant), so
	// this stays nil there; a local projection is owned by the enclosing
	// function, so the constant identity travels here.
	constant *types.Const
	// projection is the exact target basic representation of an untyped
	// constant projection. A basic kind is a canonical, comparable dedup key —
	// unlike a types.Type interface value, whose pointer identity is not a
	// stable projection key.
	projection types.BasicKind
}

// ConstantProjectionName is the exported name of an untyped constant's
// projection at one target basic representation. The `$` separator cannot occur
// in a Go source identifier, so a projection name never collides with a user
// declaration, and distinct (constant, representation) pairs never collide with
// each other. Both the declaration owner and every use site derive the name
// through this one function.
func ConstantProjectionName(base string, projection types.BasicKind) string {
	return base + "$" + types.Typ[projection].Name()
}

// NewConstantProjectionRequirement requires one untyped constant to be projected
// once at the given target basic representation.
func NewConstantProjectionRequirement(
	constant *types.Const,
	projection types.BasicKind,
) (DeclarationRequirement, error) {
	switch {
	case constant == nil:
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "constant projection constant is nil",
		}
	case projection == types.Invalid:
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "constant projection target representation is invalid",
		}
	}
	return DeclarationRequirement{
		owner:      constant,
		kind:       DeclarationRequirementConstantProjection,
		projection: projection,
	}, nil
}

// NewLocalConstantProjectionRequirement requires one untyped constant declared
// inside a function to be projected once, at the given target basic
// representation, in that function's prologue. The enclosing function owns the
// requirement because a function-local constant has no package declaration to
// reconstruct; the dedup key is the (function, constant, representation) triple.
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
	case projection == types.Invalid:
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "local constant projection target representation is invalid",
		}
	}
	return DeclarationRequirement{
		owner:      owner,
		kind:       DeclarationRequirementLocalConstantProjection,
		constant:   constant,
		projection: projection,
	}, nil
}

func NewNamedStructOperationRequirement(
	typeName *types.TypeName,
	operation NamedStructOperation,
) (DeclarationRequirement, error) {
	switch {
	case typeName == nil:
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "named-struct operation type is nil",
		}
	case !operation.Valid():
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "named-struct operation is invalid",
		}
	}
	return DeclarationRequirement{
		owner:     typeName,
		kind:      DeclarationRequirementNamedStructOperation,
		operation: operation,
	}, nil
}

func NewAddressableStorageRequirement(
	owner *types.Func,
	variable *types.Var,
) (DeclarationRequirement, error) {
	switch {
	case owner == nil:
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "addressable-storage owner is nil",
		}
	case variable == nil:
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "addressable-storage variable is nil",
		}
	case variable.IsField():
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "addressable-storage variable is a field",
		}
	}
	return DeclarationRequirement{
		owner:    owner,
		kind:     DeclarationRequirementAddressableStorage,
		variable: variable,
	}, nil
}

func (r DeclarationRequirement) Valid() bool {
	if !r.kind.Valid() {
		return false
	}
	switch r.kind {
	case DeclarationRequirementNamedStructOperation:
		if !r.operation.Valid() || r.variable != nil {
			return false
		}
		_, ok := r.owner.(*types.TypeName)
		return ok
	case DeclarationRequirementAddressableStorage:
		if r.operation != NamedStructOperationInvalid ||
			r.variable == nil ||
			r.variable.IsField() {
			return false
		}
		_, ok := r.owner.(*types.Func)
		return ok
	case DeclarationRequirementConstantProjection:
		if r.operation != NamedStructOperationInvalid ||
			r.variable != nil ||
			r.constant != nil ||
			r.projection == types.Invalid {
			return false
		}
		_, ok := r.owner.(*types.Const)
		return ok
	case DeclarationRequirementLocalConstantProjection:
		if r.operation != NamedStructOperationInvalid ||
			r.variable != nil ||
			r.constant == nil ||
			r.projection == types.Invalid {
			return false
		}
		_, ok := r.owner.(*types.Func)
		return ok
	default:
		return false
	}
}

func (r DeclarationRequirement) Owner() types.Object {
	return r.owner
}

func (r DeclarationRequirement) Kind() DeclarationRequirementKind {
	return r.kind
}

func (r DeclarationRequirement) NamedStructOperation() (
	*types.TypeName,
	NamedStructOperation,
	bool,
) {
	if !r.Valid() {
		return nil, NamedStructOperationInvalid, false
	}
	typeName, ok := r.owner.(*types.TypeName)
	return typeName, r.operation, ok
}

func (r DeclarationRequirement) AddressableStorage() (
	*types.Func,
	*types.Var,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementAddressableStorage {
		return nil, nil, false
	}
	owner, ok := r.owner.(*types.Func)
	return owner, r.variable, ok
}

func (r DeclarationRequirement) ConstantProjection() (
	*types.Const,
	types.BasicKind,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementConstantProjection {
		return nil, types.Invalid, false
	}
	constant, ok := r.owner.(*types.Const)
	return constant, r.projection, ok
}

func (r DeclarationRequirement) LocalConstantProjection() (
	*types.Func,
	*types.Const,
	types.BasicKind,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementLocalConstantProjection {
		return nil, nil, types.Invalid, false
	}
	owner, ok := r.owner.(*types.Func)
	return owner, r.constant, r.projection, ok
}
