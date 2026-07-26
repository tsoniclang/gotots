package binary

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestSignedArithmeticRequiresNativeProofForSelectedWidth(t *testing.T) {
	testCases := []struct {
		name       string
		sourceType types.Type
		arch       string
		want       bool
	}{
		{name: "64-bit int", sourceType: types.Typ[types.Int], arch: "amd64", want: true},
		{name: "32-bit int", sourceType: types.Typ[types.Int], arch: "386", want: false},
		{name: "int64 on 32-bit host", sourceType: types.Typ[types.Int64], arch: "386", want: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			source := &ast.BinaryExpr{
				X:  ast.NewIdent("left"),
				Op: token.MUL,
				Y:  ast.NewIdent("right"),
			}
			info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{
				source:   {Type: testCase.sourceType},
				source.X: {Type: testCase.sourceType},
				source.Y: {Type: testCase.sourceType},
			}}
			context, err := api.NewContext(
				api.RoleReturnResult,
				token.NewFileSet(),
				types.NewPackage("example.com/expression", "expression"),
				info,
				types.SizesFor("gc", testCase.arch),
				tsgo.Factory{},
				unusedNames{},
			)
			if err != nil {
				t.Fatal(err)
			}

			_, _, ok := operationFor(context, source)
			if ok != testCase.want {
				t.Fatalf("operation supported = %v, want %v", ok, testCase.want)
			}
		})
	}
}

type unusedNames struct{}

func (unusedNames) Declare(types.Object) (string, error) {
	panic("unused")
}

func (unusedNames) Reference(types.Object) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) TypeImport(string, string) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) Temporary(api.TemporaryKind) (string, error) {
	panic("unused")
}

func (unusedNames) ModuleExport(types.Object) (bool, error) {
	panic("unused")
}
