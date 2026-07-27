package callable

import (
	"go/ast"
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type SignatureEmission struct {
	parameters []tsgo.ParameterDeclaration
	result     tsgo.TypeNode
	requests   []api.RootRequest
}

func (e SignatureEmission) Parameters() []tsgo.ParameterDeclaration {
	return slices.Clone(e.parameters)
}

func (e SignatureEmission) Result() tsgo.TypeNode {
	return e.result
}

func (e SignatureEmission) Requests() []api.RootRequest {
	return slices.Clone(e.requests)
}

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.FuncType,
	signature *types.Signature,
	parameterRole api.Role,
	resultRole api.Role,
) (SignatureEmission, error) {
	if err := validateSyntax(
		context,
		source,
		signature,
		parameterRole,
		resultRole,
	); err != nil {
		return SignatureEmission{}, err
	}
	return emitRepresented(
		context,
		children,
		source,
		signature,
		parameterRole,
		resultRole,
	)
}

func EmitType(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	signature *types.Signature,
) (api.TypeEmission, error) {
	target, err := emitRepresented(
		context,
		children,
		source,
		signature,
		api.RoleCallableParameter,
		api.RoleCallableResult,
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return api.DirectType(
		context.Factory().FunctionTypeNode(
			nil,
			target.Parameters(),
			target.Result(),
		),
		target.Requests()...,
	), nil
}

func EmitSyntaxType(
	context api.Context,
	children api.ChildEmitter,
	source *ast.FuncType,
	signature *types.Signature,
) (api.TypeEmission, error) {
	target, err := Emit(
		context,
		children,
		source,
		signature,
		api.RoleCallableParameter,
		api.RoleCallableResult,
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return api.DirectType(
		context.Factory().FunctionTypeNode(
			nil,
			target.Parameters(),
			target.Result(),
		),
		target.Requests()...,
	), nil
}

func emitRepresented(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	signature *types.Signature,
	parameterRole api.Role,
	resultRole api.Role,
) (SignatureEmission, error) {
	if !Supports(signature) {
		return SignatureEmission{},
			api.Unsupported(context, api.CategoryType, source)
	}
	parameters := make([]tsgo.ParameterDeclaration, 0, signature.Params().Len())
	var requests []api.RootRequest
	for index := range signature.Params().Len() {
		parameter := signature.Params().At(index)
		targetType, err := children.RepresentedType(
			context.WithRole(parameterRole),
			source,
			parameter.Type(),
		)
		if err != nil {
			return SignatureEmission{}, err
		}
		name, err := context.Names().Parameter(parameter, index)
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
		requests = append(requests, targetType.Requests()...)
	}
	result, resultRequests, err := emitResultType(
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
		parameters: parameters,
		result:     result,
		requests:   requests,
	}, nil
}

func emitResultType(
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
) error {
	if source == nil {
		return &api.InvariantError{
			Role:   context.Role(),
			Reason: "callable syntax is nil",
		}
	}
	if source.Params == nil ||
		source.TypeParams != nil ||
		!Supports(signature) {
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
			field.Doc != nil ||
			field.Comment != nil ||
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
			if index >= count ||
				fieldType == nil ||
				!types.Identical(fieldType, variables.At(index).Type()) {
				return api.Unsupported(context, api.CategoryType, field)
			}
			variable := variables.At(index)
			if len(field.Names) == 0 {
				if variable.Name() != "" {
					return api.Unsupported(context, api.CategoryType, field)
				}
			} else if context.TypesInfo().Defs[field.Names[fieldIndex]] != variable {
				return api.Unsupported(context, api.CategoryType, field.Names[fieldIndex])
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
	signature, ok := types.Unalias(sourceType).(*types.Signature)
	return signature, ok && Supports(signature)
}

func Supports(signature *types.Signature) bool {
	return signature != nil &&
		!signature.Variadic() &&
		signature.TypeParams().Len() == 0 &&
		signature.RecvTypeParams().Len() == 0
}
