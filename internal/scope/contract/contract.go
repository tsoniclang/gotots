// Package contract owns the closed, versioned provider-selection contract.
// It contains policy schema and validation only: no source, definition graph,
// checker object, or selected state.
package contract

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/identity"
)

// SchemaVersion is the only admitted provider-contract schema.
const SchemaVersion = 1

// EvidenceDepth is the closed retained-evidence class selected for one
// definition. It is selection state, not source state.
type EvidenceDepth uint8

const (
	DepthInvalid EvidenceDepth = iota
	DepthFullSemantic
	DepthDeclarationContract
	DepthExternalBoundary
	DepthIntrinsic
)

func (d EvidenceDepth) Valid() bool {
	return d >= DepthFullSemantic && d <= DepthIntrinsic
}

func (d EvidenceDepth) String() string {
	switch d {
	case DepthFullSemantic:
		return "full-semantic"
	case DepthDeclarationContract:
		return "declaration-contract"
	case DepthExternalBoundary:
		return "external-boundary"
	case DepthIntrinsic:
		return "intrinsic"
	default:
		return fmt.Sprintf("contract.EvidenceDepth(%d)", uint8(d))
	}
}

// Provider is the closed implementation-owner class.
type Provider uint8

const (
	ProviderInvalid Provider = iota
	ProviderAutomaticTranslation
	ProviderGostdlib
	ProviderToolchainSource
	ProviderExternalObligation
	ProviderLanguageIntrinsic
)

func (p Provider) Valid() bool {
	return p >= ProviderAutomaticTranslation && p <= ProviderLanguageIntrinsic
}

func (p Provider) String() string {
	switch p {
	case ProviderAutomaticTranslation:
		return "automatic-translation"
	case ProviderGostdlib:
		return "gostdlib"
	case ProviderToolchainSource:
		return "toolchain-source"
	case ProviderExternalObligation:
		return "external-obligation"
	case ProviderLanguageIntrinsic:
		return "language-intrinsic"
	default:
		return fmt.Sprintf("contract.Provider(%d)", uint8(p))
	}
}

// Depth is the contract-defined provider-to-evidence-depth relation.
func (p Provider) Depth() EvidenceDepth {
	switch p {
	case ProviderAutomaticTranslation:
		return DepthFullSemantic
	case ProviderGostdlib, ProviderToolchainSource:
		return DepthDeclarationContract
	case ProviderExternalObligation:
		return DepthExternalBoundary
	case ProviderLanguageIntrinsic:
		return DepthIntrinsic
	default:
		return DepthInvalid
	}
}

// SelectionFactKind is the closed semantic fact vocabulary available to
// conditional rules.
type SelectionFactKind uint8

const (
	SelectionFactInvalid SelectionFactKind = iota
	SelectionFactCDependent
)

func (k SelectionFactKind) Valid() bool { return k == SelectionFactCDependent }

func (k SelectionFactKind) String() string {
	if k == SelectionFactCDependent {
		return "c-dependent"
	}
	return fmt.Sprintf("contract.SelectionFactKind(%d)", uint8(k))
}

// Selector is the closed rule target domain.
type Selector uint8

const (
	SelectorInvalid Selector = iota
	SelectorExactDefinition
	SelectorExactPackage
	SelectorNamespace
)

func (s Selector) Valid() bool {
	return s >= SelectorExactDefinition && s <= SelectorNamespace
}

func (s Selector) String() string {
	switch s {
	case SelectorExactDefinition:
		return "definition"
	case SelectorExactPackage:
		return "package"
	case SelectorNamespace:
		return "namespace"
	default:
		return fmt.Sprintf("contract.Selector(%d)", uint8(s))
	}
}

// Condition is the closed rule predicate. FactTrue names one requested
// semantic fact; Bodyless and Intrinsic are structural/package facts already
// present in the rule query and are not recomputed semantic facts.
type Condition uint8

const (
	ConditionInvalid Condition = iota
	ConditionAlways
	ConditionFactTrue
	ConditionBodyless
	ConditionIntrinsic
	ConditionSynthetic
)

func (c Condition) Valid() bool {
	return c >= ConditionAlways && c <= ConditionSynthetic
}

