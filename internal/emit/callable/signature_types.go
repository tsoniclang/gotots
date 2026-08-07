package callable

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func EmitSyntaxType(
	context api.Context,
	children api.ChildEmitter,
	source *ast.FuncType,
	signature *types.Signature,
) (api.TypeEmission, error) {
	if err := validateSyntax(
		context,
		source,
		signature,
		api.RoleCallableParameter,
		api.RoleCallableResult,
		false,
	); err != nil {
		return api.TypeEmission{}, err
	}
	return EmitNonNilType(context, children, source, signature)
}

func emitRepresented(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	signature *types.Signature,
	parameterRole api.Role,
	resultRole api.Role,
	parameterName func(*types.Var, int) (string, error),
	allowTypeParameters bool,
) (SignatureEmission, error) {
	if (!allowTypeParameters && !Supports(signature)) ||
		signature == nil ||
		parameterName == nil {
		return SignatureEmission{},
			api.Unsupported(context, api.CategoryType, source)
	}
	parameters := make([]tsgo.ParameterDeclaration, 0, signature.Params().Len())
	parameterNames := make([]string, 0, signature.Params().Len())
	var requests []api.RootRequest
	for index := range signature.Params().Len() {
		parameter := signature.Params().At(index)
		name, err := parameterName(parameter, index)
		if err != nil {
			return SignatureEmission{}, err
		}
		if signature.Variadic() && index == signature.Params().Len()-1 {
			parameterDeclaration, parameterRequests, err := emitVariadicParameter(
				context,
				children,
				source,
				parameter,
				name,
				parameterRole,
			)
			if err != nil {
				return SignatureEmission{}, err
			}
			parameters = append(parameters, parameterDeclaration)
			parameterNames = append(parameterNames, name)
			requests = append(requests, parameterRequests...)
			continue
		}
		parameterType := parameter.Type()
		targetType, err := children.RepresentedType(
			context.WithRole(parameterRole),
			source,
			parameterType,
		)
		if err != nil {
			return SignatureEmission{}, err
		}
		parameters = append(parameters, context.Factory().ParameterDeclaration(
			nil,
			nil,
			context.Factory().Identifier(name),
			nil,
			targetType.Value(),
			nil,
		))
		parameterNames = append(parameterNames, name)
		requests = append(requests, targetType.Requests()...)
	}
	result, resultRequests, err := EmitResultType(
		context.WithRole(resultRole),
		children,
		source,
		signature.Results(),
	)
	if err != nil {
		return SignatureEmission{}, err
	}
	requests = append(requests, resultRequests...)
	return SignatureEmission{
		parameters:     parameters,
		parameterNames: parameterNames,
		result:         result,
		requests:       requests,
	}, nil
}

func EmitResultType(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	results *types.Tuple,
) (tsgo.TypeNode, []api.RootRequest, error) {
	if results == nil || results.Len() == 0 {
		return context.Factory().KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindVoidKeyword,
		), nil, nil
	}
	elements := make([]tsgo.TypeNode, 0, results.Len())
	var requests []api.RootRequest
	for index := range results.Len() {
		target, err := children.RepresentedType(
			context,
			source,
			results.At(index).Type(),
		)
		if err != nil {
			return nil, nil, err
		}
		elements = append(elements, target.Value())
		requests = append(requests, target.Requests()...)
	}
	if len(elements) == 1 {
		return elements[0], requests, nil
	}
	return context.Factory().TupleTypeNode(elements), requests, nil
}

func validateSyntax(
	context api.Context,
	source *ast.FuncType,
	signature *types.Signature,
	parameterRole api.Role,
	resultRole api.Role,
	allowTypeParameters bool,
) error {
	if source == nil {
		return &api.InvariantError{
			Role:   context.Role(),
			Reason: "callable syntax is nil",
		}
	}
	if source.Params == nil ||
		(!allowTypeParameters && source.TypeParams != nil) ||
		(!allowTypeParameters && !Supports(signature)) {
		return api.Unsupported(context, api.CategoryType, source)
	}
	if err := validateFields(
		context.WithRole(parameterRole),
		source,
		source.Params,
		signature.Params(),
	); err != nil {
		return err
	}
	return validateFields(
		context.WithRole(resultRole),
		source,
		source.Results,
		signature.Results(),
	)
}

