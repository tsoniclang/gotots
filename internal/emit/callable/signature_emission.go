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
	target, err := EmitAdapter(context, children, source, signature)
	if err != nil {
		return SignatureEmission{}, err
	}
	return withRecoveryAuthority(context, target)
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
	names := e.parameterNames
	if e.recovery {
		names = names[:len(names)-1]
	}
	result := make([]tsgo.Expression, 0, len(names))
	for _, name := range names {
		result = append(result, factory.Identifier(name))
	}
	return result
}

func (e SignatureEmission) RecoveryAuthorityReference(
	factory tsgo.Factory,
) (tsgo.Expression, bool) {
	if !e.recovery || len(e.parameterNames) == 0 {
		return nil, false
	}
	return factory.Identifier(e.parameterNames[len(e.parameterNames)-1]), true
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
	if err := validateSyntax(
		context,
		source,
		signature,
		parameterRole,
		resultRole,
		true,
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
		true,
	)
}

func EmitType(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	signature *types.Signature,
) (api.TypeEmission, error) {
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
	profile, profiled := context.GenericCallableProfile()
	if !profiled {
		return EmitInternalNonNilType(
			context,
			children,
			source,
			signature,
		)
	}
	reference, err := ABIReference(context, signature)
	if err != nil {
		return api.TypeEmission{}, err
	}
	cooperative, selected :=
		profile.Selection().ABI(reference.Artifact())
	if !selected {
		return EmitInternalNonNilType(
			context,
			children,
			source,
			signature,
		)
	}
	target, err := emitInternalNonNilType(
		context,
		children,
		source,
		signature,
		cooperative,
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return api.DirectType(
		target.Value(),
		api.CombineRequests(
			reference.Requests(),
			target.Requests(),
		)...,
	), nil
}

func EmitNonNilType(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	signature *types.Signature,
) (api.TypeEmission, error) {
	reference, err := ABIReference(context, signature)
	if err != nil {
		return api.TypeEmission{}, err
	}
	facet, err := context.CallableABIFacet(reference.Artifact())
	if err != nil {
		return api.TypeEmission{}, err
	}
	observation, err := context.ObserveCooperativeCallable(facet)
	if err != nil {
		return api.TypeEmission{}, err
	}
	target, err := EmitInlineNonNilType(
		context,
		children,
		source,
		signature,
		observation.Cooperative(),
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return api.DirectType(
		target.Value(),
		api.CombineRequests(
			reference.Requests(),
			observation.Requests(),
			target.Requests(),
		)...,
	), nil
}

func ABIReference(
	context api.Context,
	signature *types.Signature,
) (api.CallableABIReference, error) {
	if profile, profiled := context.GenericCallableProfile(); profiled {
		return context.Names().SourceCallableABI(
			profile.Owner(),
			signature,
		)
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

func withRecoveryAuthority(
	context api.Context,
	target SignatureEmission,
) (SignatureEmission, error) {
	if target.recovery {
		return SignatureEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "callable signature already carries recovery authority",
		}
	}
	parameter, requests, err := RecoveryAuthorityParameter(context)
	if err != nil {
		return SignatureEmission{}, err
	}
	target.parameters = append(target.parameters, parameter)
	target.parameterNames = append(
		target.parameterNames,
		RecoveryAuthorityName,
	)
	target.requests = append(target.requests, requests...)
	target.recovery = true
	return target, nil
}
