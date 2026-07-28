package api

import (
	"fmt"
	"go/types"
)

type NamedStructOperation uint8

const (
	NamedStructOperationInvalid NamedStructOperation = iota
	NamedStructOperationZero
	NamedStructOperationCopy
	NamedStructOperationEqual
	NamedStructOperationHash
)

func (o NamedStructOperation) Valid() bool {
	return o == NamedStructOperationZero ||
		o == NamedStructOperationCopy ||
		o == NamedStructOperationEqual ||
		o == NamedStructOperationHash
}

func (o NamedStructOperation) String() string {
	switch o {
	case NamedStructOperationZero:
		return "zero"
	case NamedStructOperationCopy:
		return "copy"
	case NamedStructOperationEqual:
		return "equal"
	case NamedStructOperationHash:
		return "hash"
	default:
		return fmt.Sprintf("named-struct-operation(%d)", o)
	}
}

func NamedStructOperationMemberName(
	operation NamedStructOperation,
) (string, error) {
	if !operation.Valid() {
		return "", &NameError{Reason: "named-struct operation is invalid"}
	}
	return "$" + operation.String(), nil
}

type DefinedArrayOperation uint8

const (
	DefinedArrayOperationInvalid DefinedArrayOperation = iota
	DefinedArrayOperationAddressIndex
)

func (o DefinedArrayOperation) Valid() bool {
	return o == DefinedArrayOperationAddressIndex
}

type AnonymousStructDemand uint8

const (
	AnonymousStructDemandInvalid AnonymousStructDemand = iota
	AnonymousStructDemandDefinition
	AnonymousStructDemandZero
	AnonymousStructDemandCopy
	AnonymousStructDemandEqual
	AnonymousStructDemandHash
)

func (d AnonymousStructDemand) Valid() bool {
	return d >= AnonymousStructDemandDefinition &&
		d <= AnonymousStructDemandHash
}

type MapSpecializationDemand uint8

const (
	MapSpecializationDemandInvalid MapSpecializationDemand = iota
	MapSpecializationDemandDefinition
	MapSpecializationDemandStatic
	MapSpecializationDemandClear
)

func (d MapSpecializationDemand) Valid() bool {
	return d >= MapSpecializationDemandDefinition &&
		d <= MapSpecializationDemandClear
}

type DeclarationRequirementKind uint8

const (
	DeclarationRequirementInvalid DeclarationRequirementKind = iota
	DeclarationRequirementNamedStructOperation
	DeclarationRequirementAddressableStorage
	DeclarationRequirementConstantProjection
	DeclarationRequirementLocalConstantProjection
	DeclarationRequirementDefinedArrayOperation
	DeclarationRequirementAnonymousStruct
	DeclarationRequirementMapSpecialization
)

func (k DeclarationRequirementKind) Valid() bool {
	return k == DeclarationRequirementNamedStructOperation ||
		k == DeclarationRequirementAddressableStorage ||
		k == DeclarationRequirementConstantProjection ||
		k == DeclarationRequirementLocalConstantProjection ||
		k == DeclarationRequirementDefinedArrayOperation ||
		k == DeclarationRequirementAnonymousStruct ||
		k == DeclarationRequirementMapSpecialization
}

type DeclarationRequirement struct {
	owner     ArtifactOwner
	kind      DeclarationRequirementKind
	operation NamedStructOperation
	array     DefinedArrayOperation
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
	projection      types.BasicKind
	generated       *GeneratedArtifact
	anonymousDemand AnonymousStructDemand
	mapDemand       MapSpecializationDemand
}

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
		owner:     MustSourceArtifactOwner(typeName),
		kind:      DeclarationRequirementNamedStructOperation,
		operation: operation,
	}, nil
}

