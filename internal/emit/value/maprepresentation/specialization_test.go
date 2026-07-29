package maprepresentation

import (
	"context"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/storage"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestStaticSpecializationExecutesCollisionsCopiesAndNilSemantics(
	t *testing.T,
) {
	for _, testCase := range []struct {
		name    string
		integer api.IntegerRepresentation
	}{
		{name: "number", integer: api.IntegerRepresentationNumber},
		{name: "bigint", integer: api.IntegerRepresentationBigInt},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			testStaticSpecialization(t, testCase.integer)
		})
	}
}

func testStaticSpecialization(
	t *testing.T,
	integer api.IntegerRepresentation,
) {
	targetContext, key, value := staticSpecializationContext(t, integer)
	factory := targetContext.Factory()
	mapType := types.NewMap(key, value)
	specialization, err := BuildSpecialization(
		targetContext,
		nil,
		"StaticMap",
		mapType,
		factory.TypeReferenceNode(factory.Identifier("Key"), nil),
		factory.TypeReferenceNode(factory.Identifier("Box"), nil),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(specialization.Members()) != 17 {
		t.Fatalf(
			"specialization members = %d, want constructor, static operations, and map API",
			len(specialization.Members()),
		)
	}
	class := factory.ClassDeclaration(
		nil,
		factory.Identifier("StaticMap"),
		nil,
		nil,
		specialization.Members(),
	)
	printed := printSpecialization(t, factory, class)
	assertStaticSpecializationArtifact(t, printed)
	t.Logf("static aggregate map definition bytes=%d", len(printed))
	if len(printed) > 9_000 {
		t.Fatalf(
			"one aggregate map shape = %d bytes, want at most 9000",
			len(printed),
		)
	}
	keyType := "number"
	keyConversion := "new Key(x, y)"
	if integer == api.IntegerRepresentationBigInt {
		keyType = "bigint"
		keyConversion = "new Key(BigInt(x), BigInt(y))"
	}
	runner := `class Key {
    constructor(public x: ` + keyType + `, public y: ` + keyType + `) {}
}
class GoPanic {
    static raise(message: string): never { throw new Error(message); }
}
class Box {
    constructor(public value: number) {}
}
const key = (x: number, y: number): Key => ` + keyConversion + `;
` + printed + `
const nilMap = StaticMap.nil();
const missing = nilMap.lookupOk(key(1, 2));
nilMap.delete(key(1, 2));
let nilStoreFailed = false;
try { nilMap.store(key(1, 2), new Box(1)); } catch { nilStoreFailed = true; }
console.log(missing[0].value, missing[1], nilMap.length(), nilStoreFailed);

const first = StaticMap.make(0, []);
const second = StaticMap.make(0, []);
const sourceKey = key(1, 2);
const sourceValue = new Box(10);
first.store(sourceKey, sourceValue);
sourceKey.x = key(9, 0).x;
sourceValue.value = 99;
const lookupValue = first.lookup(key(1, 2));
lookupValue.value = 77;
first.store(key(3, 4), new Box(20));
second.store(key(1, 2), new Box(30));
console.log(
    first.lookup(key(1, 2)).value,
    first.lookup(key(3, 4)).value,
    first.length(),
    second.lookup(key(1, 2)).value,
);
first.delete(key(1, 2));
console.log(first.lookup(key(3, 4)).value, first.length());
`
	if output := compileAndRunSpecialization(t, runner); output !=
		"0 false 0 true\n10 20 2 30\n20 1\n" {
		t.Fatalf("specialization output = %q", output)
	}
}

func TestStaticSpecializationRejectsStoredSemanticCallbacks(t *testing.T) {
	targetContext, key, value := staticSpecializationContext(
		t,
		api.IntegerRepresentationNumber,
	)
	factory := targetContext.Factory()
	specialization, err := BuildSpecialization(
		targetContext,
		nil,
		"StaticMap",
		types.NewMap(key, value),
		factory.TypeReferenceNode(factory.Identifier("Key"), nil),
		factory.TypeReferenceNode(factory.Identifier("Box"), nil),
	)
	if err != nil {
		t.Fatal(err)
	}
	mutated := append(
		specialization.Members(),
		factory.PropertyDeclaration(
			[]tsgo.ModifierLike{factory.PrivateKeyword()},
			factory.Identifier("hash"),
			nil,
			factory.FunctionTypeNode(
				nil,
				[]tsgo.ParameterDeclaration{
					factory.ParameterDeclaration(
						nil,
						nil,
						factory.Identifier("key"),
						nil,
						factory.TypeReferenceNode(
							factory.Identifier("Key"),
							nil,
						),
						nil,
					),
				},
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindNumberKeyword,
				),
			),
			nil,
		),
	)
	if err := validateSpecialization(
		api.RoleMapReceiver,
		mutated,
	); err == nil {
		t.Fatal("stored hash callback mutation passed the specialization gate")
	}
}

