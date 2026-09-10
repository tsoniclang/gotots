package api

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	controlcontract "github.com/tsoniclang/gotots/internal/emit/api/control"
)

type NamedStructOperation uint8

const (
	NamedStructOperationInvalid NamedStructOperation = iota
	NamedStructOperationZero
	NamedStructOperationCopy
	NamedStructOperationEqual
	NamedStructOperationHash
	NamedStructOperationConvert
	NamedStructOperationStorage
	NamedStructOperationAssign
	NamedStructOperationStorageZero
)

func (o NamedStructOperation) Valid() bool {
	return o == NamedStructOperationZero ||
		o == NamedStructOperationCopy ||
		o == NamedStructOperationEqual ||
		o == NamedStructOperationHash ||
		o == NamedStructOperationConvert ||
		o == NamedStructOperationStorage ||
		o == NamedStructOperationAssign ||
		o == NamedStructOperationStorageZero
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
	case NamedStructOperationConvert:
		return "convert"
	case NamedStructOperationStorage:
		return "storage"
	case NamedStructOperationAssign:
		return "assign"
	case NamedStructOperationStorageZero:
		return "storage-zero"
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
	if operation == NamedStructOperationStorageZero {
		return StructStorageZeroMember, nil
	}
	return "$" + operation.String(), nil
}

type AnonymousStructDemand uint8

const (
	AnonymousStructDemandInvalid AnonymousStructDemand = iota
	AnonymousStructDemandDefinition
	AnonymousStructDemandZero
	AnonymousStructDemandCopy
	AnonymousStructDemandEqual
	AnonymousStructDemandHash
	AnonymousStructDemandConvert
	AnonymousStructDemandStorage
	AnonymousStructDemandAssign
)

func (d AnonymousStructDemand) Valid() bool {
	return d >= AnonymousStructDemandDefinition &&
		d <= AnonymousStructDemandAssign
}

type MapSpecializationDemand uint8

const (
	MapSpecializationDemandInvalid MapSpecializationDemand = iota
	MapSpecializationDemandDefinition
	MapSpecializationDemandStatic
)

func (d MapSpecializationDemand) Valid() bool {
	return d >= MapSpecializationDemandDefinition &&
		d <= MapSpecializationDemandStatic
}

type DeclarationRequirementKind uint8

const (
	DeclarationRequirementInvalid                            DeclarationRequirementKind = 0
	DeclarationRequirementNamedStructOperation               DeclarationRequirementKind = 1
	DeclarationRequirementConstantProjection                 DeclarationRequirementKind = 3
	DeclarationRequirementLocalConstantProjection            DeclarationRequirementKind = 4
	DeclarationRequirementGenericOperation                   DeclarationRequirementKind = 5
	DeclarationRequirementAnonymousStruct                    DeclarationRequirementKind = 6
	DeclarationRequirementMapSpecialization                  DeclarationRequirementKind = 7
	DeclarationRequirementInterfaceAdapter                   DeclarationRequirementKind = 8
	DeclarationRequirementAnonymousInterface                 DeclarationRequirementKind = 9
	DeclarationRequirementInterfaceMethodToken               DeclarationRequirementKind = 10
	DeclarationRequirementInterfaceDynamicTypeToken          DeclarationRequirementKind = 11
	DeclarationRequirementGenericCapability                  DeclarationRequirementKind = 12
	DeclarationRequirementCallableControl                    DeclarationRequirementKind = 13
	DeclarationRequirementCallableABI                        DeclarationRequirementKind = 15
	DeclarationRequirementClassMethod                        DeclarationRequirementKind = 18
	DeclarationRequirementValueReceiverCopy                  DeclarationRequirementKind = 19
	DeclarationRequirementGenericRepresentation              DeclarationRequirementKind = 20
	DeclarationRequirementInterfaceMethodCallable            DeclarationRequirementKind = 21
	DeclarationRequirementProviderInterfaceBridge            DeclarationRequirementKind = 23
	DeclarationRequirementProviderStatefulRepresentation     DeclarationRequirementKind = 24
	DeclarationRequirementDeferredCallableRegistry           DeclarationRequirementKind = 25
	DeclarationRequirementGenericConcretization              DeclarationRequirementKind = 26
	DeclarationRequirementTypeRepresentation                 DeclarationRequirementKind = 27
	DeclarationRequirementReflectionType                     DeclarationRequirementKind = 28
	DeclarationRequirementProviderInterfaceCapability        DeclarationRequirementKind = 30
	DeclarationRequirementProviderProfileInterfaceCapability DeclarationRequirementKind = 31
	DeclarationRequirementReflectionValueOperations          DeclarationRequirementKind = 32
)

