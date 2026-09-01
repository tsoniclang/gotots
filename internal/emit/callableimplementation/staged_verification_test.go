package callableimplementation

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	implementationcontract "github.com/tsoniclang/gotots/internal/contracts/implementation"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	"github.com/tsoniclang/gotots/internal/toolchain"
)

func TestStagedVerificationExactJoinsGeneratedAndManualCallables(t *testing.T) {
	fixture := newStagedVerificationFixture(t)
	verified, err := VerifyStagedGeneratedContracts(fixture.config(
		t,
		"function identity<T>(value: T): T { return value; }\n"+
			"function increment(candidate: number): number { return candidate + 1; }\n"+
			"export function addFast(candidate: number): number { return identity(increment(candidate)); }\n",
		[]string{"addFast"},
	))
	if err != nil {
		t.Fatal(err)
	}
	if len(verified) != 1 || verified[0].OutputPath() != "implementations/hot.ts" ||
		verified[0].SourceFile() == nil {
		t.Fatal("verified implementation module is incomplete")
	}
	if _, err := os.Stat(fixture.lastScratch); !os.IsNotExist(err) {
		t.Fatalf("successful verification scratch survived: %v", err)
	}
}

func TestStagedVerificationIgnoresNestedCallableParameterNames(t *testing.T) {
	fixture := newStagedVerificationFixtureWithProtocol(
		t,
		stagedGeneratedCallbackFunction(t),
	)
	verified, err := VerifyStagedGeneratedContracts(fixture.config(
		t,
		"export function addFast(callback: (candidate: number) => number): number { return callback(1); }\n",
		[]string{"addFast"},
	))
	if err != nil {
		t.Fatal(err)
	}
	if len(verified) != 1 {
		t.Fatalf("verified modules = %d, want 1", len(verified))
	}

	fixture = newStagedVerificationFixtureWithProtocol(
		t,
		stagedGeneratedCallbackFunction(t),
	)
	_, err = VerifyStagedGeneratedContracts(fixture.config(
		t,
		"export function addFast(callback: (candidate: string) => number): number { return callback(\"1\"); }\n",
		[]string{"addFast"},
	))
	if err == nil || !strings.Contains(err.Error(), "differs from implementation") {
		t.Fatalf("nested callable type mutation error = %v", err)
	}
}

func TestStagedVerificationRejectsCallableAndModuleMutations(t *testing.T) {
	t.Run("wrong checked signature", func(t *testing.T) {
		fixture := newStagedVerificationFixture(t)
		_, err := VerifyStagedGeneratedContracts(fixture.config(
			t,
			"export function addFast(value: string): number { return value.length; }\n",
			[]string{"addFast"},
		))
		if err == nil || !strings.Contains(err.Error(), "differs from implementation") {
			t.Fatalf("wrong callable signature error = %v", err)
		}
	})

	t.Run("extra authored export", func(t *testing.T) {
		fixture := newStagedVerificationFixture(t)
		_, err := VerifyStagedGeneratedContracts(fixture.config(
			t,
			"export function addFast(value: number): number { return value; }\n"+
				"export function extra(): number { return 0; }\n",
			[]string{"addFast"},
		))
		if err == nil || !strings.Contains(err.Error(), "exports") {
			t.Fatalf("extra export error = %v", err)
		}
	})

	for _, dynamicType := range []string{"any", "unknown"} {
		t.Run("forbidden "+dynamicType, func(t *testing.T) {
			fixture := newStagedVerificationFixture(t)
			_, err := VerifyStagedGeneratedContracts(fixture.config(
				t,
				"export function addFast(value: "+dynamicType+"): number { return 0; }\n",
				[]string{"addFast"},
			))
			if err == nil || !strings.Contains(err.Error(), "explicit-"+dynamicType) {
				t.Fatalf("forbidden %s error = %v", dynamicType, err)
			}
		})
	}

	t.Run("source digest drift", func(t *testing.T) {
		fixture := newStagedVerificationFixture(t)
		config := fixture.config(
			t,
			"export function addFast(value: number): number { return value; }\n",
			[]string{"addFast"},
		)
		if err := os.WriteFile(
			config.Modules[0].sourcePath,
			[]byte("export function addFast(value: number): number { return value + 1; }\n"),
			0o600,
		); err != nil {
			t.Fatal(err)
		}
		_, err := VerifyStagedGeneratedContracts(config)
		if err == nil || !strings.Contains(err.Error(), "source digest changed") {
			t.Fatalf("source digest mutation error = %v", err)
		}
	})
}