func validateFields(
	context api.Context,
	owner ast.Node,
	fields *ast.FieldList,
	variables *types.Tuple,
) error {
	count := 0
	if variables != nil {
		count = variables.Len()
	}
	if count == 0 {
		if fields == nil || len(fields.List) == 0 {
			return nil
		}
		return api.Unsupported(context, api.CategoryType, fields)
	}
	if fields == nil {
		return api.Unsupported(context, api.CategoryType, owner)
	}
	index := 0
	for _, field := range fields.List {
		if field == nil ||
			field.Tag != nil ||
			field.Type == nil {
			return api.Unsupported(context, api.CategoryType, field)
		}
		fieldCount := len(field.Names)
		if fieldCount == 0 {
			fieldCount = 1
		}
		fieldType := context.TypesInfo().TypeOf(field.Type)
		for fieldIndex := range fieldCount {
			if index >= count || !fieldMatchesVariable(
				context,
				field,
				fieldType,
				variables.At(index),
			) {
				return api.Unsupported(context, api.CategoryType, field)
			}
			variable := variables.At(index)
			if len(field.Names) == 0 {
				if variable.Name() != "" {
					return api.Unsupported(context, api.CategoryType, field)
				}
			} else if context.TypesInfo().DefOf(
				field.Names[fieldIndex],
			) != variable {
				return api.Unsupported(
					context,
					api.CategoryType,
					field.Names[fieldIndex],
				)
			}
			index++
		}
	}
	if index != count {
		return api.Unsupported(context, api.CategoryType, fields)
	}
	return nil
}

func Signature(sourceType types.Type) (*types.Signature, bool) {
	if sourceType == nil {
		return nil, false
	}
	sourceType = types.Unalias(sourceType)
	if named, ok := sourceType.(*types.Named); ok {
		sourceType = named.Underlying()
	}
	signature, ok := sourceType.(*types.Signature)
	if !ok {
		return nil, false
	}
	return ValueSignature(signature)
}

func IsValue(sourceType types.Type) bool {
	if sourceType == nil {
		return false
	}
	sourceType = types.Unalias(sourceType)
	if named, ok := sourceType.(*types.Named); ok {
		sourceType = named.Underlying()
	}
	signature, ok := sourceType.(*types.Signature)
	return ok && signature.Recv() == nil && signature.TypeParams().Len() == 0
}

func Supports(signature *types.Signature) bool {
	if signature == nil || signature.TypeParams().Len() != 0 {
		return false
	}
	if signature.RecvTypeParams().Len() == 0 {
		return true
	}
	if api.ContainsGenericTypeParameter(signature) {
		return false
	}
	if signature.Recv() == nil {
		return true
	}
	receiverType := signature.Recv().Type()
	if pointer, ok := types.Unalias(receiverType).(*types.Pointer); ok {
		receiverType = pointer.Elem()
	}
	named, ok := types.Unalias(receiverType).(*types.Named)
	return ok &&
		named.TypeParams().Len() != 0 &&
		named.TypeArgs().Len() == named.TypeParams().Len()
}

func fieldMatchesVariable(
	context api.Context,
	field *ast.Field,
	fieldType types.Type,
	variable *types.Var,
) bool {
	if field == nil || variable == nil {
		return false
	}
	if ellipsis, ok := field.Type.(*ast.Ellipsis); ok {
		if ellipsis.Elt == nil {
			return false
		}
		parameterType, ok := types.Unalias(variable.Type()).(*types.Slice)
		if !ok {
			return false
		}
		elementType := context.TypesInfo().TypeOf(ellipsis.Elt)
		return elementType != nil &&
			types.Identical(elementType, parameterType.Elem())
	}
	return fieldType != nil && types.Identical(fieldType, variable.Type())
}