func (c Condition) String() string {
	switch c {
	case ConditionAlways:
		return "always"
	case ConditionFactTrue:
		return "fact-true"
	case ConditionBodyless:
		return "bodyless"
	case ConditionIntrinsic:
		return "intrinsic"
	case ConditionSynthetic:
		return "synthetic"
	default:
		return fmt.Sprintf("contract.Condition(%d)", uint8(c))
	}
}

// Rule is one constructor-validated provider binding.
type Rule struct {
	selector   Selector
	definition identity.DefinitionID
	pkg        identity.PackageID
	namespace  identity.OwnerClass
	condition  Condition
	fact       SelectionFactKind
	provider   Provider
	id         string
}

func NewDefinitionRule(
	definition identity.DefinitionID,
	condition Condition,
	fact SelectionFactKind,
	provider Provider,
) (Rule, error) {
	if definition.IsZero() {
		return Rule{}, &Error{Reason: "exact-definition rule requires a definition identity"}
	}
	return finishRule(Rule{
		selector: SelectorExactDefinition, definition: definition,
		condition: condition, fact: fact, provider: provider,
	})
}

func NewPackageRule(
	pkg identity.PackageID,
	condition Condition,
	fact SelectionFactKind,
	provider Provider,
) (Rule, error) {
	if pkg.IsZero() {
		return Rule{}, &Error{Reason: "exact-package rule requires a package identity"}
	}
	return finishRule(Rule{
		selector: SelectorExactPackage, pkg: pkg,
		condition: condition, fact: fact, provider: provider,
	})
}

func NewNamespaceRule(
	namespace identity.OwnerClass,
	condition Condition,
	fact SelectionFactKind,
	provider Provider,
) (Rule, error) {
	if !namespace.Valid() {
		return Rule{}, &Error{Reason: "namespace rule requires an owner class"}
	}
	return finishRule(Rule{
		selector: SelectorNamespace, namespace: namespace,
		condition: condition, fact: fact, provider: provider,
	})
}

func finishRule(rule Rule) (Rule, error) {
	if !rule.selector.Valid() || !rule.condition.Valid() || !rule.provider.Valid() {
		return Rule{}, &Error{Reason: "rule has an invalid selector, condition, or provider"}
	}
	if rule.condition == ConditionFactTrue {
		if !rule.fact.Valid() {
			return Rule{}, &Error{Reason: "fact-true rule requires a valid fact kind"}
		}
	} else if rule.fact != SelectionFactInvalid {
		return Rule{}, &Error{Reason: "only fact-true rules may name a selection fact"}
	}
	id := ruleID(rule)
	if rule.id != "" && rule.id != id {
		return Rule{}, &Error{Reason: "rule has a non-canonical identity"}
	}
	rule.id = id
	return rule, nil
}

func (r Rule) Selector() Selector                { return r.selector }
func (r Rule) Definition() identity.DefinitionID { return r.definition }
func (r Rule) Package() identity.PackageID       { return r.pkg }
func (r Rule) Namespace() identity.OwnerClass    { return r.namespace }
func (r Rule) Condition() Condition              { return r.condition }
func (r Rule) FactKind() SelectionFactKind       { return r.fact }
func (r Rule) Provider() Provider                { return r.provider }

func (r Rule) ID() string {
	return r.id
}

func ruleID(r Rule) string {
	target := ""
	switch r.selector {
	case SelectorExactDefinition:
		target = "definition:" + r.definition.String()
	case SelectorExactPackage:
		target = "package:" + r.pkg.String()
	case SelectorNamespace:
		target = "namespace:" + r.namespace.String()
	}
	condition := r.condition.String()
	if r.condition == ConditionFactTrue {
		condition += ":" + r.fact.String()
	}
	return target + "|" + condition + "->" + r.provider.String()
}

func (r Rule) key() string {
	id := r.ID()
	return id[:strings.LastIndex(id, "->")]
}

func (r Rule) tier() int {
	selector := map[Selector]int{
		SelectorExactDefinition: 0,
		SelectorExactPackage:    1,
		SelectorNamespace:       2,
	}[r.selector]
	condition := 0
	if r.condition == ConditionAlways {
		condition = 1
	}
	return selector*2 + condition
}

