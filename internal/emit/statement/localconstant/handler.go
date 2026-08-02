package localconstant

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantbinding "github.com/tsoniclang/gotots/internal/emit/constant"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.DeclStmt,
) (api.StatementEmission, error) {
	declaration, ok := source.Decl.(*ast.GenDecl)
	if !ok ||
		declaration.Tok != token.CONST ||
		len(declaration.Specs) == 0 {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	var statements []tsgo.Statement
	var requests []api.RootRequest
	for _, sourceSpec := range declaration.Specs {
		spec, ok := sourceSpec.(*ast.ValueSpec)
		if !ok ||
			len(spec.Names) == 0 {
			return api.StatementEmission{},
				api.Unsupported(
					context.WithRole(api.RoleLocalDeclaration),
					api.CategoryDeclaration,
					sourceSpec,
				)
		}
		declarations := make([]tsgo.VariableDeclaration, 0, len(spec.Names))
		for _, sourceName := range spec.Names {
			selected, ok := context.TypesInfo().DefOf(sourceName).(*types.Const)
			if !ok {
				return api.StatementEmission{},
					api.Unsupported(
						context.WithRole(api.RoleLocalDeclaration),
						api.CategoryDeclaration,
						sourceName,
					)
			}
			if selected.Name() == "_" {
				continue
			}
			if constantbinding.IsUntyped(selected.Type()) {
				base, err := context.Names().Declare(selected)
				if err != nil {
					return api.StatementEmission{}, err
				}
				for _, projection := range context.LocalConstantProjections(selected) {
					projectionName, err := api.ConstantProjectionName(
						base,
						projection,
					)
					if err != nil {
						return api.StatementEmission{}, err
					}
					target, err := constantbinding.EmitProjection(
						context,
						children,
						sourceName,
						selected,
						projectionName,
						projection,
						api.RoleLocalConstantType,
						api.RoleLocalConstantValue,
					)
					if err != nil {
						return api.StatementEmission{}, err
					}
					declarations = append(
						declarations,
						target.Declaration(),
					)
					requests = append(requests, target.Requests()...)
				}
				continue
			}
			target, err := constantbinding.EmitBinding(
				context,
				children,
				sourceName,
				selected,
				api.RoleLocalConstantType,
				api.RoleLocalConstantValue,
			)
			if err != nil {
				return api.StatementEmission{}, err
			}
			declarations = append(declarations, target.Declaration())
			requests = append(requests, target.Requests()...)
		}
		if len(declarations) == 0 {
			continue
		}
		statements = append(
			statements,
			context.Factory().VariableStatement(
				nil,
				context.Factory().VariableDeclarationList(
					declarations,
					tsgo.NodeFlagsConst,
				),
			),
		)
	}
	return api.NewStatementEmission(statements, requests)
}
