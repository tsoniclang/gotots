package callable

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func EmitBody(
	context api.Context,
	children api.ChildEmitter,
	sourceType *ast.FuncType,
	sourceBody *ast.BlockStmt,
	signature *types.Signature,
	bodyRole api.Role,
) (api.BlockEmission, error) {
	if sourceType == nil || sourceBody == nil || signature == nil {
		return api.BlockEmission{}, &api.InvariantError{
			Role:   bodyRole,
			Reason: "callable body input is nil",
		}
	}
	prologue, prologueRequests, err := namedResultPrologue(
		context,
		children,
		sourceType,
		signature.Results(),
	)
	if err != nil {
		return api.BlockEmission{}, err
	}
	body, err := children.Block(
		context.
			WithRole(bodyRole).
			EnterFunction(signature.Results()),
		sourceBody,
	)
	if err != nil {
		return api.BlockEmission{}, err
	}
	statements := append(prologue, body.Value().Statements()...)
	return api.DirectBlock(
		context.Factory().Block(statements, true),
		api.CombineRequests(prologueRequests, body.Requests())...,
	), nil
}

func namedResultPrologue(
	context api.Context,
	children api.ChildEmitter,
	source *ast.FuncType,
	results *types.Tuple,
) ([]tsgo.Statement, []api.PlacementRequest, error) {
	names, err := namedResultSyntax(context, source, results)
	if err != nil || len(names) == 0 {
		return nil, nil, err
	}
	var statements []tsgo.Statement
	var requests []api.PlacementRequest
	for index, sourceName := range names {
		result := results.At(index)
		targetName, err := context.Names().Declare(result)
		if err != nil {
			return nil, nil, err
		}
		targetType, err := children.RepresentedType(
			context.WithRole(api.RoleResultType),
			sourceName,
			result.Type(),
		)
		if err != nil {
			return nil, nil, err
		}
		zero, err := context.Values().Zero(
			context.WithRole(api.RoleNamedResultZero),
			sourceName,
			result.Type(),
		)
		if err != nil {
			return nil, nil, err
		}
		statements = append(statements, zero.Before()...)
		statements = append(
			statements,
			context.Factory().VariableStatement(
				nil,
				context.Factory().VariableDeclarationList(
					[]tsgo.VariableDeclaration{
						context.Factory().VariableDeclaration(
							context.Factory().Identifier(targetName),
							nil,
							targetType.Value(),
							zero.Value(),
						),
					},
					tsgo.NodeFlagsLet,
				),
			),
		)
		requests = append(
			requests,
			targetType.Requests()...,
		)
		requests = append(requests, zero.Requests()...)
	}
	return statements, requests, nil
}

func namedResultSyntax(
	context api.Context,
	source *ast.FuncType,
	results *types.Tuple,
) ([]*ast.Ident, error) {
	if results == nil || results.Len() == 0 {
		return nil, nil
	}
	var names []*ast.Ident
	if source.Results != nil {
		for _, field := range source.Results.List {
			names = append(names, field.Names...)
		}
	}
	if len(names) == 0 {
		return nil, nil
	}
	if len(names) != results.Len() {
		return nil, api.Unsupported(
			context.WithRole(api.RoleResultType),
			api.CategoryDeclaration,
			source,
		)
	}
	for index, sourceName := range names {
		if sourceName == nil ||
			sourceName.Name == "" ||
			context.TypesInfo().Defs[sourceName] != results.At(index) ||
			results.At(index).Name() == "" {
			return nil, api.Unsupported(
				context.WithRole(api.RoleResultType),
				api.CategoryDeclaration,
				source,
			)
		}
	}
	return names, nil
}
