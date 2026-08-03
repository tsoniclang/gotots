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
	NamedStructOperationConvert
	NamedStructOperationStorage
	NamedStructOperationAssign
)

func (o NamedStructOperation) Valid() bool {
	return o == NamedStructOperationZero ||
		o == NamedStructOperationCopy ||
		o == NamedStructOperationEqual ||
		o == NamedStructOperationHash ||
		o == NamedStructOperationConvert ||
		o == NamedStructOperationStorage ||
		o == NamedStructOperationAssign
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
	DeclarationRequirementInvalid                        DeclarationRequirementKind = 0
	DeclarationRequirementNamedStructOperation           DeclarationRequirementKind = 1
	DeclarationRequirementAddressableStorage             DeclarationRequirementKind = 2
	DeclarationRequirementConstantProjection             DeclarationRequirementKind = 3
	DeclarationRequirementLocalConstantProjection        DeclarationRequirementKind = 4
	DeclarationRequirementGenericOperation               DeclarationRequirementKind = 5
	DeclarationRequirementAnonymousStruct                DeclarationRequirementKind = 6
	DeclarationRequirementMapSpecialization              DeclarationRequirementKind = 7
	DeclarationRequirementInterfaceAdapter               DeclarationRequirementKind = 8
	DeclarationRequirementAnonymousInterface             DeclarationRequirementKind = 9
	DeclarationRequirementInterfaceMethodToken           DeclarationRequirementKind = 10
	DeclarationRequirementInterfaceDynamicTypeToken      DeclarationRequirementKind = 11
	DeclarationRequirementGenericCapability              DeclarationRequirementKind = 12
	DeclarationRequirementCallableControl                DeclarationRequirementKind = 13
	DeclarationRequirementCooperativeCallable            DeclarationRequirementKind = 14
	DeclarationRequirementCallableABI                    DeclarationRequirementKind = 15
	DeclarationRequirementClassMethod                    DeclarationRequirementKind = 18
	DeclarationRequirementValueReceiverCopy              DeclarationRequirementKind = 19
	DeclarationRequirementGenericRepresentation          DeclarationRequirementKind = 20
	DeclarationRequirementInterfaceMethodCallable        DeclarationRequirementKind = 21
	DeclarationRequirementPointerRepresentation          DeclarationRequirementKind = 22
	DeclarationRequirementProviderInterfaceBridge        DeclarationRequirementKind = 23
	DeclarationRequirementProviderStatefulRepresentation DeclarationRequirementKind = 24
	DeclarationRequirementDeferredCallableRegistry       DeclarationRequirementKind = 25
	DeclarationRequirementGenericConcretization          DeclarationRequirementKind = 26
	DeclarationRequirementTypeRepresentation             DeclarationRequirementKind = 27
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
		k == DeclarationRequirementCallableControl ||
		k == DeclarationRequirementCooperativeCallable ||
		k == DeclarationRequirementCallableABI ||
		k == DeclarationRequirementClassMethod ||
		k == DeclarationRequirementValueReceiverCopy ||
		k == DeclarationRequirementGenericRepresentation ||
		k == DeclarationRequirementInterfaceMethodCallable ||
		k == DeclarationRequirementPointerRepresentation ||
		k == DeclarationRequirementProviderInterfaceBridge ||
		k == DeclarationRequirementProviderStatefulRepresentation ||
		k == DeclarationRequirementDeferredCallableRegistry ||
		k == DeclarationRequirementGenericConcretization ||
		k == DeclarationRequirementTypeRepresentation
}

type CallableControlFacet uint8

const (
	CallableControlInvalid CallableControlFacet = iota
	CallableControlDefer
	CallableControlRecovery
	CallableControlGoto
	CallableControlIteratorReturn
)

func (f CallableControlFacet) Valid() bool {
	return f == CallableControlDefer ||
		f == CallableControlRecovery ||
		f == CallableControlGoto ||
		f == CallableControlIteratorReturn
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

func validAddressableStorageOwner(
	owner ArtifactOwner,
	variable *types.Var,
) bool {
	if !owner.Valid() ||
		variable == nil ||
		variable.IsField() ||
		variable.Pkg() == nil ||
		owner.Package() != variable.Pkg() {
		return false
	}
	if source, ok := owner.Source(); ok {
		_, callable := source.(*types.Func)
		return callable
	}
	_, initializer, ok := owner.PackageInitializer()
	return ok &&
		variable.Pos().IsValid() &&
		variable.Pos() >= initializer.Rhs.Pos() &&
		variable.Pos() <= initializer.Rhs.End()
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
