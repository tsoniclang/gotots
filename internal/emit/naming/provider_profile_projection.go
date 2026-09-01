package naming

import (
	"go/types"
	"slices"
	"strconv"
	"strings"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	gostdlibsource "github.com/tsoniclang/gotots/internal/contracts/gostdlib/sourcecontract"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/generic/semanticname"
)

type providerProfileProjection struct {
	certificates map[string]gostdlib.ProviderCallableProfileInterface
	selected     map[string]gostdlib.ProviderCallableProfileInterface
	visited      map[types.Type]struct{}
}

func providerProfileBridgeClosure(
	source *types.Named,
	profile []gostdlib.ProviderCallableProfileInterface,
) ([]gostdlib.ProviderCallableProfileInterface, error) {
	if source == nil || source.Obj() == nil || len(profile) == 0 {
		return nil, &api.NameError{
			Reason: "provider-profile bridge projection input is invalid",
		}
	}
	projection := providerProfileProjection{
		certificates: make(
			map[string]gostdlib.ProviderCallableProfileInterface,
			len(profile),
		),
		selected: make(map[string]gostdlib.ProviderCallableProfileInterface),
		visited:  make(map[types.Type]struct{}),
	}
	for _, certificate := range profile {
		identity := certificate.SourceIdentity()
		if !certificate.Valid() ||
			certificate.ProviderInterface().Mode() !=
				gostdlib.ProviderInterfaceModeBridge {
			return nil, &api.NameError{
				Name:   identity,
				Reason: "provider-profile bridge certificate is invalid",
			}
		}
		if _, duplicate := projection.certificates[identity]; duplicate {
			return nil, &api.NameError{
				Name:   identity,
				Reason: "provider-profile bridge certificate is duplicated",
			}
		}
		projection.certificates[identity] = certificate
	}
	sourceIdentity, err := gostdlibsource.ObjectIdentity(source.Origin().Obj())
	if err != nil {
		return nil, err
	}
	if _, certified := projection.certificates[sourceIdentity]; !certified {
		return nil, &api.NameError{
			Name:   sourceIdentity,
			Reason: "provider-profile bridge source certificate is absent",
		}
	}
	if err := projection.collect(source); err != nil {
		return nil, err
	}
	selected := make(
		[]gostdlib.ProviderCallableProfileInterface,
		0,
		len(projection.selected),
	)
	for _, certificate := range projection.selected {
		selected = append(selected, certificate)
	}
	slices.SortFunc(selected, func(
		left gostdlib.ProviderCallableProfileInterface,
		right gostdlib.ProviderCallableProfileInterface,
	) int {
		return strings.Compare(left.SourceIdentity(), right.SourceIdentity())
	})
	return selected, nil
}

func providerProfileBridgeSemanticParts(
	selected []gostdlib.ProviderCallableProfileInterface,
) ([]string, error) {
	if len(selected) == 0 {
		return nil, &api.NameError{
			Reason: "provider-profile bridge semantic contract is empty",
		}
	}
	parts := make([]string, 0, len(selected))
	for _, current := range selected {
		part, err := providerProfileInterfaceSemanticPart(current)
		if err != nil {
			return nil, err
		}
		parts = append(parts, part)
	}
	return parts, nil
}

func providerProfileBridgeNameParts(
	selected []gostdlib.ProviderCallableProfileInterface,
) ([]string, error) {
	if len(selected) == 0 {
		return nil, &api.NameError{
			Reason: "provider-profile bridge name contract is empty",
		}
	}
	parts := make([]string, 0, len(selected))
	for _, current := range selected {
		part, err := providerProfileInterfaceNamePart(current)
		if err != nil {
			return nil, err
		}
		parts = append(parts, part)
	}
	return parts, nil
}

