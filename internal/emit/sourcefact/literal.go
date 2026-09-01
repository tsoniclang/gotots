package sourcefact

import (
	"strconv"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func text(factory tsgo.Factory, value string) tsgo.Expression {
	return factory.StringLiteral(value, tsgo.TokenFlagsNone)
}

func count(factory tsgo.Factory, value int) tsgo.Expression {
	return factory.NumericLiteral(strconv.Itoa(value), tsgo.TokenFlagsNone)
}

func truth(factory tsgo.Factory, value bool) tsgo.Expression {
	if value {
		return factory.TrueLiteral()
	}
	return factory.FalseLiteral()
}

func genericType(
	factory tsgo.Factory,
	name string,
	parameters int,
) tsgo.TypeNode {
	arguments := make([]tsgo.TypeNode, 0, parameters)
	for range parameters {
		arguments = append(arguments, factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindNeverKeyword,
		))
	}
	return factory.TypeReferenceNode(factory.Identifier(name), arguments)
}
