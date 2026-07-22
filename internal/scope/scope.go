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
	// completed gostdlib workspace. Owns standard-library units only.
	ProviderGostdlib
	// ProviderToolchainSource: selected-toolchain command source retained at
	// declaration contract depth — a distinct owner; gostdlib never silently
	// classifies toolchain packages.
	ProviderToolchainSource
	// ProviderExternalObligation: behavior is an exact external contract.
	ProviderExternalObligation
	// ProviderLanguageIntrinsic: behavior is the language/toolchain contract.
	ProviderLanguageIntrinsic

	numProviders
)

var providerNames = [numProviders]string{
	ProviderAutomaticTranslation: "automatic-translation", ProviderGostdlib: "gostdlib",
	ProviderToolchainSource:    "toolchain-source",
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
	case ProviderGostdlib, ProviderToolchainSource:
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
	id      string
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

// DefaultContractID names the initial explicit contract artifact. A request
// selects a contract by identity; the compiler never assumes one.
const DefaultContractID = "default@v1"

// defaultContractV1 is the registry's default@v1 artifact: source-available
// module units translate automatically; standard-library units are
// gostdlib-owned; toolchain command source is the distinct toolchain-source
// owner; intrinsic packages are language-owned; C-dependent units and
// bodyless automatic units are external obligations.
func defaultContractV1() ProviderContract {
	return ProviderContract{
		id:                 DefaultContractID,
		version:            ContractVersion,
		moduleProvider:     ProviderAutomaticTranslation,
		stdProvider:        ProviderGostdlib,
		toolchainProvider:  ProviderToolchainSource,
		intrinsicProvider:  ProviderLanguageIntrinsic,
		cDependentProvider: ProviderExternalObligation,
		bodylessAutomatic:  ProviderExternalObligation,
	}
}

// contractRegistry is the closed versioned contract-artifact registry.
var contractRegistry = map[string]func() ProviderContract{
	DefaultContractID: defaultContractV1,
}

// ResolveContract resolves the request-selected contract artifact by identity
// and, when a digest is supplied, verifies it. An empty identity is a typed
// error — there is no silent default.
func ResolveContract(id, digest string) (ProviderContract, error) {
	if id == "" {
		return ProviderContract{}, &SelectionError{Reason: "the compilation request selects no provider contract"}
	}
	build, known := contractRegistry[id]
	if !known {
		return ProviderContract{}, &SelectionError{Reason: "unknown provider contract " + id}
	}
	contract := build()
	if digest != "" && digest != contract.Fingerprint() {
		return ProviderContract{}, &SelectionError{Reason: "provider contract " + id +
			" digest mismatch: request " + digest + " vs artifact " + contract.Fingerprint()}
	}
	return contract, nil
}

// ID is the contract artifact identity.
func (c ProviderContract) ID() string { return c.id }

// Fingerprint is the contract's canonical fingerprint.
func (c ProviderContract) Fingerprint() string {
	canonical := fmt.Sprintf("%s|v%d|module=%s|std=%s|toolchain=%s|intrinsic=%s|cdep=%s|bodyless=%s",
		c.id, c.version, c.moduleProvider, c.stdProvider, c.toolchainProvider,
		c.intrinsicProvider, c.cDependentProvider, c.bodylessAutomatic)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(canonical)))
}

// BindingRule is the closed vocabulary of how one unit was bound.
type BindingRule uint8

const (
	RuleInvalid BindingRule = iota
	// RuleOwnerNamespace: the owner-class namespace binding applied.
	RuleOwnerNamespace
	// RuleUnitEvidence: per-unit evidence (C-dependence, bodylessness)
	// overrode the namespace binding.
	RuleUnitEvidence

	numBindingRules
)

var bindingRuleNames = [numBindingRules]string{
	RuleOwnerNamespace: "owner-namespace", RuleUnitEvidence: "unit-evidence",
}

// Valid reports whether r names a binding rule.
func (r BindingRule) Valid() bool { return r > RuleInvalid && r < numBindingRules }

// String renders r for reports.
func (r BindingRule) String() string {
	if r.Valid() {
		return bindingRuleNames[r]
	}
	return fmt.Sprintf("scope.BindingRule(%d)", uint8(r))
}

// BindingWitness records exactly which rule bound one unit.
type BindingWitness struct {
	Rule   BindingRule
	Detail string // e.g. "owner-class:standard-library", "c-dependent"
}

// SelectionError is the typed failure of scope selection.
type SelectionError struct{ Reason string }

func (e *SelectionError) Error() string { return "GOTOTS_SCOPE_SELECTION: " + e.Reason }

