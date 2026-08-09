package artifact

import (
	"slices"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestSourceContractExportsOnlyExactSourceBinding(t *testing.T) {
	factory := tsgo.NewFactory()
	resultType := factory.KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindVoidKeyword,
	)
	contract, err := ProjectSourceContract(
		factory,
		"Run",
		nil,
		[]tsgo.Statement{
			factory.FunctionDeclaration(
				[]tsgo.ModifierLike{factory.ExportKeyword()},
				nil,
				factory.Identifier("Run"),
				nil,
				nil,
				resultType,
				factory.Block(nil, true),
			),
			factory.FunctionDeclaration(
				[]tsgo.ModifierLike{factory.ExportKeyword()},
				nil,
				factory.Identifier("Run$deferred"),
				nil,
				nil,
				resultType,
				factory.Block(nil, true),
			),
			factory.TypeAliasDeclaration(
				[]tsgo.ModifierLike{factory.ExportKeyword()},
				factory.Identifier("Run$Storage"),
				nil,
				factory.TypeLiteralNode(nil),
			),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	exports, ok := contract.ExportedBindings()
	if !ok || len(exports) != 1 || exports[0] != "Run" {
		t.Fatalf("source package exports = %v, present=%t", exports, ok)
	}
}

func TestSourceContractDoesNotPublishKernelAsSourceBinding(t *testing.T) {
	factory := tsgo.NewFactory()
	contract, err := ProjectSourceContract(
		factory,
		"Apply",
		nil,
		[]tsgo.Statement{factory.FunctionDeclaration(
			[]tsgo.ModifierLike{factory.ExportKeyword()},
			nil,
			factory.Identifier("Apply$kernel"),
			nil,
			nil,
			factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindVoidKeyword,
			),
			factory.Block(nil, true),
		)},
	)
	if err != nil {
		t.Fatal(err)
	}
	exports, ok := contract.ExportedBindings()
	if !ok || len(exports) != 0 {
		t.Fatalf("kernel-only source package exports = %v, present=%t", exports, ok)
	}
}

func TestSourceContractPublishesExplicitAdditionalBindings(t *testing.T) {
	factory := tsgo.NewFactory()
	declaration := factory.VariableStatement(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{factory.VariableDeclaration(
				factory.Identifier("Width$int32"),
				nil,
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindNumberKeyword,
				),
				factory.NumericLiteral("1", tsgo.TokenFlagsNone),
			)},
			tsgo.NodeFlagsConst,
		),
	)
	contract, err := ProjectSourceContract(
		factory,
		"Width",
		[]string{"Width$int32"},
		[]tsgo.Statement{declaration},
	)
	if err != nil {
		t.Fatal(err)
	}
	exports, ok := contract.ExportedBindings()
	if !ok || len(exports) != 1 || exports[0] != "Width$int32" {
		t.Fatalf("constant projection exports = %v, present=%t", exports, ok)
	}

	if _, err := ProjectSourceContract(
		factory,
		"Width",
		[]string{"Width$uint64"},
		[]tsgo.Statement{declaration},
	); err == nil {
		t.Fatal("absent additional package binding was accepted")
	}
}

func TestSourceContractPublishesInterfaceRuntimeBindings(t *testing.T) {
	factory := tsgo.NewFactory()
	name := "Reader"
	contract, err := ProjectSourceContract(
		factory,
		name,
		[]string{
			name + api.InterfaceContractSuffix,
			name + api.InterfaceGuardSuffix,
		},
		[]tsgo.Statement{
			factory.InterfaceDeclaration(
				[]tsgo.ModifierLike{factory.ExportKeyword()},
				factory.Identifier(name),
				nil,
				nil,
				nil,
			),
			factory.VariableStatement(
				[]tsgo.ModifierLike{factory.ExportKeyword()},
				factory.VariableDeclarationList(
					[]tsgo.VariableDeclaration{factory.VariableDeclaration(
						factory.Identifier(name+api.InterfaceContractSuffix),
						nil,
						factory.ArrayTypeNode(factory.KeywordTypeNode(
							tsgo.KeywordTypeSyntaxKindUnknownKeyword,
						)),
						factory.ArrayLiteralExpression(nil, false),
					)},
					tsgo.NodeFlagsConst,
				),
			),
			factory.FunctionDeclaration(
				[]tsgo.ModifierLike{factory.ExportKeyword()},
				nil,
				factory.Identifier(name+api.InterfaceGuardSuffix),
				nil,
				nil,
				factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword),
				factory.Block(nil, true),
			),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	exports, ok := contract.ExportedBindings()
	want := []string{"Reader", "Reader$contract", "Reader$is"}
	if !ok || !slices.Equal(exports, want) {
		t.Fatalf("interface source package exports = %v, want %v", exports, want)
	}
}

func TestSourceContractPublishesEnumTypeAndValueBinding(t *testing.T) {
	factory := tsgo.NewFactory()
	contract, err := ProjectSourceContract(
		factory,
		"Flags",
		nil,
		[]tsgo.Statement{factory.EnumDeclaration(
			[]tsgo.ModifierLike{factory.ExportKeyword()},
			factory.Identifier("Flags"),
			[]tsgo.EnumMember{factory.EnumMember(
				factory.Identifier("$goType"),
				factory.NumericLiteral("1", tsgo.TokenFlagsNone),
			)},
		)},
	)
	if err != nil {
		t.Fatal(err)
	}
	exports, ok := contract.ExportedBindings()
	if !ok || len(exports) != 1 || exports[0] != "Flags" {
		t.Fatalf("enum source package exports = %v, present=%t", exports, ok)
	}
}
