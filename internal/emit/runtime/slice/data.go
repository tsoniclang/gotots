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
	target := builder{factory: factory, className: sliceName, pointerName: pointer.Export()}
	return factory.FunctionDeclaration([]tsgo.ModifierLike{factory.ExportKeyword()}, nil, target.id(contract.ExportedName()),
		[]tsgo.TypeParameterDeclaration{target.typeParameter()}, []tsgo.ParameterDeclaration{
			target.parameter("value", target.sliceType()), target.parameter("zero", factory.FunctionTypeNode(nil, nil, target.typeT())),
		}, target.optionalDataType(), factory.Block([]tsgo.Statement{
			target.returnStatement(target.call(target.id("value"), MemberName(MemberData), target.id("zero"))),
		}, true)), nil
}

func (b builder) optionalDataType() tsgo.TypeNode {
	return b.factory.UnionTypeNode([]tsgo.TypeNode{b.pointerType(b.typeT()), b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindUndefinedKeyword)})
}

func (b builder) dataMethod() tsgo.MethodDeclaration {
	return b.method(nil, MemberName(MemberData), nil,
		[]tsgo.ParameterDeclaration{b.parameter("zero", b.factory.FunctionTypeNode(nil, nil, b.typeT()))}, b.optionalDataType(),
		b.factory.IfStatement(b.call(b.factory.ThisExpression(), MemberName(MemberIsNil)), b.returnStatement(b.factory.VoidExpression(b.number("0"))), nil),
		b.factory.IfStatement(b.binary(b.thisProperty(MemberName(MemberCapacity)), tsgo.BinaryOperatorEqualsEqualsEqualsToken, b.number("0")),
			b.returnStatement(b.factory.CallExpression(b.id("allocatePointer"), nil, []tsgo.TypeNode{b.typeT()},
				[]tsgo.Expression{b.factory.CallExpression(b.id("zero"), nil, nil, nil, tsgo.NodeFlagsNone)}, tsgo.NodeFlagsNone)), nil),
		b.returnStatement(b.call(b.call(b.factory.ThisExpression(), MemberName(MemberSlice), b.number("0"), b.number("1"), b.factory.NullLiteral()),
			MemberName(MemberAddress), b.number("0"))))
}

func (b pointerSliceBuilder) pointerData() tsgo.MethodDeclaration {
	return b.method(nil, MemberName(MemberData), nil,
		[]tsgo.ParameterDeclaration{b.parameter("_zero", b.factory.FunctionTypeNode(nil, nil, b.typeT()))}, b.optionalDataType(),
		b.returnStatement(b.regionCall("goRegionAddress", b.thisProperty("region"), b.number("0"))))
}

func (b projectionBuilder) dataMethod() tsgo.MethodDeclaration {
	from := b.typeReference("F")
	to := b.typeReference("T")
	zero := b.factory.ArrowFunction(nil, nil, nil, from, b.factory.EqualsGreaterThanToken(), b.thisProperty("sourceZero"))
	data := b.call(b.source(), MemberName(MemberData), zero)
	projected := b.factory.CallExpression(b.id(b.pointerProject), nil, []tsgo.TypeNode{from, to},
		[]tsgo.Expression{data, b.thisProperty("fromSource"), b.thisProperty("toSource")}, tsgo.NodeFlagsNone)
	return b.method(MemberName(MemberData), []tsgo.ParameterDeclaration{
		b.parameter(nil, "_zero", b.factory.FunctionTypeNode(nil, nil, to)),
	}, b.factory.UnionTypeNode([]tsgo.TypeNode{b.pointerType(to), b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindUndefinedKeyword)}), b.returnStatement(projected))
}
