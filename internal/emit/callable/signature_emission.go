package callable

import (
	"go/ast"
	"go/types"
	"slices"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type SignatureEmission struct {
	parameters     []tsgo.ParameterDeclaration
	parameterNames []string
	result         tsgo.TypeNode
	requests       []api.RootRequest
	recovery       bool
}

func EmitAdapter(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	signature *types.Signature,
) (SignatureEmission, error) {
	return emitRepresented(
		context,
		children,
		source,
		signature,
		api.RoleCallableParameter,
		api.RoleCallableResult,
		func(_ *types.Var, index int) (string, error) {
			return "$argument" + strconv.Itoa(index), nil
		},
		false,
	)
}

func EmitAdapterWithRootInterfaceParameter(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	signature *types.Signature,
) (SignatureEmission, error) {
	if signature == nil || signature.Params().Len() != 1 {
		return SignatureEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "root-interface adapter requires exactly one parameter",
		}
	}
	if _, ok := signature.Params().At(0).Type().Underlying().(*types.Interface); !ok {
		return SignatureEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "root-interface adapter parameter is not an interface",
		}
	}
	return emitRepresentedWithParameterType(
		context,
		children,
		source,
		signature,
		api.RoleCallableParameter,
		api.RoleCallableResult,
		func(_ *types.Var, index int) (string, error) {
			return "$argument" + strconv.Itoa(index), nil
		},
		false,
		func(_ *types.Var, index int) (api.TypeEmission, bool, error) {
			if index != 0 {
				return api.TypeEmission{}, false, nil
			}
			runtimeValue, err := context.Names().Runtime(
				api.RuntimeInterfaceValue,
				api.ImportPhaseType,
			)
			if err != nil {
				return api.TypeEmission{}, false, err
			}
			return api.DirectType(
				context.Factory().UnionTypeNode([]tsgo.TypeNode{
					context.Factory().TypeReferenceNode(
						runtimeValue.EntityName(context.Factory()),
						nil,
					),
					context.Factory().KeywordTypeNode(
						tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
					),
				}),
				runtimeValue.Requests()...,
			), true, nil
		},
	)
}

func EmitAdapterWithSynchronousParameters(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	signature *types.Signature,
	indexes []int,
) (SignatureEmission, error) {
	selected := make(map[int]struct{}, len(indexes))
	for _, index := range indexes {
		if index < 0 || signature == nil || index >= signature.Params().Len() {
			return SignatureEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "synchronous callable parameter index is invalid",
			}
		}
		selected[index] = struct{}{}
	}
	return emitRepresentedWithParameterType(
		context,
		children,
		source,
		signature,
		api.RoleCallableParameter,
		api.RoleCallableResult,
		func(_ *types.Var, index int) (string, error) {
			return "$argument" + strconv.Itoa(index), nil
		},
		false,
		func(parameter *types.Var, index int) (api.TypeEmission, bool, error) {
			if _, ok := selected[index]; !ok {
				return api.TypeEmission{}, false, nil
			}
			if signature.Variadic() && index == signature.Params().Len()-1 {
				return api.TypeEmission{}, false, &api.InvariantError{
					Role:   context.Role(),
					Reason: "synchronous callable parameter is variadic",
				}
			}
			callableSignature, ok := Signature(parameter.Type())
			if !ok {
				return api.TypeEmission{}, false, &api.InvariantError{
					Role:   context.Role(),
					Reason: "synchronous parameter is not callable",
				}
			}
			target, err := EmitInlineNonNilType(
				context.WithRole(api.RoleCallableParameter),
				children,
				source,
				callableSignature,
				false,
			)
			if err != nil {
				return api.TypeEmission{}, false, err
			}
			return api.DirectType(
				context.Factory().UnionTypeNode([]tsgo.TypeNode{
					target.Value(),
					context.Factory().KeywordTypeNode(
						tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
					),
				}),
				target.Requests()...,
			), true, nil
		},
	)
}

func EmitEnvironmentContract(
	context api.Context,
	children api.ChildEmitter,
	signature *types.Signature,
) (SignatureEmission, error) {
	return emitRepresented(
		context,
		children,
		nil,
		signature,
		api.RoleCallableParameter,
		api.RoleCallableResult,
		func(_ *types.Var, index int) (string, error) {
			return "$argument" + strconv.Itoa(index), nil
		},
		true,
	)
}

func EmitABIAdapter(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	signature *types.Signature,
) (SignatureEmission, error) {
	return EmitAdapter(context, children, source, signature)
}

func (e SignatureEmission) Parameters() []tsgo.ParameterDeclaration {
	return slices.Clone(e.parameters)
}

func (e SignatureEmission) ParameterReferences(
	factory tsgo.Factory,
) []tsgo.Expression {
	result := make([]tsgo.Expression, 0, len(e.parameterNames))
	for _, name := range e.parameterNames {
		result = append(result, factory.Identifier(name))
	}
	return result
}