// UnitSelection is one unit's immutable selection record.
type UnitSelection struct {
	Unit     identity.SourceUnitID
	Provider Provider
	Depth    source.EvidenceDepth
	Witness  BindingWitness
}

// Selection is the immutable, total per-unit evidence-depth selection.
type Selection struct {
	contractID          string
	contractFingerprint string
	units               []UnitSelection
	depths              map[identity.SourceUnitID]source.EvidenceDepth
}

// ContractID is the identity of the request-selected contract artifact.
func (s *Selection) ContractID() string { return s.contractID }

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

// UnitProvider is the contract's own per-unit policy evaluation: it consumes
// the owner class, package disposition, unit kind, and per-unit cgo evidence
// and answers the binding. Producer and verifier both ask the contract — the
// data is the single policy owner; their independence lies in deriving the
// inputs separately.
func (c ProviderContract) UnitProvider(ownerClass identity.OwnerClass, disposition source.LanguageDisposition, kind identity.UnitKind, cDependent bool) (Provider, BindingWitness, error) {
	var binding Provider
	witness := BindingWitness{Rule: RuleOwnerNamespace}
	switch disposition {
	case source.DispositionBuiltinUniverse, source.DispositionUnsafeIntrinsic:
		binding = c.intrinsicProvider
		witness.Detail = "disposition:" + disposition.String()
	default:
		switch ownerClass {
		case identity.OwnerModule:
			binding = c.moduleProvider
		case identity.OwnerStandardLibrary:
			binding = c.stdProvider
		case identity.OwnerToolchain:
			binding = c.toolchainProvider
		case identity.OwnerLanguagePseudo:
			binding = c.intrinsicProvider
		default:
			return ProviderInvalid, BindingWitness{}, &SelectionError{Reason: "no contract binding for owner class " + ownerClass.String()}
		}
		witness.Detail = "owner-class:" + ownerClass.String()
	}
	if cDependent {
		binding = c.cDependentProvider
		witness = BindingWitness{Rule: RuleUnitEvidence, Detail: "c-dependent"}
	}
	if kind == identity.UnitBodylessDecl && binding == ProviderAutomaticTranslation {
		binding = c.bodylessAutomatic
		witness = BindingWitness{Rule: RuleUnitEvidence, Detail: "bodyless-automatic"}
	}
	return binding, witness, nil
}

// UnitDepth answers the evidence depth the contract implies for one unit.
func (c ProviderContract) UnitDepth(ownerClass identity.OwnerClass, disposition source.LanguageDisposition, kind identity.UnitKind, cDependent bool) (source.EvidenceDepth, error) {
	provider, _, err := c.UnitProvider(ownerClass, disposition, kind, cDependent)
	if err != nil {
		return source.DepthInvalid, err
	}
	depth := depthOf(provider)
	if !depth.Valid() {
		return source.DepthInvalid, &SelectionError{Reason: "no valid depth for provider " + provider.String()}
	}
	return depth, nil
}

// Select produces the immutable evidence-depth selection for every censused
// unit of the universe under the given contract. The selection is total and
// disjoint by construction: exactly one record per unit.
func Select(u *source.Universe, contract ProviderContract) (*Selection, error) {
	if contract.version != ContractVersion {
		return nil, &SelectionError{Reason: fmt.Sprintf("contract version %d unsupported", contract.version)}
	}
	out := &Selection{
		contractID:          contract.id,
		contractFingerprint: contract.Fingerprint(),
		depths:              map[identity.SourceUnitID]source.EvidenceDepth{},
	}
	for _, pkg := range u.Packages() {
		for _, file := range pkg.Files() {
			for _, unit := range file.Units() {
				provider, witness, err := contract.UnitProvider(pkg.ID().Owner().Class(), pkg.Disposition(), unit.Kind(), unit.CDependent())
				if err != nil {
					return nil, err
				}
				depth := depthOf(provider)
				if !depth.Valid() {
					return nil, &SelectionError{Reason: "no valid depth for unit " + unit.ID().String()}
				}
				if _, dup := out.depths[unit.ID()]; dup {
					return nil, &SelectionError{Reason: "duplicate selection for unit " + unit.ID().String()}
				}
				out.depths[unit.ID()] = depth
				out.units = append(out.units, UnitSelection{Unit: unit.ID(), Provider: provider, Depth: depth, Witness: witness})
			}
		}
	}
	sort.Slice(out.units, func(i, j int) bool { return out.units[i].Unit.String() < out.units[j].Unit.String() })
	return out, nil
}