func (k DeclarationRequirementKind) Valid() bool {
	return k == DeclarationRequirementNamedStructOperation ||
		k == DeclarationRequirementConstantProjection ||
		k == DeclarationRequirementLocalConstantProjection ||
		k == DeclarationRequirementGenericOperation ||
		k == DeclarationRequirementAnonymousStruct ||
		k == DeclarationRequirementMapSpecialization ||
		k == DeclarationRequirementInterfaceAdapter ||
		k == DeclarationRequirementAnonymousInterface ||
		k == DeclarationRequirementInterfaceMethodToken ||
		k == DeclarationRequirementInterfaceDynamicTypeToken ||
		k == DeclarationRequirementGenericCapability ||
		k == DeclarationRequirementCallableControl ||
		k == DeclarationRequirementCallableABI ||
		k == DeclarationRequirementClassMethod ||
		k == DeclarationRequirementValueReceiverCopy ||
		k == DeclarationRequirementGenericRepresentation ||
		k == DeclarationRequirementInterfaceMethodCallable ||
		k == DeclarationRequirementProviderInterfaceBridge ||
		k == DeclarationRequirementProviderStatefulRepresentation ||
		k == DeclarationRequirementDeferredCallableRegistry ||
		k == DeclarationRequirementGenericConcretization ||
		k == DeclarationRequirementTypeRepresentation ||
		k == DeclarationRequirementReflectionType ||
		k == DeclarationRequirementProviderInterfaceCapability ||
		k == DeclarationRequirementProviderProfileInterfaceCapability ||
		k == DeclarationRequirementReflectionValueOperations
}

type CallableControlFacet = controlcontract.CallableFacet

const (
	CallableControlInvalid          = controlcontract.CallableInvalid
	CallableControlDefer            = controlcontract.CallableDefer
	CallableControlRecovery         = controlcontract.CallableRecovery
	CallableControlGoto             = controlcontract.CallableGoto
	CallableControlIteratorReturn   = controlcontract.CallableIteratorReturn
	CallableControlIndirectMutation = controlcontract.CallableIndirectMutation
)

func NewGenericCapabilityRequirement(
	artifact *GeneratedArtifact,
) (DeclarationRequirement, error) {
	if !artifact.Valid() ||
		artifact.Kind() != GeneratedArtifactGenericCapability {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "generic capability requirement is invalid",
		}
	}
	return DeclarationRequirement{
		owner:     artifact.ReconstructionOwner(),
		kind:      DeclarationRequirementGenericCapability,
		generated: artifact,
	}, nil
}

func NewGenericCapabilityRequest(
	artifact *GeneratedArtifact,
) (RootRequest, error) {
	requirement, err := NewGenericCapabilityRequirement(artifact)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func NewGenericOperationRequirement(
	owner types.Object,
	operation *GenericOperationContract,
) (DeclarationRequirement, error) {
	if owner == nil {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "generic operation owner is nil",
		}
	}
	owner = GenericDeclarationOrigin(owner)
	if operation == nil ||
		!operation.Valid() ||
		operation.Owner() != owner ||
		len(GenericDeclarationParameters(owner)) == 0 {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "generic operation requirement is invalid",
		}
	}
	return DeclarationRequirement{
		owner:            MustSourceArtifactOwner(owner),
		kind:             DeclarationRequirementGenericOperation,
		genericOperation: operation,
	}, nil
}

