package arrayvalue_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	arraymember "github.com/tsoniclang/gotots/internal/emit/runtime/array/member"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestArrayRuntimeContractAndFixtureDemandRemainSeparate(t *testing.T) {
	emission := compileArrayFixture(t)
	directory := t.TempDir()
	target := materializeArrayProgram(t, directory, emission)
	if strings.Contains(target.printed["runtime/array.ts"], "goArrayPacked") {
		t.Fatal("ordinary array fixture emitted the packed-array operation")
	}
	expected := memberNames(arraymember.All())
	runtime := runtimeMemberNames(t, emission)
	if err := exactJoinNames("runtime", expected, runtime); err != nil {
		t.Fatal(err)
	}

	fixtureDemand := cloneNames(expected)
	delete(fixtureDemand, arraymember.Length.Name())
	source := sourceMemberNames(
		t,
		target.printed[sourceOutputPath(target)],
		expected,
	)
	if err := exactJoinNames("fixture source", fixtureDemand, source); err != nil {
		t.Fatal(err)
	}

	mutatedRuntime := cloneNames(runtime)
	delete(mutatedRuntime, arraymember.Get.Name())
	mutatedRuntime["read"] = struct{}{}
	if exactJoinNames("runtime", expected, mutatedRuntime) == nil {
		t.Fatal("one-sided runtime member-name mutation passed the contract join")
	}

	mutatedSource := cloneNames(source)
	delete(mutatedSource, arraymember.Set.Name())
	mutatedSource["store"] = struct{}{}
	if exactJoinNames("fixture source", fixtureDemand, mutatedSource) == nil {
		t.Fatal("one-sided fixture-demand mutation passed the contract join")
	}
}

func TestArrayRuntimeTopLevelContractOwnsBothArtifactSides(t *testing.T) {
	emission := compileArrayFixture(t)
	contract, err := api.RuntimeContract(api.RuntimeArray)
	if err != nil {
		t.Fatal(err)
	}
	runtimeName := runtimeClassName(t, emission)
	sourceImports := sourceRuntimeImports(t, emission, contract.OutputPath())
	if err := exactJoinTopLevelName(
		contract.ExportedName(),
		runtimeName,
		sourceImports,
	); err != nil {
		t.Fatal(err)
	}
	if exactJoinTopLevelName(
		contract.ExportedName(),
		"DriftedArray",
		sourceImports,
	) == nil {
		t.Fatal("one-sided runtime class-name mutation passed the contract join")
	}
	if exactJoinTopLevelName(
		contract.ExportedName(),
		runtimeName,
		map[string]struct{}{"DriftedArray": {}},
	) == nil {
		t.Fatal("one-sided source import-name mutation passed the contract join")
	}

	definitionPath := filepath.Join(
		repositoryRoot(),
		"internal",
		"emit",
		"runtime",
		"array",
		"definition.go",
	)
	definition, err := os.ReadFile(definitionPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(definition), "className") ||
		strings.Contains(string(definition), `"GoArray"`) {
		t.Fatal("array runtime builder retains a duplicate top-level name owner")
	}
}

func runtimeMemberNames(
	t *testing.T,
	emission emit.ProgramEmission,
) map[string]struct{} {
	t.Helper()
	expected := memberNames(arraymember.All())
	actual := make(map[string]struct{}, len(expected))
	for _, file := range emission.Files() {
		if file.OutputPath() != "runtime/array.ts" {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			class, ok := statement.(tsgo.ClassDeclaration)
			if !ok {
				continue
			}
			for _, classMember := range class.Members() {
				switch value := classMember.(type) {
				case tsgo.ConstructorDeclaration:
					for _, parameter := range value.Parameters() {
						name, ok := parameter.Name().(tsgo.Identifier)
						if ok {
							if _, exists := expected[name.Text()]; exists {
								actual[name.Text()] = struct{}{}
							}
						}
					}
				case tsgo.MethodDeclaration:
					name, ok := value.Name().(tsgo.Identifier)
					if ok {
						if _, exists := expected[name.Text()]; exists {
							actual[name.Text()] = struct{}{}
						}
					}
				}
			}
		}
	}
	return actual
}

func runtimeClassName(
	t *testing.T,
	emission emit.ProgramEmission,
) string {
	t.Helper()
	for _, file := range emission.Files() {
		if file.OutputPath() != "runtime/array.ts" {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			class, ok := statement.(tsgo.ClassDeclaration)
			if ok {
				return class.Name().Text()
			}
		}
	}
	t.Fatal("array runtime class is absent")
	return ""
}

func sourceRuntimeImports(
	t *testing.T,
	emission emit.ProgramEmission,
	outputPath string,
) map[string]struct{} {
	t.Helper()
	result := make(map[string]struct{})
	moduleSuffix := strings.TrimSuffix(outputPath, ".ts") + ".js"
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSource {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			declaration, ok := statement.(tsgo.ImportDeclaration)
			if !ok || declaration.ImportClause() == nil {
				continue
			}
			module, ok := declaration.ModuleSpecifier().(tsgo.StringLiteral)
			if !ok || !strings.HasSuffix(module.Text(), moduleSuffix) {
				continue
			}
			bindings, ok := declaration.ImportClause().
				NamedBindings().(tsgo.NamedImports)
			if !ok {
				t.Fatalf(
					"array runtime import bindings = %T",
					declaration.ImportClause().NamedBindings(),
				)
			}
			for _, binding := range bindings.Elements() {
				name := binding.Name().Text()
				if binding.PropertyName() != nil {
					name = binding.PropertyName().(tsgo.Identifier).Text()
				}
				result[name] = struct{}{}
			}
		}
	}
	return result
}

func exactJoinTopLevelName(
	expected string,
	runtime string,
	sourceImports map[string]struct{},
) error {
	if runtime != expected {
		return fmt.Errorf(
			"array runtime class = %q, contract exports %q",
			runtime,
			expected,
		)
	}
	if len(sourceImports) != 1 {
		return fmt.Errorf(
			"array runtime source imports = %d, want one",
			len(sourceImports),
		)
	}
	if _, ok := sourceImports[expected]; !ok {
		return fmt.Errorf(
			"array runtime source import has no %q",
			expected,
		)
	}
	return nil
}

func sourceMemberNames(
	t *testing.T,
	source string,
	expected map[string]struct{},
) map[string]struct{} {
	t.Helper()
	names := make([]string, 0, len(expected))
	for name := range expected {
		names = append(names, regexp.QuoteMeta(name))
	}
	pattern := regexp.MustCompile(`\.(?:` + strings.Join(names, "|") + `)\b`)
	actual := make(map[string]struct{}, len(expected))
	for _, match := range pattern.FindAllString(source, -1) {
		actual[strings.TrimPrefix(match, ".")] = struct{}{}
	}
	return actual
}

func memberNames(identities []arraymember.Identity) map[string]struct{} {
	result := make(map[string]struct{}, len(identities))
	for _, identity := range identities {
		result[identity.Name()] = struct{}{}
	}
	return result
}

func exactJoinNames(
	side string,
	expected map[string]struct{},
	actual map[string]struct{},
) error {
	for name := range expected {
		if _, ok := actual[name]; !ok {
			return fmt.Errorf("%s array member %q is missing", side, name)
		}
	}
	for name := range actual {
		if _, ok := expected[name]; !ok {
			return fmt.Errorf("%s array member %q is unexpected", side, name)
		}
	}
	return nil
}

func cloneNames(source map[string]struct{}) map[string]struct{} {
	result := make(map[string]struct{}, len(source))
	for name := range source {
		result[name] = struct{}{}
	}
	return result
}
