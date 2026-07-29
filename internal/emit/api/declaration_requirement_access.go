package api

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
)

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
	return r.typeName, r.operation, true
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

func (r DeclarationRequirement) GenericOperation() (
	types.Object,
	*GenericOperationContract,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementGenericOperation {
		return nil, nil, false
	}
	source, sourceOK := r.owner.Source()
	return source,
		r.genericOperation,
		sourceOK &&
			GenericDeclarationOrigin(source) == source
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

func (r DeclarationRequirement) InterfaceAdapter() (
	*GeneratedArtifact,
	bool,
) {
	return r.generatedDefinition(
		DeclarationRequirementInterfaceAdapter,
		GeneratedArtifactInterfaceAdapter,
	)
}

func (r DeclarationRequirement) AnonymousInterface() (
	*GeneratedArtifact,
	bool,
) {
	return r.generatedDefinition(
		DeclarationRequirementAnonymousInterface,
		GeneratedArtifactAnonymousInterface,
	)
}

func (r DeclarationRequirement) InterfaceMethodToken() (
	*GeneratedArtifact,
	bool,
) {
	return r.generatedDefinition(
		DeclarationRequirementInterfaceMethodToken,
		GeneratedArtifactInterfaceMethodToken,
	)
}

func (r DeclarationRequirement) InterfaceDynamicTypeToken() (
	*GeneratedArtifact,
	bool,
) {
	return r.generatedDefinition(
		DeclarationRequirementInterfaceDynamicTypeToken,
		GeneratedArtifactInterfaceDynamicTypeToken,
	)
}

func (r DeclarationRequirement) GenericCapability() (
	*GeneratedArtifact,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementGenericCapability ||
		r.generated.Kind() != GeneratedArtifactGenericCapability {
		return nil, false
	}
	return r.generated, true
}

func (r DeclarationRequirement) CallableControl() (
	ArtifactOwner,
	ast.Node,
	ast.Node,
	CallableControlFacet,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementCallableControl {
		return ArtifactOwner{}, nil, nil, CallableControlInvalid, false
	}
	return r.owner, r.enclosing, r.callable, r.control, true
}

func (r DeclarationRequirement) GotoControl() (
	*types.Label,
	token.Pos,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementCallableControl ||
		r.control != CallableControlGoto {
		return nil, token.NoPos, false
	}
	return r.controlLabel, r.controlPosition, true
}

func (r DeclarationRequirement) generatedDefinition(
	requirementKind DeclarationRequirementKind,
	artifactKind GeneratedArtifactKind,
) (*GeneratedArtifact, bool) {
	if !r.Valid() ||
		r.kind != requirementKind ||
		r.generated.Kind() != artifactKind {
		return nil, false
	}
	return r.generated, true
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
		DeclarationRequirementMapSpecialization,
		DeclarationRequirementInterfaceAdapter,
		DeclarationRequirementAnonymousInterface,
		DeclarationRequirementInterfaceMethodToken,
		DeclarationRequirementInterfaceDynamicTypeToken,
		DeclarationRequirementGenericCapability:
		return r.generated, true
	default:
		return nil, false
	}
}

type NamedStructOperation uint8

const (
	NamedStructOperationInvalid NamedStructOperation = iota
	NamedStructOperationZero
	NamedStructOperationCopy
	NamedStructOperationEqual
	NamedStructOperationHash
	NamedStructOperationConvert
	NamedStructOperationStorage
)

func (o NamedStructOperation) Valid() bool {
	return o == NamedStructOperationZero ||
		o == NamedStructOperationCopy ||
		o == NamedStructOperationEqual ||
		o == NamedStructOperationHash ||
		o == NamedStructOperationConvert ||
		o == NamedStructOperationStorage
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
)

func (d AnonymousStructDemand) Valid() bool {
	return d >= AnonymousStructDemandDefinition &&
		d <= AnonymousStructDemandStorage
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
	DeclarationRequirementInvalid                   DeclarationRequirementKind = 0
	DeclarationRequirementNamedStructOperation      DeclarationRequirementKind = 1
	DeclarationRequirementAddressableStorage        DeclarationRequirementKind = 2
	DeclarationRequirementConstantProjection        DeclarationRequirementKind = 3
	DeclarationRequirementLocalConstantProjection   DeclarationRequirementKind = 4
	DeclarationRequirementGenericOperation          DeclarationRequirementKind = 5
	DeclarationRequirementAnonymousStruct           DeclarationRequirementKind = 6
	DeclarationRequirementMapSpecialization         DeclarationRequirementKind = 7
	DeclarationRequirementInterfaceAdapter          DeclarationRequirementKind = 8
	DeclarationRequirementAnonymousInterface        DeclarationRequirementKind = 9
	DeclarationRequirementInterfaceMethodToken      DeclarationRequirementKind = 10
	DeclarationRequirementInterfaceDynamicTypeToken DeclarationRequirementKind = 11
	DeclarationRequirementGenericCapability         DeclarationRequirementKind = 12
	DeclarationRequirementCallableControl           DeclarationRequirementKind = 13
)

func (k DeclarationRequirementKind) Valid() bool {
	return k == DeclarationRequirementNamedStructOperation ||
		k == DeclarationRequirementAddressableStorage ||
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
		k == DeclarationRequirementCallableControl
}

type CallableControlFacet uint8

const (
	CallableControlInvalid CallableControlFacet = iota
	CallableControlDefer
	CallableControlRecovery
	CallableControlGoto
)

func (f CallableControlFacet) Valid() bool {
	return f == CallableControlDefer ||
		f == CallableControlRecovery ||
		f == CallableControlGoto
}

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
