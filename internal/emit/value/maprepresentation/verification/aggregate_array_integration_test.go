package maprepresentation_test

import (
	"fmt"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/storage"
	"github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
	"github.com/tsoniclang/gotots/internal/emit/value/representation"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

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
				factory.TypeReferenceNode(factory.Identifier("ArrayKey"), nil),
				factory.TypeReferenceNode(factory.Identifier("Box"), nil),
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
		api.ConcurrencySemanticsDisabled,
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

func (aggregateNames) Result(variable *types.Var, _ int) (string, error) {
	return variable.Name(), nil
}

func (aggregateNames) Reference(object types.Object) (api.NameReference, error) {
	return api.NewNameReference(object.Name())
}

func (aggregateNames) DefinedValueRepresentation(
	*types.TypeName,
) (api.DefinedValueRepresentation, error) {
	return api.NewDefinedValueRepresentation(
		api.DefinedValueRepresentationGeneratedWrapper,
		api.NameReference{},
	)
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

func (aggregateNames) NamedStructConstructor(
	typeName *types.TypeName,
) (api.NameReference, error) {
	return api.NewNameReference(typeName.Name())
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

func (aggregateNames) NamedStructStorage(
	*types.TypeName,
) (api.NameReference, error) {
	panic("unused")
}

func (aggregateNames) AnonymousStructStorage(
	*types.Struct,
) (api.NameReference, error) {
	panic("unused")
}

func (aggregateNames) AnonymousStruct(
	*types.Struct,
	api.AnonymousStructDemand,
	api.ImportPhase,
) (api.NameReference, error) {
	panic("unused")
}

func (aggregateNames) MapSpecialization(
	types.Type,
	api.MapSpecializationDemand,
) (api.NameReference, error) {
	panic("unused")
}

func (aggregateNames) InterfaceAdapter(
	types.Type,
	types.Type,
) (api.NameReference, error) {
	panic("unused")
}

func (aggregateNames) InterfaceContractDemand(
	types.Type,
	types.Type,
) ([]api.RootRequest, error) {
	panic("unused")
}

func (aggregateNames) InterfaceDynamicType(
	types.Type,
) (api.NameReference, error) {
	panic("unused")
}

func (aggregateNames) InterfaceType(
	types.Type,
) (api.NameReference, error) {
	panic("unused")
}

func (aggregateNames) InterfaceContract(
	types.Type,
) (api.InterfaceContractReference, error) {
	panic("unused")
}

func (aggregateNames) RecoveryCallable(
	*types.Func,
) (api.RecoveryCallableReference, bool, error) {
	return api.RecoveryCallableReference{}, false, nil
}

func (aggregateNames) DeferredCallable(
	*types.Func,
	string,
) (api.NameReference, error) {
	panic("unused")
}

func (aggregateNames) DeferredCallableRegistry(
	*types.Signature,
) (api.NameReference, error) {
	panic("unused")
}

func (aggregateNames) MethodTarget(
	*types.Func,
) (api.MethodTarget, error) {
	panic("unused")
}

func (aggregateNames) InterfaceMethodName(
	*types.Func,
) (string, error) {
	panic("unused")
}

func (aggregateNames) InterfaceMethodCallable(
	*types.Func,
) (api.InterfaceMethodCallableReference, error) {
	panic("unused")
}

func (aggregateNames) InterfaceMethodToken(
	*types.Func,
) (api.NameReference, error) {
	panic("unused")
}

func (aggregateNames) GenericCapability(
	api.GenericOperationSelection,
	*types.Signature,
) (api.GenericCapabilityReference, error) {
	panic("unused")
}

func (aggregateNames) CallableABI(
	*types.Signature,
) (api.CallableABIReference, error) {
	panic("unused")
}

func (aggregateNames) SourceCallableABI(
	types.Object,
	*types.Signature,
) (api.CallableABIReference, error) {
	panic("unused")
}

func (aggregateNames) ProviderGenericTypeArguments(
	*types.Func,
) ([]api.GenericTypeArgumentProjection, bool, error) {
	return nil, false, nil
}

func (aggregateNames) ProviderInterface(
	types.Type,
) (gostdlib.ProviderInterface, bool, error) {
	return gostdlib.ProviderInterface{}, false, nil
}

func (aggregateNames) ProviderInterfaceBridge(
	types.Type,
) (api.NameReference, bool, error) {
	return api.NameReference{}, false, nil
}

func (aggregateNames) ProviderCallableProfile(
	*types.Func,
	string,
) (api.ProviderCallableProfileReference, bool, error) {
	return api.ProviderCallableProfileReference{}, false, nil
}

func (aggregateNames) ProviderCallableProfileCandidates(
	*types.Func,
) ([]api.ProviderCallableProfileCandidate, bool, error) {
	return nil, false, nil
}

func (aggregateNames) ProviderStatefulProfileCandidates(
	*types.TypeName,
) ([]api.ProviderStatefulProfileCandidate, bool, error) {
	return nil, false, nil
}

func (aggregateNames) ProviderStatefulProfileTarget(
	*types.TypeName,
	string,
	api.ImportPhase,
) (api.NameReference, error) {
	panic("unused")
}

func (aggregateNames) ProviderRepresentationOwnsMethod(
	types.Type,
	*types.Func,
) (bool, error) {
	return false, nil
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
