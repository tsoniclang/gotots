package sourcepackage

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestRebindConsumersUsesAssemblyAndExactPrivateContracts(t *testing.T) {
	const (
		assemblyPath = "target/packages/hash/_root/package.ts"
		hasherPath   = "target/modules/hash/_root/hasher.ts"
		utilsPath    = "target/modules/hash/_root/utils.ts"
		supportPath  = "target/support/interface.ts"
	)
	paths := Paths{
		assembly: assemblyPath,
		owned: map[string]struct{}{
			assemblyPath: {}, hasherPath: {}, utilsPath: {},
		},
		bySourceFile: map[string]string{
			"hasher.go": hasherPath,
			"utils.go":  utilsPath,
		},
	}
	factory := tsgo.NewFactory()
	consumer := sourceImplementationConsumer(
		t,
		factory,
		supportPath,
		[]tsgo.Statement{
			sourceImplementationImport(factory, "../modules/hash/_root/hasher.js", 0, "Hasher"),
			sourceImplementationImport(factory, "../modules/hash/_root/utils.js", tsgo.ImportPhaseModifierSyntaxKindTypeKeyword, "str$Storage"),
		},
	)
	rebound, err := rebindConsumers(
		factory,
		paths,
		map[string]struct{}{"Hasher": {}},
		map[string][]string{utilsPath: {"str$Storage"}},
		[]Consumer{consumer},
	)
	if err != nil {
		t.Fatal(err)
	}
	statements := rebound[0].SourceFile.Statements()
	first := statements[0].(tsgo.ImportDeclaration)
	if got := first.ModuleSpecifier().(tsgo.StringLiteral).Text(); got != "../packages/hash/_root/package.js" {
		t.Fatalf("public import = %q", got)
	}
	second := statements[1].(tsgo.ImportDeclaration)
	if got := second.ModuleSpecifier().(tsgo.StringLiteral).Text(); got != "../modules/hash/_root/utils.js" {
		t.Fatalf("private import = %q", got)
	}

	for name, provided := range map[string]map[string][]string{
		"missing": {},
		"extra-module": {
			utilsPath:  {"str$Storage"},
			hasherPath: {"Hasher"},
		},
		"extra-export": {utilsPath: {"Other", "str$Storage"}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := rebindConsumers(
				factory,
				paths,
				map[string]struct{}{"Hasher": {}},
				provided,
				[]Consumer{consumer},
			); err == nil {
				t.Fatal("private contract mismatch passed")
			}
		})
	}

	privateValue := sourceImplementationConsumer(
		t,
		factory,
		supportPath,
		[]tsgo.Statement{
			sourceImplementationImport(factory, "../modules/hash/_root/utils.js", 0, "str$Storage"),
		},
	)
	if _, err := rebindConsumers(
		factory,
		paths,
		map[string]struct{}{"Hasher": {}},
		map[string][]string{utilsPath: {"str$Storage"}},
		[]Consumer{privateValue},
	); err == nil || !strings.Contains(err.Error(), "private value dependency") {
		t.Fatalf("private value dependency error = %v", err)
	}
}

func sourceImplementationImport(
	factory tsgo.Factory,
	module string,
	phase tsgo.ImportPhaseModifierSyntaxKind,
	name string,
) tsgo.Statement {
	return factory.ImportDeclaration(
		nil,
		factory.ImportClause(
			phase,
			nil,
			factory.NamedImports([]tsgo.ImportSpecifier{
				factory.ImportSpecifier(false, nil, factory.Identifier(name)),
			}),
		),
		factory.StringLiteral(module, tsgo.TokenFlagsNone),
		nil,
	)
}

func sourceImplementationConsumer(
	t *testing.T,
	factory tsgo.Factory,
	outputPath string,
	statements []tsgo.Statement,
) Consumer {
	t.Helper()
	filePath, err := tsgo.NewPath(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	return Consumer{
		OutputPath: outputPath,
		SourceFile: factory.SourceFile(
			statements,
			factory.EndOfFile(),
			tsgo.SourceFileData{
				FileName:        filePath,
				Path:            filePath,
				LanguageVariant: tsgo.LanguageVariantStandard,
				ScriptKind:      tsgo.ScriptKindTS,
			},
		),
	}
}
