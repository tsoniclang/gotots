package api

import (
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type GenericConcretization struct {
	owner        *types.Func
	arguments    []types.Type
	signature    *types.Signature
	key          string
	suffix       string
	placement    GeneratedArtifactPlacement
	lexicalOwner ArtifactOwner
	anchor       *types.TypeName
	profile      *GenericCallableProfile
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
	profile *GenericCallableProfile,
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
		profile:      profile,
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
	return err == nil && types.Identical(instantiated, c.signature) &&
		(c.profile == nil ||
			(c.profile.Valid() && c.profile.Owner() == c.owner))
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

func (c *GenericConcretization) Profile() (*GenericCallableProfile, bool) {
	if !c.Valid() || c.profile == nil {
		return nil, false
	}
	return c.profile, true
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

type GenericConcretizationReference struct {
	concretization *GenericConcretization
	name           string
	requests       []RootRequest
}

type GenericConcretizationNames interface {
	GenericConcretizationPlacement(
		*types.Func,
		TypeArgumentList,
	) (GeneratedArtifactPlacement, ArtifactOwner, *types.TypeName, error)
	GenericConcretization(
		*GenericConcretization,
	) (GenericConcretizationReference, error)
	DeferredGenericConcretization(
		*GenericConcretization,
	) (NameReference, error)
}

type GenericKernelNames interface {
	GenericKernel(
		*types.Func,
		*GenericCallableProfile,
	) (NameReference, error)
	DeferredGenericCallable(
		*types.Func,
		*GenericCallableProfile,
	) (DeferredGenericCallableReference, error)
	DeferredGenericKernel(
		*types.Func,
		*GenericCallableProfile,
	) (DeferredGenericCallableReference, error)
}

type DeferredGenericRecoveryPlacement uint8

const (
	DeferredGenericRecoveryInvalid DeferredGenericRecoveryPlacement = iota
	DeferredGenericRecoveryOmitted
	DeferredGenericRecoveryFirst
	DeferredGenericRecoveryLast
)

func (p DeferredGenericRecoveryPlacement) Valid() bool {
	return p == DeferredGenericRecoveryOmitted ||
		p == DeferredGenericRecoveryFirst ||
		p == DeferredGenericRecoveryLast
}

type DeferredGenericCallableReference struct {
	reference NameReference
	recovery  DeferredGenericRecoveryPlacement
}

func NewDeferredGenericCallableReference(
	reference NameReference,
	recovery DeferredGenericRecoveryPlacement,
) (DeferredGenericCallableReference, error) {
	if reference.Name() == "" || !recovery.Valid() {
		return DeferredGenericCallableReference{}, &NameError{
			Reason: "deferred generic callable reference is invalid",
		}
	}
	return DeferredGenericCallableReference{
		reference: reference,
		recovery:  recovery,
	}, nil
}

func (r DeferredGenericCallableReference) Valid() bool {
	return r.reference.Name() != "" && r.recovery.Valid()
}

func (r DeferredGenericCallableReference) Reference() NameReference {
	return r.reference
}

func (r DeferredGenericCallableReference) RecoveryPlacement() DeferredGenericRecoveryPlacement {
	return r.recovery
}

func (r DeferredGenericCallableReference) CallArguments(
	recovery tsgo.Expression,
	arguments []tsgo.Expression,
) ([]tsgo.Expression, error) {
	if !r.Valid() || recovery == nil {
		return nil, &NameError{
			Reason: "deferred generic callable arguments are invalid",
		}
	}
	result := make([]tsgo.Expression, 0, len(arguments)+1)
	if r.recovery == DeferredGenericRecoveryFirst {
		result = append(result, recovery)
	}
	result = append(result, arguments...)
	if r.recovery == DeferredGenericRecoveryLast {
		result = append(result, recovery)
	}
	return result, nil
}

func NewGenericConcretizationReference(
	concretization *GenericConcretization,
	name string,
	requests ...RootRequest,
) (GenericConcretizationReference, error) {
	if !concretization.Valid() || name == "" {
		return GenericConcretizationReference{}, &NameError{
			Reason: "generic concretization reference is invalid",
		}
	}
	if err := validateReferenceRequests(requests); err != nil {
		return GenericConcretizationReference{}, err
	}
	return GenericConcretizationReference{
		concretization: concretization,
		name:           name,
		requests:       slices.Clone(requests),
	}, nil
}

func (r GenericConcretizationReference) Concretization() *GenericConcretization {
	return r.concretization
}

func (r GenericConcretizationReference) Name() string {
	return r.name
}

func (r GenericConcretizationReference) Requests() []RootRequest {
	return slices.Clone(r.requests)
}

func NewGenericConcretizationRequirement(
	artifact *GeneratedArtifact,
) (DeclarationRequirement, error) {
	_, ok := artifact.GenericConcretization()
	if !ok {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "generic concretization requirement is invalid",
		}
	}
	return DeclarationRequirement{
		owner:     artifact.ReconstructionOwner(),
		kind:      DeclarationRequirementGenericConcretization,
		generated: artifact,
	}, nil
}

func NewGenericConcretizationRequest(
	artifact *GeneratedArtifact,
) (RootRequest, error) {
	requirement, err := NewGenericConcretizationRequirement(artifact)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func NewDeferredGenericConcretizationRequirement(
	artifact *GeneratedArtifact,
) (DeclarationRequirement, error) {
	concretization, ok := artifact.GenericConcretization()
	if !ok || concretization.Owner() == nil {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "deferred generic concretization requirement is invalid",
		}
	}
	return DeclarationRequirement{
		owner:                  artifact.ReconstructionOwner(),
		kind:                   DeclarationRequirementGenericConcretization,
		generated:              artifact,
		concretizationDeferred: true,
	}, nil
}

func NewDeferredGenericConcretizationRequest(
	artifact *GeneratedArtifact,
) (RootRequest, error) {
	requirement, err := NewDeferredGenericConcretizationRequirement(artifact)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func (r DeclarationRequirement) GenericConcretization() (
	*GenericConcretization,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementGenericConcretization {
		return nil, false
	}
	return r.generated.GenericConcretization()
}

func (r DeclarationRequirement) DeferredGenericConcretization() bool {
	return r.Valid() &&
		r.kind == DeclarationRequirementGenericConcretization &&
		r.concretizationDeferred
}
