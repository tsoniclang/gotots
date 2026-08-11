package certify

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/load"
)

type sourceFunction struct {
	function      *types.Func
	contract      environmentcontract.ObjectContract
	signature     *types.Signature
	modulePath    string
	moduleVersion string
	location      string
	body          bool
}

type sourcePackage struct {
	functions map[string]sourceFunction
}

func loadSourcePackages(
	config resolvedConfig,
	seeds []bindingSeed,
) (map[string]sourcePackage, error) {
	result := make(map[string]sourcePackage)
	for _, seed := range seeds {
		if _, loaded := result[seed.SourcePackage]; loaded {
			continue
		}
		selected, err := load.One(context.Background(), load.Request{
			Directory:            config.providerRoot,
			Pattern:              seed.SourcePackage,
			ContractDependencies: true,
			BuildProfile:         config.buildProfile,
			GoTool:               config.goTool,
		})
		if err != nil {
			return nil, certifyError(
				"load source package",
				seed.SourcePackage,
				err.Error(),
			)
		}
		if selected.Path() != seed.SourcePackage ||
			selected.ModulePath() == "" || selected.ModuleVersion() == "" {
			return nil, certifyError(
				"load source package",
				seed.SourcePackage,
				"selected module identity is incomplete",
			)
		}
		functions, err := packageFunctions(selected)
		if err != nil {
			return nil, err
		}
		result[seed.SourcePackage] = sourcePackage{functions: functions}
	}
	return result, nil
}

func packageFunctions(selected *load.Package) (map[string]sourceFunction, error) {
	result := make(map[string]sourceFunction)
	for _, file := range selected.Files() {
		for _, declaration := range file.Syntax().Decls {
			functionDeclaration, ok := declaration.(*ast.FuncDecl)
			if !ok || functionDeclaration.Recv != nil {
				continue
			}
			object, ok := selected.TypesInfo().Defs[functionDeclaration.Name].(*types.Func)
			if !ok || object == nil || object != object.Origin() {
				return nil, certifyError(
					"inspect source package",
					selected.Path(),
					"function declaration has no canonical object",
				)
			}
			signature, ok := object.Type().(*types.Signature)
			if !ok {
				return nil, certifyError(
					"inspect source package",
					object.Name(),
					"function signature is absent",
				)
			}
			contract, err := environmentcontract.Describe(object)
			if err != nil {
				return nil, err
			}
			position := selected.FileSet().Position(functionDeclaration.Pos())
			if !position.IsValid() {
				return nil, certifyError(
					"inspect source package",
					contract.Identity(),
					"source position is absent",
				)
			}
			if _, duplicate := result[object.Name()]; duplicate {
				return nil, certifyError(
					"inspect source package",
					object.Name(),
					"function is duplicated",
				)
			}
			result[object.Name()] = sourceFunction{
				function:      object,
				contract:      contract,
				signature:     signature,
				modulePath:    selected.ModulePath(),
				moduleVersion: selected.ModuleVersion(),
				location:      portableLocation(position),
				body:          functionDeclaration.Body != nil,
			}
		}
	}
	return result, nil
}

func portableLocation(position token.Position) string {
	return fmt.Sprintf(
		"%s:%d:%d",
		filepath.Base(position.Filename),
		position.Line,
		position.Column,
	)
}
