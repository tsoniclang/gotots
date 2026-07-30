package api

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

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
