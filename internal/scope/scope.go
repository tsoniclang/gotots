// Package scope is the analysis-scope phase: it consumes the transient typed
// universe plus the request-selected provider-contract artifact and emits the
// immutable per-unit evidence-depth selection. A contract is versioned,
// digest-bound rule DATA — exact-unit bindings, exact-package bindings, and
// explicitly declared owner-namespace rules with closed per-unit evidence
// conditions. Provenance, dispositions, and per-unit cgo evidence are inputs
// to the rules — never the policy itself. Selection is deterministic:
// precedence is by rule specificity, never by array order, and same-precedence
// disagreement fails closed as ambiguity.
package scope

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

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
const ContractVersion = 2

// BindingSelector is the closed selector vocabulary of a contract rule: what
// the rule matches a unit by.
type BindingSelector uint8

const (
	SelectorInvalid BindingSelector = iota
	// SelectorExactUnit matches one exact unit by canonical unit identity
	// (source-spanned or implicit).
	SelectorExactUnit
	// SelectorExactPackage matches every unit of one exact package identity.
	SelectorExactPackage
	// SelectorNamespace matches every unit of an explicitly declared owner
	// namespace (owner class).
	SelectorNamespace

	numSelectors
)

var selectorNames = [numSelectors]string{
	SelectorExactUnit: "unit", SelectorExactPackage: "package", SelectorNamespace: "namespace",
}

// Valid reports whether s names a selector.
func (s BindingSelector) Valid() bool { return s > SelectorInvalid && s < numSelectors }

// String renders s for reports.
func (s BindingSelector) String() string {
	if s.Valid() {
		return selectorNames[s]
	}
	return fmt.Sprintf("scope.BindingSelector(%d)", uint8(s))
}

// EvidenceCondition is the closed per-unit evidence condition a rule may
// require. Conditions consume typed unit/package evidence only — never source
// text, paths, or spellings.
type EvidenceCondition uint8

const (
	ConditionInvalid EvidenceCondition = iota
	// ConditionAlways matches unconditionally.
	ConditionAlways
	// ConditionCDependent matches units whose typed evidence marks them
	// C-dependent.
	ConditionCDependent
	// ConditionBodyless matches bodyless declaration units.
	ConditionBodyless
	// ConditionIntrinsicDisposition matches units of packages whose language
	// disposition is a builtin/unsafe intrinsic contract.
	ConditionIntrinsicDisposition

	numConditions
)

var conditionNames = [numConditions]string{
	ConditionAlways: "always", ConditionCDependent: "c-dependent",
	ConditionBodyless: "bodyless", ConditionIntrinsicDisposition: "intrinsic-disposition",
}

// Valid reports whether c names a condition.
func (c EvidenceCondition) Valid() bool { return c > ConditionInvalid && c < numConditions }

// String renders c for reports.
func (c EvidenceCondition) String() string {
	if c.Valid() {
		return conditionNames[c]
	}
	return fmt.Sprintf("scope.EvidenceCondition(%d)", uint8(c))
}

// ContractRule is one validated binding rule: a selector (exact unit, exact
// package, or declared namespace), an evidence condition, and the provider the
// rule binds. Rules are pure data with a canonical identity.
type ContractRule struct {
	selector  BindingSelector
	unit      string // canonical unit identity (SelectorExactUnit)
	pkg       string // canonical package identity (SelectorExactPackage)
	namespace identity.OwnerClass
	condition EvidenceCondition
	provider  Provider
}

// NewExactUnitRule binds one exact unit identity (source-spanned or implicit).
func NewExactUnitRule(unit string, condition EvidenceCondition, provider Provider) (ContractRule, error) {
	if unit == "" {
		return ContractRule{}, &SelectionError{Reason: "exact-unit rule requires a canonical unit identity"}
	}
	return finishRule(ContractRule{selector: SelectorExactUnit, unit: unit, condition: condition, provider: provider})
}

// NewExactPackageRule binds every unit of one exact package identity.
func NewExactPackageRule(pkg identity.PackageID, condition EvidenceCondition, provider Provider) (ContractRule, error) {
	if pkg.IsZero() {
		return ContractRule{}, &SelectionError{Reason: "exact-package rule requires a package identity"}
	}
	return finishRule(ContractRule{selector: SelectorExactPackage, pkg: pkg.String(), condition: condition, provider: provider})
}