func TestStagedVerificationRejectsEveryAuthoredSourceEscape(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		source    string
		violation string
	}{
		{
			name: "side effect import",
			source: "import \"../modules/app.js\";\n" +
				"export function addFast(value: number): number { return value; }\n",
			violation: "side-effect-import",
		},
		{
			name: "top level call",
			source: "function initialize(): void {}\ninitialize();\n" +
				"export function addFast(value: number): number { return value; }\n",
			violation: "executable-top-level",
		},
		{
			name: "top level variable",
			source: "const seed = 1;\n" +
				"export function addFast(value: number): number { return value + seed; }\n",
			violation: "executable-top-level",
		},
		{
			name: "bodyless function",
			source: "declare function helper(value: number): number;\n" +
				"export function addFast(value: number): number { return value; }\n",
			violation: "bodyless-function",
		},
		{
			name:      "as assertion",
			source:    "export function addFast(value: number): number { return \"wrong\" as never; }\n",
			violation: "unchecked-type-assertion",
		},
		{
			name:      "angle assertion",
			source:    "export function addFast(value: number): number { return <never>\"wrong\"; }\n",
			violation: "unchecked-type-assertion",
		},
		{
			name: "non-null assertion",
			source: "export function addFast(value: number): number {\n" +
				"  const candidate: number | undefined = value;\n" +
				"  return candidate!;\n}\n",
			violation: "non-null-assertion",
		},
		{
			name: "ignore directive",
			source: "export function addFast(value: number): number {\n" +
				"  // @ts-ignore\n  return value.missing;\n}\n",
			violation: "diagnostic-suppression",
		},
		{
			name: "nocheck directive",
			source: "// @ts-nocheck\n" +
				"export function addFast(value: number): number { return value; }\n",
			violation: "diagnostic-suppression",
		},
		{
			name: "expect error directive",
			source: "export function addFast(value: number): number {\n" +
				"  // @ts-expect-error\n  return value.missing;\n}\n",
			violation: "diagnostic-suppression",
		},
		{
			name: "suppression lexeme envelope",
			source: "function mention(): string { return \"@ts-ignore\"; }\n" +
				"export function addFast(value: number): number { void mention; return value; }\n",
			violation: "diagnostic-suppression",
		},
		{
			name: "inferred any",
			source: "export function addFast(value: number): number {\n" +
				"  return JSON.parse(String(value));\n}\n",
			violation: "inferred-any",
		},
		{
			name: "direct broad call inferred any argument",
			source: "export function addFast(value: number): number {\n" +
				"  return globalThis.Number(JSON.parse(String(value)));\n}\n",
			violation: "inferred-any",
		},
		{
			name: "nested inferred any",
			source: "type Dynamic = ReturnType<typeof JSON.parse>;\n" +
				"export function addFast(values: Dynamic[]): number { return values.length; }\n",
			violation: "inferred-any",
		},
		{
			name: "return-only inferred any",
			source: "type Dynamic = ReturnType<typeof JSON.parse>;\n" +
				"export function addFast(value: number): Dynamic { throw new Error(String(value)); }\n",
			violation: "inferred-any",
		},
		{
			name: "callable return inferred any",
			source: "export function addFast(value: number): number {\n" +
				"  const parser = JSON.parse; void parser; return value;\n}\n",
			violation: "inferred-any",
		},
		{
			name: "inferred unknown",
			source: "export function addFast(value: number): number {\n" +
				"  try { return value; } catch (failure) { void failure; return value; }\n}\n",
			violation: "inferred-unknown",
		},
		{
			name: "nested inferred unknown",
			source: "type Dynamic = ReturnType<<T>() => T>;\n" +
				"export function addFast(values: Dynamic[]): number { return values.length; }\n",
			violation: "inferred-unknown",
		},
		{
			name: "return-only inferred unknown",
			source: "type Dynamic = ReturnType<<T>() => T>;\n" +
				"export function addFast(value: number): Dynamic { throw new Error(String(value)); }\n",
			violation: "inferred-unknown",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := newStagedVerificationFixture(t)
			_, err := VerifyStagedGeneratedContracts(fixture.config(
				t,
				testCase.source,
				[]string{"addFast"},
			))
			if err == nil || !strings.Contains(err.Error(), testCase.violation) {
				t.Fatalf("%s error = %v", testCase.violation, err)
			}
		})
	}
}

