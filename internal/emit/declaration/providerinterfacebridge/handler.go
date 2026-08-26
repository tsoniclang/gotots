package providerinterfacebridge

import (
	"go/types"

	gostdlibsource "github.com/tsoniclang/gotots/internal/contracts/gostdlib/sourcecontract"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/typescriptclass"
	providerboundary "github.com/tsoniclang/gotots/internal/emit/value/providerboundary"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Build(
	context api.Context,
	children api.ChildEmitter,
	name string,
	source *types.Named,
	capabilities []CapabilityContract,
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
	providerType, err := providerTypeReference(context, source)
	if err != nil {
		return nil, nil, err
	}
	canonical, err := context.Names().InterfaceContract(source)
	if err != nil {
		return nil, nil, err
	}
	directProviderUse, compatibilityRequests, err :=
		providerboundary.InterfaceABIExact(context, source)
	if err != nil {
		return nil, nil, err
	}
	selectedCapabilities, capabilityRequests, err := selectCapabilities(
		context,
		name,
		source,
		capabilities,
		provider,
		directProviderUse,
	)
	if err != nil {
		return nil, nil, err
	}
	conflicts, err := capabilityConflicts(selectedCapabilities)
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
	var panicReference api.NameReference
	if !directProviderUse || len(selectedCapabilities) != 0 {
		panicReference, err = context.Names().Runtime(
			api.RuntimePanic,
			api.ImportPhaseValue,
		)
		if err != nil {
			return nil, nil, err
		}
	}
	members := make(
		[]tsgo.ClassElement,
		0,
		len(selectedCapabilities)+contract.NumMethods()+3,
	)
	for _, capability := range selectedCapabilities {
		members = append(
			members,
			capabilityFieldDeclaration(context.Factory(), capability),
		)
	}
	members = append(members,
		constructor(
			context.Factory(),
			context.Factory().TypeReferenceNode(
				providerType.EntityName(context.Factory()),
				nil,
			),
			canonical.ContractName(),
			selectedCapabilities,
			conflicts,
			panicReference.Name(),
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
			directProviderUse,
		),
	)
	requests := api.CombineRequests(
		providerType.Requests(),
		canonical.Requests(),
		runtimeBase.Requests(),
		panicReference.Requests(),
		compatibilityRequests,
		capabilityRequests,
	)
	seen := make(map[string]struct{}, contract.NumMethods())
	for index := range contract.NumMethods() {
		method := contract.Method(index)
		identity, signature, describeErr :=
			gostdlibsource.ProviderInterfaceMethod(method)
		if describeErr != nil {
			return nil, nil, describeErr
		}
		certificate, found := provider.Method(identity)
		if !found {
			return nil, nil, shapeError(
				name,
				"provider certificate omitted "+identity,
			)
		}
		if certificate.SourceSignature() != signature {
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
	capabilityMembers, methodRequests, err := emitCapabilityMethods(
		context,
		children,
		name,
		source,
		selectedCapabilities,
	)
	if err != nil {
		return nil, nil, err
	}
	members = append(members, capabilityMembers...)
	requests = append(requests, methodRequests...)
	declaration := typescriptclass.Declaration(context.Factory(),
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
			implementsHeritage(
				context.Factory(),
				implementedContractNames(
					canonical.TypeName(),
					selectedCapabilities,
				),
			),
		},
		members,
	)
	return []tsgo.Statement{declaration}, api.CombineRequests(requests), nil
}

func providerTypeReference(
	context api.Context,
	source *types.Named,
) (api.NameReference, error) {
	if source != nil && source.Obj() == types.Universe.Lookup("error") {
		return context.Names().Runtime(
			api.RuntimeBuiltinErrorType,
			api.ImportPhaseType,
		)
	}
	return context.Names().TypeReference(source.Obj())
}

func shapeError(artifact string, reason string) error {
	return &api.GeneratedArtifactShapeError{
		Artifact: artifact,
		Reason:   reason,
	}
}
