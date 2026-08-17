package api

import (
	controlcontract "github.com/tsoniclang/gotots/internal/emit/api/control"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	"go/ast"
	"go/token"
	"go/types"
	"slices"
)

type CallableControlDemand = controlcontract.CallableDemand

func (c Context) WithCallableControls(
	owner ArtifactOwner,
	enclosing ast.Node,
	requirements []DeclarationRequirement,
) (Context, error) {
	if !owner.Valid() || enclosing == nil {
		return Context{}, &InvariantError{
			Role:   c.role,
			Reason: "callable-control owner is invalid",
		}
	}
	if c.artifactOwner.Valid() && c.artifactOwner != owner {
		return Context{}, &InvariantError{
			Role:   c.role,
			Reason: "callable-control artifact owner is inconsistent",
		}
	}
	controls := make(map[ast.Node]CallableControlDemand)
	gotoUses := make(map[*types.Label][]token.Pos)
	for _, requirement := range requirements {
		if requirement.Kind() != DeclarationRequirementCallableControl {
			continue
		}
		requirementOwner, requirementEnclosing, callable, facet, ok :=
			requirement.CallableControl()
		if callable == nil &&
			requirementEnclosing == nil &&
			requirementOwner == owner {
			callable = enclosing
			requirementEnclosing = enclosing
		}
		if !ok ||
			requirementOwner != owner ||
			requirementEnclosing != enclosing {
			return Context{}, &InvariantError{
				Role:   c.role,
				Reason: "callable-control requirement is foreign to its artifact",
			}
		}
		demand := controls[callable]
		if facet == CallableControlDefer {
			source, selected := requirement.DeferControl()
			if selected {
				controls[callable] = demand.WithDefer(source)
			} else {
				controls[callable] = demand.With(facet)
			}
		} else if facet == CallableControlIteratorReturn {
			source, selected := requirement.IteratorReturnControl()
			if !selected {
				return Context{}, &InvariantError{
					Role:   c.role,
					Reason: "iterator-return control lacks exact range identity",
				}
			}
			controls[callable] = demand.WithIteratorReturn(source)
		} else {
			controls[callable] = demand.With(facet)
		}
		if facet == CallableControlGoto {
			label, position, gotoOK := requirement.GotoControl()
			if !gotoOK {
				return Context{}, &InvariantError{
					Role:   c.role,
					Reason: "goto control lacks exact edge identity",
				}
			}
			gotoUses[label] = append(gotoUses[label], position)
		}
	}
	c.artifactOwner = owner
	c.callableControls = controls
	c.callableEnclosing = enclosing
	c.gotoUses = gotoUses
	return c, nil
}

func (c Context) EnterCallable(
	callable ast.Node,
	results *types.Tuple,
) Context {
	if callable == nil {
		panic("callable-control entry is invalid")
	}
	c = c.EnterFunction(results)
	c.currentCallable = callable
	c.currentControl = c.callableControls[callable]
	return c
}

func (c Context) CurrentCallable() ast.Node {
	return c.currentCallable
}

func (c Context) IsCurrentCallableBody(body *ast.BlockStmt) bool {
	if body == nil {
		return false
	}
	switch callable := c.currentCallable.(type) {
	case *ast.FuncDecl:
		return callable.Body == body
	case *ast.FuncLit:
		return callable.Body == body
	default:
		return false
	}
}

func (c Context) CallableControl() CallableControlDemand {
	return c.currentControl
}

func (c Context) CallableControlFor(callable ast.Node) CallableControlDemand {
	return c.callableControls[callable]
}

func (c Context) WithRecoveryAuthority(name string) Context {
	if name == "" || c.recoveryAuthority != "" {
		panic("recovery authority is invalid")
	}
	c.recoveryAuthority = name
	return c
}

func (c Context) RecoveryAuthority() (string, bool) {
	return c.recoveryAuthority, c.recoveryAuthority != ""
}

func (c Context) CallableControlRequest(
	facet CallableControlFacet,
) (RootRequest, error) {
	return NewCallableControlRequest(
		c.artifactOwner,
		c.callableEnclosing,
		c.currentCallable,
		facet,
	)
}

func (c Context) DeferControlRequest(
	source *ast.DeferStmt,
) (RootRequest, error) {
	return NewDeferControlRequest(
		c.artifactOwner,
		c.callableEnclosing,
		c.currentCallable,
		source,
	)
}

func (c Context) FunctionLiteralControlRequest(
	callable *ast.FuncLit,
	facet CallableControlFacet,
) (RootRequest, error) {
	if callable == nil {
		return RootRequest{}, &InvariantError{
			Role:   c.role,
			Reason: "function-literal control request has no callable",
		}
	}
	return NewCallableControlRequest(
		c.artifactOwner,
		c.callableEnclosing,
		callable,
		facet,
	)
}

func (c Context) GotoControlRequest(
	label *types.Label,
	position token.Pos,
) (RootRequest, error) {
	return NewGotoControlRequest(
		c.artifactOwner,
		c.callableEnclosing,
		c.currentCallable,
		label,
		position,
	)
}

