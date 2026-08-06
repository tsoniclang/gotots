package artifact

import (
	"bytes"
	"errors"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestObservableInterfaceAndAliasUseInstanceTypeFacet(t *testing.T) {
	factory := tsgo.NewFactory()
	numberType := factory.KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindNumberKeyword,
	)
	interfaceDeclaration := factory.InterfaceDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.Identifier("Readable"),
		nil,
		nil,
		[]tsgo.TypeElement{factory.MethodSignatureDeclaration(
			nil,
			factory.Identifier("value"),
			nil,
			nil,
			nil,
			numberType,
		)},
	)
	aliasDeclaration := factory.TypeAliasDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.Identifier("Count"),
		nil,
		numberType,
	)
	contract, err := ProjectContract(
		factory,
		[]tsgo.Statement{interfaceDeclaration, aliasDeclaration},
	)
	if err != nil {
		t.Fatal(err)
	}
	instanceSurface, present := contract.facet(
		api.ArtifactFacetInstanceTypeSurface,
	)
	if contract.present !=
		uint16(1)<<api.ArtifactFacetInstanceTypeSurface|
			uint16(1)<<api.ArtifactFacetExportSurface ||
		!present ||
		len(instanceSurface) == 0 {
		t.Fatalf("contract facets = %#v", contract)
	}

	changed, err := ProjectContract(
		factory,
		[]tsgo.Statement{
			factory.InterfaceDeclaration(
				interfaceDeclaration.Modifiers(),
				interfaceDeclaration.Name(),
				nil,
				nil,
				[]tsgo.TypeElement{factory.MethodSignatureDeclaration(
					nil,
					factory.Identifier("value"),
					nil,
					nil,
					nil,
					factory.KeywordTypeNode(
						tsgo.KeywordTypeSyntaxKindStringKeyword,
					),
				)},
			),
			aliasDeclaration,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(
		artifactFacetBytes(contract, api.ArtifactFacetInstanceTypeSurface),
		artifactFacetBytes(changed, api.ArtifactFacetInstanceTypeSurface),
	) {
		t.Fatal("interface member change did not change instance type surface")
	}
}

func TestObservableContractRejectsInferenceDependentSurface(t *testing.T) {
	factory := tsgo.NewFactory()
	for name, statements := range map[string][]tsgo.Statement{
		"function": {
			factory.FunctionDeclaration(
				nil,
				nil,
				factory.Identifier("value"),
				nil,
				nil,
				nil,
				factory.Block([]tsgo.Statement{
					factory.ReturnStatement(
						factory.NumericLiteral("1", tsgo.TokenFlagsNone),
					),
				}, true),
			),
		},
		"variable": {
			factory.VariableStatement(
				nil,
				factory.VariableDeclarationList(
					[]tsgo.VariableDeclaration{
						factory.VariableDeclaration(
							factory.Identifier("value"),
							nil,
							nil,
							factory.NumericLiteral("1", tsgo.TokenFlagsNone),
						),
					},
					tsgo.NodeFlagsConst,
				),
			),
		},
		"property": {
			factory.ClassDeclaration(
				nil,
				factory.Identifier("Record"),
				nil,
				nil,
				[]tsgo.ClassElement{factory.PropertyDeclaration(
					nil,
					factory.Identifier("value"),
					nil,
					nil,
					factory.NumericLiteral("1", tsgo.TokenFlagsNone),
				)},
			),
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ProjectContract(factory, statements)
			var contractError *ContractError
			if !errors.As(err, &contractError) {
				t.Fatalf("error = %#v, want ContractError", err)
			}
		})
	}
}

func TestObservableContractAllowsOneTypeAndValueBindingWithTheSameName(
	t *testing.T,
) {
	factory := tsgo.NewFactory()
	name := factory.Identifier("State")
	valueType := factory.KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindNumberKeyword,
	)
	contract, err := ProjectContract(factory, []tsgo.Statement{
		factory.TypeAliasDeclaration(
			[]tsgo.ModifierLike{factory.ExportKeyword()},
			name,
			nil,
			valueType,
		),
		factory.VariableStatement(
			[]tsgo.ModifierLike{factory.ExportKeyword()},
			factory.VariableDeclarationList(
				[]tsgo.VariableDeclaration{factory.VariableDeclaration(
					name,
					nil,
					valueType,
					factory.NumericLiteral("1", tsgo.TokenFlagsNone),
				)},
				tsgo.NodeFlagsConst,
			),
		),
	})
	if err != nil {
		t.Fatal(err)
	}
	exports, ok := contract.ExportedBindings()
	if !ok || len(exports) != 1 || exports[0] != "State" {
		t.Fatalf("split declaration exports = %v, present=%t", exports, ok)
	}
}

func TestObservableContractRejectsDuplicateDeclarationSpace(t *testing.T) {
	factory := tsgo.NewFactory()
	alias := func() tsgo.Statement {
		return factory.TypeAliasDeclaration(
			[]tsgo.ModifierLike{factory.ExportKeyword()},
			factory.Identifier("State"),
			nil,
			factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindNumberKeyword,
			),
		)
	}
	_, err := ProjectContract(factory, []tsgo.Statement{alias(), alias()})
	var contractError *ContractError
	if !errors.As(err, &contractError) ||
		contractError.Reason != "export binding duplicates a target declaration space" {
		t.Fatalf("duplicate declaration error = %#v", err)
	}
}

func artifactTestFunction(
	factory tsgo.Factory,
	name string,
	body []tsgo.Statement,
) tsgo.FunctionDeclaration {
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(name),
		nil,
		nil,
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword),
		factory.Block(body, true),
	)
}

func artifactTestClass(
	factory tsgo.Factory,
	bodyValue string,
	fieldType string,
) tsgo.ClassDeclaration {
	var targetFieldType tsgo.TypeNode
	switch fieldType {
	case "number":
		targetFieldType = factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindNumberKeyword,
		)
	case "string":
		targetFieldType = factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindStringKeyword,
		)
	default:
		panic("unknown test field type")
	}
	return factory.ClassDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.Identifier("Record"),
		nil,
		nil,
		[]tsgo.ClassElement{
			factory.ConstructorDeclaration(
				nil,
				nil,
				nil,
				nil,
				factory.Block(nil, true),
			),
			factory.PropertyDeclaration(
				nil,
				factory.Identifier("value"),
				nil,
				targetFieldType,
				factory.NumericLiteral("0", tsgo.TokenFlagsNone),
			),
			factory.MethodDeclaration(
				[]tsgo.ModifierLike{factory.StaticKeyword()},
				nil,
				factory.Identifier("make"),
				nil,
				nil,
				nil,
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindNumberKeyword,
				),
				factory.Block(
					[]tsgo.Statement{factory.ReturnStatement(
						factory.Identifier(bodyValue),
					)},
					true,
				),
			),
		},
	)
}

func assertArtifactFacetEqual(
	t *testing.T,
	left Contract,
	right Contract,
	facets ...api.ArtifactFacet,
) {
	t.Helper()
	for _, facet := range facets {
		leftValue, leftOK := left.facet(facet)
		rightValue, rightOK := right.facet(facet)
		if leftOK != rightOK || !bytes.Equal(leftValue, rightValue) {
			t.Fatalf("facet %v changed unexpectedly", facet)
		}
	}
}

func artifactFacetBytes(
	contract Contract,
	facet api.ArtifactFacet,
) []byte {
	value, _ := contract.facet(facet)
	return value
}