func (e SignatureEmission) SourceParameterReferences(
	factory tsgo.Factory,
) []tsgo.Expression {
	result := make([]tsgo.Expression, 0, len(e.parameterNames))
	for _, name := range e.parameterNames {
		result = append(result, factory.Identifier(name))
	}
	return result
}

func (e SignatureEmission) RecoveryAuthorityReference(
	factory tsgo.Factory,
) (tsgo.Expression, bool) {
	return nil, false
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
		false,
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
		context.Names().Parameter,
		false,
	)
}

func EmitDeclaration(
	context api.Context,
	children api.ChildEmitter,
	source *ast.FuncType,
	signature *types.Signature,
	parameterRole api.Role,
	resultRole api.Role,
) (SignatureEmission, error) {
	sourceSignature, err := sourceCallableSignature(context, signature)
	if err != nil {
		return SignatureEmission{}, err
	}
	if err := validateSyntax(
		context,
		source,
		sourceSignature,
		parameterRole,
		resultRole,
		true,
	); err != nil {
		return SignatureEmission{}, err
	}
	parameterName := context.Names().Parameter
	if _, generic := context.GenericParameterOwner(); generic {
		parameterName = func(
			parameter *types.Var,
			index int,
		) (string, error) {
			sourceParameter := sourceSignature.Params().At(index)
			return context.Names().Parameter(sourceParameter, index)
		}
	}
	return emitRepresented(
		context,
		children,
		source,
		signature,
		parameterRole,
		resultRole,
		parameterName,
		true,
	)
}

func functionSignature(function *types.Func) (*types.Signature, bool) {
	if function == nil {
		return nil, false
	}
	signature, ok := function.Type().(*types.Signature)
	return signature, ok
}

func EmitType(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	signature *types.Signature,
) (api.TypeEmission, error) {
	signature, ok := ValueSignature(signature)
	if !ok {
		return api.TypeEmission{}, api.Unsupported(
			context,
			api.CategoryType,
			source,
		)
	}
	if context.EnvironmentContract() {
		target, err := emitEnvironmentNonNilType(
			context,
			children,
			source,
			signature,
		)
		if err != nil {
			return api.TypeEmission{}, err
		}
		return optionalCallableType(context, target), nil
	}
	target, err := EmitNonNilType(context, children, source, signature)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return optionalCallableType(context, target), nil
}

func EmitSynchronousType(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	signature *types.Signature,
) (api.TypeEmission, error) {
	signature, ok := ValueSignature(signature)
	if !ok {
		return api.TypeEmission{}, api.Unsupported(
			context,
			api.CategoryType,
			source,
		)
	}
	target, err := EmitInlineAwaitableType(
		context,
		children,
		source,
		signature,
		false,
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return optionalCallableType(context, target), nil
}

func optionalCallableType(
	context api.Context,
	target api.TypeEmission,
) api.TypeEmission {
	return api.DirectType(
		context.Factory().UnionTypeNode([]tsgo.TypeNode{
			target.Value(),
			context.Factory().KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
			),
		}),
		target.Requests()...,
	)
}

func emitEnvironmentNonNilType(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	signature *types.Signature,
) (api.TypeEmission, error) {
	return EmitInlineAwaitableType(
		context,
		children,
		source,
		signature,
		context.ConcurrencySemantics() ==
			api.ConcurrencySemanticsCooperative,
	)
}

func EmitNonNilType(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	signature *types.Signature,
) (api.TypeEmission, error) {
	return EmitInlineAwaitableType(
		context,
		children,
		source,
		signature,
		context.ConcurrencySemantics() ==
			api.ConcurrencySemanticsCooperative,
	)
}

func ABIReference(
	context api.Context,
	signature *types.Signature,
) (api.CallableABIReference, error) {
	if owner, generic := context.GenericParameterOwner(); generic {
		return context.Names().SourceCallableABI(owner, signature)
	}
	return context.Names().CallableABI(signature)
}

func EmitInternalNonNilType(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	signature *types.Signature,
) (api.TypeEmission, error) {
	return emitInternalNonNilType(
		context,
		children,
		source,
		signature,
		false,
	)
}

func emitInternalNonNilType(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	signature *types.Signature,
	cooperative bool,
) (api.TypeEmission, error) {
	target, err := emitRepresented(
		context,
		children,
		source,
		signature,
		api.RoleCallableParameter,
		api.RoleCallableResult,
		func(_ *types.Var, index int) (string, error) {
			return "$" + strconv.Itoa(index), nil
		},
		false,
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	result := target.Result()
	if cooperative {
		result = PromiseResult(context.Factory(), result)
	}
	return api.DirectType(
		context.Factory().FunctionTypeNode(
			nil,
			target.Parameters(),
			result,
		),
		target.Requests()...,
	), nil
}
