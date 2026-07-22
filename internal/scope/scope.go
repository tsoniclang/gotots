// Package scope is the analysis-scope phase: it consumes the transient typed
// universe plus the explicit environment/provider contract and emits the
// immutable per-unit evidence-depth selection. Provenance, dispositions, and
// per-unit cgo evidence are inputs to the contract's bindings — never the
// policy itself. No function here assigns depth from provenance or file state
// alone.
package scope

import (
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/source"
)

// Provider is the closed implementation-provider vocabulary of the
// environment contract.
type Provider uint8

const (
	ProviderInvalid Provider = iota
	// ProviderAutomaticTranslation: the compiler translates the Go body.
	ProviderAutomaticTranslation
	// ProviderGostdlib: behavior is supplied by the reusable manually
	// completed gostdlib workspace.
	ProviderGostdlib
	// ProviderExternalObligation: behavior is an exact external contract.
	ProviderExternalObligation
	// ProviderLanguageIntrinsic: behavior is the language/toolchain contract.
	ProviderLanguageIntrinsic

	numProviders
)

var providerNames = [numProviders]string{
	ProviderAutomaticTranslation: "automatic-translation", ProviderGostdlib: "gostdlib",
	ProviderExternalObligation: "external-obligation", ProviderLanguageIntrinsic: "language-intrinsic",
}

// Valid reports whether p names a provider.
func (p Provider) Valid() bool { return p > ProviderInvalid && p < numProviders }

// String renders p for reports.
func (p Provider) String() string {
	if p.Valid() {
		return providerNames[p]
	}
	return fmt.Sprintf("scope.Provider(%d)", uint8(p))
}

// depthOf maps one provider binding to the evidence depth it implies.
func depthOf(p Provider) source.EvidenceDepth {
	switch p {
	case ProviderAutomaticTranslation:
		return source.DepthFullSemantic
	case ProviderGostdlib:
		return source.DepthDeclarationContract
	case ProviderExternalObligation:
		return source.DepthExternalBoundary
	case ProviderLanguageIntrinsic:
		return source.DepthIntrinsic
	}
	return source.DepthInvalid
}

// ContractVersion is the provider-contract schema version.
const ContractVersion = 1

// ProviderContract is the versioned, fingerprinted environment/provider
// contract the scope phase consumes. Bindings are per owner class with exact
// per-unit evidence rules; the contract is data, and its fingerprint binds
// the selection it produced.
type ProviderContract struct {
	version int
	// bindings by owner class of the unit's package.
	moduleProvider    Provider
	stdProvider       Provider
	toolchainProvider Provider
	intrinsicProvider Provider
	// cDependentProvider overrides the binding for units whose source
	// references the cgo "C" pseudo-package (per-unit evidence).
	cDependentProvider Provider
	// bodylessAutomatic overrides bodyless declarations under an automatic
	// binding: with no Go body there is nothing to translate automatically.
	bodylessAutomatic Provider
}

// DefaultContract is the initial explicit binding: source-available module
// units translate automatically; standard-library and toolchain units are
// gostdlib-owned; intrinsic packages are language-owned; C-dependent units
// and bodyless automatic units are external obligations.
func DefaultContract() ProviderContract {
	return ProviderContract{
		version:            ContractVersion,
		moduleProvider:     ProviderAutomaticTranslation,
		stdProvider:        ProviderGostdlib,
		toolchainProvider:  ProviderGostdlib,
		intrinsicProvider:  ProviderLanguageIntrinsic,
		cDependentProvider: ProviderExternalObligation,
		bodylessAutomatic:  ProviderExternalObligation,
	}
}

// Fingerprint is the contract's canonical fingerprint.
func (c ProviderContract) Fingerprint() string {
	canonical := fmt.Sprintf("v%d|module=%s|std=%s|toolchain=%s|intrinsic=%s|cdep=%s|bodyless=%s",
		c.version, c.moduleProvider, c.stdProvider, c.toolchainProvider,
		c.intrinsicProvider, c.cDependentProvider, c.bodylessAutomatic)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(canonical)))
}