func providerProfileInterfaceNamePart(
	selected gostdlib.ProviderCallableProfileInterface,
) (string, error) {
	provider := selected.ProviderInterface()
	methods := provider.Methods()
	if !selected.Valid() ||
		provider.Mode() != gostdlib.ProviderInterfaceModeBridge ||
		len(methods) == 0 {
		return "", &api.NameError{
			Name:   selected.SourceIdentity(),
			Reason: "provider-profile bridge name contract is invalid",
		}
	}
	label, err := providerProfileInterfaceName(selected)
	if err != nil {
		return "", err
	}
	uniformEffect := methods[0].Effect()
	uniform := uniformEffect.Valid()
	for _, method := range methods {
		methodName, nameErr := providerProfileMethodName(method.SourceIdentity())
		if nameErr != nil ||
			method.Kind() != gostdlib.ProviderInterfaceMethodCallable ||
			!method.Effect().Valid() {
			return "", &api.NameError{
				Name:   selected.SourceIdentity(),
				Reason: "provider-profile bridge method name contract is invalid",
			}
		}
		if method.Effect() != uniformEffect || method.Member() != methodName {
			uniform = false
		}
	}
	if uniform {
		return label + "$" + providerProfileEffectName(uniformEffect), nil
	}
	parts := []string{label}
	for _, method := range methods {
		methodName, _ := providerProfileMethodName(method.SourceIdentity())
		part := semanticname.Identifier(methodName)
		if method.Member() != methodName {
			part += "$As$" + semanticname.Identifier(method.Member())
		}
		parts = append(
			parts,
			part+"$"+providerProfileEffectName(method.Effect()),
		)
	}
	return strings.Join(parts, "$"), nil
}

func providerProfileInterfaceName(
	selected gostdlib.ProviderCallableProfileInterface,
) (string, error) {
	if selected.SourceIdentity() == gostdlib.LanguageErrorInterfaceIdentity {
		return "Error", nil
	}
	protocol, protocolDefined := selected.Protocol()
	if protocolDefined {
		identity, err := gostdlib.BuildProviderProtocolInterfaceIdentity(protocol)
		parameter, parameterDefined := selected.ProtocolValueParameter()
		if err != nil || identity != selected.SourceIdentity() ||
			!parameterDefined || parameter < 0 {
			return "", &api.NameError{
				Name:   selected.SourceIdentity(),
				Reason: "provider-profile bridge protocol name is invalid",
			}
		}
		name, err := providerProtocolName(protocol)
		return name + "$At" + strconv.Itoa(parameter), err
	}
	const separator = "|kind=2|receiver=|name="
	owner, name, found := strings.Cut(selected.SourceIdentity(), separator)
	if !found || owner == "" || name == "" || strings.Contains(name, "|") {
		return "", &api.NameError{
			Name:   selected.SourceIdentity(),
			Reason: "provider-profile bridge named-interface identity is invalid",
		}
	}
	return semanticname.Identifier(owner) + "_" + semanticname.Identifier(name), nil
}

