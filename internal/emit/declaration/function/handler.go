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
) (tsgo.FunctionDeclaration, error) {
	if source.Doc != nil ||
		source.Recv != nil ||
		source.Type == nil ||
		source.Type.Params == nil ||
		source.Type.Results == nil ||
		source.Type.TypeParams != nil ||
		source.Body == nil {
		return nil, api.Unsupported(context, api.CategoryDeclaration, source)
	}

	functionObject, ok := context.TypesInfo().Defs[source.Name].(*types.Func)
	if !ok {
		return nil, api.Unsupported(context, api.CategoryDeclaration, source)
	}
	signature, ok := functionObject.Type().(*types.Signature)
	if !ok ||
		signature.Recv() != nil ||
		signature.TypeParams() != nil ||
		signature.RecvTypeParams() != nil ||
		signature.Variadic() ||
		signature.Results().Len() != 1 {
		return nil, api.Unsupported(context, api.CategoryDeclaration, source)
	}

	name, err := context.Names().Declare(functionObject)
	if err != nil {
		return nil, err
	}
	parameters, err := emitParameters(context, children, source.Type.Params, signature)
	if err != nil {
		return nil, err
	}
	resultType, err := emitResult(context, children, source.Type.Results, signature)
	if err != nil {
		return nil, err
	}
	body, err := children.Block(context.WithRole(api.RoleFunctionBody), source.Body)
	if err != nil {
		return nil, err
	}

	var modifiers []tsgo.ModifierLike
	if functionObject.Exported() {
		modifiers = []tsgo.ModifierLike{context.Factory().ExportKeyword()}
	}
	return context.Factory().FunctionDeclaration(
		modifiers,
		nil,
		context.Factory().Identifier(name),
		nil,
		parameters,
		resultType,
		body,
	), nil
}

func emitParameters(
	context api.Context,
	children api.ChildEmitter,
	fields *ast.FieldList,
	signature *types.Signature,
) ([]tsgo.ParameterDeclaration, error) {
	if fields == nil {
		return nil, &api.InvariantError{Role: context.Role(), Reason: "parameter list is nil"}
	}
	parameters := make([]tsgo.ParameterDeclaration, 0, signature.Params().Len())
	parameterIndex := 0
	for _, field := range fields.List {
		if field.Doc != nil || field.Comment != nil || field.Tag != nil || len(field.Names) == 0 {
			return nil, api.Unsupported(context, api.CategoryDeclaration, field)
		}
		targetType, err := children.Type(context.WithRole(api.RoleParameterType), field.Type)
		if err != nil {
			return nil, err
		}
		for _, sourceName := range field.Names {
			if parameterIndex >= signature.Params().Len() {
				return nil, api.Unsupported(context, api.CategoryDeclaration, field)
			}
			parameter := signature.Params().At(parameterIndex)
			if context.TypesInfo().Defs[sourceName] != parameter ||
				!types.Identical(parameter.Type(), context.TypesInfo().TypeOf(field.Type)) {
				return nil, api.Unsupported(context, api.CategoryDeclaration, field)
			}
			name, err := context.Names().Declare(parameter)
			if err != nil {
				return nil, err
			}
			parameters = append(parameters, context.Factory().ParameterDeclaration(
				nil,
				nil,
				context.Factory().Identifier(name),
				nil,
				targetType,
				nil,
			))
			parameterIndex++
		}
	}
	if parameterIndex != signature.Params().Len() {
		return nil, api.Unsupported(context, api.CategoryDeclaration, fields)
	}
	return parameters, nil
}

func emitResult(
	context api.Context,
	children api.ChildEmitter,
	fields *ast.FieldList,
	signature *types.Signature,
) (tsgo.TypeNode, error) {
	if fields == nil || len(fields.List) != 1 {
		if fields == nil {
			return nil, &api.InvariantError{Role: context.Role(), Reason: "result list is nil"}
		}
		return nil, api.Unsupported(context, api.CategoryDeclaration, fields)
	}
	field := fields.List[0]
	if field.Doc != nil || field.Comment != nil || field.Tag != nil || len(field.Names) != 0 ||
		!types.Identical(signature.Results().At(0).Type(), context.TypesInfo().TypeOf(field.Type)) {
		return nil, api.Unsupported(context, api.CategoryDeclaration, field)
	}
	return children.Type(context.WithRole(api.RoleResultType), field.Type)
}
