package packageconstant

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantbinding "github.com/tsoniclang/gotots/internal/emit/constant"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func EmitObject(
	context api.Context,
	children api.ChildEmitter,
	source *ast.GenDecl,
	selected *types.Const,
) (api.DeclarationEmission, error) {
	if selected == nil {
		return api.DeclarationEmission{},
			&api.InvariantError{
				Role:   context.Role(),
				Reason: "selected package constant is nil",
			}
	}
	if source.Doc != nil || source.Tok != token.CONST || len(source.Specs) == 0 {
		return api.DeclarationEmission{},
			api.Unsupported(context, api.CategoryDeclaration, source)
	}
	var declarations []tsgo.Statement
	var requests []api.RootRequest
	for _, sourceSpec := range source.Specs {
		spec, ok := sourceSpec.(*ast.ValueSpec)
		if !ok {
			return api.DeclarationEmission{},
				api.Unsupported(context, api.CategoryDeclaration, sourceSpec)
		}
		target, targetRequests, found, err := emitSpec(
			context,
			children,
			spec,
			selected,
		)
		if err != nil {
			return api.DeclarationEmission{}, err
		}
		declarations = append(declarations, target...)
		requests = append(requests, targetRequests...)
		if found {
			break
		}
	}
	if len(declarations) == 0 {
		return api.DeclarationEmission{},
			&api.InvariantError{
				Role:   context.Role(),
				Reason: "selected package constant is absent from its declaration",
			}
	}
	return api.NewDeclarationEmission(declarations, requests)
}

func emitSpec(
	context api.Context,
	children api.ChildEmitter,
	source *ast.ValueSpec,
	selected *types.Const,
) ([]tsgo.Statement, []api.RootRequest, bool, error) {
	if source.Doc != nil || source.Comment != nil || len(source.Names) == 0 {
		return nil, nil, false,
			api.Unsupported(context, api.CategoryDeclaration, source)
	}

	declarations := make([]tsgo.Statement, 0, len(source.Names))
	var requests []api.RootRequest
	selectedIndex := -1
	for index, sourceName := range source.Names {
		object, ok := context.TypesInfo().Defs[sourceName].(*types.Const)
		if ok && object == selected {
			selectedIndex = index
			break
		}
	}
	if selectedIndex < 0 {
		return nil, nil, false, nil
	}
	sourceName := source.Names[selectedIndex]
	binding, err := constantbinding.EmitBinding(
		context,
		children,
		sourceName,
		selected,
		api.RolePackageConstantType,
		api.RolePackageConstantValue,
	)
	if err != nil {
		return nil, nil, false, err
	}
	moduleExport, err := context.Names().ModuleExport(selected)
	if err != nil {
		return nil, nil, false, err
	}
	if !moduleExport {
		return nil, nil, false,
			&api.InvariantError{
				Role:   context.Role(),
				Reason: "package constant is not module-exported",
			}
	}
	declarations = append(
		declarations,
		context.Factory().VariableStatement(
			[]tsgo.ModifierLike{context.Factory().ExportKeyword()},
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{binding.Declaration()},
				tsgo.NodeFlagsConst,
			),
		),
	)
	requests = append(requests, binding.Requests()...)
	return declarations, requests, true, nil
}
