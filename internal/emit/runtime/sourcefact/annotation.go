package sourcefact

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const runtimeMemberSchema = "gotots-go-runtime-member-fact-v1"
const runtimeDeclarationSchema = "gotots-go-runtime-declaration-fact-v2"

func Annotation(
	factory tsgo.Factory,
	symbolName string,
	factName string,
	identity string,
	symbol uint16,
	declaration tsgo.Statement,
) (tsgo.Statement, error) {
	return AnnotationWithArguments(
		factory,
		symbolName,
		factName,
		declaration,
		factory.StringLiteral(runtimeDeclarationSchema, tsgo.TokenFlagsNone),
		factory.StringLiteral(identity, tsgo.TokenFlagsNone),
		factory.NumericLiteral(fmt.Sprintf("%d", symbol), tsgo.TokenFlagsNone),
	)
}

func AnnotationWithArguments(
	factory tsgo.Factory,
	symbolName string,
	factName string,
	declaration tsgo.Statement,
	arguments ...tsgo.Expression,
) (tsgo.Statement, error) {
	if symbolName == "" || factName == "" || declaration == nil || len(arguments) == 0 {
		return nil, &AnnotationError{Reason: "annotation input is incomplete"}
	}
	target, err := annotationTarget(factory, symbolName, declaration)
	if err != nil {
		return nil, err
	}
	builder := factory.CallExpression(
		factory.Identifier("attribute"),
		nil,
		[]tsgo.TypeNode{target},
		nil,
		tsgo.NodeFlagsNone,
	)
	add := factory.PropertyAccessExpression(
		builder,
		nil,
		factory.Identifier("add"),
		tsgo.NodeFlagsNone,
	)
	values := make([]tsgo.Expression, 0, len(arguments)+1)
	values = append(values, factory.Identifier(factName))
	values = append(values, arguments...)
	return factory.ExpressionStatement(factory.CallExpression(
		add,
		nil,
		nil,
		values,
		tsgo.NodeFlagsNone,
	)), nil
}

func annotationTarget(
	factory tsgo.Factory,
	name string,
	declaration tsgo.Statement,
) (tsgo.TypeNode, error) {
	switch selected := declaration.(type) {
	case tsgo.InterfaceDeclaration:
		return genericReference(factory, name, len(selected.TypeParameters())), nil
	case tsgo.TypeAliasDeclaration:
		return genericReference(factory, name, len(selected.TypeParameters())), nil
	case tsgo.ClassDeclaration,
		tsgo.FunctionDeclaration,
		tsgo.VariableStatement:
		return factory.TypeQueryNode(factory.Identifier(name), nil), nil
	default:
		return nil, &AnnotationError{Reason: "runtime declaration has no supported attribute target"}
	}
}

func genericReference(
	factory tsgo.Factory,
	name string,
	count int,
) tsgo.TypeNode {
	arguments := make([]tsgo.TypeNode, 0, count)
	for range count {
		arguments = append(arguments, factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindNeverKeyword,
		))
	}
	return factory.TypeReferenceNode(factory.Identifier(name), arguments)
}

func MemberAnnotations(
	factory tsgo.Factory,
	symbolName string,
	factName string,
	identity string,
	symbol uint16,
	declaration tsgo.Statement,
) ([]tsgo.Statement, error) {
	members, err := declarationMembers(declaration)
	if err != nil {
		return nil, err
	}
	annotations := make([]tsgo.Statement, 0, len(members))
	for ordinal, member := range members {
		target, err := memberTarget(factory, symbolName, declaration, member.static)
		if err != nil {
			return nil, err
		}
		annotation, err := memberAnnotation(
			factory,
			target,
			member.kind,
			member.name,
			factName,
			factory.StringLiteral(runtimeMemberSchema, tsgo.TokenFlagsNone),
			factory.StringLiteral(identity, tsgo.TokenFlagsNone),
			factory.NumericLiteral(fmt.Sprintf("%d", symbol), tsgo.TokenFlagsNone),
			factory.StringLiteral(member.targetName(), tsgo.TokenFlagsNone),
			factory.StringLiteral(member.kind, tsgo.TokenFlagsNone),
			factory.StringLiteral(member.name, tsgo.TokenFlagsNone),
			factory.NumericLiteral(fmt.Sprintf("%d", ordinal), tsgo.TokenFlagsNone),
		)
		if err != nil {
			return nil, err
		}
		annotations = append(annotations, annotation)
	}
	return annotations, nil
}

func memberAnnotation(
	factory tsgo.Factory,
	target tsgo.TypeNode,
	kind string,
	member string,
	factName string,
	arguments ...tsgo.Expression,
) (tsgo.Statement, error) {
	if target == nil || member == "" || factName == "" ||
		(kind != "method" && kind != "property") {
		return nil, &AnnotationError{Reason: "runtime member annotation is invalid"}
	}
	builder := factory.CallExpression(
		factory.Identifier("attribute"),
		nil,
		[]tsgo.TypeNode{target},
		nil,
		tsgo.NodeFlagsNone,
	)
	selection := factory.CallExpression(
		factory.PropertyAccessExpression(
			builder,
			nil,
			factory.Identifier(kind),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{memberSelector(factory, member)},
		tsgo.NodeFlagsNone,
	)
	add := factory.PropertyAccessExpression(
		selection,
		nil,
		factory.Identifier("add"),
		tsgo.NodeFlagsNone,
	)
	values := make([]tsgo.Expression, 0, len(arguments)+1)
	values = append(values, factory.Identifier(factName))
	values = append(values, arguments...)
	return factory.ExpressionStatement(factory.CallExpression(
		add,
		nil,
		nil,
		values,
		tsgo.NodeFlagsNone,
	)), nil
}

func memberSelector(factory tsgo.Factory, member string) tsgo.ArrowFunction {
	const target = "$go$runtimeMember"
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

func memberTarget(
	factory tsgo.Factory,
	name string,
	declaration tsgo.Statement,
	static bool,
) (tsgo.TypeNode, error) {
	if static {
		return factory.TypeQueryNode(factory.Identifier(name), nil), nil
	}
	switch selected := declaration.(type) {
	case tsgo.ClassDeclaration:
		return genericReference(factory, name, len(selected.TypeParameters())), nil
	case tsgo.InterfaceDeclaration:
		return genericReference(factory, name, len(selected.TypeParameters())), nil
	default:
		return nil, &AnnotationError{Reason: "runtime member owner is not a class or interface"}
	}
}
