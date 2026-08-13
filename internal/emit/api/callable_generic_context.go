package api

import (
	"go/ast"
	"go/types"
	"slices"
)

type GenericCallableResolver interface {
	ResolveGenericConcretization(
		*types.Func,
		TypeArgumentList,
		*types.Signature,
		GenericConcretizationEffect,
		GeneratedArtifactPlacement,
		ArtifactOwner,
		*types.TypeName,
	) (*GenericConcretization, error)
	GenericCallableSynchronousParameters(*types.Func) ([]int, bool, error)
	GenericCallableRequiresConcretization(*types.Func) (bool, error)
	ResolveGenericOperationSet(
		types.Object,
		GenericOperationConsumer,
	) (GenericOperationSet, bool, error)
	ResolveGenericOperation(
		types.Object,
		GenericOperationConsumer,
		GenericOperationSelection,
		*types.Signature,
	) (*GenericOperationContract, error)
	ResolveGenericRepresentationProfile(
		types.Object,
	) (GenericRepresentationProfile, bool, error)
}

func (c Context) GenericCallableRequiresConcretization(
	owner *types.Func,
) (bool, error) {
	if c.genericResolver == nil {
		return false, &ContextError{
			Reason: "generic concretization resolver is unavailable",
		}
	}
	return c.genericResolver.GenericCallableRequiresConcretization(owner)
}

func (c Context) ResolveGenericConcretization(
	owner *types.Func,
	arguments TypeArgumentList,
	signature *types.Signature,
	effect GenericConcretizationEffect,
) (GenericConcretizationReference, error) {
	if c.genericResolver == nil {
		return GenericConcretizationReference{}, &ContextError{
			Reason: "generic concretization resolver is unavailable",
		}
	}
	names, ok := c.names.(GenericConcretizationNames)
	if !ok {
		return GenericConcretizationReference{}, &ContextError{
			Reason: "generic concretization names are unavailable",
		}
	}
	placement, lexicalOwner, anchor, err :=
		names.GenericConcretizationPlacement(owner, arguments)
	if err != nil {
		return GenericConcretizationReference{}, err
	}
	concretization, err := c.genericResolver.ResolveGenericConcretization(
		owner,
		arguments,
		signature,
		effect,
		placement,
		lexicalOwner,
		anchor,
	)
	if err != nil {
		return GenericConcretizationReference{}, err
	}
	return names.GenericConcretization(concretization)
}

func (c Context) GenericCallableSynchronousParameters(
	owner *types.Func,
) ([]int, bool, error) {
	if c.genericResolver == nil {
		return nil, false, &ContextError{
			Reason: "generic concretization resolver is unavailable",
		}
	}
	return c.genericResolver.GenericCallableSynchronousParameters(owner)
}

func (c Context) WithGenericCallableResolver(
	resolver GenericCallableResolver,
) Context {
	if resolver == nil {
		panic("generic callable resolver is nil")
	}
	c.genericResolver = resolver
	return c
}

func (c Context) ProjectGenericOperation(
	source ast.Node,
	origin *GenericOperationContract,
	signature *types.Signature,
) (GenericOperationReference, error) {
	owner, ownerOK := c.genericSourceOwner()
	if !ownerOK ||
		!c.genericConsumer.Valid() ||
		c.genericResolver == nil ||
		source == nil ||
		!origin.Valid() ||
		!validGenericOperationSignature(signature) {
		return GenericOperationReference{}, &ContextError{
			Reason: "projected generic operation is unavailable",
		}
	}
	contract, err := c.genericResolver.ResolveGenericOperation(
		owner,
		c.genericConsumer,
		origin.Selection(),
		signature,
	)
	if err != nil {
		return GenericOperationReference{}, err
	}
	request, err := NewGenericOperationRequest(owner, contract)
	if err != nil {
		return GenericOperationReference{}, err
	}
	return NewGenericOperationReference(
		contract,
		contract.TargetName(),
		request,
	)
}

func (c Context) WithGenericParameters(
	owner types.Object,
	names map[*types.TypeParam]string,
) (Context, error) {
	owner = GenericDeclarationOrigin(owner)
	sourceOwner, sourceOwned := c.artifactOwner.Source()
	if owner == nil || !sourceOwned || sourceOwner != owner {
		return Context{}, &ContextError{
			Reason: "generic parameter owner differs from source artifact owner",
		}
	}
	switch owner.(type) {
	case *types.Func:
		c.genericConsumer = GenericFunctionOperationConsumer()
	case *types.TypeName:
		c.genericConsumer = GenericOperationConsumerInvalid
	}
	c.genericParameters = make(map[*types.TypeParam]string, len(names))
	c.genericParameterOwner = owner
	for parameter, name := range names {
		if parameter == nil ||
			name == "" ||
			!genericParameterBelongsTo(owner, parameter) {
			return Context{}, &ContextError{
				Reason: "generic parameter binding is invalid",
			}
		}
		c.genericParameters[parameter] = name
	}
	return c, nil
}

