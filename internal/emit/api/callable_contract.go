package api

import (
	"go/ast"
	"go/types"
)

type CallableFacetKind uint8

const (
	CallableFacetInvalid            CallableFacetKind = 0
	CallableFacetSource             CallableFacetKind = 1
	CallableFacetFunctionLiteral    CallableFacetKind = 2
	CallableFacetABI                CallableFacetKind = 3
	CallableFacetGenericCapability  CallableFacetKind = 4
	CallableFacetGenericOperation   CallableFacetKind = 5
	CallableFacetPackageInitializer CallableFacetKind = 6
	CallableFacetInterfaceMethod    CallableFacetKind = 8
)

func (k CallableFacetKind) Valid() bool {
	switch k {
	case CallableFacetSource,
		CallableFacetFunctionLiteral,
		CallableFacetABI,
		CallableFacetGenericCapability,
		CallableFacetGenericOperation,
		CallableFacetPackageInitializer,
		CallableFacetInterfaceMethod:
		return true
	default:
		return false
	}
}

type CallableFacet struct {
	owner     ArtifactOwner
	kind      CallableFacetKind
	function  *types.Func
	literal   *ast.FuncLit
	generated *GeneratedArtifact
	operation *GenericOperationContract
}

func NewSourceCallableFacet(function *types.Func) (CallableFacet, error) {
	if function == nil {
		return CallableFacet{}, &RootRequestError{
			Reason: "source callable facet owner is nil",
		}
	}
	function = function.Origin()
	signature, ok := function.Type().(*types.Signature)
	if !ok || signature == nil {
		return CallableFacet{}, &RootRequestError{
			Reason: "source callable facet owner has no signature",
		}
	}
	return CallableFacet{
		owner:    MustSourceArtifactOwner(function),
		kind:     CallableFacetSource,
		function: function,
	}, nil
}

func (c Context) FunctionLiteralCallableFacet(
	literal *ast.FuncLit,
) (CallableFacet, error) {
	owner := c.artifactOwner
	_, sourceOwned := owner.Source()
	_, _, initializerOwned := owner.PackageInitializer()
	if (!sourceOwned && !initializerOwned) ||
		literal == nil ||
		literal.Type == nil ||
		literal.Body == nil {
		return CallableFacet{}, &RootRequestError{
			Reason: "function-literal callable facet is invalid",
		}
	}
	return CallableFacet{
		owner:   owner,
		kind:    CallableFacetFunctionLiteral,
		literal: literal,
	}, nil
}

func NewPackageInitializerCallableFacet(
	owner ArtifactOwner,
) (CallableFacet, error) {
	if _, _, ok := owner.PackageInitializer(); !ok {
		return CallableFacet{}, &RootRequestError{
			Reason: "package-initializer callable facet is invalid",
		}
	}
	return CallableFacet{
		owner: owner,
		kind:  CallableFacetPackageInitializer,
	}, nil
}

func NewCallableABIFacet(
	artifact *GeneratedArtifact,
) (CallableFacet, error) {
	if artifact == nil ||
		artifact.Kind() != GeneratedArtifactCallableABI ||
		!artifact.Valid() {
		return CallableFacet{}, &RootRequestError{
			Reason: "callable ABI facet is invalid",
		}
	}
	return CallableFacet{
		owner:     MustGeneratedArtifactOwner(artifact),
		kind:      CallableFacetABI,
		generated: artifact,
	}, nil
}

func NewInterfaceMethodCallableFacet(
	artifact *GeneratedArtifact,
) (CallableFacet, error) {
	if artifact == nil ||
		artifact.Kind() != GeneratedArtifactInterfaceMethodCallable ||
		!artifact.Valid() {
		return CallableFacet{}, &RootRequestError{
			Reason: "interface-method callable facet is invalid",
		}
	}
	return CallableFacet{
		owner:     MustGeneratedArtifactOwner(artifact),
		kind:      CallableFacetInterfaceMethod,
		generated: artifact,
	}, nil
}

func NewGenericCapabilityCallableFacet(
	artifact *GeneratedArtifact,
) (CallableFacet, error) {
	if artifact == nil ||
		artifact.Kind() != GeneratedArtifactGenericCapability ||
		!artifact.Valid() {
		return CallableFacet{}, &RootRequestError{
			Reason: "generic-capability callable facet is invalid",
		}
	}
	return CallableFacet{
		owner:     artifact.ReconstructionOwner(),
		kind:      CallableFacetGenericCapability,
		generated: artifact,
	}, nil
}

func NewGenericOperationCallableFacet(
	operation *GenericOperationContract,
) (CallableFacet, error) {
	function, functionOwned := operationOwnerFunction(operation)
	if !operation.Valid() ||
		!functionOwned ||
		operation.Consumer() != GenericFunctionOperationConsumer() {
		return CallableFacet{}, &RootRequestError{
			Reason: "generic-operation callable facet is invalid",
		}
	}
	return CallableFacet{
		owner:     MustSourceArtifactOwner(function),
		kind:      CallableFacetGenericOperation,
		operation: operation,
	}, nil
}

