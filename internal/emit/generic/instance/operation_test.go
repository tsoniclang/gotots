package instance

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestInstantiateOperationPreservesIndependentInterfaceResult(
	t *testing.T,
) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(
		fileSet,
		"generic.go",
		`package fixture
func Adapt[T any](value T) any { return value }
var _ = Adapt[int32]
`,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Defs:      make(map[*ast.Ident]types.Object),
		Instances: make(map[*ast.Ident]types.Instance),
	}
	pkg, err := (&types.Config{}).Check(
		"example.com/fixture",
		fileSet,
		[]*ast.File{file},
		info,
	)
	if err != nil {
		t.Fatal(err)
	}
	owner, ok := pkg.Scope().Lookup("Adapt").(*types.Func)
	if !ok {
		t.Fatal("generic owner is absent")
	}
	signature := owner.Type().(*types.Signature)
	operation := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(types.NewVar(
			token.NoPos,
			pkg,
			"",
			signature.TypeParams().At(0),
		)),
		types.NewTuple(types.NewVar(
			token.NoPos,
			pkg,
			"",
			signature.Results().At(0).Type(),
		)),
		false,
	)
	selection, err := api.SelectGenericOperation(
		api.GenericOperationInterfaceAdapt,
	)
	if err != nil {
		t.Fatal(err)
	}
	contract, err := api.NewGenericOperationContract(
		owner,
		"adapt",
		"$adapt",
		api.GenericFunctionOperationConsumer(),
		selection,
		operation,
	)
	if err != nil {
		t.Fatal(err)
	}
	set, err := api.NewGenericOperationSet(
		owner,
		api.GenericFunctionOperationConsumer(),
		[]*api.GenericOperationContract{contract},
	)
	if err != nil {
		t.Fatal(err)
	}
	var arguments *types.TypeList
	for _, instance := range info.Instances {
		arguments = instance.TypeArgs
	}
	instantiated, err := InstantiateOperation(set, operation, arguments)
	if err != nil {
		t.Fatal(err)
	}
	if !types.Identical(
		instantiated.Params().At(0).Type(),
		types.Typ[types.Int32],
	) {
		t.Fatalf(
			"instantiated parameter = %s, want int32",
			instantiated.Params().At(0).Type(),
		)
	}
	if _, ok := types.Unalias(
		instantiated.Results().At(0).Type(),
	).Underlying().(*types.Interface); !ok {
		t.Fatalf(
			"instantiated result = %s, want interface",
			instantiated.Results().At(0).Type(),
		)
	}
}
