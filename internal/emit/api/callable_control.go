package api

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
)

const (
	DeferredEntrySuffix                           = "$deferred"
	GenericKernelSuffix                           = "$kernel"
	DeferredRegistryRegisterName                  = "register"
	DeferredRegistryResolveName                   = "resolve"
	DeferredRegistryRegisterMethodName            = "registerMethod"
	DeferredRegistryResolveMethodName             = "resolveMethod"
	DeferredRegistryRegisterCooperativeMethodName = "registerCooperativeMethod"
	DeferredRegistryResolveCooperativeMethodName  = "resolveCooperativeMethod"
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

type DeferControl struct {
	stack string
}

func NewDeferControl(stack string) (DeferControl, error) {
	if stack == "" {
		return DeferControl{}, &InvariantError{
			Role:   RoleBlockStatement,
			Reason: "defer stack identity is empty",
		}
	}
	return DeferControl{stack: stack}, nil
}

func (c DeferControl) Valid() bool {
	return c.stack != ""
}

func (c DeferControl) Stack() string {
	return c.stack
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

type CallableControlDemand struct {
	deferEnvelope   bool
	recovery        bool
	gotoControl     bool
	iteratorReturns map[*ast.RangeStmt]struct{}
}

func (d CallableControlDemand) With(
	facet CallableControlFacet,
) CallableControlDemand {
	switch facet {
	case CallableControlDefer:
		d.deferEnvelope = true
	case CallableControlRecovery:
		d.recovery = true
	case CallableControlGoto:
		d.gotoControl = true
	case CallableControlIteratorReturn:
		panic("iterator-return control requires an exact range")
	default:
		panic("callable-control facet is invalid")
	}
	return d
}

func (d CallableControlDemand) Has(
	facet CallableControlFacet,
) bool {
	switch facet {
	case CallableControlDefer:
		return d.deferEnvelope
	case CallableControlRecovery:
		return d.recovery
	case CallableControlGoto:
		return d.gotoControl
	case CallableControlIteratorReturn:
		return len(d.iteratorReturns) != 0
	default:
		return false
	}
}

func (d CallableControlDemand) Defer() bool {
	return d.deferEnvelope
}

func (d CallableControlDemand) Recovery() bool {
	return d.recovery
}

func (d CallableControlDemand) Goto() bool {
	return d.gotoControl
}

func (d CallableControlDemand) WithIteratorReturn(
	source *ast.RangeStmt,
) CallableControlDemand {
	if source == nil {
		panic("iterator-return range is nil")
	}
	selected := make(
		map[*ast.RangeStmt]struct{},
		len(d.iteratorReturns)+1,
	)
	for existing := range d.iteratorReturns {
		selected[existing] = struct{}{}
	}
	selected[source] = struct{}{}
	d.iteratorReturns = selected
	return d
}

func (d CallableControlDemand) IteratorReturn(
	source *ast.RangeStmt,
) bool {
	_, selected := d.iteratorReturns[source]
	return selected
}

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
		if facet == CallableControlIteratorReturn {
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
