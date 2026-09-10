package slice

import (
	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func BuildData(factory tsgo.Factory, sliceName string) (tsgo.Statement, error) {
	contract, err := api.RuntimeContract(api.RuntimeSliceData)
	if err != nil {
		return nil, err
	}
	pointer, err := tsoniccore.Resolve(tsoniccore.SymbolPointer)
	if err != nil {
		return nil, err
	}
	allocate, err := tsoniccore.Resolve(tsoniccore.SymbolAllocatePointer)
	if err != nil {
		return nil, err
	}
	target := builder{factory: factory, className: sliceName, pointerName: pointer.Export()}
	value := target.id("value")
	return factory.FunctionDeclaration([]tsgo.ModifierLike{factory.ExportKeyword()}, nil, target.id(contract.ExportedName()),
		[]tsgo.TypeParameterDeclaration{target.typeParameter()},
		[]tsgo.ParameterDeclaration{target.parameter("value", target.sliceType()), target.parameter("zero", target.typeT())},
		factory.UnionTypeNode([]tsgo.TypeNode{target.pointerType(target.typeT()), factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindUndefinedKeyword)}),
		factory.Block([]tsgo.Statement{
			factory.IfStatement(target.call(value, MemberName(MemberIsNil)),
				factory.Block([]tsgo.Statement{factory.ReturnStatement(factory.VoidExpression(target.number("0")))}, true), nil),
			factory.IfStatement(target.binary(target.property(value, MemberName(MemberCapacity)), tsgo.BinaryOperatorEqualsEqualsEqualsToken, target.number("0")),
				factory.Block([]tsgo.Statement{factory.ReturnStatement(factory.CallExpression(target.id(allocate.Export()), nil,
					[]tsgo.TypeNode{target.typeT()}, []tsgo.Expression{target.id("zero")}, tsgo.NodeFlagsNone))}, true), nil),
			factory.ReturnStatement(target.call(
				target.call(value, MemberName(MemberSlice), target.number("0"), target.number("1"), factory.NullLiteral()),
				MemberName(MemberAddress), target.number("0"))),
		}, true)), nil
}