// SelectionError is the typed failure of scope selection.
type SelectionError struct{ Reason string }

func (e *SelectionError) Error() string { return "GOTOTS_SCOPE_SELECTION: " + e.Reason }

// UnitSelection is one unit's immutable selection record.
type UnitSelection struct {
	Unit     identity.SourceUnitID
	Provider Provider
	Depth    source.EvidenceDepth
}

// Selection is the immutable, total per-unit evidence-depth selection.
type Selection struct {
	contractFingerprint string
	units               []UnitSelection
	depths              map[identity.SourceUnitID]source.EvidenceDepth
}

// ContractFingerprint binds the selection to the contract that produced it.
func (s *Selection) ContractFingerprint() string { return s.contractFingerprint }

// Units is the ordered selection ledger (immutable copy).
func (s *Selection) Units() []UnitSelection { return append([]UnitSelection(nil), s.units...) }

// Depths is the per-unit depth map consumed by source.Finalize (fresh copy).
func (s *Selection) Depths() map[identity.SourceUnitID]source.EvidenceDepth {
	out := make(map[identity.SourceUnitID]source.EvidenceDepth, len(s.depths))
	for id, depth := range s.depths {
		out[id] = depth
	}
	return out
}

// Select produces the immutable evidence-depth selection for every censused
// unit of the universe under the given contract. The selection is total and
// disjoint by construction: exactly one record per unit.
func Select(u *source.Universe, contract ProviderContract) (*Selection, error) {
	if contract.version != ContractVersion {
		return nil, &SelectionError{Reason: fmt.Sprintf("contract version %d unsupported", contract.version)}
	}
	out := &Selection{
		contractFingerprint: contract.Fingerprint(),
		depths:              map[identity.SourceUnitID]source.EvidenceDepth{},
	}
	for _, pkg := range u.Packages() {
		binding, err := bindingFor(contract, pkg)
		if err != nil {
			return nil, err
		}
		for _, file := range pkg.Files() {
			for _, unit := range file.Units() {
				provider := binding
				// Per-unit evidence rules, from the contract:
				if unit.CDependent() {
					provider = contract.cDependentProvider
				}
				if unit.Kind() == identity.UnitBodylessDecl && provider == ProviderAutomaticTranslation {
					provider = contract.bodylessAutomatic
				}
				depth := depthOf(provider)
				if !depth.Valid() {
					return nil, &SelectionError{Reason: "no valid depth for unit " + unit.ID().String()}
				}
				if _, dup := out.depths[unit.ID()]; dup {
					return nil, &SelectionError{Reason: "duplicate selection for unit " + unit.ID().String()}
				}
				out.depths[unit.ID()] = depth
				out.units = append(out.units, UnitSelection{Unit: unit.ID(), Provider: provider, Depth: depth})
			}
		}
	}
	sort.Slice(out.units, func(i, j int) bool { return out.units[i].Unit.String() < out.units[j].Unit.String() })
	return out, nil
}

// bindingFor resolves the contract's owner-class binding for one package.
func bindingFor(contract ProviderContract, pkg *source.LoadedPackage) (Provider, error) {
	switch pkg.Disposition() {
	case source.DispositionBuiltinUniverse, source.DispositionUnsafeIntrinsic:
		return contract.intrinsicProvider, nil
	}
	switch pkg.ID().Owner().Class() {
	case identity.OwnerModule:
		return contract.moduleProvider, nil
	case identity.OwnerStandardLibrary:
		return contract.stdProvider, nil
	case identity.OwnerToolchain:
		return contract.toolchainProvider, nil
	case identity.OwnerLanguagePseudo:
		return contract.intrinsicProvider, nil
	}
	return ProviderInvalid, &SelectionError{Reason: pkg.ID().String() + ": no contract binding for owner class"}
}
