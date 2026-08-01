package naming

import (
	"fmt"
	"go/types"
	"slices"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
)

type standardLibraryProvider interface {
	Valid() bool
	ToolchainKey() string
	ProviderModules() []string
	Binding(string) (gostdlib.Binding, bool)
	Facet(
		string,
		gostdlib.FacetKind,
		gostdlib.FacetCapability,
	) (gostdlib.Facet, bool)
	GenericCallableFacet(string, string) (gostdlib.Facet, bool)
	ProviderRepresentation(string, string) (gostdlib.ProviderRepresentation, bool)
	ProviderCallableProfile(string, string) (gostdlib.ProviderCallableProfile, bool)
	ProviderCallableProfiles(string) []gostdlib.ProviderCallableProfile
}

func (r *Registry) ProviderInterface(
	typeName *types.TypeName,
) (gostdlib.ProviderInterface, bool, error) {
	if r == nil || typeName == nil {
		return gostdlib.ProviderInterface{}, false, &api.NameError{
			Reason: "provider-interface identity is invalid",
		}
	}
	binding, ok := r.byObject[typeName]
	if !ok || binding.kind != targetBindingProvider {
		return gostdlib.ProviderInterface{}, false, nil
	}
	if r.provider == nil || !r.provider.Valid() {
		return gostdlib.ProviderInterface{}, true, &api.NameError{
			Name:   typeName.Name(),
			Reason: "standard-library provider certificate is invalid",
		}
	}
	contract, err := environmentcontract.Describe(typeName)
	if err != nil {
		return gostdlib.ProviderInterface{}, true, err
	}
	selected, ok := r.provider.Binding(contract.Identity())
	if !ok {
		return gostdlib.ProviderInterface{}, true, &api.NameError{
			Name:   contract.Identity(),
			Reason: "provider interface binding is absent",
		}
	}
	providerInterface, ok := selected.ProviderInterface()
	if !ok {
		return gostdlib.ProviderInterface{}, true, &api.NameError{
			Name:   contract.Identity(),
			Reason: "provider interface certificate is absent",
		}
	}
	return providerInterface, true, nil
}

func (n *File) ProviderRepresentationOwnsMethod(
	sourceType types.Type,
	method *types.Func,
) (bool, error) {
	if n == nil || n.owner == nil || sourceType == nil || method == nil ||
		method.Origin() == nil {
		return false, &api.NameError{
			Reason: "provider-representation method query is invalid",
		}
	}
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok {
		return false, nil
	}
	typeName := named.Obj()
	binding, ok := n.owner.byObject[typeName]
	if !ok && n.owner.registry != nil {
		binding, ok = n.owner.registry.byObject[typeName]
	}
	if !ok || !binding.providerRepresentation {
		return false, nil
	}
	provider := n.owner.registry.provider
	if provider == nil || !provider.Valid() {
		return false, &api.NameError{
			Name:   typeName.Name(),
			Reason: "provider representation certificate is invalid",
		}
	}
	representation, ok := provider.ProviderRepresentation(
		binding.providerModule,
		binding.providerExport,
	)
	if !ok {
		return false, &api.NameError{
			Name:   typeName.Name(),
			Reason: "provider representation is absent",
		}
	}
	typeContract, err := environmentcontract.Describe(typeName)
	if err != nil {
		return false, err
	}
	if !slices.Contains(
		representation.SourceTypes(),
		typeContract.Identity(),
	) {
		return false, &api.NameError{
			Name:   typeContract.Identity(),
			Reason: "provider representation does not own the source type",
		}
	}
	methodContract, err := environmentcontract.Describe(method.Origin())
	if err != nil {
		return false, err
	}
	selected, owns := representation.Method(methodContract.Identity())
	if !owns {
		return false, nil
	}
	methodBinding, ok := n.owner.byObject[method.Origin()]
	if !ok && n.owner.registry != nil {
		methodBinding, ok = n.owner.registry.byObject[method.Origin()]
	}
	if !ok || methodBinding.kind != targetBindingProvider ||
		!methodBinding.providerRepresentation ||
		methodBinding.providerModule != binding.providerModule ||
		methodBinding.providerExport != binding.providerExport ||
		methodBinding.providerMember != selected.Member() ||
		methodBinding.providerEffect != selected.Effect() ||
		selected.SourceIdentity() != methodContract.Identity() ||
		selected.SourceSignature() != methodContract.Signature() {
		return false, &api.NameError{
			Name:   methodContract.Identity(),
			Reason: "provider representation method certificate diverged from its target",
		}
	}
	return true, nil
}