func providerProtocolName(
	protocol gostdlib.ProviderProtocolInterfaceDocument,
) (string, error) {
	canonical, err := gostdlib.CanonicalProviderProtocolInterface(protocol)
	if err != nil {
		return "", err
	}
	parts := []string{"Protocol"}
	for _, method := range canonical.Methods {
		part := semanticname.Identifier(method.Name) + "$From"
		if len(method.Parameters) == 0 {
			part += "$Void"
		}
		for _, parameter := range method.Parameters {
			name, nameErr := providerContractTypeName(parameter)
			if nameErr != nil {
				return "", nameErr
			}
			part += "$" + name
		}
		part += "$To"
		if len(method.Results) == 0 {
			part += "$Void"
		}
		for _, result := range method.Results {
			name, nameErr := providerContractTypeName(result)
			if nameErr != nil {
				return "", nameErr
			}
			part += "$" + name
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, "$"), nil
}

func providerContractTypeName(source gostdlib.ContractTypeDocument) (string, error) {
	switch source.Kind {
	case gostdlib.ContractTypeParameter:
		if source.TypeParameter != nil {
			return "Type" + strconv.Itoa(*source.TypeParameter), nil
		}
	case gostdlib.ContractTypeCallableParameter:
		if source.CallableParameter != nil {
			return "Callable" + strconv.Itoa(*source.CallableParameter), nil
		}
	case gostdlib.ContractTypeBool:
		return "Bool", nil
	case gostdlib.ContractTypeInt:
		return "Int", nil
	case gostdlib.ContractTypeSlice:
		if source.Element != nil {
			element, err := providerContractTypeName(*source.Element)
			return "SliceOf" + element, err
		}
	case gostdlib.ContractTypeMap:
		if source.Key != nil && source.Element != nil {
			key, keyErr := providerContractTypeName(*source.Key)
			element, elementErr := providerContractTypeName(*source.Element)
			if keyErr != nil {
				return "", keyErr
			}
			return "MapOf" + key + "To" + element, elementErr
		}
	}
	return "", &api.NameError{
		Reason: "provider-profile protocol contract type name is invalid",
	}
}

func providerProfileMethodName(identity string) (string, error) {
	index := strings.LastIndex(identity, "|name=")
	separatorLength := len("|name=")
	if index < 0 {
		index = strings.LastIndex(identity, "|method=")
		separatorLength = len("|method=")
	}
	if index < 0 {
		return "", &api.NameError{
			Name:   identity,
			Reason: "provider-profile bridge method identity is invalid",
		}
	}
	name := identity[index+separatorLength:]
	if name == "" || strings.Contains(name, "|") {
		return "", &api.NameError{
			Name:   identity,
			Reason: "provider-profile bridge method identity is invalid",
		}
	}
	return name, nil
}

func providerProfileEffectName(effect gostdlib.EffectKind) string {
	return "Direct"
}

func providerProfileInterfaceSemanticPart(
	selected gostdlib.ProviderCallableProfileInterface,
) (string, error) {
	provider := selected.ProviderInterface()
	methods := provider.Methods()
	if !selected.Valid() ||
		provider.Mode() != gostdlib.ProviderInterfaceModeBridge ||
		len(methods) == 0 {
		return "", &api.NameError{
			Name:   selected.SourceIdentity(),
			Reason: "provider-profile bridge semantic contract is invalid",
		}
	}
	parts := []string{
		"Interface",
		semanticname.Identifier(selected.SourceIdentity()),
		"Mode",
		semanticname.Identifier(string(provider.Mode())),
	}
	protocol, protocolDefined := selected.Protocol()
	protocolParameter, parameterDefined := selected.ProtocolValueParameter()
	if protocolDefined {
		identity, err := gostdlib.BuildProviderProtocolInterfaceIdentity(protocol)
		if err != nil || !parameterDefined || protocolParameter < 0 ||
			identity != selected.SourceIdentity() {
			return "", &api.NameError{
				Name:   selected.SourceIdentity(),
				Reason: "provider-profile bridge protocol contract is invalid",
			}
		}
		parts = append(
			parts,
			"Protocol",
			semanticname.Identifier(identity),
			"ValueParameter",
			strconv.Itoa(protocolParameter),
		)
	} else if parameterDefined {
		return "", &api.NameError{
			Name:   selected.SourceIdentity(),
			Reason: "provider-profile bridge named contract has a protocol parameter",
		}
	} else {
		parts = append(parts, "Named")
	}
	previousMethod := ""
	for _, method := range methods {
		if method.SourceIdentity() == "" ||
			method.SourceIdentity() <= previousMethod ||
			method.Kind() != gostdlib.ProviderInterfaceMethodCallable ||
			method.Member() == "" ||
			!method.Effect().Valid() {
			return "", &api.NameError{
				Name:   selected.SourceIdentity(),
				Reason: "provider-profile bridge method contract is invalid",
			}
		}
		previousMethod = method.SourceIdentity()
		parts = append(
			parts,
			"Method",
			semanticname.Identifier(method.SourceIdentity()),
			"Kind",
			semanticname.Identifier(string(method.Kind())),
			"Member",
			semanticname.Identifier(method.Member()),
			"Effect",
			semanticname.Identifier(string(method.Effect())),
		)
	}
	return strings.Join(parts, "$"), nil
}

func (p *providerProfileProjection) collect(source types.Type) error {
	if source == nil {
		return nil
	}
	source = types.Unalias(source)
	if _, seen := p.visited[source]; seen {
		return nil
	}
	p.visited[source] = struct{}{}
	switch selected := source.(type) {
	case *types.Named:
		if contract, ok := selected.Underlying().(*types.Interface); ok {
			identity, err := gostdlibsource.ObjectIdentity(
				selected.Origin().Obj(),
			)
			if err != nil {
				return err
			}
			certificate, profiled := p.certificates[identity]
			if !profiled {
				return nil
			}
			p.selected[identity] = certificate
			return p.collectInterface(contract.Complete())
		}
		return p.collect(selected.Underlying())
	case *types.Interface:
		return p.collectInterface(selected.Complete())
	case *types.Struct:
		for index := range selected.NumFields() {
			if err := p.collect(selected.Field(index).Type()); err != nil {
				return err
			}
		}
	case *types.Pointer:
		return p.collect(selected.Elem())
	case *types.Slice:
		return p.collect(selected.Elem())
	case *types.Array:
		return p.collect(selected.Elem())
	case *types.Map:
		if err := p.collect(selected.Key()); err != nil {
			return err
		}
		return p.collect(selected.Elem())
	case *types.Chan:
		return p.collect(selected.Elem())
	case *types.Signature:
		return p.collectSignature(selected)
	case *types.Tuple:
		for index := range selected.Len() {
			if err := p.collect(selected.At(index).Type()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *providerProfileProjection) collectInterface(
	contract *types.Interface,
) error {
	if contract == nil {
		return nil
	}
	for index := range contract.NumMethods() {
		signature, ok := contract.Method(index).Type().(*types.Signature)
		if !ok {
			return &api.NameError{
				Name:   contract.Method(index).Name(),
				Reason: "provider-profile interface method signature is invalid",
			}
		}
		if err := p.collectSignature(signature); err != nil {
			return err
		}
	}
	return nil
}

func (p *providerProfileProjection) collectSignature(
	signature *types.Signature,
) error {
	if signature == nil {
		return nil
	}
	if err := p.collect(signature.Params()); err != nil {
		return err
	}
	return p.collect(signature.Results())
}

type ProviderTarget struct {
	module             string
	export             string
	member             string
	access             gostdlib.AccessKind
	representation     bool
	typeRepresentation gostdlib.RepresentationKind
	definedValue       gostdlib.DefinedValueRepresentationKind
	effect             gostdlib.EffectKind
}

func (r *Registry) ProviderTarget(object types.Object) (ProviderTarget, bool, error) {
	if r == nil || object == nil {
		return ProviderTarget{}, false, &api.NameError{
			Reason: "provider fact target identity is invalid",
		}
	}
	if function, ok := object.(*types.Func); ok {
		object = function.Origin()
	}
	binding, ok := r.byObject[object]
	if !ok || binding.kind == targetBindingEnvironment ||
		binding.kind == targetBindingMissingProvider {
		return ProviderTarget{}, false, nil
	}
	if binding.kind != targetBindingProvider ||
		binding.providerModule == "" || binding.providerExport == "" ||
		!binding.providerAccess.Valid() {
		return ProviderTarget{}, true, &api.NameError{
			Name: object.Name(), Reason: "provider fact target is incomplete",
		}
	}
	if binding.providerAccess == gostdlib.AccessExport &&
		binding.providerMember != "" {
		return ProviderTarget{}, true, &api.NameError{
			Name: object.Name(), Reason: "provider export unexpectedly names a member",
		}
	}
	if binding.providerAccess != gostdlib.AccessExport &&
		binding.providerMember == "" {
		return ProviderTarget{}, true, &api.NameError{
			Name: object.Name(), Reason: "provider member target has no member identity",
		}
	}
	return ProviderTarget{
		module:             binding.providerModule,
		export:             binding.providerExport,
		member:             binding.providerMember,
		access:             binding.providerAccess,
		representation:     binding.providerRepresentation,
		typeRepresentation: binding.providerTypeRepresentation,
		definedValue:       binding.providerDefinedValue,
		effect:             binding.providerEffect,
	}, true, nil
}

func (t ProviderTarget) Module() string              { return t.module }
func (t ProviderTarget) Export() string              { return t.export }
func (t ProviderTarget) Member() string              { return t.member }
func (t ProviderTarget) Access() gostdlib.AccessKind { return t.access }
func (t ProviderTarget) Representation() bool        { return t.representation }
func (t ProviderTarget) TypeRepresentation() gostdlib.RepresentationKind {
	return t.typeRepresentation
}
func (t ProviderTarget) DefinedValueRepresentation() gostdlib.DefinedValueRepresentationKind {
	return t.definedValue
}
func (t ProviderTarget) Effect() gostdlib.EffectKind { return t.effect }

func (n *File) ExternalProviderFunction(
	module string,
	export string,
) (api.NameReference, error) {
	return n.ProviderSymbol(module, export, api.ImportPhaseValue)
}

func (n *File) ProviderSymbol(
	module string,
	export string,
	phase api.ImportPhase,
) (api.NameReference, error) {
	if export == "" {
		return api.NameReference{}, &api.NameError{
			Reason: "external provider export is empty",
		}
	}
	qualifier, request, err := n.providerImport(module, phase)
	if err != nil {
		return api.NameReference{}, err
	}
	return api.NewQualifiedNameReference(qualifier, export, request)
}
