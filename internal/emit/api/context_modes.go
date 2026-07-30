package api

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
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

type IteratorRangeState int8

const (
	IteratorRangeStateExhausted IteratorRangeState = -2
	IteratorRangeStatePanicked  IteratorRangeState = -1
	IteratorRangeStateDone      IteratorRangeState = 0
	IteratorRangeStateReady     IteratorRangeState = 1
	IteratorRangeStateReturned  IteratorRangeState = 2
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
	case IteratorRangeStateReturned:
		return "2"
	default:
		return ""
	}
}

type IteratorRangeControl struct {
	source     *ast.RangeStmt
	stateName  string
	resultName string
	returning  bool
}

func NewIteratorRangeControl(
	source *ast.RangeStmt,
	stateName string,
	resultName string,
	returning bool,
) (IteratorRangeControl, error) {
	if source == nil ||
		source.X == nil ||
		source.Body == nil ||
		stateName == "" ||
		(!returning && resultName != "") {
		return IteratorRangeControl{}, &InvariantError{
			Role:   RoleRangeBody,
			Reason: "iterator-range control is invalid",
		}
	}
	return IteratorRangeControl{
		source:     source,
		stateName:  stateName,
		resultName: resultName,
		returning:  returning,
	}, nil
}

func (c IteratorRangeControl) Source() *ast.RangeStmt {
	return c.source
}

func (c IteratorRangeControl) StateName() string {
	return c.stateName
}

func (c IteratorRangeControl) ResultName() string {
	return c.resultName
}

func (c IteratorRangeControl) Returning() bool {
	return c.returning
}

func (c IteratorRangeControl) Valid() bool {
	return c.source != nil &&
		c.source.X != nil &&
		c.source.Body != nil &&
		c.stateName != "" &&
		(c.returning || c.resultName == "")
}

type ValueReceiverBinding struct {
	method       *types.Func
	variable     *types.Var
	original     tsgo.Expression
	selected     tsgo.Expression
	copyName     string
	copySelected bool
}

func (c Context) WithValueReceiver(
	method *types.Func,
	value tsgo.Expression,
	copyName string,
	copySelected bool,
) (Context, error) {
	if method == nil ||
		method.Origin() != method ||
		value == nil ||
		copyName == "" ||
		ValueReceiverTypeName(method) == nil {
		return Context{}, &ContextError{
			Reason: "value-receiver binding is invalid",
		}
	}
	owner, ok := c.ArtifactOwner().Source()
	signature := method.Signature()
	if !ok ||
		owner != method ||
		signature == nil ||
		signature.Recv() == nil {
		return Context{}, &ContextError{
			Reason: "value-receiver binding differs from artifact owner",
		}
	}
	selected := value
	if copySelected {
		selected = c.Factory().Identifier(copyName)
	}
	c.valueReceiver = &ValueReceiverBinding{
		method:       method,
		variable:     signature.Recv(),
		original:     value,
		selected:     selected,
		copyName:     copyName,
		copySelected: copySelected,
	}
	return c, nil
}

func (c Context) ValueReceiver(
	variable *types.Var,
) (ValueReceiverBinding, bool) {
	if c.valueReceiver == nil ||
		variable == nil ||
		c.valueReceiver.variable != variable {
		return ValueReceiverBinding{}, false
	}
	return *c.valueReceiver, true
}

func (b ValueReceiverBinding) Method() *types.Func {
	return b.method
}

func (b ValueReceiverBinding) Variable() *types.Var {
	return b.variable
}

func (b ValueReceiverBinding) Value() tsgo.Expression {
	return b.selected
}

func (b ValueReceiverBinding) OriginalValue() tsgo.Expression {
	return b.original
}

func (b ValueReceiverBinding) CopyName() string {
	return b.copyName
}

func (b ValueReceiverBinding) CopySelected() bool {
	return b.copySelected
}

func (b ValueReceiverBinding) CopyRequest() (RootRequest, error) {
	return NewValueReceiverCopyRequest(b.method)
}
