package localtype

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	definedtype "github.com/tsoniclang/gotots/internal/emit/declaration/definedtype"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.DeclStmt,
) (api.StatementEmission, error) {
	declaration, ok := source.Decl.(*ast.GenDecl)
	if !ok || declaration.Tok != token.TYPE || len(declaration.Specs) == 0 {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	typeNames := make([]*types.TypeName, 0, len(declaration.Specs))
	for _, candidate := range declaration.Specs {
		spec, ok := candidate.(*ast.TypeSpec)
		if !ok {
			return api.StatementEmission{},
				api.Unsupported(
					context.WithRole(api.RoleLocalDeclaration),
					api.CategoryDeclaration,
					candidate,
				)
		}
		typeName, ok := context.TypesInfo().Defs[spec.Name].(*types.TypeName)
		if !ok {
			return api.StatementEmission{},
				api.Unsupported(
					context.WithRole(api.RoleLocalDeclaration),
					api.CategoryDeclaration,
					spec.Name,
				)
		}
		if _, err := context.Names().Declare(typeName); err != nil {
			return api.StatementEmission{}, err
		}
		typeNames = append(typeNames, typeName)
	}

	var statements []tsgo.Statement
	var requests []api.RootRequest
	for _, typeName := range typeNames {
		target, handled, err := definedtype.Emit(
			context.WithRole(api.RoleLocalDeclaration),
			children,
			declaration,
			typeName,
			nil,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		if !handled ||
			target.Disposition() != api.DeclarationDispositionMaterialized {
			return api.StatementEmission{},
				api.Unsupported(
					context.WithRole(api.RoleLocalDeclaration),
					api.CategoryDeclaration,
					source,
				)
		}
		statements = append(statements, target.Declarations()...)
		requests = append(requests, target.Requests()...)
	}
	return api.NewStatementEmission(statements, requests)
}
