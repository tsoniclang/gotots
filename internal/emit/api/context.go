package api

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type AddressableStorage interface {
	Name(Context, *types.Var) (string, bool)
	Read(Context, *types.Var) (ExpressionEmission, bool, error)
	StoreTarget(Context, *types.Var) (StoreTargetEmission, bool, error)
	Cell(
		Context,
		ChildEmitter,
		ast.Node,
		types.Type,
		ExpressionEmission,
	) (ExpressionEmission, error)
	Requirement(Context, *types.Var) (RootRequest, error)
}

type Context struct {
	role                     Role
	fileSet                  *token.FileSet
	typesPackage             *types.Package
	typesInfo                *types.Info
	typesSizes               types.Sizes
	factory                  tsgo.Factory
	names                    Names
	values                   Values
	storage                  AddressableStorage
	integer                  IntegerRepresentation
	evaluationOrder          EvaluationOrder
	goRuntime                GoRuntimeContract
	expectedType             types.Type
	expectedResults          *types.Tuple
	functionResults          *types.Tuple
	breakDepth               uint32
	continueDepth            uint32
	breakTarget              string
	continueTarget           string
	controlLabels            map[*types.Label]ControlLabel
	statementLabel           string
	artifactOwner            ArtifactOwner
	callableControls         map[ast.Node]CallableControlDemand
	callableEnclosing        ast.Node
	currentCallable          ast.Node
	currentControl           CallableControlDemand
	deferControl             DeferControl
	returnControl            ReturnControl
	gotoUses                 map[*types.Label][]token.Pos
	gotoTargets              map[*types.Label]GotoTarget
	gotoLocals               map[*types.Var]struct{}
	storageNames             map[*types.Var]string
	localConstantProjections map[*types.Const][]types.BasicKind
	lexicalTypeRequirements  map[*types.TypeName][]DeclarationRequirement
	genericResolver          GenericCallableResolver
	genericConsumer          GenericOperationConsumer
	genericParameters        map[*types.TypeParam]string
	iteratorRangeStateName   string
}

func (c Context) WithAddressableStorage(
	owner *types.Func,
	storageNames map[*types.Var]string,
) (Context, error) {
	current, ok := c.FunctionArtifactOwner()
	if owner == nil || !ok || current != owner {
		return Context{}, &ContextError{
			Reason: "addressable-storage owner differs from source artifact owner",
		}
	}
	c.storageNames = make(map[*types.Var]string, len(storageNames))
	for variable, name := range storageNames {
		if variable == nil || variable.IsField() || name == "" {
			return Context{}, &ContextError{
				Reason: "addressable-storage selection is invalid",
			}
		}
		c.storageNames[variable] = name
	}
	return c, nil
}

func (c Context) WithLocalConstantProjections(
	owner *types.Func,
	projections map[*types.Const][]types.BasicKind,
) (Context, error) {
	current, ok := c.FunctionArtifactOwner()
	if owner == nil || !ok || current != owner {
		return Context{}, &ContextError{
			Reason: "local-constant projection owner differs from source artifact owner",
		}
	}
	c.localConstantProjections = make(
		map[*types.Const][]types.BasicKind,
		len(projections),
	)
	for selected, kinds := range projections {
		if selected == nil || len(kinds) == 0 {
			return Context{}, &ContextError{
				Reason: "local-constant projection selection is invalid",
			}
		}
		c.localConstantProjections[selected] = slices.Clone(kinds)
	}
	return c, nil
}

func (c Context) WithLexicalTypeRequirements(
	owner ArtifactOwner,
	requirements map[*types.TypeName][]DeclarationRequirement,
) Context {
	_, sourceOwned := owner.Source()
	_, _, initializerOwned := owner.PackageInitializer()
	if !sourceOwned && !initializerOwned {
		panic("lexical type-requirement owner has no source reconstruction")
	}
	c = c.withArtifactOwner(owner)
	c.lexicalTypeRequirements = make(
		map[*types.TypeName][]DeclarationRequirement,
		len(requirements),
	)
	for anchor, selected := range requirements {
		if anchor == nil || len(selected) == 0 {
			panic("lexical type-requirement selection is invalid")
		}
		for _, requirement := range selected {
			if requirement.Owner() != owner {
				panic("lexical type-requirement owner is inconsistent")
			}
			if artifact, ok := requirement.GeneratedArtifact(); ok {
				if artifact.Placement() !=
					GeneratedArtifactPlacementLexical ||
					artifact.LexicalOwner() != owner ||
					artifact.LexicalAnchor() != anchor {
					panic("lexical generated-artifact requirement is inconsistent")
				}
				continue
			}
			typeName, _, ok := requirement.NamedStructOperation()
			if !ok || typeName != anchor {
				panic("lexical named-type requirement is inconsistent")
			}
		}
		c.lexicalTypeRequirements[anchor] = slices.Clone(selected)
	}
	return c
}

