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
		declaration.Doc != nil ||
		declaration.Tok != token.CONST ||
		len(declaration.Specs) == 0 {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	var statements []tsgo.Statement
	var requests []api.PlacementRequest
	for _, sourceSpec := range declaration.Specs {
		spec, ok := sourceSpec.(*ast.ValueSpec)
		if !ok ||
			spec.Doc != nil ||
			spec.Comment != nil ||
			spec.Type == nil ||
			len(spec.Names) == 0 ||
			len(spec.Names) != len(spec.Values) {
			return api.StatementEmission{},
				api.Unsupported(
					context.WithRole(api.RoleLocalDeclaration),
					api.CategoryDeclaration,
					sourceSpec,
				)
		}
		declarations := make([]tsgo.VariableDeclaration, 0, len(spec.Names))
		for index, sourceName := range spec.Names {
			selected, ok := context.TypesInfo().Defs[sourceName].(*types.Const)
			if !ok {
				return api.StatementEmission{},
					api.Unsupported(
						context.WithRole(api.RoleLocalDeclaration),
						api.CategoryDeclaration,
						sourceName,
					)
			}
			target, err := constantbinding.EmitBinding(
				context,
				children,
				sourceName,
				spec.Type,
				spec.Values[index],
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