// NewNamespaceRule binds every unit of one explicitly declared owner
// namespace.
func NewNamespaceRule(class identity.OwnerClass, condition EvidenceCondition, provider Provider) (ContractRule, error) {
	if !class.Valid() {
		return ContractRule{}, &SelectionError{Reason: "namespace rule requires a valid owner class"}
	}
	return finishRule(ContractRule{selector: SelectorNamespace, namespace: class, condition: condition, provider: provider})
}

// finishRule validates the shared rule fields.
func finishRule(r ContractRule) (ContractRule, error) {
	if !r.condition.Valid() {
		return ContractRule{}, &SelectionError{Reason: "contract rule requires a valid evidence condition"}
	}
	if !r.provider.Valid() {
		return ContractRule{}, &SelectionError{Reason: "contract rule requires a valid provider"}
	}
	return r, nil
}

// ID is the rule's canonical identity, recorded in every witness.
func (r ContractRule) ID() string {
	target := ""
	switch r.selector {
	case SelectorExactUnit:
		target = "unit:" + r.unit
	case SelectorExactPackage:
		target = "package:" + r.pkg
	case SelectorNamespace:
		target = "namespace:" + r.namespace.String()
	}
	return target + "|" + r.condition.String() + "->" + r.provider.String()
}

// key is the rule's match target (selector+target+condition) without the
// provider; two rules sharing a key are statically ambiguous.
func (r ContractRule) key() string {
	id := r.ID()
	return id[:strings.LastIndex(id, "->")]
}

// Selector is the rule's binding selector.
func (r ContractRule) Selector() BindingSelector { return r.selector }

// Condition is the rule's evidence condition.
func (r ContractRule) Condition() EvidenceCondition { return r.condition }

// Provider is the provider the rule binds.
func (r ContractRule) Provider() Provider { return r.provider }

// tier is the rule's precedence: lower is more specific. Exact unit beats
// exact package beats namespace; within one selector, an evidence-conditioned
// rule beats an unconditional one. Precedence never depends on rule order.
func (r ContractRule) tier() int {
	selectorRank := map[BindingSelector]int{SelectorExactUnit: 0, SelectorExactPackage: 1, SelectorNamespace: 2}[r.selector]
	conditionRank := 0
	if r.condition == ConditionAlways {
		conditionRank = 1
	}
	return selectorRank*2 + conditionRank
}

// UnitQuery is the typed per-unit input to contract evaluation. Exactly one
// of Unit / Implicit is set.
type UnitQuery struct {
	Unit        identity.SourceUnitID
	Implicit    identity.ImplicitUnitID
	Package     identity.PackageID
	OwnerClass  identity.OwnerClass
	Disposition source.LanguageDisposition
	Kind        identity.UnitKind
	CDependent  bool
}

// unitString is the canonical identity of the queried unit.
func (q UnitQuery) unitString() string {
	if !q.Unit.IsZero() {
		return q.Unit.String()
	}
	return q.Implicit.String()
}

// matches reports whether the rule applies to the queried unit.
func (r ContractRule) matches(q UnitQuery) bool {
	switch r.selector {
	case SelectorExactUnit:
		if r.unit != q.unitString() {
			return false
		}
	case SelectorExactPackage:
		if r.pkg != q.Package.String() {
			return false
		}
	case SelectorNamespace:
		if r.namespace != q.OwnerClass {
			return false
		}
	default:
		return false
	}
	switch r.condition {
	case ConditionAlways:
		return true
	case ConditionCDependent:
		return q.CDependent
	case ConditionBodyless:
		return q.Kind == identity.UnitBodylessDecl
	case ConditionIntrinsicDisposition:
		return q.Disposition == source.DispositionBuiltinUniverse || q.Disposition == source.DispositionUnsafeIntrinsic
	}
	return false
}

// ProviderContract is the versioned, digest-bound provider-contract artifact:
// an identity plus validated rule data. Its fingerprint binds every selection
// it produces.
type ProviderContract struct {
	id      string
	version int
	rules   []ContractRule // sorted by canonical rule identity
}

