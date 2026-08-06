package unsafeoperation

import (
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (b builder) sliceFunction(name string) tsgo.FunctionDeclaration {
	logical := b.typeL()
	storage := b.typeS()
	pointer := b.id("pointer")
	length := b.id("length")
	return b.factory.FunctionDeclaration(
		[]tsgo.ModifierLike{b.factory.ExportKeyword()}, nil, b.id(name),
		[]tsgo.TypeParameterDeclaration{
			b.typeParameter("L", nil),
			b.typeParameter("S", nil),
		},
		[]tsgo.ParameterDeclaration{
			b.parameter("pointer", b.optionalPointerType(logical, storage)),
			b.parameter("length", b.integerType()),
		},
		b.sliceType(storage),
		b.factory.Block([]tsgo.Statement{b.factory.ReturnStatement(
			b.directCall(
				b.sliceRegionName,
				[]tsgo.TypeNode{storage},
				b.pointerRegion(logical, storage, pointer, length),
				length,
			),
		)}, true),
	)
}

func (b builder) sliceDataFunction(name string) tsgo.FunctionDeclaration {
	logical := b.typeL()
	storage := b.typeS()
	location := b.id("location")
	return b.factory.FunctionDeclaration(
		[]tsgo.ModifierLike{b.factory.ExportKeyword()}, nil, b.id(name),
		[]tsgo.TypeParameterDeclaration{
			b.typeParameter("L", nil),
			b.typeParameter("S", nil),
		},
		[]tsgo.ParameterDeclaration{b.parameter("value", b.sliceType(storage))},
		b.optionalPointerType(logical, storage),
		b.factory.Block([]tsgo.Statement{
			b.variable(
				tsgo.NodeFlagsConst,
				"location",
				b.optionalRegionType(storage),
				b.call(b.id("value"), runtimeslice.MemberName(runtimeslice.MemberArrayLocation), []tsgo.TypeNode{b.numberType()}, b.number("0")),
			),
			b.factory.IfStatement(
				b.binary(location, tsgo.BinaryOperatorEqualsEqualsEqualsToken, b.undefined()),
				b.factory.Block([]tsgo.Statement{b.factory.ReturnStatement(b.undefined())}, true),
				nil,
			),
			b.factory.ReturnStatement(b.call(
				b.id(b.pointerName),
				pointerruntime.ElementName,
				[]tsgo.TypeNode{logical, storage},
				location,
			)),
		}, true),
	)
}

func (b builder) sliceHeaderFunction(name string) tsgo.FunctionDeclaration {
	logical := b.typeL()
	storage := b.typeS()
	whole := b.id("whole")
	return b.factory.FunctionDeclaration(
		[]tsgo.ModifierLike{b.factory.ExportKeyword()}, nil, b.id(name),
		[]tsgo.TypeParameterDeclaration{
			b.typeParameter("L", nil),
			b.typeParameter("S", nil),
		},
		[]tsgo.ParameterDeclaration{
			b.parameter("pointer", b.optionalPointerType(logical, storage)),
			b.parameter("length", b.integerType()),
			b.parameter("capacity", b.integerType()),
		},
		b.sliceType(storage),
		b.factory.Block([]tsgo.Statement{
			b.variable(
				tsgo.NodeFlagsConst,
				"whole",
				b.sliceType(storage),
				b.directCall(
					b.unsafeSliceName,
					[]tsgo.TypeNode{logical, storage},
					b.id("pointer"),
					b.id("capacity"),
				),
			),
			b.factory.ReturnStatement(b.call(
				whole,
				runtimeslice.MemberName(runtimeslice.MemberSlice),
				nil,
				b.number("0"),
				b.id("length"),
				b.factory.NullLiteral(),
			)),
		}, true),
	)
}