func selectProviderBinding(
	sourcePackage *load.Package,
	object types.Object,
	base targetBinding,
	provider standardLibraryProvider,
) (targetBinding, error) {
	if provider == nil ||
		sourcePackage.Kind() != load.PackageStandardLibraryContract {
		return base, nil
	}
	if !provider.Valid() ||
		provider.ToolchainKey() != sourcePackage.ToolchainKey() {
		return targetBinding{}, &api.NameError{
			Name:   sourcePackage.Path(),
			Reason: "standard-library provider toolchain identity is invalid",
		}
	}
	contract, err := environmentcontract.Describe(object)
	if err != nil {
		return targetBinding{}, err
	}
	selected, ok := provider.Binding(contract.Identity())
	if !ok {
		return selectProviderRepresentationBinding(
			sourcePackage,
			object,
			contract,
			base,
			provider,
		)
	}
	if err := validateProviderBinding(contract, object, sourcePackage, selected); err != nil {
		return targetBinding{}, err
	}
	base.kind = targetBindingProvider
	base.providerModule = selected.ModuleSpecifier()
	base.providerExport = selected.Export()
	base.providerMember = selected.Member()
	base.providerAccess = selected.Access()
	base.providerTypeRepresentation = selected.Representation()
	base.providerDefinedValue = selected.DefinedValueRepresentation()
	base.providerEffect = selected.Effect()
	base.providerGenericTypeArguments = selected.GenericTypeArguments()
	base.providerGenericOperations = selected.GenericOperations()
	return base, nil
}

func selectProviderRepresentationBinding(
	sourcePackage *load.Package,
	object types.Object,
	contract environmentcontract.ObjectContract,
	base targetBinding,
	provider standardLibraryProvider,
) (targetBinding, error) {
	base.kind = targetBindingMissingProvider
	typeName, ok := object.(*types.TypeName)
	if !ok || typeName.IsAlias() {
		return base, nil
	}
	facet, ok := provider.Facet(
		contract.Identity(),
		gostdlib.FacetNamedStructOperations,
		gostdlib.FacetCapabilityRepresentation,
	)
	if !ok {
		return base, nil
	}
	if facet.SourceIdentity() != contract.Identity() ||
		facet.Kind() != gostdlib.FacetNamedStructOperations {
		return targetBinding{}, &api.NameError{
			Name:   contract.Identity(),
			Reason: "provider representation facet is invalid",
		}
	}
	representation, ok := facet.Representation()
	if !ok || !slices.Contains(
		representation.SourceTypes(),
		contract.Identity(),
	) {
		return targetBinding{}, &api.NameError{
			Name:   contract.Identity(),
			Reason: "provider representation does not own the source type",
		}
	}
	selectedRepresentation, ok := provider.ProviderRepresentation(
		representation.ModuleSpecifier(),
		representation.Export(),
	)
	if !ok || selectedRepresentation.TargetFingerprint() == "" ||
		selectedRepresentation.ImplementationOwner() == "" {
		return targetBinding{}, &api.NameError{
			Name:   contract.Identity(),
			Reason: "provider representation certificate is invalid",
		}
	}
	base.kind = targetBindingProvider
	base.providerModule = representation.ModuleSpecifier()
	base.providerExport = representation.Export()
	base.providerAccess = gostdlib.AccessExport
	base.providerRepresentation = true
	base.providerTypeRepresentation = gostdlib.RepresentationDirect
	return base, nil
}

func selectProviderRepresentationMethod(
	method *types.Func,
	base targetBinding,
	owner targetBinding,
	provider standardLibraryProvider,
) (targetBinding, error) {
	if provider == nil || !owner.providerRepresentation {
		return targetBinding{}, &api.NameError{
			Name:   method.Name(),
			Reason: "provider representation owner is invalid",
		}
	}
	representation, ok := provider.ProviderRepresentation(
		owner.providerModule,
		owner.providerExport,
	)
	if !ok {
		return targetBinding{}, &api.NameError{
			Name:   method.Name(),
			Reason: "provider representation is absent",
		}
	}
	contract, err := environmentcontract.Describe(method.Origin())
	if err != nil {
		return targetBinding{}, err
	}
	selected, ok := representation.Method(contract.Identity())
	if !ok || selected.SourceIdentity() != contract.Identity() ||
		selected.SourceSignature() != contract.Signature() {
		return targetBinding{}, &api.NameError{
			Name:   method.Name(),
			Reason: "provider representation does not own the concrete method",
		}
	}
	base.kind = targetBindingProvider
	base.providerModule = owner.providerModule
	base.providerExport = owner.providerExport
	base.providerMember = selected.Member()
	base.providerAccess = gostdlib.AccessInstanceMethod
	base.providerEffect = selected.Effect()
	base.providerRepresentation = true
	return base, nil
}

