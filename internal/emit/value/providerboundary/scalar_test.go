package providerboundary

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestProviderScalarBoundaryConvertsOnlyDifferentCarriers(t *testing.T) {
	context := scalarBoundaryContext(
		t,
		"amd64",
		api.IntegerRepresentationNumber,
		api.IntegerRepresentationBigInt,
	)
	value := api.DirectExpression(context.Factory().Identifier("value"))

	toProvider, changed, err := ToProviderValue(
		context,
		nil,
		nil,
		"",
		types.Typ[types.Int64],
		value,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("number-product int64 did not cross to the bigint provider ABI")
	}
	assertBigIntNormalizer(t, toProvider.Value(), "asIntN", true)

	fromProvider, changed, err := FromProviderValue(
		context,
		nil,
		nil,
		"",
		types.Typ[types.Uint64],
		value,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("bigint-provider uint64 did not cross to the number product ABI")
	}
	outer := requireCall(t, fromProvider.Value(), 1)
	number := requireProperty(t, outer.Expression(), api.TargetGlobalAnchorName, "Number")
	_ = number
	assertBigIntNormalizer(t, outer.Arguments()[0], "asUintN", false)

	unchanged, changed, err := ToProviderValue(
		context,
		nil,
		nil,
		"",
		types.Typ[types.Int32],
		value,
	)
	if err != nil {
		t.Fatal(err)
	}
	if changed || unchanged.Value() != value.Value() {
		t.Fatal("equal number carriers acquired a provider-boundary conversion")
	}
}

func TestProviderScalarBoundaryUsesUnderlyingIntegerIdentity(t *testing.T) {
	context := scalarBoundaryContext(
		t,
		"amd64",
		api.IntegerRepresentationNumber,
		api.IntegerRepresentationBigInt,
	)
	declared := types.NewNamed(
		types.NewTypeName(token.NoPos, context.TypesPackage(), "Counter", nil),
		types.Typ[types.Int64],
		nil,
	)
	target, changed, err := ToProviderValue(
		context,
		nil,
		nil,
		"",
		declared,
		api.DirectExpression(context.Factory().Identifier("counter")),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("defined int64 did not use its checker-owned underlying scalar identity")
	}
	assertBigIntNormalizer(t, target.Value(), "asIntN", true)
}

func TestProviderScalarBoundaryPreservesProviderDefinedValue(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/boundary", "boundary")
	typeName := types.NewTypeName(
		token.NoPos,
		sourcePackage,
		"Kind",
		nil,
	)
	defined := types.NewNamed(typeName, types.Typ[types.Uint], nil)
	operations, err := api.NewNameReference("KindOperations")
	if err != nil {
		t.Fatal(err)
	}
	representation, err := api.NewDefinedValueRepresentation(
		api.DefinedValueRepresentationProviderOperations,
		operations,
	)
	if err != nil {
		t.Fatal(err)
	}
	context := scalarBoundaryContextWithNames(
		t,
		"amd64",
		api.IntegerRepresentationNumber,
		api.IntegerRepresentationBigInt,
		sourcePackage,
		scalarBoundaryNames{representations: map[*types.TypeName]api.DefinedValueRepresentation{
			typeName: representation,
		}},
	)
	value := api.DirectExpression(context.Factory().Identifier("kind"))

	toProvider, changed, err := ToProviderValue(
		context,
		nil,
		nil,
		"",
		defined,
		value,
	)
	if err != nil {
		t.Fatal(err)
	}
	if changed || toProvider.Value() != value.Value() {
		t.Fatal("provider-defined value was reduced to its underlying scalar")
	}

	fromProvider, changed, err := FromProviderValue(
		context,
		nil,
		nil,
		"",
		defined,
		value,
	)
	if err != nil {
		t.Fatal(err)
	}
	if changed || fromProvider.Value() != value.Value() {
		t.Fatal("provider-defined result was rebuilt from its underlying scalar")
	}
}

