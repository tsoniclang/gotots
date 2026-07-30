package artifact

import (
	"bytes"
	"errors"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestObservableContractRejectsGenericEmptyButAcceptsTypedCoverage(t *testing.T) {
	factory := tsgo.NewFactory()
	if _, err := ProjectContract(factory, nil); err == nil {
		t.Fatal("generic empty materialized contract was accepted")
	}
	contract, err := ProjectCoverageContract(nil)
	if err != nil {
		t.Fatal(err)
	}
	if contract == nil || len(contract) != 0 {
		t.Fatalf("coverage contract = %#v, want explicit empty contract", contract)
	}
	statement := artifactTestFunction(factory, "value", nil)
	if _, err := ProjectCoverageContract(
		[]tsgo.Statement{statement},
	); err == nil {
		t.Fatal("coverage-only contract accepted a target declaration")
	}
}

func TestObservableFunctionContractIgnoresBody(t *testing.T) {
	factory := tsgo.NewFactory()
	first := artifactTestFunction(factory, "value", nil)
	second := artifactTestFunction(
		factory,
		"value",
		[]tsgo.Statement{factory.ReturnStatement(
			factory.NumericLiteral("1", tsgo.TokenFlagsNone),
		)},
	)
	firstContract, err := ProjectContract(factory, []tsgo.Statement{first})
	if err != nil {
		t.Fatal(err)
	}
	secondContract, err := ProjectContract(factory, []tsgo.Statement{second})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(
		firstContract[api.ArtifactFacetCallableSignature],
		secondContract[api.ArtifactFacetCallableSignature],
	) {
		t.Fatal("function body changed the observable callable signature")
	}

	changed := factory.FunctionDeclaration(
		first.Modifiers(),
		nil,
		first.Name(),
		nil,
		[]tsgo.ParameterDeclaration{factory.ParameterDeclaration(
			nil,
			nil,
			factory.Identifier("flag"),
			nil,
			factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword),
			nil,
		)},
		first.Type(),
		first.Body(),
	)
	changedContract, err := ProjectContract(
		factory,
		[]tsgo.Statement{changed},
	)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(
		firstContract[api.ArtifactFacetCallableSignature],
		changedContract[api.ArtifactFacetCallableSignature],
	) {
		t.Fatal("parameter change was absent from the callable signature")
	}
}

func TestObservableClassContractPartitionsStaticAndInstanceFacets(t *testing.T) {
	factory := tsgo.NewFactory()
	class := artifactTestClass(factory, "one", "number")
	baseline, err := ProjectContract(factory, []tsgo.Statement{class})
	if err != nil {
		t.Fatal(err)
	}

	bodyOnly := artifactTestClass(factory, "two", "number")
	bodyContract, err := ProjectContract(
		factory,
		[]tsgo.Statement{bodyOnly},
	)
	if err != nil {
		t.Fatal(err)
	}
	assertArtifactFacetEqual(
		t,
		baseline,
		bodyContract,
		api.ArtifactFacetConstructorSurface,
		api.ArtifactFacetInstanceTypeSurface,
		api.ArtifactFacetStaticSurface,
	)

	constructorMembers := class.Members()
	constructorMembers[0] = factory.ConstructorDeclaration(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{factory.ParameterDeclaration(
			nil,
			nil,
			factory.Identifier("value"),
			nil,
			factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindNumberKeyword,
			),
			nil,
		)},
		nil,
		factory.Block(nil, true),
	)
	constructorChanged := factory.ClassDeclaration(
		class.Modifiers(),
		class.Name(),
		class.TypeParameters(),
		class.HeritageClauses(),
		constructorMembers,
	)
	constructorContract, err := ProjectContract(
		factory,
		[]tsgo.Statement{constructorChanged},
	)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(
		baseline[api.ArtifactFacetConstructorSurface],
		constructorContract[api.ArtifactFacetConstructorSurface],
	) {
		t.Fatal("constructor parameter change did not change constructor surface")
	}
	assertArtifactFacetEqual(
		t,
		baseline,
		constructorContract,
		api.ArtifactFacetInstanceTypeSurface,
		api.ArtifactFacetStaticSurface,
	)

	instanceChanged := artifactTestClass(factory, "two", "string")
	instanceContract, err := ProjectContract(
		factory,
		[]tsgo.Statement{instanceChanged},
	)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(
		baseline[api.ArtifactFacetInstanceTypeSurface],
		instanceContract[api.ArtifactFacetInstanceTypeSurface],
	) {
		t.Fatal("instance field type change did not change instance surface")
	}
	assertArtifactFacetEqual(
		t,
		baseline,
		instanceContract,
		api.ArtifactFacetConstructorSurface,
		api.ArtifactFacetStaticSurface,
	)

	members := class.Members()
	members = append(members, factory.MethodDeclaration(
		[]tsgo.ModifierLike{factory.StaticKeyword()},
		nil,
		factory.Identifier("extra"),
		nil,
		nil,
		nil,
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword),
		factory.Block(nil, true),
	))
	staticChanged := factory.ClassDeclaration(
		class.Modifiers(),
		class.Name(),
		class.TypeParameters(),
		class.HeritageClauses(),
		members,
	)
	staticContract, err := ProjectContract(
		factory,
		[]tsgo.Statement{staticChanged},
	)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(
		baseline[api.ArtifactFacetStaticSurface],
		staticContract[api.ArtifactFacetStaticSurface],
	) {
		t.Fatal("static member addition did not change static surface")
	}
	assertArtifactFacetEqual(
		t,
		baseline,
		staticContract,
		api.ArtifactFacetConstructorSurface,
		api.ArtifactFacetInstanceTypeSurface,
	)
}

