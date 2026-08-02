package deferredregistry

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type callableContract struct {
	source                    tsgo.TypeNode
	deferred                  tsgo.TypeNode
	methodDeferred            tsgo.TypeNode
	cooperativeMethodDeferred tsgo.TypeNode
	interfaceValue            tsgo.TypeNode
	requests                  []api.RootRequest
}

func Build(
	context api.Context,
	children api.ChildEmitter,
	artifact *api.GeneratedArtifact,
) (tsgo.Statement, tsgo.TypeNode, []api.RootRequest, error) {
	signature, ok := artifact.DeferredCallableRegistry()
	if !ok {
		return nil, nil, nil,
			invariant(context, "registry source signature is invalid")
	}
	contract, err := observedCallableTypes(
		context,
		children,
		nil,
		signature,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	runtimeRegistry, err := context.Names().Runtime(
		api.RuntimeDeferredRegistry,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	factory := context.Factory()
	typeArguments := []tsgo.TypeNode{
		contract.source,
		contract.deferred,
		contract.methodDeferred,
		contract.cooperativeMethodDeferred,
	}
	registryType := factory.TypeReferenceNode(
		runtimeRegistry.EntityName(factory),
		typeArguments,
	)
	statement := factory.VariableStatement(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{factory.VariableDeclaration(
				factory.Identifier(artifact.TargetName()),
				nil,
				nil,
				factory.NewExpression(
					runtimeRegistry.Expression(factory),
					typeArguments,
					nil,
				),
			)},
			tsgo.NodeFlagsConst,
		),
	)
	return statement, registryType, api.CombineRequests(
		contract.requests,
		runtimeRegistry.Requests(),
	), nil
}

func callableTypes(
	context api.Context,
	children api.ChildEmitter,
	signature *types.Signature,
	cooperative bool,
) (callableContract, error) {
	target, err := callable.EmitAdapter(context, children, nil, signature)
	if err != nil {
		return callableContract{}, err
	}
	result := target.Result()
	sourceResult := result
	if cooperative {
		sourceResult = callable.PromiseResult(context.Factory(), result)
	}
	source := context.Factory().FunctionTypeNode(
		nil,
		target.Parameters(),
		sourceResult,
	)
	recovery, recoveryRequests, err :=
		callable.RecoveryAuthorityParameter(context)
	if err != nil {
		return callableContract{}, err
	}
	deferred := context.Factory().FunctionTypeNode(
		nil,
		append(
			[]tsgo.ParameterDeclaration{recovery},
			target.Parameters()...,
		),
		sourceResult,
	)
	runtimeValue, err := context.Names().Runtime(
		api.RuntimeInterfaceValue,
		api.ImportPhaseType,
	)
	if err != nil {
		return callableContract{}, err
	}
	interfaceValue := context.Factory().TypeReferenceNode(
		runtimeValue.EntityName(context.Factory()),
		nil,
	)
	methodDeferred := context.Factory().FunctionTypeNode(
		nil,
		append(
			[]tsgo.ParameterDeclaration{
				recovery,
				parameter(
					context.Factory(),
					context.Factory().Identifier("receiver"),
					interfaceValue,
				),
			},
			target.Parameters()...,
		),
		result,
	)
	cooperativeMethodDeferred := context.Factory().FunctionTypeNode(
		nil,
		append(
			[]tsgo.ParameterDeclaration{
				recovery,
				parameter(
					context.Factory(),
					context.Factory().Identifier("receiver"),
					interfaceValue,
				),
			},
			target.Parameters()...,
		),
		callable.PromiseResult(context.Factory(), result),
	)
	return callableContract{
		source:                    source,
		deferred:                  deferred,
		methodDeferred:            methodDeferred,
		cooperativeMethodDeferred: cooperativeMethodDeferred,
		interfaceValue:            interfaceValue,
		requests: api.CombineRequests(
			target.Requests(),
			recoveryRequests,
			runtimeValue.Requests(),
		),
	}, nil
}

func parameter(
	factory tsgo.Factory,
	name tsgo.Identifier,
	targetType tsgo.TypeNode,
) tsgo.ParameterDeclaration {
	return factory.ParameterDeclaration(nil, nil, name, nil, targetType, nil)
}

func invariant(context api.Context, reason string) error {
	return &api.InvariantError{Role: context.Role(), Reason: reason}
}