func TestProviderScalarBoundaryReifiesGeneratedNumericIdentity(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/boundary", "boundary")
	typeName := types.NewTypeName(
		token.NoPos,
		sourcePackage,
		"Kind",
		nil,
	)
	defined := types.NewNamed(typeName, types.Typ[types.Uint32], nil)
	representation, err := api.NewDefinedValueRepresentation(
		api.DefinedValueRepresentationGeneratedNumeric,
		api.NameReference{},
	)
	if err != nil {
		t.Fatal(err)
	}
	context := scalarBoundaryContextWithNames(
		t,
		"amd64",
		api.IntegerRepresentationNumber,
		api.IntegerRepresentationBigInt,
		sourcePackage,
		scalarBoundaryNames{representations: map[*types.TypeName]api.DefinedValueRepresentation{
			typeName: representation,
		}},
	)
	value := api.DirectExpression(context.Factory().Identifier("kind"))

	toProvider, changed, err := ToProviderValue(
		context,
		nil,
		nil,
		"",
		defined,
		value,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !changed || toProvider.Value() != value.Value() {
		t.Fatal("generated numeric did not project directly at the provider boundary")
	}

	fromProvider, changed, err := FromProviderValue(
		context,
		nil,
		nil,
		"",
		defined,
		value,
	)
	if err != nil {
		t.Fatal(err)
	}
	wrapped, ok := fromProvider.Value().(tsgo.BinaryExpression)
	if !changed || !ok ||
		wrapped.OperatorToken().Kind() != tsgo.SyntaxKindAsteriskToken {
		t.Fatalf("generated numeric provider result = %T, want nominal wrap", fromProvider.Value())
	}
	requireProperty(t, wrapped.Right(), "Kind", "$goType")
}

func TestProviderScalarBoundaryFailsWithoutCertifiedABI(t *testing.T) {
	context, err := api.NewContext(
		api.RoleCallArgument,
		token.NewFileSet(),
		types.NewPackage("example.com/boundary", "boundary"),
		&types.Info{},
		types.SizesFor("gc", "amd64"),
		api.MemoryByteOrderLittleEndian,
		tsgo.NewFactory(),
		scalarBoundaryNames{},
		scalarBoundaryValues{},
		scalarBoundaryStorage{},
		api.IntegerRepresentationNumber,
		api.EvaluationOrderDirect,
		api.ConcurrencySemanticsDisabled,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = ToProviderValue(
		context,
		nil,
		nil,
		"",
		types.Typ[types.Int64],
		api.DirectExpression(context.Factory().Identifier("value")),
	)
	if err == nil {
		t.Fatal("integer provider boundary accepted an absent provider scalar ABI")
	}
}

func scalarBoundaryContext(
	t *testing.T,
	arch string,
	product api.IntegerRepresentation,
	provider api.IntegerRepresentation,
) api.Context {
	t.Helper()
	return scalarBoundaryContextWithNames(
		t,
		arch,
		product,
		provider,
		types.NewPackage("example.com/boundary", "boundary"),
		scalarBoundaryNames{},
	)
}

func scalarBoundaryContextWithNames(
	t *testing.T,
	arch string,
	product api.IntegerRepresentation,
	provider api.IntegerRepresentation,
	sourcePackage *types.Package,
	names scalarBoundaryNames,
) api.Context {
	t.Helper()
	sizes := types.SizesFor("gc", arch)
	context, err := api.NewContext(
		api.RoleCallArgument,
		token.NewFileSet(),
		sourcePackage,
		&types.Info{},
		sizes,
		api.MemoryByteOrderLittleEndian,
		tsgo.NewFactory(),
		names,
		scalarBoundaryValues{},
		scalarBoundaryStorage{},
		product,
		api.EvaluationOrderDirect,
		api.ConcurrencySemanticsDisabled,
	)
	if err != nil {
		t.Fatal(err)
	}
	providerABI, err := api.NewScalarABIFromSizes(provider, sizes)
	if err != nil {
		t.Fatal(err)
	}
	return context.WithProviderScalarABI(providerABI)
}

func assertBigIntNormalizer(
	t *testing.T,
	expression tsgo.Expression,
	member string,
	numberInput bool,
) {
	t.Helper()
	call := requireCall(t, expression, 2)
	requireProperty(t, call.Expression(), "BigInt", member)
	width, ok := call.Arguments()[0].(tsgo.NumericLiteral)
	if !ok || width.Text() != "64" {
		t.Fatalf("provider scalar width = %T %v, want numeric 64", call.Arguments()[0], call.Arguments()[0])
	}
	if numberInput {
		converted := requireCall(t, call.Arguments()[1], 1)
		callee, ok := converted.Expression().(tsgo.Identifier)
		if !ok || callee.Text() != "goNumberToBigInt" {
			t.Fatal("number-to-bigint provider boundary bypassed the shared runtime conversion")
		}
	}
}

func requireCall(
	t *testing.T,
	expression tsgo.Expression,
	argumentCount int,
) tsgo.CallExpression {
	t.Helper()
	call, ok := expression.(tsgo.CallExpression)
	if !ok || len(call.Arguments()) != argumentCount {
		t.Fatalf("expression = %T, want %d-argument call", expression, argumentCount)
	}
	return call
}

func requireProperty(
	t *testing.T,
	expression tsgo.Expression,
	receiver string,
	member string,
) tsgo.PropertyAccessExpression {
	t.Helper()
	property, ok := expression.(tsgo.PropertyAccessExpression)
	if !ok {
		t.Fatalf("callee = %T, want property access", expression)
	}
	selectedReceiver, receiverOK := property.Expression().(tsgo.Identifier)
	selectedMember, memberOK := property.Name().(tsgo.Identifier)
	if !receiverOK || !memberOK ||
		selectedReceiver.Text() != receiver || selectedMember.Text() != member {
		t.Fatalf("property = %T.%T, want %s.%s", property.Expression(), property.Name(), receiver, member)
	}
	return property
}

type scalarBoundaryNames struct {
	api.Names
	representations map[*types.TypeName]api.DefinedValueRepresentation
}

func (scalarBoundaryNames) Reference(
	object types.Object,
) (api.NameReference, error) {
	return api.NewNameReference(object.Name())
}

func (n scalarBoundaryNames) DefinedValueRepresentation(
	typeName *types.TypeName,
) (api.DefinedValueRepresentation, error) {
	if selected, ok := n.representations[typeName]; ok {
		return selected, nil
	}
	return api.NewDefinedValueRepresentation(
		api.DefinedValueRepresentationGeneratedWrapper,
		api.NameReference{},
	)
}

func (scalarBoundaryNames) Runtime(
	symbol api.RuntimeSymbol,
	phase api.ImportPhase,
) (api.NameReference, error) {
	if symbol != api.RuntimeNumberToBigInt || phase != api.ImportPhaseValue {
		panic("unexpected provider scalar runtime reference")
	}
	return api.NewNameReference("goNumberToBigInt")
}

type scalarBoundaryValues struct {
	api.Values
}

type scalarBoundaryStorage struct {
	api.AddressableStorage
}
