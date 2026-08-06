package deferredregistry

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type callableContract struct {
	source         tsgo.TypeNode
	deferred       tsgo.TypeNode
	methodDeferred tsgo.TypeNode
	interfaceValue tsgo.TypeNode
	requests       []api.RootRequest
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
	sourceResult := api.DirectType(result)
	if cooperative {
		sourceResult, err = callable.IndirectResult(context, result)
		if err != nil {
			return callableContract{}, err
		}
	}
	source := context.Factory().FunctionTypeNode(
		nil,
		target.Parameters(),
		sourceResult.Value(),
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
		sourceResult.Value(),
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
	methodResult, err := callable.IndirectResult(context, result)
	if err != nil {
		return callableContract{}, err
	}
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
		methodResult.Value(),
	)
	return callableContract{
		source:         source,
		deferred:       deferred,
		methodDeferred: methodDeferred,
		interfaceValue: interfaceValue,
		requests: api.CombineRequests(
			target.Requests(),
			sourceResult.Requests(),
			methodResult.Requests(),
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
