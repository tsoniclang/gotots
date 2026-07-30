package naming

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	interfacecontract "github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue/contract"
	"github.com/tsoniclang/gotots/internal/emit/type/methodidentity"
)

func (n *File) InterfaceMethodName(method *types.Func) (string, error) {
	if method == nil {
		return "", &api.NameError{Reason: "interface method is nil"}
	}
	if _, ok := methodidentity.Signature(method); !ok {
		return "", &api.NameError{
			Name:   method.Name(),
			Reason: "interface method signature is invalid",
		}
	}
	if method.Exported() {
		return portableIdentifier(method.Name()), nil
	}
	artifactKey, err := methodidentity.BuildKey(
		method,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return "", err
	}
	return "$go$private_" + artifactKey[:interfaceTargetNameHexLength], nil
}

func (n *File) MethodTarget(
	method *types.Func,
) (api.MethodTarget, error) {
	if method == nil {
		return api.MethodTarget{}, &api.NameError{
			Reason: "method target identity is nil",
		}
	}
	method = method.Origin()
	binding, ok := n.owner.byObject[method]
	if !ok && n.owner.registry != nil {
		binding, ok = n.owner.registry.byObject[method]
	}
	if !ok {
		return api.MethodTarget{}, &api.NameError{
			Name:   method.Name(),
			Reason: "method target has no emitted declaration",
		}
	}
	switch binding.kind {
	case targetBindingSource:
		member, err := n.InterfaceMethodName(method)
		if err != nil {
			return api.MethodTarget{}, err
		}
		dependency, err := api.NewArtifactDependencyRequest(
			method,
			api.ArtifactFacetCallableSignature,
		)
		if err != nil {
			return api.MethodTarget{}, err
		}
		return api.NewMethodTarget(
			api.MethodTargetClassMember,
			member,
			dependency,
		)
	case targetBindingEnvironment:
		reference, err := n.reference(
			method,
			api.ImportPhaseValue,
			api.ArtifactFacetCallableSignature,
		)
		if err != nil {
			return api.MethodTarget{}, err
		}
		return api.NewMethodTarget(
			api.MethodTargetEnvironmentFunction,
			reference.Name(),
			reference.Requests()...,
		)
	default:
		return api.MethodTarget{}, &api.NameError{
			Name:   method.Name(),
			Reason: "method target has no supported ownership",
		}
	}
}

func (n *File) InterfaceMethodToken(
	method *types.Func,
) (api.NameReference, error) {
	if symbol, ok := runtimeInterfaceMethodToken(method); ok {
		return n.Runtime(symbol, api.ImportPhaseValue)
	}
	signature, ok := methodidentity.Signature(method)
	if !ok {
		return api.NameReference{}, &api.NameError{
			Name:   objectName(method),
			Reason: "interface method signature is invalid",
		}
	}
	artifactKey, err := methodidentity.BuildKey(
		method,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	binding, err := n.owner.registry.internInterfaceMethodToken(
		artifactKey,
		method,
		signature,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	requirement, err := api.NewInterfaceMethodTokenRequest(binding.owner)
	if err != nil {
		return api.NameReference{}, err
	}
	return n.generatedValueReference(
		binding.owner,
		binding.name,
		requirement,
		api.ArtifactFacetValueSurface,
	)
}

func runtimeInterfaceMethodToken(
	method *types.Func,
) (api.RuntimeSymbol, bool) {
	switch interfacecontract.Method(method) {
	case interfacecontract.MethodError:
		return api.RuntimeErrorMethodToken, true
	case interfacecontract.MethodRuntimeError:
		return api.RuntimeRuntimeErrorToken, true
	default:
		return api.RuntimeInvalid, false
	}
}
