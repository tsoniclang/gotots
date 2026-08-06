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

func Infer(
	program *load.Program,
	scalar api.ScalarABI,
	manual *sourceimplementation.Certificate,
) (map[*types.Func]callableabi.Callable, error) {
	result := make(map[*types.Func]callableabi.Callable)
	uses := directCallableUses(program)
	for _, sourcePackage := range program.Packages() {
		if sourcePackage.Kind() != load.PackageSource {
			continue
		}
		for _, sourceFile := range sourcePackage.Files() {
			for _, declaration := range sourceFile.Syntax().Decls {
				functionDeclaration, ok := declaration.(*ast.FuncDecl)
				if !ok || functionDeclaration.Recv != nil ||
					functionDeclaration.Body == nil || functionDeclaration.Name == nil {
					continue
				}
				function, ok := sourcePackage.TypesInfo().Defs[functionDeclaration.Name].(*types.Func)
				if !ok || function.Signature().TypeParams().Len() != 0 ||
					function.Signature().Variadic() {
					continue
				}
				if manual != nil {
					if _, selected := manual.ResolveCallableABI(
						function.Pkg().Path(),
						function.Name(),
					); selected {
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
						directReturnedPointee(
							functionDeclaration.Body,
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
				identity, err := callableabi.PackageFunctionIdentity(
					function.Pkg().Path(),
					function.Name(),
				)
				if err != nil {
					return nil, err
				}
				selected, err := callableabi.New(identity, parameters)
				if err != nil {
					return nil, err
				}
				result[function] = selected
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
			if !ok || function.Signature().Recv() != nil {
				continue
			}
			evidence := result[function]
			evidence.observed = true
			result[function] = evidence
		}
	}
	return result
}

func directReturnedPointee(
	body *ast.BlockStmt,
	parameter *types.Var,
	info *types.Info,
) bool {
	if body == nil || parameter == nil || info == nil || len(body.List) != 1 {
		return false
	}
	result, ok := body.List[0].(*ast.ReturnStmt)
	if !ok || len(result.Results) != 1 {
		return false
	}
	expression := unparenthesized(result.Results[0])
	dereference, ok := expression.(*ast.StarExpr)
	if !ok {
		return false
	}
	identifier, ok := unparenthesized(dereference.X).(*ast.Ident)
	return ok && info.Uses[identifier] == parameter
}

func unparenthesized(source ast.Expr) ast.Expr {
	for {
		parenthesized, ok := source.(*ast.ParenExpr)
		if !ok {
			return source
		}
		source = parenthesized.X
	}
}

func packagePath(source *types.Package) string {
	if source == nil {
		return ""
	}
	return source.Path()
}
