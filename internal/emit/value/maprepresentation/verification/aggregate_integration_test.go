package maprepresentation_test

import (
	"crypto/sha256"
	"fmt"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/storage"
	"github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
	"github.com/tsoniclang/gotots/internal/emit/value/representation"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestScalarMapArtifactsStayAtTheImmutableBaseline(t *testing.T) {
	artifacts := materialize(
		t,
		compileExported(t, loadMapValuesProject(t)),
		t.TempDir(),
	)
	for path, expected := range map[string]string{
		"source.ts":      "039d67c3463d6034acf6a4dda7364e61e6555c75eb31ab520011eae0ea29e656",
		"runtime/map.ts": "b08ead3648e95400dc2aabe9727ff5dc07462e7177cec534e864f76f5abbdb2e",
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
		factory.TypeReferenceNode(factory.Identifier("Box"), nil),
		maprepresentation.SpecializationCapabilities{},
	)
	if err != nil {
		t.Fatal(err)
	}
	operations := make(map[string]int)
	requirements := make(map[api.DeclarationRequirement]int)
	for _, request := range specialization.Requests() {
		requirement, ok := request.DeclarationRequirement()
		if !ok {
			continue
		}
		if requirement.Kind() != api.DeclarationRequirementNamedStructOperation {
			t.Fatalf(
				"aggregate specialization introduced requirement kind %d",
				requirement.Kind(),
			)
		}
		requirements[requirement]++
		typeName, operation, ok := requirement.NamedStructOperation()
		if !ok {
			t.Fatal("named-struct requirement lost its typed operation")
		}
		operations[typeName.Name()+"/"+operation.String()]++
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
		"clear(): void",
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("aggregate specialization contains %q:\n%s", forbidden, source)
		}
	}
}

func TestUnnamedArrayKeyOperationsInlineStaticTypedSemantics(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		integer    api.IntegerRepresentation
		hashMember string
	}{
		{
			name:       "number",
			integer:    api.IntegerRepresentationNumber,
			hashMember: "GoMapHash.number(",
		},
		{
			name:       "bigint",
			integer:    api.IntegerRepresentationBigInt,
			hashMember: "GoMapHash.bigint(",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			targetContext, _, value := productionAggregateContext(
				t,
				testCase.integer,
			)
			factory := targetContext.Factory()
			key := types.NewArray(types.Typ[types.Int32], 2)
			specialization, err := maprepresentation.BuildSpecialization(
				targetContext,
				nil,
				"ArrayMap",
				types.NewMap(key, value),
				factory.TypeReferenceNode(factory.Identifier("ArrayKey"), nil),
				factory.TypeReferenceNode(factory.Identifier("Box"), nil),
				maprepresentation.SpecializationCapabilities{},
			)
			if err != nil {
				t.Fatal(err)
			}
			class := factory.ClassDeclaration(
				nil,
				factory.Identifier("ArrayMap"),
				nil,
				nil,
				specialization.Members(),
			)
			source := printAggregateSpecialization(t, factory, class)
			for _, required := range []string{
				"private static $hash($key: ArrayKey): number",
				"for (let __gotots_array_hash_",
				testCase.hashMember,
				"private static $equal($left: ArrayKey, $right: ArrayKey): boolean",
				"private static $copyKey($key: ArrayKey): ArrayKey",
				"private static $copyValue($value: Box): Box",
				"return Box.$copy($value)",
			} {
				if !strings.Contains(source, required) {
					t.Fatalf(
						"array-key specialization lacks %q:\n%s",
						required,
						source,
					)
				}
			}
			for _, forbidden := range []string{
				"=>",
				"hash: (",
				"equal: (",
				"copyKey: (",
				"any",
				"unknown",
			} {
				if strings.Contains(source, forbidden) {
					t.Fatalf(
						"array-key specialization contains %q:\n%s",
						forbidden,
						source,
					)
				}
			}
		})
	}
}

