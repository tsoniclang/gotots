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
	sourceCallable ast.Node,
	sourceType *ast.FuncType,
	sourceBody *ast.BlockStmt,
	signature *types.Signature,
	bodyRole api.Role,
) (api.BlockEmission, error) {
	if sourceCallable == nil ||
		sourceType == nil ||
		sourceBody == nil ||
		signature == nil {
		return api.BlockEmission{}, &api.InvariantError{
			Role:   bodyRole,
			Reason: "callable body input is nil",
		}
	}
	sourceSignature, err := sourceCallableSignature(context, signature)
	if err != nil {
		return api.BlockEmission{}, err
	}
	parameterPrologue, parameterRequests, err := addressableParameterPrologue(
		context,
		children,
		sourceType,
		signature,
	)
	if err != nil {
		return api.BlockEmission{}, err
	}
	prologue, prologueRequests, namedResults, err := namedResultPrologue(
		context,
		children,
		sourceType,
		sourceSignature.Results(),
	)
	if err != nil {
		return api.BlockEmission{}, err
	}
	callableContext := context.
		WithRole(bodyRole).
		EnterCallable(sourceCallable, signature.Results())
	var body api.BlockEmission
	if callableContext.CallableControl().Defer() {
		body, err = emitDeferredBody(
			callableContext,
			children,
			sourceBody,
			signature,
			namedResults,
		)
	} else {
		body, err = children.Block(callableContext, sourceBody)
	}
	if err != nil {
		return api.BlockEmission{}, err
	}
	statements := append(parameterPrologue, prologue...)
	statements = append(statements, body.Value().Statements()...)
	return api.DirectBlock(
		context.Factory().Block(statements, true),
		api.CombineRequests(
			parameterRequests,
			prologueRequests,
			body.Requests(),
		)...,
	), nil
}

func namedResultPrologue(
	context api.Context,
	children api.ChildEmitter,
	source *ast.FuncType,
	results *types.Tuple,
) ([]tsgo.Statement, []api.RootRequest, bool, error) {
	names, err := namedResultSyntax(context, source, results)
	if err != nil || len(names) == 0 {
		return nil, nil, false, err
	}
	var statements []tsgo.Statement
	var requests []api.RootRequest
	for index, sourceName := range names {
		result := results.At(index)
		resultType := context.TypesInfo().TypeOfObject(result)
		targetName, selected := context.AddressableStorage().Name(context, result)
		if !selected {
			targetName, err = context.Names().Result(result, index)
		}
		if err != nil {
			return nil, nil, false, err
		}
		var declarationType tsgo.TypeNode
		if !selected {
			targetType, err := children.RepresentedType(
				context.WithRole(api.RoleResultType),
				sourceName,
				resultType,
			)
			if err != nil {
				return nil, nil, false, err
			}
			declarationType = targetType.Value()
			requests = append(requests, targetType.Requests()...)
		}
		zero, err := context.Values().Zero(
			context.WithRole(api.RoleNamedResultZero),
			sourceName,
			resultType,
		)
		if err != nil {
			return nil, nil, false, err
		}
		if selected {
			zero, err = context.AddressableStorage().Cell(
				context,
				children,
				sourceName,
				resultType,
				zero,
			)
			if err != nil {
				return nil, nil, false, err
			}
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
							declarationType,
							zero.Value(),
						),
					},
					tsgo.NodeFlagsLet,
				),
			),
		)
		requests = append(requests, zero.Requests()...)
	}
	return statements, requests, true, nil
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
			context.TypesInfo().DefOf(sourceName) != results.At(index) ||
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