func TestStagedVerificationUsesOnlyCertifiedDeclarationSources(t *testing.T) {
	t.Run("executable source is rejected", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "runtime-contract.ts")
		if _, err := implementationcontract.NewCertificationSource(path, strings.Repeat("0", 64)); err == nil {
			t.Fatal("executable certification source was accepted")
		}
	})

	t.Run("ambient unknown declaration enables exact authored call", func(t *testing.T) {
		fixture := newStagedVerificationFixture(t)
		config, certificationPath := fixture.configWithCertificationSource(t)
		verified, err := VerifyStagedGeneratedContracts(config)
		if err != nil {
			t.Fatal(err)
		}
		if len(verified) != 1 {
			t.Fatalf("verified modules = %d, want 1", len(verified))
		}
		if _, err := os.Stat(certificationPath); err != nil {
			t.Fatalf("certification source was modified: %v", err)
		}
	})

	t.Run("ambient explicit any is rejected", func(t *testing.T) {
		fixture := newStagedVerificationFixture(t)
		config, certificationPath := fixture.configWithCertificationSource(t)
		payload := []byte(
			"declare module \"@fixture/runtime.js\" {\n" +
				"  export function identity(value: any): number;\n" +
				"}\n",
		)
		if err := replaceCertificationSource(
			config.Modules,
			certificationPath,
			payload,
		); err != nil {
			t.Fatal(err)
		}
		_, err := VerifyStagedGeneratedContracts(config)
		if err == nil || !strings.Contains(err.Error(), "explicit-any") {
			t.Fatalf("certification any policy error = %v", err)
		}
	})

	t.Run("declaration digest drift fails", func(t *testing.T) {
		fixture := newStagedVerificationFixture(t)
		config, certificationPath := fixture.configWithCertificationSource(t)
		if err := os.WriteFile(
			certificationPath,
			[]byte("declare module \"@fixture/runtime.js\" {}\n"),
			0o600,
		); err != nil {
			t.Fatal(err)
		}
		_, err := VerifyStagedGeneratedContracts(config)
		if err == nil || !strings.Contains(err.Error(), "source digest changed") {
			t.Fatalf("certification source mutation error = %v", err)
		}
	})

	t.Run("declaration source policy fails", func(t *testing.T) {
		fixture := newStagedVerificationFixture(t)
		config, certificationPath := fixture.configWithCertificationSource(t)
		payload := []byte(
			"// @ts-nocheck\n" +
				"declare module \"@fixture/runtime.js\" {\n" +
				"  export function identity(value: number): number;\n" +
				"}\n",
		)
		if err := replaceCertificationSource(
			config.Modules,
			certificationPath,
			payload,
		); err != nil {
			t.Fatal(err)
		}
		_, err := VerifyStagedGeneratedContracts(config)
		if err == nil || !strings.Contains(err.Error(), "diagnostic-suppression") {
			t.Fatalf("certification source policy error = %v", err)
		}
	})
}

type stagedVerificationFixture struct {
	repository  string
	root        string
	tool        tsgo.Tool
	generated   StagedTarget
	lastScratch string
}

func newStagedVerificationFixture(t *testing.T) *stagedVerificationFixture {
	t.Helper()
	return newStagedVerificationFixtureWithProtocol(t, stagedGeneratedFunction(t))
}

func newStagedVerificationFixtureWithProtocol(
	t *testing.T,
	protocol []byte,
) *stagedVerificationFixture {
	t.Helper()
	repository, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	selectedGo, err := toolchain.ResolveGo(
		"",
		filepath.Join(repository, ".temp", "cache", "toolchain-tests"),
	)
	if err != nil {
		t.Fatal(err)
	}
	tool, err := tsgo.ResolveTool(selectedGo, repository, "")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	protocolPath := filepath.Join(root, "generated.ast")
	if err := os.WriteFile(protocolPath, protocol, 0o600); err != nil {
		t.Fatal(err)
	}
	generated, err := NewStagedTarget(
		"modules/app.ts",
		protocolPath,
		sha256.Sum256(protocol),
	)
	if err != nil {
		t.Fatal(err)
	}
	return &stagedVerificationFixture{
		repository: repository,
		root:       root,
		tool:       tool,
		generated:  generated,
	}
}

