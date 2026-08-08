package maprepresentation_test

import (
	"crypto/sha256"
	"fmt"
	"go/types"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
)

func TestScalarMapArtifactsStayAtTheImmutableBaseline(t *testing.T) {
	artifacts := materialize(
		t,
		compileExported(t, loadMapValuesProject(t)),
		t.TempDir(),
	)
	for path, expected := range map[string]string{
		"source.ts":      "31232dade49dffef3db2784eda9911abff538e950f93c2e3acbf2f6f312cf2c0",
		"runtime/map.ts": "7f26493efc6f9213e59853a6e485061ae24fa2d0cd41a277f6d54e9399c3fc6e",
	} {
		content := readFile(t, artifacts.file(t, path))
		actual := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))
		t.Logf("%s bytes=%d sha256=%s", path, len(content), actual)
		if actual != expected {
			t.Fatalf(
				"%s sha256 = %s, want immutable baseline %s",
				path,
				actual,
				expected,
			)
		}
	}
}

func TestProductionAggregateKeyOperationsAreStaticAndTyped(t *testing.T) {
	targetContext, key, value := productionAggregateContext(
		t,
		api.IntegerRepresentationNumber,
	)
	factory := targetContext.Factory()
	specialization, err := maprepresentation.BuildSpecialization(
		targetContext,
		nil,
		"AggregateMap",
		types.NewMap(key, value),
		factory.TypeReferenceNode(factory.Identifier("Key"), nil),
		factory.TypeReferenceNode(factory.Identifier("Key"), nil),
		factory.TypeReferenceNode(factory.Identifier("Box"), nil),
	)
	if err != nil {
		t.Fatal(err)
	}
	operations := make(map[string]int)
	requirements := make(map[api.DeclarationRequirement]int)
	err = api.WalkRootRequests(
		specialization.Requests(),
		func(request api.RootRequest) error {
			requirement, ok := request.DeclarationRequirement()
			if !ok {
				return nil
			}
			if requirement.Kind() != api.DeclarationRequirementNamedStructOperation {
				return fmt.Errorf(
					"aggregate specialization introduced requirement kind %d",
					requirement.Kind(),
				)
			}
			requirements[requirement]++
			typeName, operation, ok := requirement.NamedStructOperation()
			if !ok {
				return fmt.Errorf(
					"named-struct requirement lost its typed operation",
				)
			}
			operations[typeName.Name()+"/"+operation.String()]++
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{
		"Key/copy",
		"Key/equal",
		"Key/hash",
		"Box/zero",
		"Box/copy",
	} {
		if operations[operation] != 1 {
			t.Fatalf(
				"aggregate operation %s requests = %d, want one",
				operation,
				operations[operation],
			)
		}
	}
	if len(requirements) != 5 {
		t.Fatalf(
			"aggregate declaration requirements = %d, want five",
			len(requirements),
		)
	}
	class := factory.ClassDeclaration(
		nil,
		factory.Identifier("AggregateMap"),
		nil,
		nil,
		specialization.Members(),
	)
	source := printAggregateSpecialization(t, factory, class)
	t.Logf("production-shaped aggregate map bytes=%d", len(source))
	if len(source) > 7_000 {
		t.Fatalf(
			"production-shaped aggregate map = %d bytes, want at most 7000",
			len(source),
		)
	}
	for _, required := range []string{
		"return Box.$zero()",
		"return Key.$hash($key)",
		"return Key.$equal($left, $right)",
		"return Key.$copy($key)",
		"return Box.$copy($value)",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("aggregate specialization lacks %q:\n%s", required, source)
		}
	}
	for _, forbidden := range []string{
		"hash: (",
		"equal: (",
		"copyKey: (",
		"private readonly hash",
		"private readonly equal",
		"private readonly copyKey",
		"any",
		"unknown",
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("aggregate specialization contains %q:\n%s", forbidden, source)
		}
	}
	for _, required := range []string{"clear(): void", "keys(): Key[]"} {
		if !strings.Contains(source, required) {
			t.Fatalf("aggregate specialization lacks %q:\n%s", required, source)
		}
	}
}
