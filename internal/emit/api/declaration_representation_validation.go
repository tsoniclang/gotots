package api

import (
	"go/types"
)

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
	if r.kind != DeclarationRequirementTypeRepresentation &&
		r.typeRepresentation != TypeRepresentationInvalid {
		return false
	}
	if r.kind != DeclarationRequirementPointerRepresentation &&
		r.pointerCarrier {
		return false
	}
	if r.kind != DeclarationRequirementGenericConcretization &&
		r.concretizationDeferred {
		return false
	}
	if r.kind != DeclarationRequirementCooperativeCallable &&
		!r.callableFacet.empty() {
		return false
	}
	if r.kind != DeclarationRequirementInterfaceAdapter &&
		r.kind != DeclarationRequirementProviderInterfaceCapability &&
		(r.interfaceContract != nil || r.interfaceContractKey != "") {
		return false
	}
	if r.kind != DeclarationRequirementProviderProfileInterfaceCapability &&
		r.providerProfileTarget != nil {
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
	case DeclarationRequirementGenericConcretization:
		if r.operation != NamedStructOperationInvalid ||
			r.typeName != nil ||
			r.variable != nil ||
			r.constant != nil ||
			r.projection != types.Invalid ||
			r.generated == nil ||
			r.anonymousDemand != AnonymousStructDemandInvalid ||
			r.mapDemand != MapSpecializationDemandInvalid ||
			r.genericOperation != nil {
			return false
		}
		concretization, ok := r.generated.GenericConcretization()
		return ok && concretization.Valid() &&
			r.owner == r.generated.ReconstructionOwner()
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
	case DeclarationRequirementTypeRepresentation:
		return r.validTypeRepresentation()
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
	case DeclarationRequirementReflectionType,
		DeclarationRequirementReflectionValueOperations:
		return r.validGeneratedDefinition(GeneratedArtifactReflectionType)
	case DeclarationRequirementUnsafeCodec:
		return r.validGeneratedDefinition(GeneratedArtifactUnsafeCodec)
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
	case DeclarationRequirementProviderInterfaceBridge:
		return r.validGeneratedDefinition(
			GeneratedArtifactProviderInterfaceBridge,
		)
	case DeclarationRequirementProviderInterfaceCapability:
		return r.operation == NamedStructOperationInvalid &&
			r.typeName == nil &&
			r.variable == nil &&
			r.constant == nil &&
			r.projection == types.Invalid &&
			r.generated.Valid() &&
			r.generated.Kind() == GeneratedArtifactProviderInterfaceBridge &&
			r.interfaceContract != nil &&
			r.interfaceContract.Complete().IsMethodSet() &&
			r.interfaceContractKey != "" &&
			r.anonymousDemand == AnonymousStructDemandInvalid &&
			r.mapDemand == MapSpecializationDemandInvalid &&
			r.owner == r.generated.ReconstructionOwner()
	case DeclarationRequirementProviderProfileInterfaceCapability:
		if r.operation != NamedStructOperationInvalid ||
			r.typeName != nil ||
			r.variable != nil ||
			r.constant != nil ||
			r.projection != types.Invalid ||
			!r.generated.Valid() ||
			r.generated.Kind() != GeneratedArtifactProviderInterfaceBridge ||
			r.interfaceContract != nil ||
			r.providerProfileTarget == nil ||
			!r.providerProfileTarget.Valid() ||
			r.providerProfileTarget.Kind() != GeneratedArtifactProviderInterfaceBridge ||
			r.interfaceContractKey != "" ||
			r.anonymousDemand != AnonymousStructDemandInvalid ||
			r.mapDemand != MapSpecializationDemandInvalid ||
			r.owner != r.generated.ReconstructionOwner() {
			return false
		}
		base, profile, profiled := r.generated.ProviderProfileInterfaceBridge()
		target, targetProfile, targetProfiled :=
			r.providerProfileTarget.ProviderProfileInterfaceBridge()
		if !profiled || len(profile) == 0 ||
			!targetProfiled || len(targetProfile) == 0 {
			return false
		}
		baseContract, baseInterface := base.Underlying().(*types.Interface)
		targetContract, targetInterface := target.Underlying().(*types.Interface)
		return baseInterface && targetInterface &&
			types.Implements(target, baseContract.Complete()) &&
			!types.Implements(base, targetContract.Complete())
	case DeclarationRequirementProviderStatefulRepresentation:
		return r.validGeneratedDefinition(
			GeneratedArtifactProviderStatefulRepresentation,
		)
	case DeclarationRequirementDeferredCallableRegistry:
		return r.validGeneratedDefinition(
			GeneratedArtifactDeferredCallableRegistry,
		)
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

type TypeRepresentationFacet uint8

const (
	TypeRepresentationInvalid          TypeRepresentationFacet = 0
	TypeRepresentationStorage          TypeRepresentationFacet = 1
	TypeRepresentationContainerStorage TypeRepresentationFacet = 2
	TypeRepresentationPointer          TypeRepresentationFacet = 3
)

func (f TypeRepresentationFacet) Valid() bool {
	return f == TypeRepresentationStorage ||
		f == TypeRepresentationContainerStorage ||
		f == TypeRepresentationPointer
}

func (f TypeRepresentationFacet) String() string {
	switch f {
	case TypeRepresentationStorage:
		return "storage"
	case TypeRepresentationContainerStorage:
		return "container-storage"
	case TypeRepresentationPointer:
		return "pointer"
	default:
		return "invalid"
	}
}

func SupportsTypeRepresentation(typeName *types.TypeName) bool {
	if typeName == nil || typeName.IsAlias() {
		return false
	}
	named, ok := types.Unalias(typeName.Type()).(*types.Named)
	if !ok || named.Obj() != typeName || named.Origin() == nil {
		return false
	}
	switch named.Underlying().(type) {
	case *types.Basic,
		*types.Array,
		*types.Slice,
		*types.Pointer,
		*types.Signature,
		*types.Map,
		*types.Chan,
		*types.Struct:
		return true
	default:
		return false
	}
}

func NewTypeRepresentationRequirement(
	typeName *types.TypeName,
	facet TypeRepresentationFacet,
) (DeclarationRequirement, error) {
	if !SupportsTypeRepresentation(typeName) {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "type-representation owner is unsupported",
		}
	}
	return newTypeRepresentationRequirement(
		MustSourceArtifactOwner(typeName),
		typeName,
		nil,
		facet,
	)
}