func (c Context) WithEnvironmentGenericParameters(
	owner types.Object,
	names map[*types.TypeParam]string,
) (Context, error) {
	owner = GenericDeclarationOrigin(owner)
	if owner == nil ||
		owner.Pkg() == nil ||
		owner.Pkg() != c.typesPackage ||
		c.artifactOwner.Valid() {
		return Context{}, &ContextError{
			Reason: "environment generic parameter owner is invalid",
		}
	}
	c.genericParameters = make(map[*types.TypeParam]string, len(names))
	c.genericParameterOwner = owner
	for parameter, name := range names {
		if parameter == nil ||
			name == "" ||
			!genericParameterBelongsTo(owner, parameter) {
			return Context{}, &ContextError{
				Reason: "environment generic parameter binding is invalid",
			}
		}
		c.genericParameters[parameter] = name
	}
	return c, nil
}

func (c Context) WithGenericNamedStructOperation(
	operation NamedStructOperation,
) Context {
	owner, ownerOK := c.genericSourceOwner()
	if !ownerOK {
		panic("generic named-struct operation has no source owner")
	}
	if _, ok := owner.(*types.TypeName); !ok {
		panic("generic named-struct operation has no type owner")
	}
	consumer, err := GenericNamedStructOperationConsumer(operation)
	if err != nil {
		panic(err)
	}
	c.genericConsumer = consumer
	return c
}

func (c Context) GenericOperation(
	source ast.Node,
	operation GenericOperation,
	signature *types.Signature,
) (GenericOperationReference, error) {
	selection, err := SelectGenericOperation(operation)
	if err != nil {
		return GenericOperationReference{}, err
	}
	return c.genericOperation(source, selection, signature)
}

func (c Context) GenericConstraintMethod(
	source ast.Node,
	method *types.Func,
	signature *types.Signature,
) (GenericOperationReference, error) {
	selection, err := SelectGenericConstraintMethod(method)
	if err != nil {
		return GenericOperationReference{}, err
	}
	return c.genericOperation(source, selection, signature)
}

func (c Context) genericOperation(
	source ast.Node,
	selection GenericOperationSelection,
	signature *types.Signature,
) (GenericOperationReference, error) {
	owner, ownerOK := c.genericSourceOwner()
	switch {
	case !ownerOK:
		return GenericOperationReference{}, &ContextError{
			Reason: "generic operation has no source artifact owner",
		}
	case !c.genericConsumer.Valid():
		return GenericOperationReference{}, &ContextError{
			Reason: "generic operation has no target consumer",
		}
	case c.genericResolver == nil:
		return GenericOperationReference{}, &ContextError{
			Reason: "generic operation has no resolver",
		}
	case source == nil &&
		selection.Operation() == GenericOperationConstraintMethod:
		return GenericOperationReference{}, &ContextError{
			Reason: "generic constraint-method operation has no source construct",
		}
	case !selection.Valid():
		return GenericOperationReference{}, &ContextError{
			Reason: "generic operation selection is invalid",
		}
	case !validGenericOperationSignature(signature):
		return GenericOperationReference{}, &ContextError{
			Reason: "generic operation signature is invalid",
		}
	}
	contract, err := c.genericResolver.ResolveGenericOperation(
		owner,
		c.genericConsumer,
		selection,
		signature,
	)
	if err != nil {
		return GenericOperationReference{}, err
	}
	request, err := NewGenericOperationRequest(owner, contract)
	if err != nil {
		return GenericOperationReference{}, err
	}
	return NewGenericOperationReference(
		contract,
		contract.TargetName(),
		request,
	)
}

func (c Context) ResolveGenericCallable(
	function *types.Func,
) (GenericOperationSet, bool, error) {
	if c.genericResolver == nil {
		return GenericOperationSet{}, false, &ContextError{
			Reason: "generic callable resolver is unavailable",
		}
	}
	if function == nil {
		return GenericOperationSet{}, false, nil
	}
	return c.genericResolver.ResolveGenericOperationSet(
		function.Origin(),
		GenericFunctionOperationConsumer(),
	)
}

func (c Context) ResolveGenericNamedStructOperation(
	owner *types.TypeName,
	operation NamedStructOperation,
) (GenericOperationSet, bool, error) {
	if c.genericResolver == nil {
		return GenericOperationSet{}, false, &ContextError{
			Reason: "generic named-struct operation resolver is unavailable",
		}
	}
	consumer, err := GenericNamedStructOperationConsumer(operation)
	if err != nil {
		return GenericOperationSet{}, false, err
	}
	return c.genericResolver.ResolveGenericOperationSet(owner, consumer)
}

func (c Context) GenericParameterName(
	parameter *types.TypeParam,
) (string, bool) {
	name, ok := c.genericParameters[parameter]
	return name, ok
}