// Contract is one immutable provider-contract artifact.
type Contract struct {
	id          string
	version     int
	rules       []Rule
	fingerprint string
}

func New(id string, rules []Rule) (Contract, error) {
	if id == "" || strings.ContainsAny(id, " \t\n|") {
		return Contract{}, &Error{Reason: "contract identity must be a non-empty token"}
	}
	if len(rules) == 0 {
		return Contract{}, &Error{Reason: "contract declares no rules"}
	}
	canonical := append([]Rule(nil), rules...)
	sort.Slice(canonical, func(i, j int) bool { return canonical[i].ID() < canonical[j].ID() })
	seen := map[string]bool{}
	for _, rule := range canonical {
		validated, err := finishRule(rule)
		if err != nil || validated.ID() != rule.ID() {
			return Contract{}, &Error{Reason: "contract contains an unvalidated rule"}
		}
		if seen[rule.key()] {
			return Contract{}, &Error{Reason: "contract duplicates rule target " + rule.key()}
		}
		seen[rule.key()] = true
	}
	return Contract{
		id: id, version: SchemaVersion, rules: canonical,
		fingerprint: fingerprint(id, SchemaVersion, canonical),
	}, nil
}

func (c Contract) ID() string    { return c.id }
func (c Contract) Version() int  { return c.version }
func (c Contract) Rules() []Rule { return append([]Rule(nil), c.rules...) }
func (c Contract) Fingerprint() string {
	return c.fingerprint
}

func fingerprint(id string, version int, rules []Rule) string {
	var text strings.Builder
	fmt.Fprintf(&text, "%s|v%d", id, version)
	for _, rule := range rules {
		text.WriteByte('\n')
		text.WriteString(rule.ID())
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(text.String())))
}

// Query is the complete typed rule input for one definition.
type Query struct {
	Definition identity.DefinitionID
	Package    identity.PackageID
	Intrinsic  bool
	Facts      map[SelectionFactKind]bool
}

func (r Rule) matches(query Query) bool {
	switch r.selector {
	case SelectorExactDefinition:
		if r.definition != query.Definition {
			return false
		}
	case SelectorExactPackage:
		if r.pkg != query.Package {
			return false
		}
	case SelectorNamespace:
		if r.namespace != query.Package.Owner().Class() {
			return false
		}
	default:
		return false
	}
	switch r.condition {
	case ConditionAlways:
		return true
	case ConditionFactTrue:
		return query.Facts[r.fact]
	case ConditionBodyless:
		return query.Definition.Kind() == identity.DefinitionBodylessDecl
	case ConditionIntrinsic:
		return query.Intrinsic
	case ConditionSynthetic:
		return query.Definition.SyntheticRole().Valid()
	default:
		return false
	}
}

// Witness records the exact rule and fact evidence used by a binding.
type Witness struct {
	RuleID    string
	Selector  Selector
	Condition Condition
	Facts     []SelectionFactKind
}

// Bind applies deterministic specificity and rejects same-tier disagreement.
func (c Contract) Bind(query Query) (Provider, Witness, error) {
	if err := c.validateQuery(query); err != nil {
		return ProviderInvalid, Witness{}, err
	}
	best := int(^uint(0) >> 1)
	var matches []Rule
	for _, rule := range c.rules {
		if !rule.matches(query) {
			continue
		}
		tier := rule.tier()
		if tier < best {
			best = tier
			matches = nil
		}
		if tier == best {
			matches = append(matches, rule)
		}
	}
	if len(matches) == 0 {
		return ProviderInvalid, Witness{}, &Error{
			Reason: "contract " + c.id + " binds no rule to " + query.Definition.String(),
		}
	}
	for _, candidate := range matches[1:] {
		if candidate.provider != matches[0].provider {
			return ProviderInvalid, Witness{}, &Error{
				Reason: "same-tier rules disagree for " + query.Definition.String(),
			}
		}
	}
	winner := matches[0]
	witness := Witness{
		RuleID: winner.ID(), Selector: winner.selector, Condition: winner.condition,
	}
	if winner.condition == ConditionFactTrue {
		witness.Facts = []SelectionFactKind{winner.fact}
	}
	return winner.provider, witness, nil
}