func stagedGeneratedCallbackFunction(t *testing.T) []byte {
	t.Helper()
	factory := tsgo.NewFactory()
	numberType := factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword)
	callbackParameter := factory.ParameterDeclaration(
		nil,
		nil,
		factory.Identifier("generated"),
		nil,
		numberType,
		nil,
	)
	parameter := factory.ParameterDeclaration(
		nil,
		nil,
		factory.Identifier("callback"),
		nil,
		factory.FunctionTypeNode(
			nil,
			[]tsgo.ParameterDeclaration{callbackParameter},
			numberType,
		),
		nil,
	)
	declaration := factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier("Add"),
		nil,
		[]tsgo.ParameterDeclaration{parameter},
		numberType,
		factory.Block(
			[]tsgo.Statement{factory.ReturnStatement(
				factory.NumericLiteral("0", tsgo.TokenFlagsNone),
			)},
			true,
		),
	)
	path, err := tsgo.NewPath("/modules/app.ts")
	if err != nil {
		t.Fatal(err)
	}
	source := factory.SourceFile(
		[]tsgo.Statement{declaration},
		factory.EndOfFile(),
		tsgo.SourceFileData{
			FileName:        path,
			Path:            path,
			LanguageVariant: tsgo.LanguageVariantStandard,
			ScriptKind:      tsgo.ScriptKindTS,
		},
	)
	payload, err := tsgo.EncodeSourceFile(source)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func (f *stagedVerificationFixture) config(
	t *testing.T,
	source string,
	exports []string,
) StagedVerificationConfig {
	t.Helper()
	sourcePath := filepath.Join(f.root, "hot.ts")
	payload := []byte(source)
	if err := os.WriteFile(sourcePath, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(payload)
	module, err := NewStagedModule(
		sourcePath,
		"implementations/hot.ts",
		hex.EncodeToString(digest[:]),
		exports,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	generated, err := NewGeneratedModuleTarget(
		"example.test/app|kind=5|receiver=|name=Add",
		VariantSource,
		"modules/app.ts",
		"Add",
	)
	if err != nil {
		t.Fatal(err)
	}
	callable, err := NewStagedCallable(
		generated.SourceIdentity(),
		"func(value int) int|params=value|results=",
		strings.Repeat("0", 64),
		VariantSource,
		module.outputPath,
		"addFast",
		generated,
	)
	if err != nil {
		t.Fatal(err)
	}
	f.lastScratch = filepath.Join(f.root, "scratch-"+strings.ReplaceAll(t.Name(), "/", "-"))
	return StagedVerificationConfig{
		RepositoryRoot: f.repository,
		ScratchRoot:    f.lastScratch,
		TSGoTool:       f.tool,
		Generated:      []StagedTarget{f.generated},
		Modules:        []StagedModule{module},
		Callables:      []StagedCallable{callable},
	}
}

func (f *stagedVerificationFixture) configWithCertificationSource(
	t *testing.T,
) (StagedVerificationConfig, string) {
	t.Helper()
	config := f.config(
		t,
		"import { identity } from \"@fixture/runtime.js\";\n"+
			"export function addFast(value: number): number { return identity(value); }\n",
		[]string{"addFast"},
	)
	certificationPath := filepath.Join(f.root, "runtime-contract.d.ts")
	payload := []byte(
		"declare module \"@fixture/runtime.js\" {\n" +
			"  export function identity(value: unknown): number;\n" +
			"}\n",
	)
	if err := os.WriteFile(certificationPath, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(payload)
	certificationSource, err := implementationcontract.NewCertificationSource(
		certificationPath,
		hex.EncodeToString(digest[:]),
	)
	if err != nil {
		t.Fatal(err)
	}
	module := config.Modules[0]
	module, err = NewStagedModule(
		module.sourcePath,
		module.outputPath,
		module.sourceDigest,
		module.exports,
		[]implementationcontract.CertificationSource{certificationSource},
	)
	if err != nil {
		t.Fatal(err)
	}
	config.Modules[0] = module
	return config, certificationPath
}

func replaceCertificationSource(
	modules []StagedModule,
	certificationPath string,
	payload []byte,
) error {
	if len(modules) != 1 {
		return fmt.Errorf("staged module denominator is %d, want 1", len(modules))
	}
	if err := os.WriteFile(certificationPath, payload, 0o600); err != nil {
		return err
	}
	digest := sha256.Sum256(payload)
	changed, err := implementationcontract.NewCertificationSource(
		certificationPath,
		hex.EncodeToString(digest[:]),
	)
	if err != nil {
		return err
	}
	module := modules[0]
	module, err = NewStagedModule(
		module.sourcePath,
		module.outputPath,
		module.sourceDigest,
		module.exports,
		[]implementationcontract.CertificationSource{changed},
	)
	if err != nil {
		return err
	}
	modules[0] = module
	return nil
}

func stagedGeneratedFunction(t *testing.T) []byte {
	t.Helper()
	factory := tsgo.NewFactory()
	numberType := factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword)
	parameter := factory.ParameterDeclaration(
		nil,
		nil,
		factory.Identifier("value"),
		nil,
		numberType,
		nil,
	)
	declaration := factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier("Add"),
		nil,
		[]tsgo.ParameterDeclaration{parameter},
		numberType,
		factory.Block(
			[]tsgo.Statement{factory.ReturnStatement(factory.Identifier("value"))},
			true,
		),
	)
	path, err := tsgo.NewPath("/modules/app.ts")
	if err != nil {
		t.Fatal(err)
	}
	source := factory.SourceFile(
		[]tsgo.Statement{declaration},
		factory.EndOfFile(),
		tsgo.SourceFileData{
			FileName:        path,
			Path:            path,
			LanguageVariant: tsgo.LanguageVariantStandard,
			ScriptKind:      tsgo.ScriptKindTS,
		},
	)
	payload, err := tsgo.EncodeSourceFile(source)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}
