package unsafeoperation

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type builder struct {
	factory           tsgo.Factory
	pointerName       string
	pointerRegionName string
	sliceName         string
	sliceRegionName   string
	denseIndexName    string
}

func Build(factory tsgo.Factory, symbol api.RuntimeSymbol) (tsgo.Statement, error) {
	pointer, err := api.RuntimeContract(api.RuntimePointer)
	if err != nil {
		return nil, err
	}
	pointerRegion, err := api.RuntimeContract(api.RuntimePointerRegion)
	if err != nil {
		return nil, err
	}
	slice, err := api.RuntimeContract(api.RuntimeSlice)
	if err != nil {
		return nil, err
	}
	sliceRegion, err := api.RuntimeContract(api.RuntimeSliceRegion)
	if err != nil {
		return nil, err
	}
	denseIndex, err := api.RuntimeContract(api.RuntimeDenseIndex)
	if err != nil {
		return nil, err
	}
	contract, err := api.RuntimeContract(symbol)
	if err != nil {
		return nil, err
	}
	b := builder{
		factory:           factory,
		pointerName:       pointer.ExportedName(),
		pointerRegionName: pointerRegion.ExportedName(),
		sliceName:         slice.ExportedName(),
		sliceRegionName:   sliceRegion.ExportedName(),
		denseIndexName:    denseIndex.ExportedName(),
	}
	switch symbol {
	case api.RuntimeUnsafeString:
		return b.stringFunction(contract.ExportedName()), nil
	case api.RuntimeUnsafeSlice:
		return b.sliceFunction(contract.ExportedName()), nil
	case api.RuntimeUnsafeStringData:
		return b.stringDataFunction(contract.ExportedName()), nil
	case api.RuntimeUnsafeSliceData:
		return b.sliceDataFunction(contract.ExportedName()), nil
	default:
		return nil, &api.RuntimeSymbolError{Symbol: symbol}
	}
}

func (b builder) id(name string) tsgo.Identifier {
	return b.factory.Identifier(name)
}

func (b builder) typeReference(name string, arguments ...tsgo.TypeNode) tsgo.TypeReferenceNode {
	return b.factory.TypeReferenceNode(b.id(name), arguments)
}

func (b builder) typeI() tsgo.TypeNode { return b.typeReference("I") }
func (b builder) typeL() tsgo.TypeNode { return b.typeReference("L") }
func (b builder) typeS() tsgo.TypeNode { return b.typeReference("S") }

func (b builder) numberType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword)
}

func (b builder) bigintType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBigIntKeyword)
}

func (b builder) integerType() tsgo.TypeNode {
	return b.factory.UnionTypeNode([]tsgo.TypeNode{b.numberType(), b.bigintType()})
}

func (b builder) stringType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindStringKeyword)
}

func (b builder) undefinedType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindUndefinedKeyword)
}

func (b builder) pointerType(logical, storage tsgo.TypeNode) tsgo.TypeNode {
	return b.typeReference(b.pointerName, logical, storage)
}

func (b builder) optionalPointerType(logical, storage tsgo.TypeNode) tsgo.TypeNode {
	return b.factory.UnionTypeNode([]tsgo.TypeNode{
		b.pointerType(logical, storage),
		b.undefinedType(),
	})
}

func (b builder) sliceType(storage tsgo.TypeNode) tsgo.TypeNode {
	return b.typeReference(b.sliceName, storage)
}

func (b builder) typeParameter(name string, constraint tsgo.TypeNode) tsgo.TypeParameterDeclaration {
	return b.factory.TypeParameterDeclaration(nil, b.id(name), constraint, nil, nil)
}

func (b builder) parameter(name string, target tsgo.TypeNode) tsgo.ParameterDeclaration {
	return b.factory.ParameterDeclaration(nil, nil, b.id(name), nil, target, nil)
}

func (b builder) property(receiver tsgo.Expression, name string) tsgo.PropertyAccessExpression {
	return b.factory.PropertyAccessExpression(receiver, nil, b.id(name), tsgo.NodeFlagsNone)
}

func (b builder) call(
	receiver tsgo.Expression,
	name string,
	typeArguments []tsgo.TypeNode,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return b.factory.CallExpression(
		b.property(receiver, name),
		nil,
		typeArguments,
		arguments,
		tsgo.NodeFlagsNone,
	)
}

func (b builder) directCall(
	name string,
	typeArguments []tsgo.TypeNode,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return b.factory.CallExpression(
		b.id(name),
		nil,
		typeArguments,
		arguments,
		tsgo.NodeFlagsNone,
	)
}

func (b builder) globalCall(
	name string,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return b.factory.CallExpression(
		b.property(b.id("globalThis"), name),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
}

func (b builder) binary(left tsgo.Expression, operator tsgo.BinaryOperator, right tsgo.Expression) tsgo.BinaryExpression {
	return b.factory.BinaryExpression(nil, left, nil, b.factory.BinaryOperatorToken(operator), right)
}

func (b builder) variable(flags tsgo.NodeFlags, name string, target tsgo.TypeNode, value tsgo.Expression) tsgo.VariableStatement {
	return b.factory.VariableStatement(nil, b.factory.VariableDeclarationList(
		[]tsgo.VariableDeclaration{b.factory.VariableDeclaration(b.id(name), nil, target, value)},
		flags,
	))
}

func (b builder) number(value string) tsgo.NumericLiteral {
	return b.factory.NumericLiteral(value, tsgo.TokenFlagsNone)
}

func (b builder) undefined() tsgo.VoidExpression {
	return b.factory.VoidExpression(b.number("0"))
}

func (b builder) regionType(storage tsgo.TypeNode) tsgo.TypeNode {
	return b.factory.TypeOperatorNode(
		tsgo.TypeOperatorNodeOperatorKindReadonlyKeyword,
		b.factory.TupleTypeNode([]tsgo.TypeNode{
			b.factory.ArrayTypeNode(storage),
			b.numberType(),
		}),
	)
}

func (b builder) optionalRegionType(storage tsgo.TypeNode) tsgo.TypeNode {
	return b.factory.UnionTypeNode([]tsgo.TypeNode{b.regionType(storage), b.undefinedType()})
}

func (b builder) pointerRegion(logical, storage tsgo.TypeNode, pointer, length tsgo.Expression) tsgo.CallExpression {
	return b.directCall(
		b.pointerRegionName,
		[]tsgo.TypeNode{logical, storage},
		pointer,
		length,
	)
}

func (b builder) loop(limit tsgo.Expression, statements ...tsgo.Statement) tsgo.ForStatement {
	index := b.id("index")
	return b.factory.ForStatement(
		b.factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{b.factory.VariableDeclaration(index, nil, nil, b.number("0"))},
			tsgo.NodeFlagsLet,
		),
		b.binary(index, tsgo.BinaryOperatorLessThanToken, limit),
		b.factory.PostfixUnaryExpression(index, tsgo.PostfixUnaryExpressionOperatorKindPlusPlusToken),
		b.factory.Block(statements, true),
	)
}
