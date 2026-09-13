package slice

import (
	"github.com/tsoniclang/gotots/internal/emit/runtime/memoryview"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func BuildElementRegion(factory tsgo.Factory, functionName, sliceName, panicName string) tsgo.FunctionDeclaration {
	target := builder{factory: factory, className: sliceName, panicName: panicName}
	index := target.id("index")
	value := target.id("value")
	location := target.id("location")
	invalid := target.binary(target.binary(index, tsgo.BinaryOperatorLessThanToken, target.number("0")),
		tsgo.BinaryOperatorBarBarToken, target.binary(index, tsgo.BinaryOperatorGreaterThanEqualsToken,
			target.call(value, MemberName(MemberSourceLength))))
	return factory.FunctionDeclaration([]tsgo.ModifierLike{factory.ExportKeyword()}, nil, target.id(functionName),
		[]tsgo.TypeParameterDeclaration{target.typeParameter()}, []tsgo.ParameterDeclaration{
			target.parameter("value", target.sliceType()), target.parameter("index", target.integerInputType()),
		}, memoryview.RegionType(factory, target.typeT()), factory.Block([]tsgo.Statement{
			factory.IfStatement(invalid, target.throwBounds(), nil),
			target.variable(tsgo.NodeFlagsConst, "location", target.call(value, MemberName(MemberArrayLocation), target.number("0"))),
			factory.IfStatement(target.binary(location, tsgo.BinaryOperatorEqualsEqualsEqualsToken, factory.VoidExpression(target.number("0"))),
				target.throwBounds(), nil),
			target.returnStatement(factory.CallExpression(target.id("goRegionView"), nil, []tsgo.TypeNode{target.typeT()},
				[]tsgo.Expression{location, index}, tsgo.NodeFlagsNone)),
		}, true))
}

func BuildRegion(factory tsgo.Factory, functionName, sliceName, pointerSliceName, panicName string) tsgo.FunctionDeclaration {
	target := builder{factory: factory, className: sliceName, panicName: panicName}
	location := target.id("location")
	length := target.id("length")
	undefined := factory.VoidExpression(target.number("0"))
	fail := func(message string) tsgo.Statement {
		return factory.ExpressionStatement(panicruntime.Call(factory, panicName, factory.StringLiteral(message, tsgo.TokenFlagsNone)))
	}
	return factory.FunctionDeclaration([]tsgo.ModifierLike{factory.ExportKeyword()}, nil, target.id(functionName),
		[]tsgo.TypeParameterDeclaration{target.typeParameter()}, []tsgo.ParameterDeclaration{
			target.parameter("location", factory.UnionTypeNode([]tsgo.TypeNode{memoryview.RegionType(factory, target.typeT()),
				factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindUndefinedKeyword)})),
			target.parameter("length", target.integerInputType()),
		}, target.sliceType(), factory.Block([]tsgo.Statement{
			factory.IfStatement(target.binary(length, tsgo.BinaryOperatorLessThanToken, target.number("0")), fail("unsafe slice length is negative"), nil),
			factory.IfStatement(target.binary(location, tsgo.BinaryOperatorEqualsEqualsEqualsToken, undefined), factory.Block([]tsgo.Statement{
				factory.IfStatement(target.binary(length, tsgo.BinaryOperatorEqualsEqualsToken, target.number("0")), factory.Block([]tsgo.Statement{
					target.returnStatement(factory.CallExpression(target.property(target.id(sliceName), MemberName(MemberNil)), nil,
						[]tsgo.TypeNode{target.typeT()}, nil, tsgo.NodeFlagsNone)),
				}, true), nil), fail("unsafe slice on nil pointer"),
			}, true), nil),
			factory.IfStatement(target.binary(length, tsgo.BinaryOperatorEqualsEqualsToken, target.number("0")),
				target.returnStatement(factory.NewExpression(target.id(pointerSliceName), []tsgo.TypeNode{target.typeT()},
					[]tsgo.Expression{location, length, length})), nil),
			target.returnStatement(factory.CallExpression(target.id("goSliceFromRegion"), nil, []tsgo.TypeNode{target.typeT()},
				[]tsgo.Expression{location, length, length}, tsgo.NodeFlagsNone)),
		}, true))
}
