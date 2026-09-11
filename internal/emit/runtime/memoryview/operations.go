package memoryview

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/runtime/indexedstorage"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func BuildOperation(factory tsgo.Factory, symbol api.RuntimeSymbol, name, panicName string) (tsgo.Statement, error) {
	if symbol == api.RuntimeStorageRegion {
		return BuildRegion(factory, name), nil
	}
	if symbol != api.RuntimeRegionAddress && symbol != api.RuntimeRegionRead && symbol != api.RuntimeRegionWrite && symbol != api.RuntimeRegionView {
		return nil, &api.RuntimeSymbolError{Symbol: symbol}
	}
	element := factory.TypeReferenceNode(factory.Identifier("Element"), nil)
	region := factory.Identifier("region")
	index := factory.Identifier("index")
	member := func(value tsgo.Expression, name string) tsgo.Expression {
		return factory.PropertyAccessExpression(value, nil, factory.Identifier(name), tsgo.NodeFlagsNone)
	}
	call := func(name string, arguments ...tsgo.Expression) tsgo.Expression {
		return factory.CallExpression(factory.Identifier(name), nil, nil, arguments, tsgo.NodeFlagsNone)
	}
	binary := func(left tsgo.Expression, operator tsgo.BinaryOperator, right tsgo.Expression) tsgo.Expression {
		return factory.BinaryExpression(nil, left, nil, factory.BinaryOperatorToken(operator), right)
	}
	offset := member(region, OffsetMember)
	numericIndex := binary(offset, tsgo.BinaryOperatorPlusToken, call("Number", index))
	indexed := binary(member(region, KindMember), tsgo.BinaryOperatorEqualsEqualsEqualsToken,
		factory.StringLiteral(IndexedKind, tsgo.TokenFlagsNone))
	pointerIndex := binary(call("BigInt", offset), tsgo.BinaryOperatorPlusToken, call("BigInt", index))
	address := factory.CallExpression(member(region, LocateMember), nil, nil, []tsgo.Expression{pointerIndex}, tsgo.NodeFlagsNone)
	var direct, indirect tsgo.Expression
	var resultType tsgo.TypeNode
	parameters := []tsgo.ParameterDeclaration{
		factory.ParameterDeclaration(nil, nil, region, nil, RegionType(factory, element), nil),
		factory.ParameterDeclaration(nil, nil, index, nil, CountType(factory), nil),
	}
	switch symbol {
	case api.RuntimeRegionAddress:
		resultType = PointerType(factory, element)
		direct = factory.CallExpression(factory.Identifier("addressOf"), nil, []tsgo.TypeNode{element}, []tsgo.Expression{
			factory.ElementAccessExpression(member(region, ValuesMember), nil, numericIndex, tsgo.NodeFlagsNone),
		}, tsgo.NodeFlagsNone)
		indirect = address
	case api.RuntimeRegionRead:
		resultType = element
		direct = indexedstorage.Element(factory, panicName, member(region, ValuesMember), numericIndex, element)
		indirect = factory.CallExpression(factory.Identifier("loadPointer"), nil, []tsgo.TypeNode{element}, []tsgo.Expression{address}, tsgo.NodeFlagsNone)
	case api.RuntimeRegionWrite:
		resultType = element
		value := factory.Identifier("value")
		parameters = append(parameters, factory.ParameterDeclaration(nil, nil, value, nil, element, nil))
		direct = binary(factory.ElementAccessExpression(member(region, ValuesMember), nil, numericIndex, tsgo.NodeFlagsNone), tsgo.BinaryOperatorEqualsToken, value)
		indirect = factory.CallExpression(factory.Identifier("storePointer"), nil, []tsgo.TypeNode{element}, []tsgo.Expression{address, value}, tsgo.NodeFlagsNone)
	case api.RuntimeRegionView:
		resultType = RegionType(factory, element)
		direct = Indexed(factory, element, member(region, ValuesMember), numericIndex)
		indirect = Pointer(factory, element, member(region, LocateMember), pointerIndex)
	}
	indirectBody := []tsgo.Statement{factory.ReturnStatement(indirect)}
	if symbol == api.RuntimeRegionWrite {
		indirectBody = []tsgo.Statement{factory.ExpressionStatement(indirect), factory.ReturnStatement(factory.Identifier("value"))}
	}
	statements := []tsgo.Statement{factory.IfStatement(indexed, factory.Block([]tsgo.Statement{factory.ReturnStatement(direct)}, true), nil)}
	statements = append(statements, indirectBody...)
	return factory.FunctionDeclaration([]tsgo.ModifierLike{factory.ExportKeyword()}, nil, factory.Identifier(name),
		[]tsgo.TypeParameterDeclaration{factory.TypeParameterDeclaration(nil, factory.Identifier("Element"), nil, nil, nil)},
		parameters, resultType, factory.Block(statements, true)), nil
}
