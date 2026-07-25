package function

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.FuncDecl,
) (api.DeclarationEmission, error) {
	if source.Doc != nil ||
		source.Recv != nil ||
		source.Type == nil ||
		source.Type.Params == nil ||
		source.Type.Results == nil ||
		source.Type.TypeParams != nil ||
		source.Body == nil {
		return api.DeclarationEmission{},
			api.Unsupported(context, api.CategoryDeclaration, source)
	}

	functionObject, ok := context.TypesInfo().Defs[source.Name].(*types.Func)
	if !ok {
		return api.DeclarationEmission{},
			api.Unsupported(context, api.CategoryDeclaration, source)
	}
	signature, ok := functionObject.Type().(*types.Signature)
	if !ok ||
		signature.Recv() != nil ||
		signature.TypeParams() != nil ||
		signature.RecvTypeParams() != nil ||
		signature.Variadic() ||
		signature.Results().Len() != 1 {
		return api.DeclarationEmission{},
			api.Unsupported(context, api.CategoryDeclaration, source)
	}

	name, err := context.Names().Declare(functionObject)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	parameters, parameterRequests, err := emitParameters(
		context,
		children,
		source.Type.Params,
		signature,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	resultType, err := emitResult(context, children, source.Type.Results, signature)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	body, err := children.Block(
		context.
			WithRole(api.RoleFunctionBody).
			EnterFunction(signature.Results()),
		source.Body,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}

	moduleExport, err := context.Names().ModuleExport(functionObject)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	var modifiers []tsgo.ModifierLike
	if moduleExport {
		modifiers = []tsgo.ModifierLike{context.Factory().ExportKeyword()}
	}
	target := context.Factory().FunctionDeclaration(
		modifiers,
		nil,
		context.Factory().Identifier(name),
		nil,
		parameters,
		resultType.Value(),
		body.Value(),
	)
	return api.DirectDeclaration(
		target,
		api.CombineRequests(
			parameterRequests,
			resultType.Requests(),
			body.Requests(),
		)...,
	), nil
}

func emitParameters(
	context api.Context,
	children api.ChildEmitter,
	fields *ast.FieldList,
	signature *types.Signature,
) ([]tsgo.ParameterDeclaration, []api.PlacementRequest, error) {
	if fields == nil {
		return nil, nil,
			&api.InvariantError{Role: context.Role(), Reason: "parameter list is nil"}
	}
	parameters := make([]tsgo.ParameterDeclaration, 0, signature.Params().Len())
	var requests []api.PlacementRequest
	parameterIndex := 0
	for _, field := range fields.List {
		if field.Doc != nil || field.Comment != nil || field.Tag != nil || len(field.Names) == 0 {
			return nil, nil, api.Unsupported(context, api.CategoryDeclaration, field)
		}
		targetType, err := children.Type(context.WithRole(api.RoleParameterType), field.Type)
		if err != nil {
			return nil, nil, err
		}
		requests = append(requests, targetType.Requests()...)
		for _, sourceName := range field.Names {
			if parameterIndex >= signature.Params().Len() {
				return nil, nil, api.Unsupported(context, api.CategoryDeclaration, field)
			}
			parameter := signature.Params().At(parameterIndex)
			if context.TypesInfo().Defs[sourceName] != parameter ||
				!types.Identical(parameter.Type(), context.TypesInfo().TypeOf(field.Type)) {
				return nil, nil, api.Unsupported(context, api.CategoryDeclaration, field)
			}
			name, err := context.Names().Declare(parameter)
			if err != nil {
				return nil, nil, err
			}
			parameters = append(parameters, context.Factory().ParameterDeclaration(
				nil,
				nil,
				context.Factory().Identifier(name),
				nil,
				targetType.Value(),
				nil,
			))
			parameterIndex++
		}
	}
	if parameterIndex != signature.Params().Len() {
		return nil, nil, api.Unsupported(context, api.CategoryDeclaration, fields)
	}
	return parameters, requests, nil
}

func emitResult(
	context api.Context,
	children api.ChildEmitter,
	fields *ast.FieldList,
	signature *types.Signature,
) (api.TypeEmission, error) {
	if fields == nil || len(fields.List) != 1 {
		if fields == nil {
			return api.TypeEmission{},
				&api.InvariantError{Role: context.Role(), Reason: "result list is nil"}
		}
		return api.TypeEmission{},
			api.Unsupported(context, api.CategoryDeclaration, fields)
	}
	field := fields.List[0]
	if field.Doc != nil || field.Comment != nil || field.Tag != nil || len(field.Names) != 0 ||
		!types.Identical(signature.Results().At(0).Type(), context.TypesInfo().TypeOf(field.Type)) {
		return api.TypeEmission{},
			api.Unsupported(context, api.CategoryDeclaration, field)
	}
	return children.Type(context.WithRole(api.RoleResultType), field.Type)
}