func (c Contract) validateQuery(query Query) error {
	if query.Definition.IsZero() || query.Package.IsZero() {
		return &Error{Reason: "binding query requires definition and package identities"}
	}
	if query.Definition.Kind() == identity.DefinitionImplicit {
		if query.Definition.Package() != query.Package {
			return &Error{Reason: "implicit definition and package identities disagree"}
		}
	} else if query.Definition.File().Owner() != query.Package.Owner() {
		return &Error{Reason: "source definition and package owners disagree"}
	}
	required := c.RequestedFacts(query.Definition, query.Package)
	if len(query.Facts) != len(required) {
		return &Error{Reason: "binding query does not provide the exact requested fact set"}
	}
	for _, kind := range required {
		if _, present := query.Facts[kind]; !present {
			return &Error{Reason: "binding query omits requested fact " + kind.String()}
		}
	}
	for kind := range query.Facts {
		if !kind.Valid() {
			return &Error{Reason: "binding query contains an invalid fact kind"}
		}
	}
	return nil
}

// RequestedFacts returns the finite closed fact set needed for a definition
// candidate. It is derived solely from declared rules.
func (c Contract) RequestedFacts(definition identity.DefinitionID, pkg identity.PackageID) []SelectionFactKind {
	set := map[SelectionFactKind]bool{}
	for _, rule := range c.rules {
		if rule.condition != ConditionFactTrue {
			continue
		}
		switch rule.selector {
		case SelectorExactDefinition:
			if rule.definition != definition {
				continue
			}
		case SelectorExactPackage:
			if rule.pkg != pkg {
				continue
			}
		case SelectorNamespace:
			if rule.namespace != pkg.Owner().Class() {
				continue
			}
		}
		set[rule.fact] = true
	}
	out := make([]SelectionFactKind, 0, len(set))
	for fact := range set {
		out = append(out, fact)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Error is the typed contract validation/selection failure.
type Error struct{ Reason string }

func (e *Error) Error() string { return "GOTOTS_PROVIDER_CONTRACT: " + e.Reason }

// DefaultID names the built-in project-independent contract.
const DefaultID = "portable@v1"

func Default() (Contract, error) {
	spec := []struct {
		owner     identity.OwnerClass
		condition Condition
		fact      SelectionFactKind
		provider  Provider
	}{
		{identity.OwnerModule, ConditionAlways, SelectionFactInvalid, ProviderAutomaticTranslation},
		{identity.OwnerStandardLibrary, ConditionAlways, SelectionFactInvalid, ProviderGostdlib},
		{identity.OwnerToolchain, ConditionAlways, SelectionFactInvalid, ProviderToolchainSource},
		{identity.OwnerLanguagePseudo, ConditionAlways, SelectionFactInvalid, ProviderLanguageIntrinsic},
		{identity.OwnerModule, ConditionFactTrue, SelectionFactCDependent, ProviderExternalObligation},
		{identity.OwnerStandardLibrary, ConditionFactTrue, SelectionFactCDependent, ProviderExternalObligation},
		{identity.OwnerToolchain, ConditionFactTrue, SelectionFactCDependent, ProviderExternalObligation},
		{identity.OwnerModule, ConditionBodyless, SelectionFactInvalid, ProviderExternalObligation},
		{identity.OwnerModule, ConditionSynthetic, SelectionFactInvalid, ProviderExternalObligation},
		{identity.OwnerStandardLibrary, ConditionSynthetic, SelectionFactInvalid, ProviderExternalObligation},
		{identity.OwnerToolchain, ConditionSynthetic, SelectionFactInvalid, ProviderExternalObligation},
		{identity.OwnerStandardLibrary, ConditionIntrinsic, SelectionFactInvalid, ProviderLanguageIntrinsic},
	}
	rules := make([]Rule, 0, len(spec))
	for _, item := range spec {
		rule, err := NewNamespaceRule(item.owner, item.condition, item.fact, item.provider)
		if err != nil {
			return Contract{}, err
		}
		rules = append(rules, rule)
	}
	return New(DefaultID, rules)
}
