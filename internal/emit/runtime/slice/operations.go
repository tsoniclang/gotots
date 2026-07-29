package slice

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func BuildOperation(
	factory tsgo.Factory,
	symbol api.RuntimeSymbol,
) (tsgo.Statement, error) {
	contract, err := api.RuntimeContract(symbol)
	if err != nil {
		return nil, err
	}
	sliceContract, err := api.RuntimeContract(api.RuntimeSlice)
	if err != nil {
		return nil, err
	}
	target := builder{
		factory:   factory,
		className: sliceContract.ExportedName(),
	}
	switch symbol {
	case api.RuntimeSliceStorage:
		return target.storageExport(contract.ExportedName()), nil
	case api.RuntimeSliceAppendSlice:
		return target.appendSliceExport(contract.ExportedName()), nil
	case api.RuntimeSliceClear:
		return target.clearExport(contract.ExportedName()), nil
	default:
		return nil, &api.RuntimeSymbolError{Symbol: symbol}
	}
}

func (b builder) storageExport(name string) tsgo.FunctionDeclaration {
	return b.exportedFunction(
		name,
		[]tsgo.ParameterDeclaration{
			b.parameter("length", b.integerInputType()),
			b.parameter("capacity", b.optionalIntegerInputType()),
		},
		b.sliceType(),
		b.factory.ReturnStatement(
			b.factory.CallExpression(
				b.property(b.id(b.className), StorageAllocateMember),
				nil,
				[]tsgo.TypeNode{b.typeT()},
				[]tsgo.Expression{
					b.id("length"),
					b.id("capacity"),
				},
				tsgo.NodeFlagsNone,
			),
		),
	)
}

func (b builder) appendSliceExport(name string) tsgo.FunctionDeclaration {
	return b.exportedFunction(
		name,
		[]tsgo.ParameterDeclaration{
			b.parameter("target", b.sliceType()),
			b.parameter("source", b.sliceType()),
			b.parameter("zero", b.typeT()),
		},
		b.sliceType(),
		b.factory.ReturnStatement(b.call(
			b.id("target"),
			MemberName(MemberAppendSlice),
			b.id("zero"),
			b.id("source"),
		)),
	)
}

func (b builder) clearExport(name string) tsgo.FunctionDeclaration {
	return b.exportedFunction(
		name,
		[]tsgo.ParameterDeclaration{
			b.parameter("source", b.sliceType()),
			b.parameter("zero", b.typeT()),
		},
		b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword),
		b.factory.ExpressionStatement(b.call(
			b.id("source"),
			MemberName(MemberClear),
			b.id("zero"),
		)),
	)
}

func (b builder) exportedFunction(
	name string,
	parameters []tsgo.ParameterDeclaration,
	result tsgo.TypeNode,
	statement tsgo.Statement,
) tsgo.FunctionDeclaration {
	return b.factory.FunctionDeclaration(
		[]tsgo.ModifierLike{b.factory.ExportKeyword()},
		nil,
		b.id(name),
		[]tsgo.TypeParameterDeclaration{b.typeParameter()},
		parameters,
		result,
		b.factory.Block([]tsgo.Statement{statement}, true),
	)
}
