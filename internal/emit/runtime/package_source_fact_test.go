package runtime

import (
	"slices"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/runtime/sourcefact"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestEveryRuntimeDeclarationHasOneClosedSourceFactDisposition(t *testing.T) {
	valid := 0
	for symbol := api.RuntimeSymbol(1); symbol <= api.RuntimeSourceImplementationFact; symbol++ {
		contract, err := api.RuntimeContract(symbol)
		if err != nil {
			continue
		}
		valid++
		facts := sourcefact.FactSymbols(symbol)
		if contract.Module() == api.RuntimeModuleSourceFact {
			if len(facts) != 0 {
				t.Fatalf("source-fact declaration %d recursively acquired facts %v", symbol, facts)
			}
			continue
		}
		primary, selected := sourcefact.FactSymbol(symbol)
		if !selected || len(facts) == 0 || !slices.Contains(facts, primary) {
			t.Fatalf("runtime declaration %d has no exact source-fact disposition", symbol)
		}
		if contract.TypeUsable() &&
			primary != api.RuntimeSourceOperationFact &&
			!slices.Contains(facts, api.RuntimeSourceOperationFact) {
			t.Fatalf("runtime type declaration %d omits its operation disposition", symbol)
		}
	}
	if valid == 0 {
		t.Fatal("runtime source-fact denominator was empty")
	}
}

func TestScalarPackageUsesSharedPrimitiveWithoutRedundantCompanion(t *testing.T) {
	assembled, err := AssemblePackage(
		tsgo.NewFactory(),
		testScalarABI(t, api.IntegerRepresentationNumber),
		nil,
		[]api.PrimitiveAlias{api.PrimitiveInt32},
	)
	if err != nil {
		t.Fatal(err)
	}
	source := runtimePackageFile(t, assembled, "runtime/scalars.ts")
	alias := scalarAlias(t, source, "int32")
	reference, ok := alias.Type().(tsgo.TypeReferenceNode)
	if !ok {
		t.Fatalf("int32 underlying type = %T, want type reference", alias.Type())
	}
	identifier, ok := reference.TypeName().(tsgo.Identifier)
	if !ok || identifier.Text() != "$go$core$int32" {
		t.Fatalf("int32 underlying reference = %T %v", reference.TypeName(), reference.TypeName())
	}
	if got := importedBinding(source, "@tsonic/core/types.js", "int32"); got != "$go$core$int32" {
		t.Fatalf("shared int32 import = %q", got)
	}
	for _, statement := range source.Statements() {
		if _, annotation := statement.(tsgo.ExpressionStatement); annotation {
			t.Fatal("fixed int32 emitted a redundant Go companion fact")
		}
	}
}

func TestScalarPackageAddsOnlyRequiredGoCompanions(t *testing.T) {
	assembled, err := AssemblePackage(
		tsgo.NewFactory(),
		testScalarABI(t, api.IntegerRepresentationBigInt),
		nil,
		[]api.PrimitiveAlias{
			api.PrimitiveInt32,
			api.PrimitiveInt,
			api.PrimitiveString,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	source := runtimePackageFile(t, assembled, "runtime/scalars.ts")
	annotationCount := 0
	for _, statement := range source.Statements() {
		if _, annotation := statement.(tsgo.ExpressionStatement); annotation {
			annotationCount++
		}
	}
	if annotationCount != 2 {
		t.Fatalf("Go scalar companion annotations = %d, want native-int and Go string", annotationCount)
	}
	if importedBinding(source, "@tsonic/core/lang.js", "attribute") == "" {
		t.Fatal("Go scalar companions omit the canonical attribute import")
	}
	if importedBinding(source, "./source-fact.js", "GoBasicFact") == "" {
		t.Fatal("Go scalar companions omit their exact fact declaration")
	}
}

func runtimePackageFile(
	t *testing.T,
	assembled Package,
	outputPath string,
) tsgo.SourceFile {
	t.Helper()
	for _, file := range assembled.Files() {
		if file.OutputPath() == outputPath {
			return file.SourceFile()
		}
	}
	t.Fatalf("runtime package omits %s", outputPath)
	return nil
}

func scalarAlias(
	t *testing.T,
	source tsgo.SourceFile,
	name string,
) tsgo.TypeAliasDeclaration {
	t.Helper()
	for _, statement := range source.Statements() {
		alias, ok := statement.(tsgo.TypeAliasDeclaration)
		if ok && alias.Name().Text() == name {
			return alias
		}
	}
	t.Fatalf("scalar package omits alias %s", name)
	return nil
}

func importedBinding(
	source tsgo.SourceFile,
	moduleName string,
	exportedName string,
) string {
	for _, statement := range source.Statements() {
		declaration, ok := statement.(tsgo.ImportDeclaration)
		if !ok {
			continue
		}
		module, ok := declaration.ModuleSpecifier().(tsgo.StringLiteral)
		if !ok || module.Text() != moduleName || declaration.ImportClause() == nil {
			continue
		}
		bindings, ok := declaration.ImportClause().NamedBindings().(tsgo.NamedImports)
		if !ok {
			continue
		}
		for _, binding := range bindings.Elements() {
			imported := binding.Name().Text()
			if binding.PropertyName() != nil {
				imported = binding.PropertyName().(tsgo.Identifier).Text()
			}
			if imported == exportedName {
				return binding.Name().Text()
			}
		}
	}
	return ""
}
