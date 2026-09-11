package slice

import (
	"github.com/tsoniclang/gotots/internal/emit/runtime/memoryview"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func BuildRegion(factory tsgo.Factory, functionName, sliceName, panicName string) tsgo.FunctionDeclaration {
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
			target.returnStatement(factory.CallExpression(target.id("goSliceFromRegion"), nil, []tsgo.TypeNode{target.typeT()},
				[]tsgo.Expression{location, length, length}, tsgo.NodeFlagsNone)),
		}, true))
}
