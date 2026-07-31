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
	projection           types.BasicKind
	generated            *GeneratedArtifact
	interfaceContract    *types.Interface
	interfaceContractKey string
	anonymousDemand      AnonymousStructDemand
	mapDemand            MapSpecializationDemand
	genericOperation     *GenericOperationContract
	genericParameter     *types.TypeParam
	genericFacet         GenericRepresentationFacet
	pointerCarrier       bool
	genericProfile       *GenericCallableProfile
	environmentBuiltin   *types.Builtin
	environmentSignature *types.Signature
	enclosing            ast.Node
	callable             ast.Node
	control              CallableControlFacet
	controlLabel         *types.Label
	controlPosition      token.Pos
	controlRange         *ast.RangeStmt
	callableFacet        CallableFacet
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

func (r DeclarationRequirement) Valid() bool {
	if !r.kind.Valid() {
		return false
	}
	if r.kind != DeclarationRequirementCallableControl &&
		(r.enclosing != nil ||
			r.callable != nil ||
			r.control != CallableControlInvalid ||
			r.controlLabel != nil ||
			r.controlPosition.IsValid() ||
			r.controlRange != nil) {
		return false
	}
	if r.kind != DeclarationRequirementGenericOperation &&
		r.kind != DeclarationRequirementGenericCapability &&
		r.genericOperation != nil {
		return false
	}
	if r.kind != DeclarationRequirementGenericRepresentation &&
		(r.genericParameter != nil ||
			r.genericFacet != GenericRepresentationInvalid) {
		return false
	}
	if r.kind != DeclarationRequirementPointerRepresentation &&
		r.pointerCarrier {
		return false
	}
	if r.kind != DeclarationRequirementGenericCallableProfile &&
		r.genericProfile != nil {
		return false
	}
	if r.kind != DeclarationRequirementCooperativeCallable &&
		!r.callableFacet.empty() {
		return false
	}
	if r.kind != DeclarationRequirementEnvironmentBuiltin &&
		(r.environmentBuiltin != nil || r.environmentSignature != nil) {
		return false
	}
	if r.kind != DeclarationRequirementInterfaceAdapter &&
		(r.interfaceContract != nil || r.interfaceContractKey != "") {
		return false
	}
	if r.kind != DeclarationRequirementClassMethod &&
		r.classMethod != nil {
		return false
	}
	switch r.kind {
	case DeclarationRequirementValueReceiverCopy:
		if r.operation != NamedStructOperationInvalid ||
			r.typeName != nil ||
			r.variable != nil ||
			r.constant != nil ||
			r.projection != types.Invalid ||
			r.generated != nil ||
			r.anonymousDemand != AnonymousStructDemandInvalid ||
			r.mapDemand != MapSpecializationDemandInvalid {
			return false
		}
		source, sourceOK := r.owner.Source()
		method, methodOK := source.(*types.Func)
		return sourceOK &&
			methodOK &&
			method.Origin() == method &&
			ValueReceiverTypeName(method) != nil
	case DeclarationRequirementClassMethod:
		if r.operation != NamedStructOperationInvalid ||
			r.typeName == nil ||
			r.classMethod == nil ||
			r.variable != nil ||
			r.constant != nil ||
			r.projection != types.Invalid ||
			r.generated != nil ||
			r.anonymousDemand != AnonymousStructDemandInvalid ||
			r.mapDemand != MapSpecializationDemandInvalid {
			return false
		}
		source, sourceOK := r.owner.Source()
		owner, ownerOK := source.(*types.TypeName)
		return sourceOK &&
			ownerOK &&
			owner == r.typeName &&
			MethodReceiverTypeName(r.classMethod) == owner
	case DeclarationRequirementNamedStructOperation:
		if !r.operation.Valid() ||
			r.typeName == nil ||
			r.variable != nil ||
			r.constant != nil ||
			r.projection != types.Invalid ||
			r.generated != nil ||
			r.anonymousDemand != AnonymousStructDemandInvalid ||
			r.mapDemand != MapSpecializationDemandInvalid {
			return false
		}
		source, sourceOK := r.owner.Source()
		if sourceType, ok := source.(*types.TypeName); sourceOK && ok {
			return sourceType == r.typeName
		}
		return validLexicalNamedStructOwner(r.owner, r.typeName)
	case DeclarationRequirementAddressableStorage:
		if r.operation != NamedStructOperationInvalid ||
			r.typeName != nil ||
			r.variable == nil ||
			r.variable.IsField() ||
			r.constant != nil ||
			r.projection != types.Invalid ||
			r.generated != nil ||
			r.anonymousDemand != AnonymousStructDemandInvalid ||
			r.mapDemand != MapSpecializationDemandInvalid {
			return false
		}
		return validAddressableStorageOwner(r.owner, r.variable)
	case DeclarationRequirementConstantProjection:
		if r.operation != NamedStructOperationInvalid ||
			r.typeName != nil ||
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
			r.typeName != nil ||
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
	case DeclarationRequirementGenericOperation:
		if r.operation != NamedStructOperationInvalid ||
			r.typeName != nil ||
			r.variable != nil ||
			r.constant != nil ||
			r.projection != types.Invalid ||
			r.generated != nil ||
			r.anonymousDemand != AnonymousStructDemandInvalid ||
			r.mapDemand != MapSpecializationDemandInvalid ||
			!r.genericOperation.Valid() {
			return false
		}
		source, sourceOK := r.owner.Source()
		return sourceOK &&
			GenericDeclarationOrigin(source) == source &&
			len(GenericDeclarationParameters(source)) != 0 &&
			r.genericOperation.Owner() == source
	case DeclarationRequirementGenericCallableProfile:
		if !r.owner.Valid() ||
			r.operation != NamedStructOperationInvalid ||
			r.typeName != nil ||
			r.variable != nil ||
			r.constant != nil ||
			r.projection != types.Invalid ||
			r.generated != nil ||
			r.anonymousDemand != AnonymousStructDemandInvalid ||
			r.mapDemand != MapSpecializationDemandInvalid ||
			r.genericOperation != nil ||
			!r.genericProfile.Valid() {
			return false
		}
		source, sourceOK := r.owner.Source()
		return sourceOK &&
			source == r.genericProfile.Owner()
	case DeclarationRequirementGenericRepresentation:
		if r.operation != NamedStructOperationInvalid ||
			r.typeName != nil ||
			r.variable != nil ||
			r.constant != nil ||
			r.projection != types.Invalid ||
			r.generated != nil ||
			r.anonymousDemand != AnonymousStructDemandInvalid ||
			r.mapDemand != MapSpecializationDemandInvalid ||
			r.genericOperation != nil ||
			!r.genericFacet.Valid() {
			return false
		}
		source, sourceOK := r.owner.Source()
		_, indexed := GenericDeclarationParameterIndex(
			source,
			r.genericParameter,
		)
		return sourceOK &&
			GenericDeclarationOrigin(source) == source &&
			indexed
	case DeclarationRequirementAnonymousStruct:
		return r.operation == NamedStructOperationInvalid &&
			r.typeName == nil &&
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
			r.typeName == nil &&
			r.variable == nil &&
			r.constant == nil &&
			r.projection == types.Invalid &&
			r.generated.Valid() &&
			r.generated.Kind() == GeneratedArtifactMapSpecialization &&
			r.anonymousDemand == AnonymousStructDemandInvalid &&
			r.mapDemand.Valid() &&
			r.owner == r.generated.ReconstructionOwner()
	case DeclarationRequirementInterfaceAdapter:
		if !r.validGeneratedDefinition(
			GeneratedArtifactInterfaceAdapter,
		) {
			return false
		}
		if r.interfaceContract == nil {
			return r.interfaceContractKey == ""
		}
		sourceType, ok := r.generated.InterfaceAdapterType()
		return ok &&
			r.interfaceContractKey != "" &&
			r.interfaceContract.Complete().IsMethodSet() &&
			types.Implements(sourceType, r.interfaceContract)
	case DeclarationRequirementAnonymousInterface:
		return r.validGeneratedDefinition(
			GeneratedArtifactAnonymousInterface,
		)
	case DeclarationRequirementInterfaceMethodToken:
		return r.validGeneratedDefinition(
			GeneratedArtifactInterfaceMethodToken,
		)
	case DeclarationRequirementInterfaceMethodCallable:
		return r.validGeneratedDefinition(
			GeneratedArtifactInterfaceMethodCallable,
		)
	case DeclarationRequirementInterfaceDynamicTypeToken:
		return r.validGeneratedDefinition(
			GeneratedArtifactInterfaceDynamicTypeToken,
		)
	case DeclarationRequirementGenericCapability:
		return r.operation == NamedStructOperationInvalid &&
			r.typeName == nil &&
			r.variable == nil &&
			r.constant == nil &&
			r.projection == types.Invalid &&
			r.generated.Valid() &&
			r.generated.Kind() == GeneratedArtifactGenericCapability &&
			r.anonymousDemand == AnonymousStructDemandInvalid &&
			r.mapDemand == MapSpecializationDemandInvalid &&
			r.genericOperation == nil &&
			r.owner == r.generated.ReconstructionOwner()
	case DeclarationRequirementCallableControl:
		if !r.owner.Valid() ||
			r.operation != NamedStructOperationInvalid ||
			r.typeName != nil ||
			r.variable != nil ||
			r.constant != nil ||
			r.projection != types.Invalid ||
			r.generated != nil ||
			r.anonymousDemand != AnonymousStructDemandInvalid ||
			r.mapDemand != MapSpecializationDemandInvalid ||
			r.genericOperation != nil ||
			!validCallableControlOwner(r.owner, r.enclosing, r.callable) ||
			!r.control.Valid() {
			return false
		}
		if r.control == CallableControlGoto {
			return r.controlLabel != nil &&
				r.controlPosition.IsValid() &&
				r.controlRange == nil &&
				r.callable != nil &&
				r.controlPosition >= r.callable.Pos() &&
				r.controlPosition <= r.callable.End()
		}
		if r.control == CallableControlIteratorReturn {
			return r.controlLabel == nil &&
				!r.controlPosition.IsValid() &&
				validIteratorReturnRange(r.callable, r.controlRange)
		}
		return r.controlLabel == nil &&
			!r.controlPosition.IsValid() &&
			r.controlRange == nil
	case DeclarationRequirementCooperativeCallable:
		return r.validCooperativeCallable()
	case DeclarationRequirementCallableABI:
		return r.validGeneratedDefinition(
			GeneratedArtifactCallableABI,
		)
	case DeclarationRequirementPointerRepresentation:
		return r.validGeneratedDefinition(
			GeneratedArtifactPointerRepresentation,
		)
	case DeclarationRequirementEnvironmentBuiltin:
		if !r.owner.Valid() ||
			r.operation != NamedStructOperationInvalid ||
			r.typeName != nil ||
			r.variable != nil ||
			r.constant != nil ||
			r.projection != types.Invalid ||
			r.generated != nil ||
			r.anonymousDemand != AnonymousStructDemandInvalid ||
			r.mapDemand != MapSpecializationDemandInvalid ||
			r.genericOperation != nil ||
			r.environmentBuiltin == nil ||
			!validEnvironmentBuiltinSignature(r.environmentSignature) {
			return false
		}
		source, sourceOK := r.owner.Source()
		return sourceOK &&
			source == r.environmentBuiltin &&
			r.environmentBuiltin.Pkg() != nil &&
			r.environmentBuiltin.Parent() ==
				r.environmentBuiltin.Pkg().Scope() &&
			r.environmentBuiltin.Parent().Lookup(
				r.environmentBuiltin.Name(),
			) == r.environmentBuiltin
	default:
		return false
	}
}

func (r DeclarationRequirement) validGeneratedDefinition(
	kind GeneratedArtifactKind,
) bool {
	return r.operation == NamedStructOperationInvalid &&
		r.typeName == nil &&
		r.variable == nil &&
		r.constant == nil &&
		r.projection == types.Invalid &&
		r.generated.Valid() &&
		r.generated.Kind() == kind &&
		r.anonymousDemand == AnonymousStructDemandInvalid &&
		r.mapDemand == MapSpecializationDemandInvalid &&
		r.owner == r.generated.ReconstructionOwner()
}