func productionAggregateContext(
	t *testing.T,
	integer api.IntegerRepresentation,
) (api.Context, *types.Named, *types.Named) {
	t.Helper()
	pkg := types.NewPackage("example.com/aggregate", "aggregate")
	pair := types.NewArray(types.Typ[types.Int32], 2)
	key := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Key", nil),
		types.NewStruct(
			[]*types.Var{
				types.NewField(
					token.NoPos,
					pkg,
					"Label",
					types.Typ[types.Int32],
					false,
				),
				types.NewField(token.NoPos, pkg, "Pair", pair, false),
			},
			nil,
		),
		nil,
	)
	value := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "Box", nil),
		types.NewStruct(
			[]*types.Var{
				types.NewField(
					token.NoPos,
					pkg,
					"Value",
					types.Typ[types.Int32],
					false,
				),
			},
			nil,
		),
		nil,
	)
	targetContext, err := api.NewContext(
		api.RoleMapReceiver,
		token.NewFileSet(),
		pkg,
		&types.Info{},
		types.SizesFor("gc", "amd64"),
		tsgo.NewFactory(),
		&aggregateNames{},
		representation.Owner{},
		storage.Owner{},
		integer,
		api.EvaluationOrderPreserveGo,
	)
	if err != nil {
		t.Fatal(err)
	}
	return targetContext, key, value
}

func printAggregateSpecialization(
	t *testing.T,
	factory tsgo.Factory,
	class tsgo.ClassDeclaration,
) string {
	t.Helper()
	path, err := tsgo.NewPath("aggregate-map.ts")
	if err != nil {
		t.Fatal(err)
	}
	source := factory.SourceFile(
		[]tsgo.Statement{class},
		factory.EndOfFile(),
		tsgo.SourceFileData{
			FileName:        path,
			Path:            path,
			LanguageVariant: tsgo.LanguageVariantStandard,
			ScriptKind:      tsgo.ScriptKindTS,
		},
	)
	client, err := tsgo.StartClient(repositoryRoot(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	printed, err := client.PrintNode(source, tsgo.PrintOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return printed
}

type aggregateNames struct {
	next int
}

func (aggregateNames) Declare(object types.Object) (string, error) {
	return object.Name(), nil
}

func (aggregateNames) Parameter(variable *types.Var, _ int) (string, error) {
	return variable.Name(), nil
}

func (aggregateNames) Reference(object types.Object) (api.NameReference, error) {
	return api.NewNameReference(object.Name())
}

func (aggregateNames) TypeReference(
	object types.Object,
) (api.NameReference, error) {
	return api.NewNameReference(object.Name())
}

func (aggregateNames) PackageVariable(
	*types.Var,
) (api.PackageVariableReference, error) {
	panic("unused")
}

func (aggregateNames) NamedStructOperation(
	typeName *types.TypeName,
	operation api.NamedStructOperation,
) (api.NameReference, error) {
	request, err := api.NewNamedStructOperationRequest(typeName, operation)
	if err != nil {
		return api.NameReference{}, err
	}
	return api.NewNameReference(typeName.Name(), request)
}

func (aggregateNames) AnonymousStruct(
	*types.Struct,
	api.AnonymousStructDemand,
) (api.NameReference, error) {
	panic("unused")
}

func (aggregateNames) MapSpecialization(
	types.Type,
	api.MapSpecializationDemand,
) (api.NameReference, error) {
	panic("unused")
}

func (aggregateNames) ConstantProjection(
	*types.Const,
	types.BasicKind,
) (api.NameReference, error) {
	panic("unused")
}

func (aggregateNames) Member(variable *types.Var) (string, error) {
	return variable.Name(), nil
}

func (aggregateNames) Primitive(
	alias api.PrimitiveAlias,
) (api.NameReference, error) {
	name, err := api.PrimitiveAliasName(alias)
	if err != nil {
		return api.NameReference{}, err
	}
	return api.NewNameReference(name)
}

func (aggregateNames) Runtime(
	symbol api.RuntimeSymbol,
	_ api.ImportPhase,
) (api.NameReference, error) {
	contract, err := api.RuntimeContract(symbol)
	if err != nil {
		return api.NameReference{}, err
	}
	return api.NewNameReference(contract.ExportedName())
}

func (n *aggregateNames) Temporary(kind api.TemporaryKind) (string, error) {
	prefix, err := api.TemporaryPrefix(kind)
	if err != nil {
		return "", err
	}
	name := fmt.Sprintf("%s%d", prefix, n.next)
	n.next++
	return name, nil
}

func (aggregateNames) ModuleExport(types.Object) (bool, error) {
	return false, nil
}