func NewDefinedArrayOperationRequirement(
	typeName *types.TypeName,
	operation DefinedArrayOperation,
) (DeclarationRequirement, error) {
	switch {
	case typeName == nil:
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "defined-array operation type is nil",
		}
	case !operation.Valid():
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "defined-array operation is invalid",
		}
	}
	return DeclarationRequirement{
		owner: MustSourceArtifactOwner(typeName),
		kind:  DeclarationRequirementDefinedArrayOperation,
		array: operation,
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
		owner:    MustSourceArtifactOwner(owner),
		kind:     DeclarationRequirementAddressableStorage,
		variable: variable,
	}, nil
}

func NewAnonymousStructRequirement(
	artifact *GeneratedArtifact,
	demand AnonymousStructDemand,
) (DeclarationRequirement, error) {
	if !artifact.Valid() ||
		artifact.Kind() != GeneratedArtifactAnonymousStruct ||
		!demand.Valid() {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "anonymous-struct requirement is invalid",
		}
	}
	return DeclarationRequirement{
		owner:           artifact.ReconstructionOwner(),
		kind:            DeclarationRequirementAnonymousStruct,
		generated:       artifact,
		anonymousDemand: demand,
	}, nil
}

func NewMapSpecializationRequirement(
	artifact *GeneratedArtifact,
	demand MapSpecializationDemand,
) (DeclarationRequirement, error) {
	if !artifact.Valid() ||
		artifact.Kind() != GeneratedArtifactMapSpecialization ||
		!demand.Valid() {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "map-specialization requirement is invalid",
		}
	}
	return DeclarationRequirement{
		owner:     artifact.ReconstructionOwner(),
		kind:      DeclarationRequirementMapSpecialization,
		generated: artifact,
		mapDemand: demand,
	}, nil
}

func (r DeclarationRequirement) Valid() bool {
	if !r.kind.Valid() {
		return false
	}
	switch r.kind {
	case DeclarationRequirementNamedStructOperation:
		if !r.operation.Valid() ||
			r.array != DefinedArrayOperationInvalid ||
			r.variable != nil ||
			r.constant != nil ||
			r.projection != types.Invalid ||
			r.generated != nil ||
			r.anonymousDemand != AnonymousStructDemandInvalid ||
			r.mapDemand != MapSpecializationDemandInvalid {
			return false
		}
		source, sourceOK := r.owner.Source()
		_, ok := source.(*types.TypeName)
		return sourceOK && ok
	case DeclarationRequirementDefinedArrayOperation:
		if r.operation != NamedStructOperationInvalid ||
			!r.array.Valid() ||
			r.variable != nil ||
			r.constant != nil ||
			r.projection != types.Invalid ||
			r.generated != nil ||
			r.anonymousDemand != AnonymousStructDemandInvalid ||
			r.mapDemand != MapSpecializationDemandInvalid {
			return false
		}
		source, sourceOK := r.owner.Source()
		_, ok := source.(*types.TypeName)
		return sourceOK && ok
	case DeclarationRequirementAddressableStorage:
		if r.operation != NamedStructOperationInvalid ||
			r.array != DefinedArrayOperationInvalid ||
			r.variable == nil ||
			r.variable.IsField() ||
			r.constant != nil ||
			r.projection != types.Invalid ||
			r.generated != nil ||
			r.anonymousDemand != AnonymousStructDemandInvalid ||
			r.mapDemand != MapSpecializationDemandInvalid {
			return false
		}
		source, sourceOK := r.owner.Source()
		_, ok := source.(*types.Func)
		return sourceOK && ok
	case DeclarationRequirementConstantProjection:
		if r.operation != NamedStructOperationInvalid ||
			r.array != DefinedArrayOperationInvalid ||
			r.variable != nil ||
			r.constant != nil ||
			!validConstantProjection(r.projection) ||
			r.generated != nil ||
			r.anonymousDemand != AnonymousStructDemandInvalid ||
			r.mapDemand != MapSpecializationDemandInvalid {
			return false
		}
		source, sourceOK := r.owner.Source()
		_, ok := source.(*types.Const)
		return sourceOK && ok
	case DeclarationRequirementLocalConstantProjection:
		if r.operation != NamedStructOperationInvalid ||
			r.array != DefinedArrayOperationInvalid ||
			r.variable != nil ||
			r.constant == nil ||
			!validConstantProjection(r.projection) ||
			r.generated != nil ||
			r.anonymousDemand != AnonymousStructDemandInvalid ||
			r.mapDemand != MapSpecializationDemandInvalid {
			return false
		}
		source, sourceOK := r.owner.Source()
		_, ok := source.(*types.Func)
		return sourceOK && ok
	case DeclarationRequirementAnonymousStruct:
		return r.operation == NamedStructOperationInvalid &&
			r.array == DefinedArrayOperationInvalid &&
			r.variable == nil &&
			r.constant == nil &&
			r.projection == types.Invalid &&
			r.generated.Valid() &&
			r.generated.Kind() == GeneratedArtifactAnonymousStruct &&
			r.anonymousDemand.Valid() &&
			r.mapDemand == MapSpecializationDemandInvalid &&
			r.owner == r.generated.ReconstructionOwner()
	case DeclarationRequirementMapSpecialization:
		return r.operation == NamedStructOperationInvalid &&
			r.array == DefinedArrayOperationInvalid &&
			r.variable == nil &&
			r.constant == nil &&
			r.projection == types.Invalid &&
			r.generated.Valid() &&
			r.generated.Kind() == GeneratedArtifactMapSpecialization &&
			r.anonymousDemand == AnonymousStructDemandInvalid &&
			r.mapDemand.Valid() &&
			r.owner == r.generated.ReconstructionOwner()
	default:
		return false
	}
}

