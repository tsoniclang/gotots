package slice

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func BuildAggregateOperation(
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
	case api.RuntimeSliceNilWith:
		return target.aggregateNilExport(contract.ExportedName()), nil
	case api.RuntimeSliceMakeWith:
		return target.aggregateMakeExport(contract.ExportedName()), nil
	case api.RuntimeSliceLiteralWith:
		return target.aggregateLiteralExport(contract.ExportedName()), nil
	case api.RuntimeSliceAppendWith:
		return target.aggregateAppendExport(contract.ExportedName()), nil
	case api.RuntimeSliceAppendSliceWith:
		return target.aggregateAppendSliceExport(contract.ExportedName()), nil
	case api.RuntimeSliceCopyWith:
		return target.aggregateCopyExport(contract.ExportedName()), nil
	default:
		return nil, &api.RuntimeSymbolError{Symbol: symbol}
	}
}

func (b builder) aggregateAppendSliceExport(name string) tsgo.FunctionDeclaration {
	return b.aggregateExport(
		name,
		[]tsgo.ParameterDeclaration{
			b.parameter("source", b.sliceType()),
			b.parameter("zero", b.valueFactoryType()),
			b.parameter("copyValue", b.valueCopyType()),
			b.parameter("appended", b.sliceType()),
		},
		b.sliceType(),
		b.call(
			b.id("source"),
			MemberName(MemberAppendSliceWith),
			b.id("zero"),
			b.id("copyValue"),
			b.id("appended"),
		),
	)
}

func (b builder) aggregateNilExport(name string) tsgo.FunctionDeclaration {
	return b.aggregateExport(
		name,
		[]tsgo.ParameterDeclaration{
			b.parameter("zero", b.valueFactoryType()),
		},
		b.sliceType(),
		b.factory.CallExpression(
			b.property(b.id(b.className), aggregateNilMember),
			nil,
			[]tsgo.TypeNode{b.typeT()},
			[]tsgo.Expression{b.id("zero")},
			tsgo.NodeFlagsNone,
		),
	)
}

func (b builder) aggregateMakeExport(name string) tsgo.FunctionDeclaration {
	return b.aggregateExport(
		name,
		[]tsgo.ParameterDeclaration{
			b.parameter("length", b.integerInputType()),
			b.parameter("capacity", b.optionalIntegerInputType()),
			b.parameter("zero", b.valueFactoryType()),
		},
		b.sliceType(),
		b.factory.CallExpression(
			b.property(b.id(b.className), aggregateMakeMember),
			nil,
			[]tsgo.TypeNode{b.typeT()},
			[]tsgo.Expression{
				b.id("length"),
				b.id("capacity"),
				b.id("zero"),
			},
			tsgo.NodeFlagsNone,
		),
	)
}

func (b builder) aggregateAppendExport(name string) tsgo.FunctionDeclaration {
	values := b.factory.ParameterDeclaration(
		nil,
		b.factory.DotDotDotToken(),
		b.id("values"),
		nil,
		b.factory.ArrayTypeNode(b.typeT()),
		nil,
	)
	return b.aggregateExport(
		name,
		[]tsgo.ParameterDeclaration{
			b.parameter("source", b.sliceType()),
			b.parameter("zero", b.valueFactoryType()),
			b.parameter("copyValue", b.valueCopyType()),
			values,
		},
		b.sliceType(),
		b.factory.CallExpression(
			b.property(b.id("source"), aggregateAppendMember),
			nil,
			nil,
			[]tsgo.Expression{
				b.id("zero"),
				b.id("copyValue"),
				b.factory.SpreadElement(b.id("values")),
			},
			tsgo.NodeFlagsNone,
		),
	)
}

func (b builder) aggregateLiteralExport(name string) tsgo.FunctionDeclaration {
	values := b.factory.ParameterDeclaration(
		nil,
		b.factory.DotDotDotToken(),
		b.id("values"),
		nil,
		b.factory.ArrayTypeNode(b.typeT()),
		nil,
	)
	return b.aggregateExport(
		name,
		[]tsgo.ParameterDeclaration{
			b.parameter("zero", b.valueFactoryType()),
			values,
		},
		b.sliceType(),
		b.factory.CallExpression(
			b.property(b.id(b.className), aggregateLiteralMember),
			nil,
			[]tsgo.TypeNode{b.typeT()},
			[]tsgo.Expression{
				b.id("zero"),
				b.factory.SpreadElement(b.id("values")),
			},
			tsgo.NodeFlagsNone,
		),
	)
}

func (b builder) aggregateCopyExport(name string) tsgo.FunctionDeclaration {
	return b.aggregateExport(
		name,
		[]tsgo.ParameterDeclaration{
			b.parameter("target", b.sliceType()),
			b.parameter("source", b.sliceType()),
			b.parameter("copyValue", b.valueCopyType()),
		},
		b.numberType(),
		b.factory.CallExpression(
			b.property(b.id(b.className), aggregateCopyMember),
			nil,
			[]tsgo.TypeNode{b.typeT()},
			[]tsgo.Expression{
				b.id("target"),
				b.id("source"),
				b.id("copyValue"),
			},
			tsgo.NodeFlagsNone,
		),
	)
}

func (b builder) aggregateExport(
	name string,
	parameters []tsgo.ParameterDeclaration,
	result tsgo.TypeNode,
	value tsgo.Expression,
) tsgo.FunctionDeclaration {
	return b.factory.FunctionDeclaration(
		[]tsgo.ModifierLike{b.factory.ExportKeyword()},
		nil,
		b.id(name),
		[]tsgo.TypeParameterDeclaration{b.typeParameter()},
		parameters,
		result,
		b.factory.Block(
			[]tsgo.Statement{b.returnStatement(value)},
			true,
		),
	)
}
