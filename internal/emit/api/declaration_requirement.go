package api

import (
	"go/ast"
	"go/token"
	"go/types"
)

type DeclarationRequirement struct {
	owner       ArtifactOwner
	kind        DeclarationRequirementKind
	operation   NamedStructOperation
	typeName    *types.TypeName
	classMethod *types.Func
	variable    *types.Var
	// constant is the untyped constant a local projection materializes. A
	// package projection owns the constant directly (owner is the constant), so
	// this stays nil there; a local projection is owned by the enclosing
	// function, so the constant identity travels here.
	constant *types.Const
	// projection is the exact target basic representation of an untyped
	// constant projection. A basic kind is a canonical, comparable dedup key —
	// unlike a types.Type interface value, whose pointer identity is not a
	// stable projection key.
	projection             types.BasicKind
	generated              *GeneratedArtifact
	interfaceContract      *types.Interface
	interfaceContractKey   string
	anonymousDemand        AnonymousStructDemand
	mapDemand              MapSpecializationDemand
	genericOperation       *GenericOperationContract
	genericParameter       *types.TypeParam
	genericFacet           GenericRepresentationFacet
	typeRepresentation     TypeRepresentationFacet
	pointerCarrier         bool
	concretizationDeferred bool
	enclosing              ast.Node
	callable               ast.Node
	control                CallableControlFacet
	controlLabel           *types.Label
	controlPosition        token.Pos
	controlRange           *ast.RangeStmt
	callableFacet          CallableFacet
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
		typeName:  typeName,
	}, nil
}

func NewLexicalNamedStructOperationRequirement(
	owner ArtifactOwner,
	typeName *types.TypeName,
	operation NamedStructOperation,
) (DeclarationRequirement, error) {
	if !validLexicalNamedStructOwner(owner, typeName) ||
		!operation.Valid() {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "lexical named-struct operation is invalid",
		}
	}
	return DeclarationRequirement{
		owner:     owner,
		kind:      DeclarationRequirementNamedStructOperation,
		operation: operation,
		typeName:  typeName,
	}, nil
}

func NewAddressableStorageRequirement(
	owner ArtifactOwner,
	variable *types.Var,
) (DeclarationRequirement, error) {
	switch {
	case !owner.Valid():
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "addressable-storage owner is invalid",
		}
	case variable == nil:
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "addressable-storage variable is nil",
		}
	case variable.IsField():
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "addressable-storage variable is a field",
		}
	case !validAddressableStorageOwner(owner, variable):
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "addressable-storage owner does not contain the variable",
		}
	}
	return DeclarationRequirement{
		owner:    owner,
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

func NewInterfaceAdapterRequirement(
	artifact *GeneratedArtifact,
) (DeclarationRequirement, error) {
	return newGeneratedDefinitionRequirement(
		artifact,
		GeneratedArtifactInterfaceAdapter,
		DeclarationRequirementInterfaceAdapter,
		"interface-adapter",
	)
}

func NewInterfaceAdapterContractRequirement(
	artifact *GeneratedArtifact,
	contract *types.Interface,
	contractKey string,
) (DeclarationRequirement, error) {
	if !artifact.Valid() ||
		artifact.Kind() != GeneratedArtifactInterfaceAdapter ||
		contract == nil ||
		!contract.Complete().IsMethodSet() ||
		contractKey == "" {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "interface-adapter contract requirement is invalid",
		}
	}
	sourceType, ok := artifact.InterfaceAdapterType()
	if !ok || !types.Implements(sourceType, contract) {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "interface-adapter source does not implement its demanded contract",
		}
	}
	return DeclarationRequirement{
		owner:                artifact.ReconstructionOwner(),
		kind:                 DeclarationRequirementInterfaceAdapter,
		generated:            artifact,
		interfaceContract:    contract,
		interfaceContractKey: contractKey,
	}, nil
}

func NewAnonymousInterfaceRequirement(
	artifact *GeneratedArtifact,
) (DeclarationRequirement, error) {
	return newGeneratedDefinitionRequirement(
		artifact,
		GeneratedArtifactAnonymousInterface,
		DeclarationRequirementAnonymousInterface,
		"anonymous-interface",
	)
}

func NewInterfaceMethodTokenRequirement(
	artifact *GeneratedArtifact,
) (DeclarationRequirement, error) {
	return newGeneratedDefinitionRequirement(
		artifact,
		GeneratedArtifactInterfaceMethodToken,
		DeclarationRequirementInterfaceMethodToken,
		"interface-method-token",
	)
}

func NewInterfaceMethodCallableRequirement(
	artifact *GeneratedArtifact,
) (DeclarationRequirement, error) {
	return newGeneratedDefinitionRequirement(
		artifact,
		GeneratedArtifactInterfaceMethodCallable,
		DeclarationRequirementInterfaceMethodCallable,
		"interface-method callable",
	)
}

func NewInterfaceDynamicTypeTokenRequirement(
	artifact *GeneratedArtifact,
) (DeclarationRequirement, error) {
	return newGeneratedDefinitionRequirement(
		artifact,
		GeneratedArtifactInterfaceDynamicTypeToken,
		DeclarationRequirementInterfaceDynamicTypeToken,
		"interface-dynamic-type-token",
	)
}

func NewReflectionTypeRequirement(
	artifact *GeneratedArtifact,
) (DeclarationRequirement, error) {
	return newGeneratedDefinitionRequirement(
		artifact,
		GeneratedArtifactReflectionType,
		DeclarationRequirementReflectionType,
		"reflection type",
	)
}

func NewProviderInterfaceBridgeRequirement(
	artifact *GeneratedArtifact,
) (DeclarationRequirement, error) {
	return newGeneratedDefinitionRequirement(
		artifact,
		GeneratedArtifactProviderInterfaceBridge,
		DeclarationRequirementProviderInterfaceBridge,
		"provider-interface bridge",
	)
}

func NewProviderStatefulRepresentationRequirement(
	artifact *GeneratedArtifact,
) (DeclarationRequirement, error) {
	return newGeneratedDefinitionRequirement(
		artifact,
		GeneratedArtifactProviderStatefulRepresentation,
		DeclarationRequirementProviderStatefulRepresentation,
		"provider stateful representation",
	)
}

func NewDeferredCallableRegistryRequirement(
	artifact *GeneratedArtifact,
) (DeclarationRequirement, error) {
	return newGeneratedDefinitionRequirement(
		artifact,
		GeneratedArtifactDeferredCallableRegistry,
		DeclarationRequirementDeferredCallableRegistry,
		"deferred-callable registry",
	)
}

func NewDeferredCallableRegistryRequest(
	artifact *GeneratedArtifact,
) (RootRequest, error) {
	requirement, err := NewDeferredCallableRegistryRequirement(artifact)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func newGeneratedDefinitionRequirement(
	artifact *GeneratedArtifact,
	artifactKind GeneratedArtifactKind,
	requirementKind DeclarationRequirementKind,
	name string,
) (DeclarationRequirement, error) {
	if !artifact.Valid() || artifact.Kind() != artifactKind {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: name + " requirement is invalid",
		}
	}
	return DeclarationRequirement{
		owner:     artifact.ReconstructionOwner(),
		kind:      requirementKind,
		generated: artifact,
	}, nil
}
