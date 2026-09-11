package slice

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/runtime/memoryview"
	"github.com/tsoniclang/gotots/internal/emit/typescriptclass"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type pointerSliceBuilder struct {
	builder
	pointerSliceName string
}

func BuildPointerSlice(factory tsgo.Factory, name, sliceName, panicName string, capabilities Capabilities) tsgo.ClassDeclaration {
	target := pointerSliceBuilder{builder: builder{factory: factory, className: sliceName, panicName: panicName, pointerName: "Pointer"}, pointerSliceName: name}
	members := []tsgo.ClassElement{target.pointerConstructor(),
		target.pointerCount(MemberSourceLength, "count"), target.pointerCount(MemberSourceCapacity, "extent"),
		target.method(nil, MemberName(MemberIsNil), nil, nil, target.booleanType(), target.returnStatement(factory.FalseLiteral())),
		target.pointerAccess(MemberGet), target.pointerAccess(MemberSet), target.pointerAccess(MemberAddress),
		target.pointerSlice(), target.pointerAppend(), target.pointerLocation(), target.pointerInitialize(), target.pointerWithLength(),
	}
	if capabilities.AppendSlice {
		members = append(members, target.pointerAppendSlice())
	}
	if capabilities.Clear {
		members = append(members, target.pointerClear())
	}
	if capabilities.Data {
		members = append(members, target.pointerData())
	}
	return typescriptclass.Declaration(factory, []tsgo.ModifierLike{factory.ExportKeyword()}, factory.Identifier(name),
		[]tsgo.TypeParameterDeclaration{target.typeParameter()}, []tsgo.HeritageClause{factory.HeritageClause(
			tsgo.HeritageClauseTokenKindExtendsKeyword, []tsgo.ExpressionWithTypeArguments{
				factory.ExpressionWithTypeArguments(factory.Identifier(sliceName), []tsgo.TypeNode{target.typeT()}),
			})}, members)
}

func (b pointerSliceBuilder) pointerConstructor() tsgo.ConstructorDeclaration {
	readonly := []tsgo.ModifierLike{b.factory.PrivateKeyword(), b.factory.ReadonlyKeyword()}
	return b.factory.ConstructorDeclaration(nil, nil, []tsgo.ParameterDeclaration{
		b.factory.ParameterDeclaration(readonly, nil, b.id("region"), nil, memoryview.RegionType(b.factory, b.typeT()), nil),
		b.factory.ParameterDeclaration(readonly, nil, b.id("count"), nil, b.integerInputType(), nil),
		b.factory.ParameterDeclaration(readonly, nil, b.id("extent"), nil, b.integerInputType(), nil),
	}, nil, b.factory.Block([]tsgo.Statement{b.factory.ExpressionStatement(b.factory.CallExpression(b.factory.SuperExpression(), nil, nil,
		[]tsgo.Expression{b.factory.NullLiteral(), b.number("0"), b.toNumber(b.id("count")), b.toNumber(b.id("extent"))}, tsgo.NodeFlagsNone))}, true))
}

func (b pointerSliceBuilder) pointerCount(member Member, field string) tsgo.MethodDeclaration {
	return b.method(nil, MemberName(member), nil, nil, b.integerInputType(), b.returnStatement(b.thisProperty(field)))
}

func (b pointerSliceBuilder) method(modifiers []tsgo.ModifierLike, name string, generics []tsgo.TypeParameterDeclaration,
	parameters []tsgo.ParameterDeclaration, result tsgo.TypeNode, statements ...tsgo.Statement) tsgo.MethodDeclaration {
	return b.builder.method(append(modifiers, b.factory.OverrideKeyword()), name, generics, parameters, result, statements...)
}

func (b pointerSliceBuilder) regionCall(name string, arguments ...tsgo.Expression) tsgo.Expression {
	return b.factory.CallExpression(b.id(name), nil, []tsgo.TypeNode{b.typeT()}, arguments, tsgo.NodeFlagsNone)
}

func (b pointerSliceBuilder) newPointerSlice(region, length, capacity tsgo.Expression) tsgo.Expression {
	return b.factory.NewExpression(b.id(b.pointerSliceName), []tsgo.TypeNode{b.typeT()}, []tsgo.Expression{region, length, capacity})
}

