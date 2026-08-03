package unsafepointer

import (
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (b builder) bindPointerSync(
	pointer tsgo.Expression,
	location tsgo.Expression,
) tsgo.Expression {
	callback := func(member string) tsgo.ArrowFunction {
		return b.factory.ArrowFunction(
			nil,
			nil,
			nil,
			b.voidType(),
			b.factory.EqualsGreaterThanToken(),
			b.call(location, member),
		)
	}
	return b.typedCall(
		b.id(b.pointerName),
		pointerruntime.UnsafeBindName,
		nil,
		pointer,
		callback("flush"),
		callback("refresh"),
	)
}