// NewProviderContract validates one contract artifact: a non-empty identity
// and rules with no duplicate match targets (two rules sharing selector,
// target, and condition are statically ambiguous regardless of provider).
func NewProviderContract(id string, rules []ContractRule) (ProviderContract, error) {
	if id == "" || strings.ContainsAny(id, " \t\n|") {
		return ProviderContract{}, &SelectionError{Reason: "contract identity must be a non-empty token"}
	}
	if len(rules) == 0 {
		return ProviderContract{}, &SelectionError{Reason: "contract " + id + " declares no rules"}
	}
	sorted := append([]ContractRule(nil), rules...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID() < sorted[j].ID() })
	seen := map[string]string{}
	for _, rule := range sorted {
		if rule.ID() != finishedRuleID(rule) {
			return ProviderContract{}, &SelectionError{Reason: "contract " + id + " holds an unvalidated rule"}
		}
		if prior, dup := seen[rule.key()]; dup {
			return ProviderContract{}, &SelectionError{Reason: "contract " + id + " rules " + prior +
				" and " + rule.ID() + " share one match target"}
		}
		seen[rule.key()] = rule.ID()
	}
	return ProviderContract{id: id, version: ContractVersion, rules: sorted}, nil
}

// finishedRuleID revalidates a rule's components and returns its identity;
// a rule fabricated outside the constructors yields a mismatch.
func finishedRuleID(r ContractRule) string {
	validated, err := finishRule(r)
	if err != nil || !r.selector.Valid() {
		return ""
	}
	switch r.selector {
	case SelectorExactUnit:
		if r.unit == "" {
			return ""
		}
	case SelectorExactPackage:
		if r.pkg == "" {
			return ""
		}
	case SelectorNamespace:
		if !r.namespace.Valid() {
			return ""
		}
	}
	return validated.ID()
}

// ID is the contract artifact identity.
func (c ProviderContract) ID() string { return c.id }

// Rules is the validated rule data (immutable copy).
func (c ProviderContract) Rules() []ContractRule { return append([]ContractRule(nil), c.rules...) }

// Fingerprint is the contract's canonical fingerprint over identity, schema
// version, and every rule.
func (c ProviderContract) Fingerprint() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s|v%d", c.id, c.version)
	for _, rule := range c.rules {
		b.WriteString("\n")
		b.WriteString(rule.ID())
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(b.String())))
}

// DefaultContractID names the built-in explicit contract artifact. A request
// selects a contract by identity or artifact path; the compiler never assumes
// one.
const DefaultContractID = "default@v1"

// defaultContractV1 is the registry's default@v1 artifact: source-available
// module units translate automatically; standard-library units are
// gostdlib-owned; toolchain command source is the distinct toolchain-source
// owner; language-pseudo and intrinsic-disposition units are language-owned;
// C-dependent units and bodyless module declarations are external obligations.
func defaultContractV1() (ProviderContract, error) {
	spec := []struct {
		class     identity.OwnerClass
		condition EvidenceCondition
		provider  Provider
	}{
		{identity.OwnerModule, ConditionAlways, ProviderAutomaticTranslation},
		{identity.OwnerStandardLibrary, ConditionAlways, ProviderGostdlib},
		{identity.OwnerToolchain, ConditionAlways, ProviderToolchainSource},
		{identity.OwnerLanguagePseudo, ConditionAlways, ProviderLanguageIntrinsic},
		{identity.OwnerModule, ConditionCDependent, ProviderExternalObligation},
		{identity.OwnerStandardLibrary, ConditionCDependent, ProviderExternalObligation},
		{identity.OwnerToolchain, ConditionCDependent, ProviderExternalObligation},
		{identity.OwnerModule, ConditionBodyless, ProviderExternalObligation},
		{identity.OwnerStandardLibrary, ConditionIntrinsicDisposition, ProviderLanguageIntrinsic},
	}
	rules := make([]ContractRule, 0, len(spec))
	for _, s := range spec {
		rule, err := NewNamespaceRule(s.class, s.condition, s.provider)
		if err != nil {
			return ProviderContract{}, err
		}
		rules = append(rules, rule)
	}
	return NewProviderContract(DefaultContractID, rules)
}

// contractRegistry is the closed built-in contract-artifact registry.
var contractRegistry = map[string]func() (ProviderContract, error){
	DefaultContractID: defaultContractV1,
}

// ResolveContract resolves the request-selected contract: a built-in registry
// identity, or the path of a versioned contract-artifact file. When a digest
// is supplied it must match the resolved artifact's fingerprint. An empty
// selection is a typed error — there is no silent default.
func ResolveContract(selection, digest string) (ProviderContract, error) {
	if selection == "" {
		return ProviderContract{}, &SelectionError{Reason: "the compilation request selects no provider contract"}
	}
	var contract ProviderContract
	if build, known := contractRegistry[selection]; known {
		built, err := build()
		if err != nil {
			return ProviderContract{}, err
		}
		contract = built
	} else {
		loaded, err := loadContractArtifact(selection)
		if err != nil {
			return ProviderContract{}, err
		}
		contract = loaded
	}
	if digest != "" && digest != contract.Fingerprint() {
		return ProviderContract{}, &SelectionError{Reason: "provider contract " + selection +
			" digest mismatch: request " + digest + " vs artifact " + contract.Fingerprint()}
	}
	return contract, nil
}