func validateProviderBinding(
	contract environmentcontract.ObjectContract,
	object types.Object,
	sourcePackage *load.Package,
	binding gostdlib.Binding,
) error {
	if binding.Identity() != contract.Identity() ||
		binding.GoImportPath() != sourcePackage.Path() ||
		binding.SourceSignature() != contract.Signature() ||
		binding.SourceValue() != contract.Value() {
		return &api.NameError{
			Name:   contract.Identity(),
			Reason: "provider source contract does not match the selected Go object",
		}
	}
	wantKind, err := providerBindingKind(contract.Kind())
	if err != nil {
		return err
	}
	if binding.Kind() != wantKind {
		return &api.NameError{
			Name:   contract.Identity(),
			Reason: "provider binding kind does not match the selected Go object",
		}
	}
	signature, method := object.Type().(*types.Signature)
	method = method && signature.Recv() != nil
	switch {
	case method:
		_, pointer := signature.Recv().Type().(*types.Pointer)
		want := gostdlib.AccessInstanceMethod
		if pointer {
			want = gostdlib.AccessStaticMethod
		}
		if binding.Access() != want || binding.Member() != object.Name() {
			return &api.NameError{
				Name:   contract.Identity(),
				Reason: "provider method access does not match receiver semantics",
			}
		}
	case binding.Kind() == gostdlib.BindingVariable:
		if binding.Access() != gostdlib.AccessStateMember {
			return &api.NameError{
				Name:   contract.Identity(),
				Reason: "provider variable is not package-state owned",
			}
		}
	default:
		if binding.Access() != gostdlib.AccessExport || binding.Member() != "" {
			return &api.NameError{
				Name:   contract.Identity(),
				Reason: "provider declaration is not a direct export",
			}
		}
	}
	return nil
}

func providerBindingKind(
	kind environmentcontract.ObjectKind,
) (gostdlib.BindingKind, error) {
	switch kind {
	case environmentcontract.ObjectConstant:
		return gostdlib.BindingConstant, nil
	case environmentcontract.ObjectType:
		return gostdlib.BindingType, nil
	case environmentcontract.ObjectVariable:
		return gostdlib.BindingVariable, nil
	case environmentcontract.ObjectFunction:
		return gostdlib.BindingFunction, nil
	case environmentcontract.ObjectBuiltin:
		return gostdlib.BindingBuiltin, nil
	default:
		return gostdlib.BindingInvalid, &api.NameError{
			Name:   fmt.Sprint(kind),
			Reason: "provider object kind is unsupported",
		}
	}
}

func (r *Registry) ProviderGenericCallableEffect(
	owner *types.Func,
	profileKey string,
) (gostdlib.EffectKind, bool, error) {
	if r == nil || owner == nil || profileKey == "" {
		return gostdlib.EffectInvalid, false, &api.NameError{
			Reason: "provider generic-callable identity is invalid",
		}
	}
	owner = owner.Origin()
	binding, ok := r.byObject[owner]
	if !ok {
		return gostdlib.EffectInvalid, false, &api.NameError{
			Name:   owner.Name(),
			Reason: "provider generic-callable owner has no target binding",
		}
	}
	if binding.kind != targetBindingProvider &&
		binding.kind != targetBindingMissingProvider {
		return gostdlib.EffectInvalid, false, nil
	}
	if r.provider == nil || !r.provider.Valid() {
		return gostdlib.EffectInvalid, true, &api.NameError{
			Name:   owner.Name(),
			Reason: "standard-library provider certificate is invalid",
		}
	}
	contract, err := environmentcontract.Describe(owner)
	if err != nil {
		return gostdlib.EffectInvalid, true, err
	}
	selected, ok := r.provider.GenericCallableFacet(
		contract.Identity(),
		profileKey,
	)
	if !ok || selected.SourceIdentity() != contract.Identity() ||
		selected.ProfileKey() != profileKey || !selected.Effect().Valid() {
		return gostdlib.EffectInvalid, true, &api.NameError{
			Name:   contract.Identity(),
			Reason: "certified provider generic-callable effect is absent or inconsistent",
		}
	}
	return selected.Effect(), true, nil
}

func (r *Registry) ProviderCallableEffect(
	owner *types.Func,
) (gostdlib.EffectKind, bool, error) {
	if r == nil || owner == nil {
		return gostdlib.EffectInvalid, false, &api.NameError{
			Reason: "provider callable identity is invalid",
		}
	}
	owner = owner.Origin()
	binding, ok := r.byObject[owner]
	if !ok || binding.kind == targetBindingMissingProvider {
		return gostdlib.EffectInvalid, false, nil
	}
	if binding.kind != targetBindingProvider {
		return gostdlib.EffectInvalid, false, nil
	}
	if !binding.providerEffect.Valid() {
		return gostdlib.EffectInvalid, true, &api.NameError{
			Name:   owner.Name(),
			Reason: "certified provider callable effect is absent",
		}
	}
	return binding.providerEffect, true, nil
}
