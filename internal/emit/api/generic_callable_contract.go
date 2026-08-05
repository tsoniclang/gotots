package api

import (
	"go/ast"
	"go/types"
	"slices"
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

type GenericConcretization struct {
	owner        *types.Func
	arguments    []types.Type
	signature    *types.Signature
	key          string
	suffix       string
	placement    GeneratedArtifactPlacement
	lexicalOwner ArtifactOwner
	anchor       *types.TypeName
}

func NewGenericConcretization(
	owner *types.Func,
	arguments []types.Type,
	signature *types.Signature,
	key string,
	suffix string,
	placement GeneratedArtifactPlacement,
	lexicalOwner ArtifactOwner,
	anchor *types.TypeName,
) (*GenericConcretization, error) {
	target := &GenericConcretization{
		owner:        owner,
		arguments:    slices.Clone(arguments),
		signature:    signature,
		key:          key,
		suffix:       suffix,
		placement:    placement,
		lexicalOwner: lexicalOwner,
		anchor:       anchor,
	}
	if !target.Valid() {
		return nil, &InvariantError{
			Role:   RoleCallTypeArgument,
			Reason: "generic concretization identity is invalid",
		}
	}
	return target, nil
}

func (c *GenericConcretization) Valid() bool {
	if c == nil || c.owner == nil || c.owner.Origin() != c.owner ||
		c.key == "" || c.suffix == "" || c.signature == nil {
		return false
	}
	source, ok := c.owner.Type().(*types.Signature)
	parameters := GenericDeclarationParameters(c.owner)
	if !ok || len(parameters) == 0 || len(parameters) != len(c.arguments) ||
		(source.Recv() == nil) != (c.signature.Recv() == nil) {
		return false
	}
	for _, argument := range c.arguments {
		if argument == nil || ContainsGenericTypeParameter(argument) {
			return false
		}
	}
	components := make([]*types.TypeName, 0)
	for _, argument := range c.arguments {
		components = append(
			components,
			localTypeComponents(argument)...,
		)
	}
	switch c.placement {
	case GeneratedArtifactPlacementCompilation:
		if len(components) != 0 || c.lexicalOwner.Valid() || c.anchor != nil {
			return false
		}
	case GeneratedArtifactPlacementLexical:
		if len(components) == 0 || !c.lexicalOwner.Valid() || c.anchor == nil ||
			c.lexicalOwner.Package() == nil ||
			c.anchor.Pkg() != c.lexicalOwner.Package() ||
			c.anchor.Parent() == nil ||
			c.anchor.Parent() == c.anchor.Pkg().Scope() {
			return false
		}
		anchored := false
		for _, component := range components {
			if component == c.anchor {
				anchored = true
				break
			}
		}
		if !anchored {
			return false
		}
	default:
		return false
	}
	instantiated, err := instantiateGenericCallable(c.owner, c.arguments)
	return err == nil && types.Identical(instantiated, c.signature)
}

func localTypeComponents(sourceType types.Type) []*types.TypeName {
	seen := make(map[*types.TypeName]struct{})
	var result []*types.TypeName
	collectLocalTypeComponents(sourceType, seen, &result)
	return result
}

func collectLocalTypeComponents(
	sourceType types.Type,
	seen map[*types.TypeName]struct{},
	result *[]*types.TypeName,
) {
	if sourceType == nil {
		return
	}
	sourceType = types.Unalias(sourceType)
	switch source := sourceType.(type) {
	case *types.Named:
		object := source.Obj()
		if object != nil && object.Pkg() != nil &&
			object.Parent() != object.Pkg().Scope() {
			if _, duplicate := seen[object]; !duplicate {
				seen[object] = struct{}{}
				*result = append(*result, object)
			}
		}
		for index := range source.TypeArgs().Len() {
			collectLocalTypeComponents(source.TypeArgs().At(index), seen, result)
		}
	case *types.Pointer:
		collectLocalTypeComponents(source.Elem(), seen, result)
	case *types.Slice:
		collectLocalTypeComponents(source.Elem(), seen, result)
	case *types.Array:
		collectLocalTypeComponents(source.Elem(), seen, result)
	case *types.Map:
		collectLocalTypeComponents(source.Key(), seen, result)
		collectLocalTypeComponents(source.Elem(), seen, result)
	case *types.Chan:
		collectLocalTypeComponents(source.Elem(), seen, result)
	case *types.Struct:
		for index := range source.NumFields() {
			collectLocalTypeComponents(source.Field(index).Type(), seen, result)
		}
	case *types.Signature:
		collectLocalTupleTypeComponents(source.Params(), seen, result)
		collectLocalTupleTypeComponents(source.Results(), seen, result)
	case *types.Interface:
		source = source.Complete()
		for index := range source.NumMethods() {
			collectLocalTypeComponents(source.Method(index).Type(), seen, result)
		}
	}
}

func collectLocalTupleTypeComponents(
	tuple *types.Tuple,
	seen map[*types.TypeName]struct{},
	result *[]*types.TypeName,
) {
	if tuple == nil {
		return
	}
	for index := range tuple.Len() {
		collectLocalTypeComponents(tuple.At(index).Type(), seen, result)
	}
}

func (c *GenericConcretization) Owner() *types.Func {
	if !c.Valid() {
		return nil
	}
	return c.owner
}

func (c *GenericConcretization) Arguments() []types.Type {
	if !c.Valid() {
		return nil
	}
	return slices.Clone(c.arguments)
}

func (c *GenericConcretization) Signature() *types.Signature {
	if !c.Valid() {
		return nil
	}
	return c.signature
}

func (c *GenericConcretization) Key() string {
	if !c.Valid() {
		return ""
	}
	return c.key
}

func (c *GenericConcretization) Suffix() string {
	if !c.Valid() {
		return ""
	}
	return c.suffix
}

func (c *GenericConcretization) Placement() GeneratedArtifactPlacement {
	if !c.Valid() {
		return GeneratedArtifactPlacementInvalid
	}
	return c.placement
}

func (c *GenericConcretization) LexicalOwner() ArtifactOwner {
	if !c.Valid() || c.placement != GeneratedArtifactPlacementLexical {
		return ArtifactOwner{}
	}
	return c.lexicalOwner
}

func (c *GenericConcretization) LexicalAnchor() *types.TypeName {
	if !c.Valid() || c.placement != GeneratedArtifactPlacementLexical {
		return nil
	}
	return c.anchor
}

func InstantiateGenericCallable(
	owner *types.Func,
	arguments TypeArgumentList,
) (*types.Signature, error) {
	return instantiateGenericCallable(owner, arguments.Values())
}

func instantiateGenericCallable(
	owner *types.Func,
	arguments []types.Type,
) (*types.Signature, error) {
	if owner == nil || owner.Origin() != owner {
		return nil, &InvariantError{
			Role:   RoleCallTypeArgument,
			Reason: "generic callable owner is invalid",
		}
	}
	source, ok := owner.Type().(*types.Signature)
	parameters := GenericDeclarationParameters(owner)
	if !ok || len(parameters) == 0 || len(parameters) != len(arguments) {
		return nil, &InvariantError{
			Role:   RoleCallTypeArgument,
			Reason: "generic callable arguments do not match its declaration",
		}
	}
	replacements := make(map[*types.TypeParam]types.Type, len(parameters))
	for index, parameter := range parameters {
		argument := arguments[index]
		if argument == nil || ContainsGenericTypeParameter(argument) {
			return nil, &InvariantError{
				Role:   RoleCallTypeArgument,
				Reason: "generic callable argument remains open",
			}
		}
		replacements[parameter] = argument
	}
	target, err := SubstituteType(source, replacements)
	if err != nil {
		return nil, err
	}
	signature, ok := target.(*types.Signature)
	if !ok {
		return nil, &InvariantError{
			Role:   RoleCallTypeArgument,
			Reason: "generic callable did not produce a signature",
		}
	}
	return signature, nil
}