func validConstantProjection(projection types.BasicKind) bool {
	_, ok := ConstantProjectionType(projection)
	return ok
}

func (r DeclarationRequirement) Owner() ArtifactOwner {
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
	if !r.Valid() ||
		r.kind != DeclarationRequirementNamedStructOperation {
		return nil, NamedStructOperationInvalid, false
	}
	source, sourceOK := r.owner.Source()
	typeName, ok := source.(*types.TypeName)
	return typeName, r.operation, sourceOK && ok
}

func (r DeclarationRequirement) DefinedArrayOperation() (
	*types.TypeName,
	DefinedArrayOperation,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementDefinedArrayOperation {
		return nil, DefinedArrayOperationInvalid, false
	}
	source, sourceOK := r.owner.Source()
	typeName, ok := source.(*types.TypeName)
	return typeName, r.array, sourceOK && ok
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
	source, sourceOK := r.owner.Source()
	owner, ok := source.(*types.Func)
	return owner, r.variable, sourceOK && ok
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
	source, sourceOK := r.owner.Source()
	constant, ok := source.(*types.Const)
	return constant, r.projection, sourceOK && ok
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
	source, sourceOK := r.owner.Source()
	owner, ok := source.(*types.Func)
	return owner, r.constant, r.projection, sourceOK && ok
}

func (r DeclarationRequirement) AnonymousStruct() (
	*GeneratedArtifact,
	AnonymousStructDemand,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementAnonymousStruct {
		return nil, AnonymousStructDemandInvalid, false
	}
	return r.generated, r.anonymousDemand, true
}

func (r DeclarationRequirement) MapSpecialization() (
	*GeneratedArtifact,
	MapSpecializationDemand,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementMapSpecialization {
		return nil, MapSpecializationDemandInvalid, false
	}
	return r.generated, r.mapDemand, true
}

func (r DeclarationRequirement) GeneratedArtifact() (
	*GeneratedArtifact,
	bool,
) {
	if !r.Valid() {
		return nil, false
	}
	switch r.kind {
	case DeclarationRequirementAnonymousStruct,
		DeclarationRequirementMapSpecialization:
		return r.generated, true
	default:
		return nil, false
	}
}
