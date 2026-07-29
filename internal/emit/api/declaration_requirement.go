package api

import (
	"go/ast"
	"go/token"
	"go/types"
)

type DeclarationRequirement struct {
	owner     ArtifactOwner
	kind      DeclarationRequirementKind
	operation NamedStructOperation
	typeName  *types.TypeName
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
	projection       types.BasicKind
	generated        *GeneratedArtifact
	anonymousDemand  AnonymousStructDemand
	mapDemand        MapSpecializationDemand
	genericOperation *GenericOperationContract
	enclosing        ast.Node
	callable         ast.Node
	control          CallableControlFacet
	controlLabel     *types.Label
	controlPosition  token.Pos
	callableFacet    CallableFacet
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
			r.controlPosition.IsValid()) {
		return false
	}
	if r.kind != DeclarationRequirementGenericOperation &&
		r.kind != DeclarationRequirementGenericCapability &&
		r.genericOperation != nil {
		return false
	}
	if r.kind != DeclarationRequirementCooperativeCallable &&
		!r.callableFacet.empty() {
		return false
	}
	switch r.kind {
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
		source, sourceOK := r.owner.Source()
		_, ok := source.(*types.Func)
		return sourceOK && ok
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
		return r.validGeneratedDefinition(
			GeneratedArtifactInterfaceAdapter,
		)
	case DeclarationRequirementAnonymousInterface:
		return r.validGeneratedDefinition(
			GeneratedArtifactAnonymousInterface,
		)
	case DeclarationRequirementInterfaceMethodToken:
		return r.validGeneratedDefinition(
			GeneratedArtifactInterfaceMethodToken,
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
				r.callable != nil &&
				r.controlPosition >= r.callable.Pos() &&
				r.controlPosition <= r.callable.End()
		}
		return r.controlLabel == nil && !r.controlPosition.IsValid()
	case DeclarationRequirementCooperativeCallable:
		return r.validCooperativeCallable()
	case DeclarationRequirementCallableABI:
		return r.validGeneratedDefinition(
			GeneratedArtifactCallableABI,
		)
	default:
		return false
	}
}

func NewDirectCallableControlRequirement(
	owner *types.Func,
	control CallableControlFacet,
) (DeclarationRequirement, error) {
	if owner == nil ||
		owner.Origin() != owner ||
		!control.Valid() ||
		control == CallableControlGoto {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "direct callable-control requirement is invalid",
		}
	}
	return DeclarationRequirement{
		owner:   MustSourceArtifactOwner(owner),
		kind:    DeclarationRequirementCallableControl,
		control: control,
	}, nil
}

func NewCallableControlRequirement(
	owner ArtifactOwner,
	enclosing ast.Node,
	callable ast.Node,
	control CallableControlFacet,
) (DeclarationRequirement, error) {
	if !validCallableControlAnchor(owner, enclosing, callable) ||
		!control.Valid() ||
		control == CallableControlGoto {
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

func NewGotoControlRequirement(
	owner ArtifactOwner,
	enclosing ast.Node,
	callable ast.Node,
	label *types.Label,
	position token.Pos,
) (DeclarationRequirement, error) {
	if !validCallableControlAnchor(owner, enclosing, callable) ||
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

func validCallableControlAnchor(
	owner ArtifactOwner,
	enclosing ast.Node,
	callable ast.Node,
) bool {
	if !owner.Valid() ||
		enclosing == nil ||
		callable == nil ||
		callable.Pos() < enclosing.Pos() ||
		callable.End() > enclosing.End() {
		return false
	}
	switch callable := callable.(type) {
	case *ast.FuncDecl:
		source, ok := owner.Source()
		function, functionOK := source.(*types.Func)
		return ok &&
			functionOK &&
			enclosing == callable &&
			callable.Type != nil &&
			callable.Body != nil &&
			function.Pos() >= callable.Pos() &&
			function.Pos() <= callable.End()
	case *ast.FuncLit:
		if callable.Type == nil || callable.Body == nil {
			return false
		}
		if source, ok := owner.Source(); ok {
			function, functionOK := source.(*types.Func)
			return functionOK &&
				function.Pos() >= enclosing.Pos() &&
				function.Pos() <= enclosing.End()
		}
		_, initializer, ok := owner.PackageInitializer()
		return ok &&
			initializer.Rhs != nil &&
			enclosing == initializer.Rhs
	default:
		return false
	}
}

func validCallableControlOwner(
	owner ArtifactOwner,
	enclosing ast.Node,
	callable ast.Node,
) bool {
	if enclosing != nil || callable != nil {
		return validCallableControlAnchor(owner, enclosing, callable)
	}
	source, ok := owner.Source()
	function, functionOK := source.(*types.Func)
	return ok && functionOK && function.Origin() == function
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