func NewContext(
	role Role,
	fileSet *token.FileSet,
	typesPackage *types.Package,
	typesInfo *types.Info,
	typesSizes types.Sizes,
	factory tsgo.Factory,
	names Names,
	values Values,
	storage AddressableStorage,
	integer IntegerRepresentation,
	evaluationOrder EvaluationOrder,
) (Context, error) {
	switch {
	case role == "":
		return Context{}, &ContextError{Reason: "role is empty"}
	case fileSet == nil:
		return Context{}, &ContextError{Reason: "file set is nil"}
	case typesPackage == nil:
		return Context{}, &ContextError{Reason: "types package is nil"}
	case typesInfo == nil:
		return Context{}, &ContextError{Reason: "types info is nil"}
	case typesSizes == nil:
		return Context{}, &ContextError{Reason: "types sizes are nil"}
	case names == nil:
		return Context{}, &ContextError{Reason: "name owner is nil"}
	case values == nil:
		return Context{}, &ContextError{Reason: "value owner is nil"}
	case storage == nil:
		return Context{}, &ContextError{Reason: "addressable-storage owner is nil"}
	case !integer.Valid():
		return Context{}, &ContextError{Reason: "integer representation is invalid"}
	case !evaluationOrder.Valid():
		return Context{}, &ContextError{Reason: "evaluation order is invalid"}
	}
	return Context{
		role:            role,
		fileSet:         fileSet,
		typesPackage:    typesPackage,
		typesInfo:       typesInfo,
		typesSizes:      typesSizes,
		factory:         factory,
		names:           names,
		values:          values,
		storage:         storage,
		integer:         integer,
		evaluationOrder: evaluationOrder,
	}, nil
}

func (c Context) WithRole(role Role) Context {
	c.role = role
	return c
}

func (c Context) WithExpectedType(expectedType types.Type) Context {
	c.expectedType = expectedType
	c.expectedResults = nil
	return c
}

func (c Context) WithExpectedResults(expectedResults *types.Tuple) Context {
	if expectedResults == nil || expectedResults.Len() < 2 {
		panic("expected result tuple has fewer than two elements")
	}
	c.expectedType = nil
	c.expectedResults = expectedResults
	return c
}

func (c Context) EnterFunction(results *types.Tuple) Context {
	c.functionResults = results
	c.expectedType = nil
	c.expectedResults = nil
	c.breakDepth = 0
	c.continueDepth = 0
	c.breakTarget = ""
	c.continueTarget = ""
	c.controlLabels = nil
	c.statementLabel = ""
	c.iteratorRangeStateName = ""
	c.currentCallable = nil
	c.currentControl = CallableControlDemand{}
	c.deferControl = DeferControl{}
	c.returnControl = ReturnControl{}
	c.gotoTargets = nil
	c.gotoLocals = nil
	return c
}

func (c Context) EnterLoop() Context {
	c.breakDepth++
	c.continueDepth++
	return c
}

func (c Context) EnterLoopTarget(name string) Context {
	if name == "" {
		panic("loop control target is empty")
	}
	c = c.EnterLoop()
	c.breakTarget = name
	c.continueTarget = name
	return c
}

func (c Context) EnterBreakable() Context {
	c.breakDepth++
	return c
}

func (c Context) EnterIteratorRange(stateName string) Context {
	if stateName == "" {
		panic("iterator-range state name is empty")
	}
	c.breakDepth = 0
	c.continueDepth = 0
	c.controlLabels = nil
	c.statementLabel = ""
	c.iteratorRangeStateName = stateName
	return c
}

func (c Context) EnterBreakableTarget(name string) Context {
	if name == "" {
		panic("breakable control target is empty")
	}
	c = c.EnterBreakable()
	c.breakTarget = name
	return c
}

func (c Context) WithControlLabel(
	label *types.Label,
	target ControlLabel,
) Context {
	if label == nil || !target.Valid() {
		panic("control-label capability is invalid")
	}
	labels := make(map[*types.Label]ControlLabel, len(c.controlLabels)+1)
	for existing, capability := range c.controlLabels {
		labels[existing] = capability
	}
	labels[label] = target
	c.controlLabels = labels
	return c
}

func (c Context) WithStatementLabel(name string) Context {
	if name == "" || c.statementLabel != "" {
		panic("statement-label capability is invalid")
	}
	c.statementLabel = name
	return c
}

func (c Context) TakeStatementLabel() (Context, string) {
	name := c.statementLabel
	c.statementLabel = ""
	return c, name
}

func (c Context) Role() Role {
	return c.role
}

func (c Context) FileSet() *token.FileSet {
	return c.fileSet
}

func (c Context) TypesPackage() *types.Package {
	return c.typesPackage
}

func (c Context) TypesInfo() *types.Info {
	return c.typesInfo
}

