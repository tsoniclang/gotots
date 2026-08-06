package callableprojection

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/callableabi"
	representationcontract "github.com/tsoniclang/gotots/internal/contracts/representation"
	"github.com/tsoniclang/gotots/internal/contracts/sourceimplementation"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
)

type callableUseEvidence struct {
	observed bool
}

type Binding struct {
	function *types.Func
	callable callableabi.Callable
}

func (b Binding) Function() *types.Func          { return b.function }
func (b Binding) Callable() callableabi.Callable { return b.callable }

func Select(
	program *load.Program,
	scalar api.ScalarABI,
	manual *sourceimplementation.Certificate,
) ([]Binding, error) {
	var result []Binding
	uses := directCallableUses(program)
	for _, sourcePackage := range program.Packages() {
		if sourcePackage.Kind() != load.PackageSource {
			continue
		}
		for _, sourceFile := range sourcePackage.Files() {
			for _, declaration := range sourceFile.Syntax().Decls {
				functionDeclaration, ok := declaration.(*ast.FuncDecl)
				if !ok || functionDeclaration.Body == nil ||
					functionDeclaration.Name == nil {
					continue
				}
				function, ok := sourcePackage.TypesInfo().Defs[functionDeclaration.Name].(*types.Func)
				if !ok || function.Signature().TypeParams().Len() != 0 ||
					function.Signature().RecvTypeParams().Len() != 0 ||
					function.Signature().Variadic() {
					continue
				}
				if manual != nil && function.Signature().Recv() == nil {
					if selected, ok := manual.ResolveCallableABI(
						function.Pkg().Path(),
						function.Name(),
					); ok {
						result = append(result, Binding{
							function: function,
							callable: selected,
						})
						continue
					}
				}
				use := uses[function]
				if !use.observed {
					continue
				}
				parameters := make([]callableabi.Parameter, function.Signature().Params().Len())
				projected := false
				for index := range function.Signature().Params().Len() {
					parameter := function.Signature().Params().At(index)
					projection := callableabi.ProjectionIdentity
					targetType := types.TypeString(parameter.Type(), packagePath)
					if pointer, ok := types.Unalias(parameter.Type()).(*types.Pointer); ok &&
						callableabi.PointeeValueReadAtEntry(
							functionDeclaration,
							parameter,
							sourcePackage.TypesInfo(),
						) {
						if primitive, ok := representationcontract.PrimitiveTypeScriptType(
							pointer.Elem(),
							scalar,
						); ok {
							projection = callableabi.ProjectionPointeeValue
							targetType = primitive
							projected = true
						}
					}
					nilPolicy := callableabi.NilPolicyNotApplicable
					if projection == callableabi.ProjectionPointeeValue {
						nilPolicy = callableabi.NilPolicyRejectAtBoundary
					}
					var err error
					parameters[index], err = callableabi.NewParameter(
						projection,
						nilPolicy,
						targetType,
					)
					if err != nil {
						return nil, err
					}
				}
				if !projected {
					continue
				}
				identity, err := callableabi.SourceCallableIdentity(function)
				if err != nil {
					return nil, err
				}
				selected, err := callableabi.New(identity, parameters)
				if err != nil {
					return nil, err
				}
				result = append(result, Binding{
					function: function,
					callable: selected,
				})
			}
		}
	}
	return result, nil
}

func directCallableUses(program *load.Program) map[*types.Func]callableUseEvidence {
	result := make(map[*types.Func]callableUseEvidence)
	for _, sourcePackage := range program.Packages() {
		if sourcePackage.Kind() != load.PackageSource {
			continue
		}
		for _, object := range sourcePackage.TypesInfo().Uses {
			function, ok := object.(*types.Func)
			if !ok {
				continue
			}
			evidence := result[function]
			evidence.observed = true
			result[function] = evidence
		}
	}
	return result
}

func packagePath(source *types.Package) string {
	if source == nil {
		return ""
	}
	return source.Path()
}