func (f CallableFacet) Valid() bool {
	if !f.owner.Valid() || !f.kind.Valid() {
		return false
	}
	switch f.kind {
	case CallableFacetSource:
		source, sourceOwned := f.owner.Source()
		function, callable := source.(*types.Func)
		signature, signatureOK := functionType(function)
		return sourceOwned &&
			callable &&
			signatureOK &&
			function.Origin() == function &&
			f.function == function &&
			f.literal == nil &&
			f.generated == nil &&
			f.operation == nil &&
			signature != nil
	case CallableFacetFunctionLiteral:
		_, sourceOwned := f.owner.Source()
		_, _, initializerOwned := f.owner.PackageInitializer()
		return (sourceOwned || initializerOwned) &&
			f.function == nil &&
			f.literal != nil &&
			f.literal.Type != nil &&
			f.literal.Body != nil &&
			f.generated == nil &&
			f.operation == nil
	case CallableFacetABI:
		generated, generatedOwned := f.owner.Generated()
		return generatedOwned &&
			generated == f.generated &&
			f.function == nil &&
			f.literal == nil &&
			f.generated != nil &&
			f.generated.Kind() == GeneratedArtifactCallableABI &&
			f.generated.Valid() &&
			f.operation == nil
	case CallableFacetGenericCapability:
		return f.generated != nil &&
			f.owner == f.generated.ReconstructionOwner() &&
			f.function == nil &&
			f.literal == nil &&
			f.generated.Kind() == GeneratedArtifactGenericCapability &&
			f.generated.Valid() &&
			f.operation == nil
	case CallableFacetGenericOperation:
		source, sourceOwned := f.owner.Source()
		function, functionOwned := operationOwnerFunction(f.operation)
		return sourceOwned &&
			functionOwned &&
			source == function &&
			f.operation.Valid() &&
			f.function == nil &&
			f.literal == nil &&
			f.generated == nil &&
			f.operation.Consumer() ==
				GenericFunctionOperationConsumer()
	case CallableFacetPackageInitializer:
		_, _, initializerOwned := f.owner.PackageInitializer()
		return initializerOwned &&
			f.function == nil &&
			f.literal == nil &&
			f.generated == nil &&
			f.operation == nil
	case CallableFacetInterfaceMethod:
		generated, generatedOwned := f.owner.Generated()
		return generatedOwned &&
			f.function == nil &&
			f.literal == nil &&
			f.generated == generated &&
			f.generated.Kind() ==
				GeneratedArtifactInterfaceMethodCallable &&
			f.generated.Valid() &&
			f.operation == nil
	default:
		return false
	}
}

func (f CallableFacet) empty() bool {
	return !f.owner.Valid() &&
		f.kind == CallableFacetInvalid &&
		f.function == nil &&
		f.literal == nil &&
		f.generated == nil &&
		f.operation == nil
}

func (f CallableFacet) Owner() ArtifactOwner {
	return f.owner
}

func (f CallableFacet) Kind() CallableFacetKind {
	return f.kind
}

func (f CallableFacet) SourceFunction() (*types.Func, bool) {
	return f.function, f.Valid() && f.kind == CallableFacetSource
}

func (f CallableFacet) FunctionLiteral() (*ast.FuncLit, bool) {
	return f.literal, f.Valid() && f.kind == CallableFacetFunctionLiteral
}

func (f CallableFacet) ABI() (*GeneratedArtifact, bool) {
	return f.generated, f.Valid() && f.kind == CallableFacetABI
}

func (f CallableFacet) InterfaceMethod() (*GeneratedArtifact, bool) {
	return f.generated,
		f.Valid() && f.kind == CallableFacetInterfaceMethod
}

func (f CallableFacet) GenericCapability() (*GeneratedArtifact, bool) {
	return f.generated,
		f.Valid() && f.kind == CallableFacetGenericCapability
}

func (f CallableFacet) GenericOperation() (
	*GenericOperationContract,
	bool,
) {
	return f.operation,
		f.Valid() && f.kind == CallableFacetGenericOperation
}

func (f CallableFacet) PackageInitializer() (ArtifactOwner, bool) {
	return f.owner,
		f.Valid() && f.kind == CallableFacetPackageInitializer
}

func functionType(function *types.Func) (*types.Signature, bool) {
	if function == nil {
		return nil, false
	}
	signature, ok := function.Type().(*types.Signature)
	return signature, ok
}

func operationOwnerFunction(
	operation *GenericOperationContract,
) (*types.Func, bool) {
	if operation == nil {
		return nil, false
	}
	function, ok := operation.Owner().(*types.Func)
	return function, ok && function != nil && function.Origin() == function
}
