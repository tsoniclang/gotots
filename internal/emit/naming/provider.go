package naming

import (
	"fmt"
	"go/types"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
)

type standardLibraryProvider interface {
	Valid() bool
	ToolchainKey() string
	Binding(string) (gostdlib.Binding, bool)
	Facet(
		string,
		gostdlib.FacetKind,
		gostdlib.FacetCapability,
	) (gostdlib.Facet, bool)
	GenericCallableFacet(string, string) (gostdlib.Facet, bool)
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
		base.kind = targetBindingMissingProvider
		return base, nil
	}
	if err := validateProviderBinding(contract, object, sourcePackage, selected); err != nil {
		return targetBinding{}, err
	}
	base.kind = targetBindingProvider
	base.providerModule = selected.ModuleSpecifier()
	base.providerExport = selected.Export()
	base.providerMember = selected.Member()
	base.providerAccess = selected.Access()
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