func (c Context) IteratorReturnControlRequests() (
	[]RootRequest,
	error,
) {
	if len(c.iteratorRangeControls) == 0 {
		return nil, &InvariantError{
			Role:   c.role,
			Reason: "iterator-return request has no range boundary",
		}
	}
	requests := make([]RootRequest, 0, len(c.iteratorRangeControls))
	for _, control := range c.iteratorRangeControls {
		request, err := NewIteratorReturnControlRequest(
			c.artifactOwner,
			c.callableEnclosing,
			c.currentCallable,
			control.Source(),
		)
		if err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	return requests, nil
}

func (c Context) GotoUses(label *types.Label) []token.Pos {
	return slices.Clone(c.gotoUses[label])
}

func (c Context) WithDeferControl(control DeferControl) Context {
	if !control.Valid() {
		panic("defer control is invalid")
	}
	c.deferControl = control
	return c
}

func (c Context) DeferControl() (DeferControl, bool) {
	return c.deferControl, c.deferControl.Valid()
}

func (c Context) WithReturnControl(control ReturnControl) Context {
	if !control.Valid() {
		panic("return control is invalid")
	}
	c.returnControl = control
	return c
}

func (c Context) ReturnControl() (ReturnControl, bool) {
	return c.returnControl, c.returnControl.Valid()
}

func (c Context) WithoutReturnControl() Context {
	c.returnControl = ReturnControl{}
	return c
}

func (c Context) WithGotoTarget(
	label *types.Label,
	target GotoTarget,
) Context {
	if label == nil || !target.Valid() {
		panic("goto target is invalid")
	}
	targets := make(
		map[*types.Label]GotoTarget,
		len(c.gotoTargets)+1,
	)
	for existing, value := range c.gotoTargets {
		targets[existing] = value
	}
	if _, duplicate := targets[label]; duplicate {
		panic("goto target is duplicated")
	}
	targets[label] = target
	c.gotoTargets = targets
	return c
}

func (c Context) GotoTarget(
	label *types.Label,
) (GotoTarget, bool) {
	if label == nil {
		return GotoTarget{}, false
	}
	target, ok := c.gotoTargets[label]
	return target, ok
}

func (c Context) WithGotoLocals(
	variables []*types.Var,
) Context {
	locals := make(map[*types.Var]struct{}, len(variables))
	for _, variable := range variables {
		if variable == nil || variable.IsField() {
			panic("goto local identity is invalid")
		}
		locals[variable] = struct{}{}
	}
	c.gotoLocals = locals
	return c
}

func (c Context) IsGotoLocal(variable *types.Var) bool {
	if variable == nil {
		return false
	}
	_, selected := c.gotoLocals[variable]
	return selected
}

type GenericConcretizationReference struct {
	concretization *GenericConcretization
	name           string
	suffix         string
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
	GenericKernel(*types.Func) (NameReference, error)
	SynchronousGenericKernel(*types.Func) (NameReference, error)
	DeferredGenericCallable(*types.Func) (DeferredGenericCallableReference, error)
	DeferredGenericKernel(*types.Func) (DeferredGenericCallableReference, error)
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
	suffix string,
	requests ...RootRequest,
) (GenericConcretizationReference, error) {
	if !concretization.Valid() || name == "" || suffix == "" {
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
		suffix:         suffix,
		requests:       slices.Clone(requests),
	}, nil
}

func (r GenericConcretizationReference) Concretization() *GenericConcretization {
	return r.concretization
}

func (r GenericConcretizationReference) Name() string {
	return r.name
}

func (r GenericConcretizationReference) Suffix() string {
	return r.suffix
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

func NewClassMethodRequest(
	owner *types.TypeName,
	method *types.Func,
) (RootRequest, error) {
	requirement, err := NewClassMethodRequirement(owner, method)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func NewValueReceiverCopyRequest(
	method *types.Func,
) (RootRequest, error) {
	requirement, err := NewValueReceiverCopyRequirement(method)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func NewCallableControlRequest(
	owner ArtifactOwner,
	enclosing ast.Node,
	callable ast.Node,
	control CallableControlFacet,
) (RootRequest, error) {
	requirement, err := NewCallableControlRequirement(
		owner,
		enclosing,
		callable,
		control,
	)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func NewDeferControlRequest(
	owner ArtifactOwner,
	enclosing ast.Node,
	callable ast.Node,
	source *ast.DeferStmt,
) (RootRequest, error) {
	requirement, err := NewDeferControlRequirement(
		owner,
		enclosing,
		callable,
		source,
	)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func NewGotoControlRequest(
	owner ArtifactOwner,
	enclosing ast.Node,
	callable ast.Node,
	label *types.Label,
	position token.Pos,
) (RootRequest, error) {
	requirement, err := NewGotoControlRequirement(
		owner,
		enclosing,
		callable,
		label,
		position,
	)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func NewDirectCallableControlRequest(
	owner *types.Func,
	control CallableControlFacet,
) (RootRequest, error) {
	requirement, err := NewDirectCallableControlRequirement(owner, control)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}