func TestClassMemberContractSeparatesSignatureFromImplementation(t *testing.T) {
	factory := tsgo.NewFactory()
	member := func(
		parameterType tsgo.TypeNode,
		bodyValue string,
	) tsgo.MethodDeclaration {
		return factory.MethodDeclaration(
			nil,
			nil,
			factory.Identifier("Read"),
			nil,
			nil,
			[]tsgo.ParameterDeclaration{factory.ParameterDeclaration(
				nil,
				nil,
				factory.Identifier("value"),
				nil,
				parameterType,
				nil,
			)},
			factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindNumberKeyword,
			),
			factory.Block(
				[]tsgo.Statement{factory.ReturnStatement(
					factory.Identifier(bodyValue),
				)},
				true,
			),
		)
	}
	numberType := factory.KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindNumberKeyword,
	)
	baseline, err := ProjectClassMemberContract(
		factory,
		[]tsgo.ClassElement{member(numberType, "first")},
	)
	if err != nil {
		t.Fatal(err)
	}
	bodyChanged, err := ProjectClassMemberContract(
		factory,
		[]tsgo.ClassElement{member(numberType, "second")},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(
		baseline[api.ArtifactFacetCallableSignature],
		bodyChanged[api.ArtifactFacetCallableSignature],
	) {
		t.Fatal("class-member body changed its callable signature")
	}
	if bytes.Equal(
		baseline[api.ArtifactFacetImplementation],
		bodyChanged[api.ArtifactFacetImplementation],
	) {
		t.Fatal("class-member body was absent from its implementation facet")
	}

	signatureChanged, err := ProjectClassMemberContract(
		factory,
		[]tsgo.ClassElement{member(
			factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindStringKeyword,
			),
			"first",
		)},
	)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(
		baseline[api.ArtifactFacetCallableSignature],
		signatureChanged[api.ArtifactFacetCallableSignature],
	) {
		t.Fatal("class-member parameter type did not change its callable signature")
	}
}

func TestObservableValueContractIgnoresInitializer(t *testing.T) {
	factory := tsgo.NewFactory()
	makeStatement := func(
		targetType tsgo.TypeNode,
		initializer tsgo.Expression,
	) tsgo.VariableStatement {
		return factory.VariableStatement(
			[]tsgo.ModifierLike{factory.ExportKeyword()},
			factory.VariableDeclarationList(
				[]tsgo.VariableDeclaration{factory.VariableDeclaration(
					factory.Identifier("Value"),
					nil,
					targetType,
					initializer,
				)},
				tsgo.NodeFlagsConst,
			),
		)
	}
	numberType := factory.KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindNumberKeyword,
	)
	first, err := ProjectContract(
		factory,
		[]tsgo.Statement{makeStatement(
			numberType,
			factory.NumericLiteral("1", tsgo.TokenFlagsNone),
		)},
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ProjectContract(
		factory,
		[]tsgo.Statement{makeStatement(
			numberType,
			factory.NumericLiteral("2", tsgo.TokenFlagsNone),
		)},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(
		first[api.ArtifactFacetValueSurface],
		second[api.ArtifactFacetValueSurface],
	) {
		t.Fatal("variable initializer changed the observable value surface")
	}
	changed, err := ProjectContract(
		factory,
		[]tsgo.Statement{makeStatement(
			factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindStringKeyword),
			factory.StringLiteral("value", tsgo.TokenFlagsNone),
		)},
	)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(
		first[api.ArtifactFacetValueSurface],
		changed[api.ArtifactFacetValueSurface],
	) {
		t.Fatal("variable type change was absent from the value surface")
	}
}

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
	if len(contract) != 1 ||
		len(contract[api.ArtifactFacetInstanceTypeSurface]) == 0 {
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
		contract[api.ArtifactFacetInstanceTypeSurface],
		changed[api.ArtifactFacetInstanceTypeSurface],
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
		if !bytes.Equal(left[facet], right[facet]) {
			t.Fatalf("facet %v changed unexpectedly", facet)
		}
	}
}