func (b pointerSliceBuilder) pointerAccess(member Member) tsgo.MethodDeclaration {
	parameters := []tsgo.ParameterDeclaration{b.parameter("index", b.integerInputType())}
	result := b.typeT()
	operation := "goRegionRead"
	arguments := []tsgo.Expression{b.thisProperty("region"), b.id("index")}
	if member == MemberSet {
		parameters = append(parameters, b.parameter("value", b.typeT()))
		arguments = append(arguments, b.id("value"))
		operation = "goRegionWrite"
	}
	if member == MemberAddress {
		operation = "goRegionAddress"
		result = b.pointerType(b.typeT())
	}
	return b.method(nil, MemberName(member), nil, parameters, result,
		b.factory.IfStatement(b.invalidPointerIndex(b.id("index"), b.thisProperty("count")), b.throwIndexBounds(b.id("index")), nil),
		b.returnStatement(b.regionCall(operation, arguments...)))
}

func (b pointerSliceBuilder) invalidPointerIndex(index, count tsgo.Expression) tsgo.Expression {
	return b.binary(b.binary(index, tsgo.BinaryOperatorLessThanToken, b.number("0")), tsgo.BinaryOperatorBarBarToken,
		b.binary(index, tsgo.BinaryOperatorGreaterThanEqualsToken, count))
}

func BuildFromRegion(factory tsgo.Factory, sliceName string) (tsgo.Statement, error) {
	contract, err := api.RuntimeContract(api.RuntimeSliceFromRegion)
	if err != nil {
		return nil, err
	}
	pointerContract, err := api.RuntimeContract(api.RuntimeSlicePointer)
	if err != nil {
		return nil, err
	}
	panicContract, err := api.RuntimeContract(api.RuntimePanic)
	if err != nil {
		return nil, err
	}
	target := builder{factory: factory, className: sliceName, panicName: panicContract.ExportedName()}
	region := target.id("region")
	length, capacity := target.id("length"), target.id("capacity")
	invalid := target.binary(target.binary(length, tsgo.BinaryOperatorLessThanToken, target.number("0")), tsgo.BinaryOperatorBarBarToken,
		target.binary(capacity, tsgo.BinaryOperatorLessThanToken, length))
	indexed := target.binary(target.property(region, memoryview.KindMember), tsgo.BinaryOperatorEqualsEqualsEqualsToken,
		factory.StringLiteral(memoryview.IndexedKind, tsgo.TokenFlagsNone))
	remaining := target.subtract(target.globalCall("BigInt", target.property(target.property(region, memoryview.ValuesMember), "length")),
		target.globalCall("BigInt", target.property(region, memoryview.OffsetMember)))
	return factory.FunctionDeclaration([]tsgo.ModifierLike{factory.ExportKeyword()}, nil, target.id(contract.ExportedName()),
		[]tsgo.TypeParameterDeclaration{target.typeParameter()}, []tsgo.ParameterDeclaration{
			target.parameter("region", memoryview.RegionType(factory, target.typeT())),
			target.parameter("length", target.integerInputType()), target.parameter("capacity", target.integerInputType()),
		}, target.sliceType(), factory.Block([]tsgo.Statement{
			factory.IfStatement(invalid, target.throwBounds(), nil),
			factory.IfStatement(indexed, factory.Block([]tsgo.Statement{
				factory.IfStatement(target.binary(capacity, tsgo.BinaryOperatorGreaterThanToken, remaining), target.throwBounds(), nil),
				target.returnStatement(factory.CallExpression(target.property(target.id(sliceName), ArrayViewMember), nil, []tsgo.TypeNode{target.typeT()},
					[]tsgo.Expression{target.property(region, memoryview.ValuesMember), target.property(region, memoryview.OffsetMember), target.toNumber(length), target.toNumber(capacity)}, tsgo.NodeFlagsNone)),
			}, true), nil),
			target.returnStatement(factory.NewExpression(target.id(pointerContract.ExportedName()), []tsgo.TypeNode{target.typeT()}, []tsgo.Expression{region, length, capacity})),
		}, true)), nil
}
