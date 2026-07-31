package api

import (
	"go/ast"
	"go/types"
	"slices"
	"sort"
	"strings"
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
	CallableFacetGenericProfile     CallableFacetKind = 7
	CallableFacetInterfaceMethod    CallableFacetKind = 8
)

func (k CallableFacetKind) Valid() bool {
	return k >= CallableFacetSource &&
		k <= CallableFacetInterfaceMethod
}

type CallableFacet struct {
	owner     ArtifactOwner
	kind      CallableFacetKind
	function  *types.Func
	literal   *ast.FuncLit
	generated *GeneratedArtifact
	operation *GenericOperationContract
	profile   *GenericCallableProfile
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
	profile := c.genericCallableProfile
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
	if profile != nil {
		source, sourceProfile := owner.Source()
		if !sourceProfile ||
			!profile.Valid() ||
			source != profile.Owner() {
			return CallableFacet{}, &RootRequestError{
				Reason: "function-literal callable profile is invalid",
			}
		}
	}
	return CallableFacet{
		owner:   owner,
		kind:    CallableFacetFunctionLiteral,
		literal: literal,
		profile: profile,
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

func NewGenericCallableProfileFacet(
	profile *GenericCallableProfile,
) (CallableFacet, error) {
	if !profile.Valid() {
		return CallableFacet{}, &RootRequestError{
			Reason: "generic callable profile facet is invalid",
		}
	}
	return CallableFacet{
		owner:   MustSourceArtifactOwner(profile.Owner()),
		kind:    CallableFacetGenericProfile,
		profile: profile,
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

func NewGenericProfileCallableABIFacet(
	profile *GenericCallableProfile,
	artifact *GeneratedArtifact,
) (CallableFacet, error) {
	if !profile.Valid() ||
		artifact == nil ||
		artifact.Kind() != GeneratedArtifactCallableABI ||
		!artifact.Valid() {
		return CallableFacet{}, &RootRequestError{
			Reason: "generic-profile callable ABI facet is invalid",
		}
	}
	return CallableFacet{
		owner:     MustSourceArtifactOwner(profile.Owner()),
		kind:      CallableFacetABI,
		generated: artifact,
		profile:   profile,
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
			f.profile == nil &&
			signature != nil
	case CallableFacetFunctionLiteral:
		source, sourceOwned := f.owner.Source()
		_, _, initializerOwned := f.owner.PackageInitializer()
		profileValid := f.profile == nil
		if f.profile != nil {
			profileValid = sourceOwned &&
				f.profile.Valid() &&
				source == f.profile.Owner()
		}
		return (sourceOwned || initializerOwned) &&
			f.function == nil &&
			f.literal != nil &&
			f.literal.Type != nil &&
			f.literal.Body != nil &&
			f.generated == nil &&
			f.operation == nil &&
			profileValid
	case CallableFacetABI:
		generated, generatedOwned := f.owner.Generated()
		source, sourceOwned := f.owner.Source()
		global := f.profile == nil &&
			generatedOwned &&
			generated == f.generated
		profiled := f.profile != nil &&
			sourceOwned &&
			f.profile.Valid() &&
			source == f.profile.Owner()
		return (global || profiled) &&
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
			f.operation == nil &&
			f.profile == nil
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
			f.profile == nil &&
			f.operation.Consumer() ==
				GenericFunctionOperationConsumer()
	case CallableFacetPackageInitializer:
		_, _, initializerOwned := f.owner.PackageInitializer()
		return initializerOwned &&
			f.function == nil &&
			f.literal == nil &&
			f.generated == nil &&
			f.operation == nil &&
			f.profile == nil
	case CallableFacetGenericProfile:
		source, sourceOwned := f.owner.Source()
		return sourceOwned &&
			f.profile.Valid() &&
			source == f.profile.Owner() &&
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
			f.operation == nil &&
			f.profile == nil
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
		f.operation == nil &&
		f.profile == nil
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

func (f CallableFacet) FunctionLiteralProfile() (
	*GenericCallableProfile,
	bool,
) {
	return f.profile,
		f.Valid() &&
			f.kind == CallableFacetFunctionLiteral &&
			f.profile != nil
}

func (f CallableFacet) ABI() (*GeneratedArtifact, bool) {
	return f.generated, f.Valid() && f.kind == CallableFacetABI
}

func (f CallableFacet) GenericProfileABI() (
	*GenericCallableProfile,
	*GeneratedArtifact,
	bool,
) {
	return f.profile,
		f.generated,
		f.Valid() &&
			f.kind == CallableFacetABI &&
			f.profile != nil
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

func (f CallableFacet) GenericProfile() (
	*GenericCallableProfile,
	bool,
) {
	return f.profile,
		f.Valid() && f.kind == CallableFacetGenericProfile
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

type GenericCallableABISelection struct {
	artifact    *GeneratedArtifact
	cooperative bool
}

func NewGenericCallableABISelection(
	artifact *GeneratedArtifact,
	cooperative bool,
) (GenericCallableABISelection, error) {
	if artifact == nil ||
		!artifact.Valid() ||
		artifact.Kind() != GeneratedArtifactCallableABI {
		return GenericCallableABISelection{}, &InvariantError{
			Role:   RoleCallArgument,
			Reason: "generic callable ABI selection is invalid",
		}
	}
	return GenericCallableABISelection{
		artifact:    artifact,
		cooperative: cooperative,
	}, nil
}

func (s GenericCallableABISelection) Artifact() *GeneratedArtifact {
	return s.artifact
}

func (s GenericCallableABISelection) Cooperative() bool {
	return s.cooperative
}

func (s GenericCallableABISelection) valid() bool {
	return s.artifact != nil &&
		s.artifact.Valid() &&
		s.artifact.Kind() == GeneratedArtifactCallableABI
}

type GenericCallableProfileSelection struct {
	abis        []GenericCallableABISelection
	key         string
	cooperative bool
}

func NewGenericCallableProfileSelection(
	selections []GenericCallableABISelection,
) (GenericCallableProfileSelection, error) {
	merged := make(map[*GeneratedArtifact]bool, len(selections))
	for _, selection := range selections {
		if !selection.valid() {
			return GenericCallableProfileSelection{}, &InvariantError{
				Role:   RoleCallArgument,
				Reason: "generic callable profile selection is invalid",
			}
		}
		merged[selection.artifact] =
			merged[selection.artifact] || selection.cooperative
	}
	canonical := make(
		[]GenericCallableABISelection,
		0,
		len(merged),
	)
	for artifact, cooperative := range merged {
		canonical = append(canonical, GenericCallableABISelection{
			artifact:    artifact,
			cooperative: cooperative,
		})
	}
	sort.Slice(canonical, func(left, right int) bool {
		return canonical[left].artifact.ArtifactKey() <
			canonical[right].artifact.ArtifactKey()
	})
	var key strings.Builder
	for _, selection := range canonical {
		if key.Len() != 0 {
			key.WriteByte('|')
		}
		key.WriteString(selection.artifact.ArtifactKey())
		if selection.cooperative {
			key.WriteString("=cooperative")
		} else {
			key.WriteString("=synchronous")
		}
	}
	if key.Len() == 0 {
		key.WriteString("synchronous")
	}
	profile := GenericCallableProfileSelection{
		abis: slices.Clone(canonical),
		key:  key.String(),
	}
	for _, selection := range canonical {
		profile.cooperative =
			profile.cooperative || selection.cooperative
	}
	return profile, nil
}

func (s GenericCallableProfileSelection) Valid() bool {
	if s.key == "" {
		return false
	}
	previous := ""
	cooperative := false
	for _, selection := range s.abis {
		if !selection.valid() ||
			previous >= selection.artifact.ArtifactKey() {
			return false
		}
		previous = selection.artifact.ArtifactKey()
		cooperative = cooperative || selection.cooperative
	}
	return cooperative == s.cooperative
}

func (s GenericCallableProfileSelection) ABIs() []GenericCallableABISelection {
	return slices.Clone(s.abis)
}

func (s GenericCallableProfileSelection) Key() string {
	return s.key
}

func (s GenericCallableProfileSelection) Cooperative() bool {
	return s.cooperative
}

func (s GenericCallableProfileSelection) ABI(
	artifact *GeneratedArtifact,
) (bool, bool) {
	if artifact == nil {
		return false, false
	}
	index, found := slices.BinarySearchFunc(
		s.abis,
		artifact.ArtifactKey(),
		func(
			selection GenericCallableABISelection,
			key string,
		) int {
			return strings.Compare(selection.artifact.ArtifactKey(), key)
		},
	)
	if !found || s.abis[index].artifact != artifact {
		return false, false
	}
	return s.abis[index].cooperative, true
}