func assertStaticSpecializationArtifact(t *testing.T, source string) {
	t.Helper()
	for _, required := range []string{
		"private readonly buckets: Map<number,",
		"private static $hash($key: Key): number",
		"private static $equal($left: Key, $right: Key): boolean",
		"private static $copyKey($key: Key): Key",
		"private static $copyValue($value: Box): Box",
		"StaticMap.$hash(key)",
		"StaticMap.$equal(entry[0], key)",
		"StaticMap.$copyKey(key)",
		"StaticMap.$copyValue(value)",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("specialization lacks %q:\n%s", required, source)
		}
	}
	for _, operation := range []string{
		"$hash(",
		"$equal(",
		"$copyKey(",
		"$copyValue(",
	} {
		if strings.Count(source, "static "+operation) != 1 {
			t.Fatalf(
				"specialization operation %q definitions = %d, want one:\n%s",
				operation,
				strings.Count(source, "static "+operation),
				source,
			)
		}
	}
	for _, forbidden := range []string{
		"private readonly hash",
		"private readonly equal",
		"private readonly copyKey",
		"private readonly copyValue",
		"=>",
		"any",
		"unknown",
		"JSON.",
		"Reflect.",
		".call(",
		".apply(",
		".bind(",
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("specialization contains %q:\n%s", forbidden, source)
		}
	}
}

