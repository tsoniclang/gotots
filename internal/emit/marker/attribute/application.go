package attribute

import (
	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type MemberKind uint8

const (
	MemberInvalid MemberKind = iota
	MemberProperty
	MemberMethod
)

func (k MemberKind) selector() (string, bool) {
	switch k {
	case MemberProperty:
		return "property", true
	case MemberMethod:
		return "method", true
	default:
		return "", false
	}
}

func Apply(
	context api.Context,
	target tsgo.TypeNode,
	fact api.RuntimeSymbol,
	arguments ...tsgo.Expression,
) (api.StatementEmission, error) {
	if target == nil {
		return api.StatementEmission{}, &Error{Reason: "target type is nil"}
	}
	attribute, err := context.Names().TsonicCore(tsoniccore.SymbolAttribute)
	if err != nil {
		return api.StatementEmission{}, err
	}
	factType, err := context.Names().Runtime(fact, api.ImportPhaseValue)
	if err != nil {
		return api.StatementEmission{}, err
	}
	builder := context.Factory().CallExpression(
		attribute.Expression(context.Factory()),
		nil,
		[]tsgo.TypeNode{target},
		nil,
		tsgo.NodeFlagsNone,
	)
	add := context.Factory().PropertyAccessExpression(
		builder,
		nil,
		context.Factory().Identifier("add"),
		tsgo.NodeFlagsNone,
	)
	values := make([]tsgo.Expression, 0, len(arguments)+1)
	values = append(values, factType.Expression(context.Factory()))
	values = append(values, arguments...)
	statement := context.Factory().ExpressionStatement(
		context.Factory().CallExpression(
			add,
			nil,
			nil,
			values,
			tsgo.NodeFlagsNone,
		),
	)
	return api.NewStatementEmission(
		[]tsgo.Statement{statement},
		api.CombineRequests(
			attribute.Requests(),
			factType.Requests(),
		),
	)
}

func ApplyMember(
	context api.Context,
	target tsgo.TypeNode,
	kind MemberKind,
	member string,
	fact api.RuntimeSymbol,
	arguments ...tsgo.Expression,
) (api.StatementEmission, error) {
	selector, ok := kind.selector()
	if target == nil || !ok || member == "" {
		return api.StatementEmission{}, &Error{Reason: "member target is invalid"}
	}
	attribute, err := context.Names().TsonicCore(tsoniccore.SymbolAttribute)
	if err != nil {
		return api.StatementEmission{}, err
	}
	factType, err := context.Names().Runtime(fact, api.ImportPhaseValue)
	if err != nil {
		return api.StatementEmission{}, err
	}
	factory := context.Factory()
	builder := factory.CallExpression(
		attribute.Expression(factory),
		nil,
		[]tsgo.TypeNode{target},
		nil,
		tsgo.NodeFlagsNone,
	)
	selected := factory.CallExpression(
		factory.PropertyAccessExpression(
			builder,
			nil,
			factory.Identifier(selector),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{memberSelector(factory, member)},
		tsgo.NodeFlagsNone,
	)
	add := factory.PropertyAccessExpression(
		selected,
		nil,
		factory.Identifier("add"),
		tsgo.NodeFlagsNone,
	)
	values := make([]tsgo.Expression, 0, len(arguments)+1)
	values = append(values, factType.Expression(factory))
	values = append(values, arguments...)
	statement := factory.ExpressionStatement(factory.CallExpression(
		add,
		nil,
		nil,
		values,
		tsgo.NodeFlagsNone,
	))
	return api.NewStatementEmission(
		[]tsgo.Statement{statement},
		api.CombineRequests(attribute.Requests(), factType.Requests()),
	)
}

func memberSelector(
	factory tsgo.Factory,
	member string,
) tsgo.ArrowFunction {
	const target = "$go$attributeTarget"
	return factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{factory.ParameterDeclaration(
			nil,
			nil,
			factory.Identifier(target),
			nil,
			nil,
			nil,
		)},
		nil,
		factory.EqualsGreaterThanToken(),
		factory.PropertyAccessExpression(
			factory.Identifier(target),
			nil,
			factory.Identifier(member),
			tsgo.NodeFlagsNone,
		),
	)
}
