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

func (n *File) InterfaceMethodCallable(
	method *types.Func,
) (api.InterfaceMethodCallableReference, error) {
	if method == nil {
		return api.InterfaceMethodCallableReference{}, &api.NameError{
			Reason: "interface method identity is nil",
		}
	}
	origin := method.Origin()
	originBinding, err := n.interfaceMethodCallableBinding(origin)
	if err != nil {
		return api.InterfaceMethodCallableReference{}, err
	}
	artifacts := []*api.GeneratedArtifact{originBinding.owner}
	signature, ok := methodidentity.Signature(method)
	if !ok {
		return api.InterfaceMethodCallableReference{}, &api.NameError{
			Name:   objectName(method),
			Reason: "interface method signature is invalid",
		}
	}
	if method != origin && !api.ContainsGenericTypeParameter(signature) {
		concrete, concreteErr :=
			n.interfaceMethodCallableBinding(method)
		if concreteErr != nil {
			return api.InterfaceMethodCallableReference{}, concreteErr
		}
		if concrete.owner != originBinding.owner {
			artifacts = append(artifacts, concrete.owner)
		}
	}
	var requests []api.RootRequest
	for _, artifact := range artifacts {
		requirement, requirementErr :=
			api.NewInterfaceMethodCallableRequest(artifact)
		if requirementErr != nil {
			return api.InterfaceMethodCallableReference{}, requirementErr
		}
		requests = append(requests, requirement)
	}
	return api.NewInterfaceMethodCallableReference(
		artifacts,
		requests...,
	)
}

func (n *File) interfaceMethodCallableBinding(
	method *types.Func,
) (interfaceMethodCallableBinding, error) {
	signature, ok := methodidentity.Signature(method)
	if !ok {
		return interfaceMethodCallableBinding{}, &api.NameError{
			Name:   objectName(method),
			Reason: "interface method signature is invalid",
		}
	}
	artifactKey, err := methodidentity.BuildKey(
		method,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return interfaceMethodCallableBinding{}, err
	}
	return n.owner.registry.internInterfaceMethodCallable(
		artifactKey,
		method,
		signature,
	)
}

func (n *File) InterfaceMethodToken(
	method *types.Func,
) (api.NameReference, error) {
	signature, ok := methodidentity.Signature(method)
	if !ok || api.ContainsGenericTypeParameter(signature) {
		return api.NameReference{}, &api.NameError{
			Name:   objectName(method),
			Reason: "runtime interface-method token requires a closed signature",
		}
	}
	runtime, _ := runtimeInterfaceMethodToken(method)
	binding, err := n.interfaceMethodTokenBinding(method, runtime)
	if err != nil {
		return api.NameReference{}, err
	}
	requirement, err := api.NewInterfaceMethodTokenRequest(binding.owner)
	if err != nil {
		return api.NameReference{}, err
	}
	reference, err := n.generatedValueReference(
		binding.owner,
		binding.name,
		requirement,
		api.ArtifactFacetValueSurface,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	return reference, nil
}

func (n *File) interfaceMethodTokenBinding(
	method *types.Func,
	runtime api.RuntimeSymbol,
) (interfaceMethodTokenBinding, error) {
	signature, ok := methodidentity.Signature(method)
	if !ok {
		return interfaceMethodTokenBinding{}, &api.NameError{
			Name:   objectName(method),
			Reason: "interface method signature is invalid",
		}
	}
	artifactKey, err := methodidentity.BuildKey(
		method,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return interfaceMethodTokenBinding{}, err
	}
	binding, err := n.owner.registry.internInterfaceMethodToken(
		artifactKey,
		method,
		signature,
		runtime,
	)
	if err != nil {
		return interfaceMethodTokenBinding{}, err
	}
	return binding, nil
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