// BindingWitness records exactly which rule bound one unit.
type BindingWitness struct {
	RuleID    string
	Selector  BindingSelector
	Condition EvidenceCondition
}

// SelectionError is the typed failure of scope selection.
type SelectionError struct{ Reason string }

func (e *SelectionError) Error() string { return "GOTOTS_SCOPE_SELECTION: " + e.Reason }

// Bind evaluates the contract's rules for one unit: the most specific
// matching tier wins; same-tier disagreement on the provider is ambiguity and
// fails closed with both rule identities; no matching rule fails closed.
func (c ProviderContract) Bind(q UnitQuery) (Provider, BindingWitness, error) {
	bestTier := int(^uint(0) >> 1)
	var matches []ContractRule
	for _, rule := range c.rules {
		if !rule.matches(q) {
			continue
		}
		tier := rule.tier()
		if tier < bestTier {
			bestTier = tier
			matches = matches[:0]
		}
		if tier == bestTier {
			matches = append(matches, rule)
		}
	}
	if len(matches) == 0 {
		return ProviderInvalid, BindingWitness{}, &SelectionError{Reason: "contract " + c.id +
			" binds no rule to unit " + q.unitString()}
	}
	// matches inherit the contract's canonical rule order, so the witness is
	// deterministic; any same-tier provider disagreement is ambiguity.
	for _, m := range matches[1:] {
		if m.provider != matches[0].provider {
			return ProviderInvalid, BindingWitness{}, &SelectionError{Reason: "ambiguous binding for unit " +
				q.unitString() + ": rules " + matches[0].ID() + " and " + m.ID() + " disagree"}
		}
	}
	winner := matches[0]
	return winner.provider, BindingWitness{RuleID: winner.ID(), Selector: winner.selector, Condition: winner.condition}, nil
}

// UnitDepth answers the evidence depth the contract implies for one unit.
func (c ProviderContract) UnitDepth(q UnitQuery) (source.EvidenceDepth, error) {
	provider, _, err := c.Bind(q)
	if err != nil {
		return source.DepthInvalid, err
	}
	depth := depthOf(provider)
	if !depth.Valid() {
		return source.DepthInvalid, &SelectionError{Reason: "no valid depth for provider " + provider.String()}
	}
	return depth, nil
}

// AcquisitionPolicy derives the census/acquisition policy the contract
// implies for ordinary compilation: namespaces and exact packages bound to
// automatic translation are censused recursively (interiors derive locally);
// provider-owned source consumes the request's verified unit manifest. The
// policy is contract data resolved BEFORE the universe loads.
func (c ProviderContract) AcquisitionPolicy() (source.AcquisitionPolicy, error) {
	return c.acquisitionPolicy(source.CensusManifest)
}

// AuditAcquisitionPolicy derives the manifest-producing gate run's policy:
// every file is censused recursively, so the produced artifact carries the
// complete provider unit manifest.
func (c ProviderContract) AuditAcquisitionPolicy() (source.AcquisitionPolicy, error) {
	return c.acquisitionPolicy(source.CensusRecursive)
}

func (c ProviderContract) acquisitionPolicy(nonAutomatic source.CensusMode) (source.AcquisitionPolicy, error) {
	byClass := map[identity.OwnerClass]source.CensusMode{}
	byPackage := map[string]source.CensusMode{}
	mode := func(p Provider) source.CensusMode {
		if p == ProviderAutomaticTranslation {
			return source.CensusRecursive
		}
		return nonAutomatic
	}
	for _, rule := range c.rules {
		if rule.condition != ConditionAlways {
			continue // evidence conditions are unit-level; acquisition is file-level
		}
		switch rule.selector {
		case SelectorNamespace:
			byClass[rule.namespace] = mode(rule.provider)
		case SelectorExactPackage:
			byPackage[rule.pkg] = mode(rule.provider)
		}
	}
	return source.NewAcquisitionPolicy(byClass, byPackage)
}