func genericParameterBelongsTo(
	owner types.Object,
	parameter *types.TypeParam,
) bool {
	for _, selected := range GenericDeclarationParameters(owner) {
		if selected == parameter {
			return true
		}
	}
	return false
}

func (c Context) genericSourceOwner() (types.Object, bool) {
	source, ok := c.artifactOwner.Source()
	if !ok {
		return nil, false
	}
	owner := GenericDeclarationOrigin(source)
	return owner, owner != nil && owner == source
}

const (
	DeferredEntrySuffix                = "$deferred"
	GenericKernelSuffix                = "$kernel"
	DeferredRegistryRegisterName       = "register"
	DeferredRegistryResolveName        = "resolve"
	DeferredRegistryRegisterMethodName = "registerMethod"
	DeferredRegistryResolveMethodName  = "resolveMethod"
)

type ControlLabel struct {
	name        string
	breakable   bool
	continuable bool
}

func NewControlLabel(
	name string,
	breakable bool,
	continuable bool,
) (ControlLabel, error) {
	if name == "" || continuable && !breakable {
		return ControlLabel{}, &InvariantError{
			Role:   RoleLabelTarget,
			Reason: "control-label target is invalid",
		}
	}
	return ControlLabel{
		name:        name,
		breakable:   breakable,
		continuable: continuable,
	}, nil
}

func (l ControlLabel) Valid() bool {
	return l.name != "" && (!l.continuable || l.breakable)
}

func (l ControlLabel) Name() string {
	return l.name
}

func (l ControlLabel) Breakable() bool {
	return l.breakable
}

func (l ControlLabel) Continuable() bool {
	return l.continuable
}

type ReturnControl struct {
	label        string
	resultTarget string
	namedTargets []StoreTargetEmission
}

func NewReturnControl(
	label string,
	resultTarget string,
) (ReturnControl, error) {
	if label == "" {
		return ReturnControl{}, &InvariantError{
			Role:   RoleReturnResult,
			Reason: "return-control label is empty",
		}
	}
	return ReturnControl{
		label:        label,
		resultTarget: resultTarget,
	}, nil
}

func NewNamedReturnControl(
	label string,
	targets []StoreTargetEmission,
) (ReturnControl, error) {
	if label == "" || len(targets) == 0 {
		return ReturnControl{}, &InvariantError{
			Role:   RoleReturnResult,
			Reason: "named return control is invalid",
		}
	}
	for _, target := range targets {
		if !target.Valid() {
			return ReturnControl{}, &InvariantError{
				Role:   RoleReturnResult,
				Reason: "named return-control target is invalid",
			}
		}
	}
	return ReturnControl{
		label:        label,
		namedTargets: slices.Clone(targets),
	}, nil
}

func (c ReturnControl) Valid() bool {
	return c.label != "" &&
		(c.resultTarget == "" || len(c.namedTargets) == 0)
}

func (c ReturnControl) Label() string {
	return c.label
}

func (c ReturnControl) ResultTarget() string {
	return c.resultTarget
}

func (c ReturnControl) NamedTargets() []StoreTargetEmission {
	return slices.Clone(c.namedTargets)
}

func (c ReturnControl) Named() bool {
	return len(c.namedTargets) != 0
}

type GotoTargetKind uint8

const (
	GotoTargetInvalid GotoTargetKind = iota
	GotoTargetBreak
	GotoTargetContinue
	GotoTargetState
)

type GotoTarget struct {
	kind      GotoTargetKind
	label     string
	stateName string
	state     int
}

func NewDirectGotoTarget(
	kind GotoTargetKind,
	label string,
) (GotoTarget, error) {
	if (kind != GotoTargetBreak && kind != GotoTargetContinue) ||
		label == "" {
		return GotoTarget{}, &InvariantError{
			Role:   RoleLabelTarget,
			Reason: "direct goto target is invalid",
		}
	}
	return GotoTarget{kind: kind, label: label}, nil
}

func NewStateGotoTarget(
	dispatchLabel string,
	stateName string,
	state int,
) (GotoTarget, error) {
	if dispatchLabel == "" || stateName == "" || state < 0 {
		return GotoTarget{}, &InvariantError{
			Role:   RoleLabelTarget,
			Reason: "state goto target is invalid",
		}
	}
	return GotoTarget{
		kind:      GotoTargetState,
		label:     dispatchLabel,
		stateName: stateName,
		state:     state,
	}, nil
}

func (t GotoTarget) Valid() bool {
	switch t.kind {
	case GotoTargetBreak, GotoTargetContinue:
		return t.label != "" &&
			t.stateName == "" &&
			t.state == 0
	case GotoTargetState:
		return t.label != "" &&
			t.stateName != "" &&
			t.state >= 0
	default:
		return false
	}
}

func (t GotoTarget) Kind() GotoTargetKind {
	return t.kind
}

func (t GotoTarget) Label() string {
	return t.label
}

func (t GotoTarget) StateName() string {
	return t.stateName
}

func (t GotoTarget) State() int {
	return t.state
}

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