func NewGenericOperationRequest(
	owner types.Object,
	operation *GenericOperationContract,
) (RootRequest, error) {
	requirement, err := NewGenericOperationRequirement(owner, operation)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func GenericDeclarationParameters(owner types.Object) []*types.TypeParam {
	var parameters []*types.TypeParam
	var lists []*types.TypeParamList
	switch source := owner.(type) {
	case *types.Func:
		signature, _ := source.Type().(*types.Signature)
		if signature != nil {
			lists = []*types.TypeParamList{
				signature.RecvTypeParams(),
				signature.TypeParams(),
			}
		}
	case *types.TypeName:
		switch declared := source.Type().(type) {
		case *types.Named:
			lists = []*types.TypeParamList{declared.TypeParams()}
		case *types.Alias:
			lists = []*types.TypeParamList{declared.TypeParams()}
		}
	}
	for _, list := range lists {
		for index := range list.Len() {
			parameters = append(parameters, list.At(index))
		}
	}
	return parameters
}

func GenericDeclarationOrigin(owner types.Object) types.Object {
	switch source := owner.(type) {
	case *types.Func:
		return source.Origin()
	case *types.TypeName:
		switch declared := source.Type().(type) {
		case *types.Named:
			return declared.Origin().Obj()
		case *types.Alias:
			return declared.Origin().Obj()
		}
	}
	return nil
}

func validLexicalNamedStructOwner(
	owner ArtifactOwner,
	typeName *types.TypeName,
) bool {
	if !owner.Valid() ||
		typeName == nil ||
		typeName.Pkg() == nil ||
		typeName.Parent() == nil ||
		typeName.Parent() == typeName.Pkg().Scope() ||
		owner.Package() != typeName.Pkg() {
		return false
	}
	if source, ok := owner.Source(); ok {
		_, function := source.(*types.Func)
		return function
	}
	_, _, initializer := owner.PackageInitializer()
	return initializer
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

func NewCallableControlRequirement(
	owner ArtifactOwner,
	enclosing ast.Node,
	callable ast.Node,
	control CallableControlFacet,
) (DeclarationRequirement, error) {
	if !controlcontract.ValidAnchor(owner, enclosing, callable) ||
		!control.Valid() ||
		control == CallableControlGoto ||
		control == CallableControlIteratorReturn ||
		control == CallableControlIndirectMutation {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "callable-control requirement is invalid",
		}
	}
	return DeclarationRequirement{
		owner:     owner,
		kind:      DeclarationRequirementCallableControl,
		enclosing: enclosing,
		callable:  callable,
		control:   control,
	}, nil
}

func NewDeferControlRequirement(
	owner ArtifactOwner,
	enclosing ast.Node,
	callable ast.Node,
	source *ast.DeferStmt,
) (DeclarationRequirement, error) {
	if !controlcontract.ValidAnchor(owner, enclosing, callable) ||
		!controlcontract.ValidDefer(callable, source) {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "defer control requirement is invalid",
		}
	}
	return DeclarationRequirement{
		owner:        owner,
		kind:         DeclarationRequirementCallableControl,
		enclosing:    enclosing,
		callable:     callable,
		control:      CallableControlDefer,
		controlDefer: source,
	}, nil
}

func NewIteratorReturnControlRequirement(
	owner ArtifactOwner,
	enclosing ast.Node,
	callable ast.Node,
	source *ast.RangeStmt,
) (DeclarationRequirement, error) {
	if !controlcontract.ValidAnchor(owner, enclosing, callable) ||
		!controlcontract.ValidIteratorRange(callable, source) {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "iterator-return control requirement is invalid",
		}
	}
	return DeclarationRequirement{
		owner:        owner,
		kind:         DeclarationRequirementCallableControl,
		enclosing:    enclosing,
		callable:     callable,
		control:      CallableControlIteratorReturn,
		controlRange: source,
	}, nil
}

func NewGotoControlRequirement(
	owner ArtifactOwner,
	enclosing ast.Node,
	callable ast.Node,
	label *types.Label,
	position token.Pos,
) (DeclarationRequirement, error) {
	if !controlcontract.ValidAnchor(owner, enclosing, callable) ||
		label == nil ||
		!position.IsValid() ||
		position < callable.Pos() ||
		position > callable.End() {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "goto control requirement is invalid",
		}
	}
	return DeclarationRequirement{
		owner:           owner,
		kind:            DeclarationRequirementCallableControl,
		enclosing:       enclosing,
		callable:        callable,
		control:         CallableControlGoto,
		controlLabel:    label,
		controlPosition: position,
	}, nil
}

func NewCallableControlRequest(
	owner ArtifactOwner,
	enclosing ast.Node,
	callable ast.Node,
	control CallableControlFacet,
) (RootRequest, error) {
	requirement, err := NewCallableControlRequirement(
		owner,
		enclosing,
		callable,
		control,
	)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func NewDeferControlRequest(
	owner ArtifactOwner,
	enclosing ast.Node,
	callable ast.Node,
	source *ast.DeferStmt,
) (RootRequest, error) {
	requirement, err := NewDeferControlRequirement(
		owner,
		enclosing,
		callable,
		source,
	)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func NewGotoControlRequest(
	owner ArtifactOwner,
	enclosing ast.Node,
	callable ast.Node,
	label *types.Label,
	position token.Pos,
) (RootRequest, error) {
	requirement, err := NewGotoControlRequirement(
		owner,
		enclosing,
		callable,
		label,
		position,
	)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func NewDirectCallableControlRequest(
	owner *types.Func,
	control CallableControlFacet,
) (RootRequest, error) {
	requirement, err := NewDirectCallableControlRequirement(owner, control)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}
