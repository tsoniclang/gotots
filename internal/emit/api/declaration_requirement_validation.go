package api

import "go/types"

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
	case DeclarationRequirementReflectionType:
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