func (c Context) TypesSizes() types.Sizes {
	return c.typesSizes
}

func (c Context) Factory() tsgo.Factory {
	return c.factory
}

func (c Context) Names() Names {
	return c.names
}

func (c Context) Values() Values {
	return c.values
}

func (c Context) AddressableStorage() AddressableStorage {
	return c.storage
}

func (c Context) IntegerRepresentation() IntegerRepresentation {
	return c.integer
}

func (c Context) EvaluationOrder() EvaluationOrder {
	return c.evaluationOrder
}

func (c Context) ExpectedType() types.Type {
	return c.expectedType
}

func (c Context) ExpectedResults() *types.Tuple {
	return c.expectedResults
}

func (c Context) FunctionResults() *types.Tuple {
	return c.functionResults
}

func (c Context) CanBreak() bool {
	return c.breakDepth != 0
}

func (c Context) CanContinue() bool {
	return c.continueDepth != 0
}

func (c Context) BreakTarget() string {
	return c.breakTarget
}

func (c Context) ContinueTarget() string {
	return c.continueTarget
}

func (c Context) SelectControlTarget(existing string) (string, error) {
	if existing != "" || !c.currentControl.Goto() {
		return existing, nil
	}
	return c.names.Temporary(TemporaryControlTarget)
}

func (c Context) ControlLabel(label *types.Label) (ControlLabel, bool) {
	if label == nil {
		return ControlLabel{}, false
	}
	target, ok := c.controlLabels[label]
	return target, ok
}

func (c Context) IteratorRangeControl() (IteratorRangeControl, bool) {
	if c.iteratorRangeStateName == "" {
		return IteratorRangeControl{}, false
	}
	return IteratorRangeControl{
		stateName: c.iteratorRangeStateName,
	}, true
}

func (c Context) ArtifactOwner() ArtifactOwner {
	return c.artifactOwner
}

func (c Context) FunctionArtifactOwner() (*types.Func, bool) {
	source, ok := c.artifactOwner.Source()
	owner, function := source.(*types.Func)
	return owner, ok && function
}

func (c Context) LocalConstantProjections(
	selected *types.Const,
) []types.BasicKind {
	return slices.Clone(c.localConstantProjections[selected])
}

func (c Context) LexicalTypeRequirements(
	anchor *types.TypeName,
) []DeclarationRequirement {
	return slices.Clone(c.lexicalTypeRequirements[anchor])
}

func (c Context) AddressableStorageName(variable *types.Var) (string, bool) {
	if variable == nil {
		return "", false
	}
	name, ok := c.storageNames[variable]
	return name, ok
}

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

type IteratorRangeState int8

const (
	IteratorRangeStateExhausted IteratorRangeState = -2
	IteratorRangeStatePanicked  IteratorRangeState = -1
	IteratorRangeStateDone      IteratorRangeState = 0
	IteratorRangeStateReady     IteratorRangeState = 1
)

func (s IteratorRangeState) Literal() string {
	switch s {
	case IteratorRangeStateExhausted:
		return "-2"
	case IteratorRangeStatePanicked:
		return "-1"
	case IteratorRangeStateDone:
		return "0"
	case IteratorRangeStateReady:
		return "1"
	default:
		return ""
	}
}

type IteratorRangeControl struct {
	stateName string
}

func (c IteratorRangeControl) StateName() string {
	return c.stateName
}

func (c IteratorRangeControl) Valid() bool {
	return c.stateName != ""
}

type ChildEmitter interface {
	Block(Context, *ast.BlockStmt) (BlockEmission, error)
	Statements(Context, ast.Node, []ast.Stmt) (StatementEmission, error)
	Statement(Context, ast.Stmt) (StatementEmission, error)
	Expression(Context, ast.Expr) (ExpressionEmission, error)
	Address(Context, ast.Expr) (ExpressionEmission, error)
	StoreTarget(Context, ast.Expr) (StoreTargetEmission, error)
	DiscardedCall(Context, *ast.CallExpr) (ExpressionEmission, error)
	Condition(Context, ast.Expr) (ExpressionEmission, error)
	IntegerConstant(Context, ast.Expr) (ExpressionEmission, error)
	ScopedInitializer(Context, ast.Stmt) (StatementEmission, error)
	IfAlternate(Context, *ast.IfStmt) (StatementEmission, error)
	Type(Context, ast.Expr) (TypeEmission, error)
	RepresentedType(Context, ast.Node, types.Type) (TypeEmission, error)
}

func (c Context) withArtifactOwner(owner ArtifactOwner) Context {
	if !owner.Valid() {
		panic("artifact owner is invalid")
	}
	if c.artifactOwner.Valid() && c.artifactOwner != owner {
		panic("artifact owner is inconsistent")
	}
	c.artifactOwner = owner
	return c
}
