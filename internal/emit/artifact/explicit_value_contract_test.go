package artifact

import (
	"bytes"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestExplicitValueContractProjectsOneTypedObservableBinding(t *testing.T) {
	factory := tsgo.NewFactory()
	statement := factory.VariableStatement(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{factory.VariableDeclaration(
				factory.Identifier("registry"),
				nil,
				nil,
				factory.NewExpression(
					factory.Identifier("Registry"),
					[]tsgo.TypeNode{factory.KeywordTypeNode(
						tsgo.KeywordTypeSyntaxKindStringKeyword,
					)},
					nil,
				),
			)},
			tsgo.NodeFlagsConst,
		),
	)
	targetType := factory.TypeReferenceNode(
		factory.Identifier("Registry"),
		[]tsgo.TypeNode{factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindStringKeyword,
		)},
	)
	contract, err := ProjectExplicitValueContract(
		factory,
		statement,
		targetType,
	)
	if err != nil {
		t.Fatal(err)
	}
	exports, ok := contract.ExportedBindings()
	if !ok || len(exports) != 1 || exports[0] != "registry" {
		t.Fatalf("explicit value exports = %v/%v", exports, ok)
	}
	if !contract.hasFacet(api.ArtifactFacetValueSurface) ||
		!contract.hasFacet(api.ArtifactFacetExportSurface) {
		t.Fatal("explicit value contract omitted an observable facet")
	}

	changedInitializer := factory.VariableStatement(
		statement.Modifiers(),
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{factory.VariableDeclaration(
				factory.Identifier("registry"),
				nil,
				nil,
				factory.NewExpression(
					factory.Identifier("DifferentRegistry"),
					nil,
					nil,
				),
			)},
			tsgo.NodeFlagsConst,
		),
	)
	implementationOnly, err := ProjectExplicitValueContract(
		factory,
		changedInitializer,
		targetType,
	)
	if err != nil {
		t.Fatal(err)
	}
	currentValue, _ := contract.facet(api.ArtifactFacetValueSurface)
	implementationValue, _ := implementationOnly.facet(
		api.ArtifactFacetValueSurface,
	)
	if !bytes.Equal(currentValue, implementationValue) {
		t.Fatal("initializer-only change altered the observable contract")
	}

	changedType, err := ProjectExplicitValueContract(
		factory,
		statement,
		factory.TypeReferenceNode(
			factory.Identifier("Registry"),
			[]tsgo.TypeNode{factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindNumberKeyword,
			)},
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	changedValue, _ := changedType.facet(api.ArtifactFacetValueSurface)
	if bytes.Equal(currentValue, changedValue) {
		t.Fatal("explicit type change did not alter the value facet")
	}
}

func TestExplicitValueContractRejectsAmbiguousImplementations(t *testing.T) {
	factory := tsgo.NewFactory()
	targetType := factory.KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindStringKeyword,
	)
	cases := []struct {
		name      string
		statement tsgo.Statement
		typeNode  tsgo.TypeNode
	}{
		{
			name: "missing explicit contract type",
			statement: inferredValue(
				factory,
				[]tsgo.BindingName{factory.Identifier("value")},
			),
		},
		{
			name: "multiple declarations",
			statement: inferredValue(
				factory,
				[]tsgo.BindingName{
					factory.Identifier("first"),
					factory.Identifier("second"),
				},
			),
			typeNode: targetType,
		},
		{
			name: "annotated implementation",
			statement: factory.VariableStatement(
				nil,
				factory.VariableDeclarationList(
					[]tsgo.VariableDeclaration{factory.VariableDeclaration(
						factory.Identifier("value"),
						nil,
						targetType,
						factory.StringLiteral("value", 0),
					)},
					tsgo.NodeFlagsConst,
				),
			),
			typeNode: targetType,
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ProjectExplicitValueContract(
				factory,
				test.statement,
				test.typeNode,
			); err == nil {
				t.Fatal("invalid explicit value contract was accepted")
			}
		})
	}
}

func inferredValue(
	factory tsgo.Factory,
	names []tsgo.BindingName,
) tsgo.Statement {
	declarations := make([]tsgo.VariableDeclaration, len(names))
	for index, name := range names {
		declarations[index] = factory.VariableDeclaration(
			name,
			nil,
			nil,
			factory.StringLiteral("value", 0),
		)
	}
	return factory.VariableStatement(
		nil,
		factory.VariableDeclarationList(
			declarations,
			tsgo.NodeFlagsConst,
		),
	)
}
