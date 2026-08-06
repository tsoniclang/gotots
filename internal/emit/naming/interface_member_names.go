package naming

import (
	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	interfacecontract "github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue/contract"
	"github.com/tsoniclang/gotots/internal/emit/type/methodidentity"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
	"github.com/tsoniclang/gotots/internal/output"
	"go/types"
)

func (n *File) derivedSourceReference(
	object types.Object,
	suffix string,
	facet api.ArtifactFacet,
) (api.NameReference, error) {
	if object == nil || suffix == "" {
		return api.NameReference{}, &api.NameError{
			Reason: "derived source reference is invalid",
		}
	}
	binding, ok := n.owner.byObject[object]
	if !ok {
		binding, ok = n.owner.registry.byObject[object]
	}
	if !ok || !binding.scheduled() {
		return api.NameReference{}, &api.NameError{
			Name:   object.Name(),
			Reason: "derived source reference has no declaration",
		}
	}
	if err := n.requireUse(
		object,
		environmentcontract.UseDemandRuntimeFacet,
	); err != nil {
		return api.NameReference{}, err
	}
	exportedName := binding.name + suffix
	var requests []api.RootRequest
	if binding.sourceOwned() && n.artifactOwner.Valid() {
		dependency, err := api.NewArtifactDependencyRequest(object, facet)
		if err != nil {
			return api.NameReference{}, err
		}
		requests = append(requests, dependency)
	}
	if binding.sourcePath == n.targetPath {
		return api.NewNameReference(exportedName, requests...)
	}
	referencePath := binding.sourcePath
	crossPackage := object.Pkg() != nil &&
		(n.packageScope == nil || object.Pkg().Scope() != n.packageScope)
	localName := exportedName
	if crossPackage {
		key := referencePath + "\x00" + exportedName
		localName = n.derivedImports[key]
		if localName == "" {
			localName = exportedName
			if n.owner.registry.sourceBindingNameCollides(binding.name) ||
				n.sourceNameExists(exportedName) ||
				n.hasImportName(exportedName) {
				qualifier, qualifierErr := n.packageImportQualifier(object.Pkg())
				if qualifierErr != nil {
					return api.NameReference{}, qualifierErr
				}
				localName = n.allocateImportName(exportedName, qualifier)
			} else {
				n.importNames[localName] = struct{}{}
			}
			n.derivedImports[key] = localName
		}
	}
	modulePath, err := output.ModuleSpecifier(n.targetPath, referencePath)
	if err != nil {
		return api.NameReference{}, err
	}
	request, err := api.NewImportRequest(
		n.factory,
		api.ImportPhaseValue,
		modulePath,
		exportedName,
		localName,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	requests = append(requests, request)
	return api.NewNameReference(localName, requests...)
}

func (n *File) generatedValueReference(
	artifact *api.GeneratedArtifact,
	name string,
	requirement api.RootRequest,
	facet api.ArtifactFacet,
) (api.NameReference, error) {
	return n.generatedReference(
		artifact,
		name,
		requirement,
		facet,
		api.ImportPhaseValue,
	)
}

func (n *File) generatedReference(
	artifact *api.GeneratedArtifact,
	name string,
	requirement api.RootRequest,
	facet api.ArtifactFacet,
	phase api.ImportPhase,
) (api.NameReference, error) {
	requests := []api.RootRequest{requirement}
	if artifact.Placement() == api.GeneratedArtifactPlacementLexical {
		return api.NewNameReference(name, requests...)
	}
	if n.artifactOwner.Valid() &&
		n.artifactOwner != api.MustGeneratedArtifactOwner(artifact) {
		dependency, err := api.NewGeneratedArtifactDependencyRequest(
			artifact,
			facet,
		)
		if err != nil {
			return api.NameReference{}, err
		}
		requests = append(requests, dependency)
	}
	if artifact.OutputPath() == n.targetPath {
		return api.NewNameReference(name, requests...)
	}
	modulePath, err := output.ModuleSpecifier(
		n.targetPath,
		artifact.OutputPath(),
	)
	if err != nil {
		return api.NameReference{}, err
	}
	request, err := api.NewImportRequest(
		n.factory,
		phase,
		modulePath,
		name,
		name,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	requests = append(requests, request)
	return api.NewNameReference(name, requests...)
}

func (n *File) interfaceContractImports(
	outputPath string,
	baseName string,
	qualifier string,
) ([]api.RootRequest, error) {
	modulePath, err := output.ModuleSpecifier(n.targetPath, outputPath)
	if err != nil {
		return nil, err
	}
	exports := []struct {
		name  string
		phase api.ImportPhase
	}{
		{baseName, api.ImportPhaseType},
		{interfaceContractName(baseName), api.ImportPhaseValue},
		{interfaceGuardName(baseName), api.ImportPhaseValue},
	}
	requests := make([]api.RootRequest, 0, len(exports))
	for _, exported := range exports {
		localName := exported.name
		if qualifier != "" {
			localName = n.allocateImportName(exported.name, qualifier)
		}
		request, requestErr := api.NewImportRequest(
			n.factory,
			exported.phase,
			modulePath,
			exported.name,
			localName,
		)
		if requestErr != nil {
			return nil, requestErr
		}
		requests = append(requests, request)
	}
	return requests, nil
}

func interfaceContractName(base string) string {
	return base + "$contract"
}

func interfaceGuardName(base string) string {
	return base + "$is"
}

func namedInterface(
	sourceType types.Type,
) (*types.TypeName, *types.Interface, bool) {
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok || named.Origin() == nil || named.Origin().Obj() == nil {
		return nil, nil, false
	}
	parameters := named.Origin().TypeParams().Len()
	arguments := named.TypeArgs().Len()
	if parameters != arguments {
		return nil, nil, false
	}
	source, ok := named.Underlying().(*types.Interface)
	if !ok {
		return nil, nil, false
	}
	source = source.Complete()
	return named.Origin().Obj(), source, source.IsMethodSet()
}

func predeclaredError(typeName *types.TypeName) bool {
	return typeName != nil && typeName == types.Universe.Lookup("error")
}

func anonymousInterface(sourceType types.Type) (*types.Interface, bool) {
	source, ok := types.Unalias(sourceType).(*types.Interface)
	if !ok {
		return nil, false
	}
	source = source.Complete()
	return source, source.IsMethodSet()
}

func (n *File) canonicalInterfaceContract(
	sourceType types.Type,
) (*types.Interface, string, error) {
	var source *types.Interface
	if _, selected, ok := namedInterface(sourceType); ok {
		source = selected
	} else {
		var ok bool
		source, ok = anonymousInterface(sourceType)
		if !ok {
			return nil, "", &api.NameError{
				Name:   types.TypeString(sourceType, nil),
				Reason: "interface contract demand type is invalid",
			}
		}
	}
	key, err := typeidentity.BuildKey(
		source,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return nil, "", err
	}
	source, err = n.owner.registry.internInterfaceContract(key, source)
	if err != nil {
		return nil, "", err
	}
	return source, key, nil
}

func interfaceAdapterSource(sourceType types.Type) bool {
	if sourceType == nil {
		return false
	}
	switch types.Unalias(sourceType).Underlying().(type) {
	case *types.Interface, *types.Tuple, *types.TypeParam, *types.Union:
		return false
	default:
		return true
	}
}

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
			api.MethodReceiverABISourceRepresentation,
			false,
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
			api.MethodReceiverABISourceRepresentation,
			false,
			reference.Requests()...,
		)
	case targetBindingProvider:
		if binding.providerMember == "" ||
			(binding.providerAccess != gostdlib.AccessStaticMethod &&
				binding.providerAccess != gostdlib.AccessInstanceMethod) {
			return api.MethodTarget{}, &api.NameError{
				Name:   method.Name(),
				Reason: "provider method target is invalid",
			}
		}
		if err := n.requireUse(
			method,
			environmentcontract.UseDemandCallable,
		); err != nil {
			return api.MethodTarget{}, err
		}
		receiver := api.MethodReceiverTypeName(method)
		receiverBinding, receiverOK := n.owner.byObject[receiver]
		if !receiverOK && n.owner.registry != nil {
			receiverBinding, receiverOK = n.owner.registry.byObject[receiver]
		}
		if !receiverOK ||
			receiverBinding.kind != targetBindingProvider ||
			receiverBinding.providerTypeRepresentation !=
				gostdlib.RepresentationDirect {
			return api.MethodTarget{}, &api.NameError{
				Name:   method.Name(),
				Reason: "provider method receiver representation is not certified direct",
			}
		}
		return api.NewMethodTarget(
			api.MethodTargetClassMember,
			binding.providerMember,
			api.MethodReceiverABIContractDirect,
			true,
		)
	case targetBindingMissingProvider:
		contract, err := environmentcontract.Describe(method)
		if err != nil {
			return api.MethodTarget{}, err
		}
		return api.MethodTarget{}, &api.NameError{
			Name:   contract.Identity(),
			Reason: "selected standard-library method has no provider binding",
		}
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
	var correspondences []api.InterfaceMethodCallableCorrespondence
	originSignature, originOK := methodidentity.Signature(origin)
	if method != origin &&
		originOK &&
		!types.Identical(originSignature, signature) {
		correspondence, correspondenceErr :=
			api.NewInterfaceMethodCallableCorrespondence(
				api.MethodReceiverTypeName(origin),
				originSignature,
				signature,
			)
		if correspondenceErr != nil {
			return api.InterfaceMethodCallableReference{},
				correspondenceErr
		}
		correspondences = append(correspondences, correspondence)
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
		correspondences,
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
