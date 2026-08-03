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
		return api.DirectType(
			context.Factory().UnionTypeNode([]tsgo.TypeNode{
				target.Value(),
				context.Factory().KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
				),
			}),
			target.Requests()...,
		), nil
	}
	target, err := EmitNonNilType(context, children, source, signature)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return api.DirectType(
		context.Factory().UnionTypeNode([]tsgo.TypeNode{
			target.Value(),
			context.Factory().KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
			),
		}),
		target.Requests()...,
	), nil
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