func NewLexicalTypeRepresentationRequirement(
	owner ArtifactOwner,
	typeName *types.TypeName,
	facet TypeRepresentationFacet,
) (DeclarationRequirement, error) {
	if !SupportsTypeRepresentation(typeName) ||
		!validLexicalNamedStructOwner(owner, typeName) {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "lexical type-representation owner is invalid",
		}
	}
	return newTypeRepresentationRequirement(
		owner,
		typeName,
		nil,
		facet,
	)
}

func NewGeneratedTypeRepresentationRequirement(
	artifact *GeneratedArtifact,
	facet TypeRepresentationFacet,
) (DeclarationRequirement, error) {
	if artifact == nil ||
		artifact.Kind() != GeneratedArtifactAnonymousStruct {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "generated type-representation owner is invalid",
		}
	}
	return newTypeRepresentationRequirement(
		artifact.ReconstructionOwner(),
		nil,
		artifact,
		facet,
	)
}

func newTypeRepresentationRequirement(
	owner ArtifactOwner,
	typeName *types.TypeName,
	artifact *GeneratedArtifact,
	facet TypeRepresentationFacet,
) (DeclarationRequirement, error) {
	requirement := DeclarationRequirement{
		owner:              owner,
		kind:               DeclarationRequirementTypeRepresentation,
		typeName:           typeName,
		generated:          artifact,
		typeRepresentation: facet,
	}
	if !requirement.validTypeRepresentation() {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "type-representation requirement is invalid",
		}
	}
	return requirement, nil
}

func NewTypeRepresentationRequest(
	typeName *types.TypeName,
	facet TypeRepresentationFacet,
) (RootRequest, error) {
	requirement, err := NewTypeRepresentationRequirement(typeName, facet)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func NewLexicalTypeRepresentationRequest(
	owner ArtifactOwner,
	typeName *types.TypeName,
	facet TypeRepresentationFacet,
) (RootRequest, error) {
	requirement, err := NewLexicalTypeRepresentationRequirement(
		owner,
		typeName,
		facet,
	)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func NewGeneratedTypeRepresentationRequest(
	artifact *GeneratedArtifact,
	facet TypeRepresentationFacet,
) (RootRequest, error) {
	requirement, err := NewGeneratedTypeRepresentationRequirement(
		artifact,
		facet,
	)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func (r DeclarationRequirement) TypeRepresentation() (
	*types.TypeName,
	*GeneratedArtifact,
	TypeRepresentationFacet,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementTypeRepresentation {
		return nil, nil, TypeRepresentationInvalid, false
	}
	return r.typeName, r.generated, r.typeRepresentation, true
}

func (r DeclarationRequirement) validTypeRepresentation() bool {
	if !r.owner.Valid() ||
		!r.typeRepresentation.Valid() ||
		r.operation != NamedStructOperationInvalid ||
		r.variable != nil ||
		r.constant != nil ||
		r.projection != types.Invalid ||
		r.anonymousDemand != AnonymousStructDemandInvalid ||
		r.mapDemand != MapSpecializationDemandInvalid ||
		r.genericOperation != nil ||
		r.genericParameter != nil ||
		r.genericFacet != GenericRepresentationInvalid {
		return false
	}
	if r.typeName != nil {
		if r.generated != nil || r.typeName.IsAlias() {
			return false
		}
		source, sourceOK := r.owner.Source()
		if sourceOK && source == r.typeName {
			return true
		}
		return validLexicalNamedStructOwner(r.owner, r.typeName)
	}
	return r.generated != nil &&
		r.generated.Kind() == GeneratedArtifactAnonymousStruct &&
		r.owner == r.generated.ReconstructionOwner()
}
