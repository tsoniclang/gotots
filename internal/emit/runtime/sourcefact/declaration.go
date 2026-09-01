package sourcefact

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/typescriptclass"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Build(
	factory tsgo.Factory,
	symbol api.RuntimeSymbol,
	name string,
) (tsgo.Statement, error) {
	brand, ok := brandName(symbol)
	if !ok || name == "" {
		return nil, &Error{Symbol: symbol}
	}
	return typescriptclass.Declaration(
		factory,
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.Identifier(name),
		nil,
		nil,
		[]tsgo.ClassElement{factory.PropertyDeclaration(
			[]tsgo.ModifierLike{
				factory.DeclareKeyword(),
				factory.PrivateKeyword(),
				factory.ReadonlyKeyword(),
			},
			factory.Identifier(brand),
			nil,
			factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword),
			nil,
		)},
	), nil
}

func brandName(symbol api.RuntimeSymbol) (string, bool) {
	switch symbol {
	case api.RuntimeSourceCompilationFact:
		return "$go$compilationFact", true
	case api.RuntimeSourceDeclarationFact:
		return "$go$declarationFact", true
	case api.RuntimeSourceBasicFact:
		return "$go$basicFact", true
	case api.RuntimeSourceAggregateFact:
		return "$go$aggregateFact", true
	case api.RuntimeSourceCallableFact:
		return "$go$callableFact", true
	case api.RuntimeSourceInterfaceFact:
		return "$go$interfaceFact", true
	case api.RuntimeSourceStorageFact:
		return "$go$storageFact", true
	case api.RuntimeSourceOperationFact:
		return "$go$operationFact", true
	case api.RuntimeSourceImplementationFact:
		return "$go$implementationFact", true
	default:
		return "", false
	}
}
