package providerinterfacebridge

import (
	"go/types"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Build(
	context api.Context,
	children api.ChildEmitter,
	name string,
	source *types.Named,
	modifiers []tsgo.ModifierLike,
) ([]tsgo.Statement, []api.RootRequest, error) {
	if name == "" || source == nil || source.Obj() == nil {
		return nil, nil, shapeError(name, "bridge identity is invalid")
	}
	contract, ok := source.Underlying().(*types.Interface)
	if !ok || !contract.Complete().IsMethodSet() {
		return nil, nil, shapeError(name, "bridge source is not an interface")
	}
	provider, selected, err := context.Names().ProviderInterface(source)
	if err != nil {
		return nil, nil, err
	}
	if !selected {
		return nil, nil, shapeError(name, "bridge source has no provider certificate")
	}
	providerType, err := context.Names().TypeReference(source.Obj())
	if err != nil {
		return nil, nil, err
	}
	canonical, err := context.Names().InterfaceContract(source)
	if err != nil {
		return nil, nil, err
	}
	runtimeBase, err := context.Names().Runtime(
		api.RuntimeProviderInterfaceBridge,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, nil, err
	}
	panicReference, err := context.Names().Runtime(
		api.RuntimePanic,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, nil, err
	}
	members := []tsgo.ClassElement{
		constructor(
			context.Factory(),
			providerType,
			canonical.ContractName(),
		),
		fromMethod(
			context.Factory(),
			name,
			providerType,
			canonical.TypeName(),
		),
		toMethod(
			context.Factory(),
			name,
			providerType,
			canonical.TypeName(),
			panicReference.Name(),
		),
	}
	requests := api.CombineRequests(
		providerType.Requests(),
		canonical.Requests(),
		runtimeBase.Requests(),
		panicReference.Requests(),
	)
	seen := make(map[string]struct{}, contract.NumMethods())
	for index := range contract.NumMethods() {
		method := contract.Method(index)
		descriptor, describeErr := environmentcontract.Describe(method)
		if describeErr != nil {
			return nil, nil, describeErr
		}
		certificate, found := provider.Method(descriptor.Identity())
		if !found {
			return nil, nil, shapeError(
				name,
				"provider certificate omitted "+descriptor.Identity(),
			)
		}
		if certificate.SourceSignature() != descriptor.Signature() {
			return nil, nil, shapeError(
				name,
				"provider method signature certificate drifted",
			)
		}
		seen[certificate.SourceIdentity()] = struct{}{}
		member, methodRequests, methodErr := emitMethod(
			context,
			children,
			name,
			source,
			method,
			certificate,
		)
		if methodErr != nil {
			return nil, nil, methodErr
		}
		members = append(members, member)
		requests = append(requests, methodRequests...)
	}
	if len(seen) != len(provider.Methods()) {
		return nil, nil, shapeError(name, "provider certificate has foreign methods")
	}
	declaration := context.Factory().ClassDeclaration(
		modifiers,
		context.Factory().Identifier(name),
		nil,
		[]tsgo.HeritageClause{
			context.Factory().HeritageClause(
				tsgo.HeritageClauseTokenKindExtendsKeyword,
				[]tsgo.ExpressionWithTypeArguments{
					context.Factory().ExpressionWithTypeArguments(
						context.Factory().Identifier(runtimeBase.Name()),
						[]tsgo.TypeNode{
							context.Factory().TypeReferenceNode(
								providerType.EntityName(context.Factory()),
								nil,
							),
						},
					),
				},
			),
			context.Factory().HeritageClause(
				tsgo.HeritageClauseTokenKindImplementsKeyword,
				[]tsgo.ExpressionWithTypeArguments{
					context.Factory().ExpressionWithTypeArguments(
						context.Factory().Identifier(canonical.TypeName()),
						nil,
					),
				},
			),
		},
		members,
	)
	return []tsgo.Statement{declaration}, api.CombineRequests(requests), nil
}

func shapeError(artifact string, reason string) error {
	return &api.GeneratedArtifactShapeError{
		Artifact: artifact,
		Reason:   reason,
	}
}

func providerCooperative(method gostdlib.ProviderInterfaceMethod) bool {
	return method.Effect() == gostdlib.EffectAsynchronous
}