func printSpecialization(
	t *testing.T,
	factory tsgo.Factory,
	class tsgo.ClassDeclaration,
) string {
	t.Helper()
	path, err := tsgo.NewPath("specialization.ts")
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
	client, err := tsgo.StartClient(specializationRepositoryRoot(), t.TempDir())
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

func compileAndRunSpecialization(t *testing.T, source string) string {
	t.Helper()
	directory := t.TempDir()
	sourcePath := filepath.Join(directory, "specialization.ts")
	if err := os.WriteFile(sourcePath, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := tsgo.Compile(
		ctx,
		specializationRepositoryRoot(),
		directory,
		[]string{
			"--target", "es2022",
			"--module", "nodenext",
			"--moduleResolution", "nodenext",
			"--strict",
			"--outDir", "out",
			sourcePath,
		},
	); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("node", filepath.Join(directory, "out", "specialization.js"))
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("specialization execution failed: %v\n%s", err, output)
	}
	return string(output)
}

func staticSpecializationContext(
	t *testing.T,
	integer api.IntegerRepresentation,
) (api.Context, *types.Struct, *types.Struct) {
	t.Helper()
	key := types.NewStruct(
		[]*types.Var{
			types.NewField(token.NoPos, nil, "x", types.Typ[types.Int32], false),
			types.NewField(token.NoPos, nil, "y", types.Typ[types.Int32], false),
		},
		nil,
	)
	value := types.NewStruct(
		[]*types.Var{
			types.NewField(
				token.NoPos,
				nil,
				"value",
				types.Typ[types.Int32],
				false,
			),
		},
		nil,
	)
	values := staticSpecializationValues{key: key, value: value}
	targetContext, err := api.NewContext(
		api.RoleMapReceiver,
		token.NewFileSet(),
		types.NewPackage("example.com/specialization", "specialization"),
		&types.Info{},
		types.SizesFor("gc", "amd64"),
		tsgo.NewFactory(),
		staticSpecializationNames{},
		values,
		storage.Owner{},
		integer,
		api.EvaluationOrderPreserveGo,
	)
	if err != nil {
		t.Fatal(err)
	}
	return targetContext, key, value
}

type staticSpecializationValues struct {
	key   types.Type
	value types.Type
}

func (v staticSpecializationValues) RequiresCustomEquality(
	api.Context,
	types.Type,
) bool {
	return false
}

func (v staticSpecializationValues) RequiresExplicitType(
	api.Context,
	types.Type,
) bool {
	return false
}

func (v staticSpecializationValues) RequiresStructuralCopy(
	_ api.Context,
	sourceType types.Type,
) bool {
	return sourceType == v.key || sourceType == v.value
}

func (v staticSpecializationValues) SupportsHash(
	_ api.Context,
	sourceType types.Type,
) bool {
	return sourceType == v.key
}

func (staticSpecializationValues) RequiresStorageProjection(
	api.Context,
	types.Type,
) bool {
	return false
}

func (v staticSpecializationValues) StorageType(
	context api.Context,
	_ ast.Node,
	sourceType types.Type,
) (api.TypeEmission, error) {
	switch sourceType {
	case v.key:
		return api.DirectType(
			context.Factory().TypeReferenceNode(
				context.Factory().Identifier("Key"),
				nil,
			),
		), nil
	case v.value:
		return api.DirectType(
			context.Factory().TypeReferenceNode(
				context.Factory().Identifier("Box"),
				nil,
			),
		), nil
	default:
		panic("unexpected specialization storage type")
	}
}

func (v staticSpecializationValues) ToStorage(
	_ api.Context,
	_ ast.Node,
	_ types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	return value, nil
}

func (v staticSpecializationValues) FromStorage(
	_ api.Context,
	_ ast.Node,
	_ types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	return value, nil
}

func (v staticSpecializationValues) Zero(
	context api.Context,
	_ ast.Node,
	sourceType types.Type,
) (api.ExpressionEmission, error) {
	if sourceType != v.value {
		panic("unexpected specialization zero type")
	}
	return api.DirectExpression(
		context.Factory().NewExpression(
			context.Factory().Identifier("Box"),
			nil,
			[]tsgo.Expression{
				context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
			},
		),
	), nil
}

func (v staticSpecializationValues) Copy(
	context api.Context,
	_ ast.Node,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	var className string
	var fields []tsgo.Expression
	switch sourceType {
	case v.key:
		className = "Key"
		fields = []tsgo.Expression{
			staticField(context, value.Value(), "x"),
			staticField(context, value.Value(), "y"),
		}
	case v.value:
		className = "Box"
		fields = []tsgo.Expression{
			staticField(context, value.Value(), "value"),
		}
	default:
		panic("unexpected specialization copy type")
	}
	return api.NewExpressionEmission(
		value.Before(),
		context.Factory().NewExpression(
			context.Factory().Identifier(className),
			nil,
			fields,
		),
		value.Requests(),
	)
}

func (v staticSpecializationValues) Assign(
	api.Context,
	ast.Node,
	types.Type,
	tsgo.Expression,
	api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	panic("unused")
}

func (v staticSpecializationValues) Equal(
	context api.Context,
	_ ast.Node,
	sourceType types.Type,
	left tsgo.Expression,
	right tsgo.Expression,
) (api.ExpressionEmission, error) {
	if sourceType != v.key {
		panic("unexpected specialization equality type")
	}
	return api.DirectExpression(
		context.Factory().BinaryExpression(
			nil,
			context.Factory().BinaryExpression(
				nil,
				staticField(context, left, "x"),
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				),
				staticField(context, right, "x"),
			),
			nil,
			context.Factory().BinaryOperatorToken(
				tsgo.BinaryOperatorAmpersandAmpersandToken,
			),
			context.Factory().BinaryExpression(
				nil,
				staticField(context, left, "y"),
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				),
				staticField(context, right, "y"),
			),
		),
	), nil
}

func (v staticSpecializationValues) Hash(
	context api.Context,
	_ ast.Node,
	sourceType types.Type,
	value tsgo.Expression,
) (api.ExpressionEmission, error) {
	if sourceType != v.key {
		panic("unexpected specialization hash type")
	}
	one, err := api.IntegerLiteral(
		context.Factory(),
		context.IntegerRepresentation(),
		"1",
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	hash := tsgo.Expression(context.Factory().BinaryExpression(
		nil,
		staticField(context, value, "x"),
		nil,
		context.Factory().BinaryOperatorToken(
			tsgo.BinaryOperatorAmpersandToken,
		),
		one,
	))
	if context.IntegerRepresentation() == api.IntegerRepresentationBigInt {
		hash = context.Factory().CallExpression(
			context.Factory().Identifier("Number"),
			nil,
			nil,
			[]tsgo.Expression{hash},
			tsgo.NodeFlagsNone,
		)
	}
	return api.DirectExpression(
		hash,
	), nil
}

func (v staticSpecializationValues) BinaryUpdate(
	api.Context,
	ast.Node,
	ast.Expr,
	types.Type,
	types.Type,
	token.Token,
	tsgo.Expression,
	api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	panic("unused")
}

func (v staticSpecializationValues) Increment(
	api.Context,
	ast.Node,
	types.Type,
	token.Token,
	tsgo.Expression,
) (api.ExpressionEmission, bool, error) {
	panic("unused")
}
