package sourceplan

import (
	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/source"
)

type ruleCandidate struct {
	definition identity.DefinitionID
	pkg        identity.PackageID
	intrinsic  bool
	synthetic  bool
	bodyless   bool
	facts      map[contract.SelectionFactKind]bool
}

func packageMayTranslate(
	pkg *source.LoadedPackage,
	selected contract.Contract,
) bool {
	for _, bodyless := range []bool{false, true} {
		if automaticPossible(selected, ruleCandidate{
			pkg: pkg.ID(),
			intrinsic: pkg.Disposition() ==
				source.DispositionUnsafeIntrinsic,
			bodyless: bodyless,
		}) {
			return true
		}
	}
	return false
}

func exactDefinitionMayTranslate(
	file identity.FileID,
	pkg *source.LoadedPackage,
	selected contract.Contract,
) bool {
	seen := map[identity.DefinitionID]bool{}
	for _, rule := range selected.Rules() {
		if rule.Selector() != contract.SelectorExactDefinition ||
			rule.Definition().File() != file ||
			seen[rule.Definition()] {
			continue
		}
		seen[rule.Definition()] = true
		if automaticPossible(selected, ruleCandidate{
			definition: rule.Definition(),
			pkg:        pkg.ID(),
			intrinsic: pkg.Disposition() ==
				source.DispositionUnsafeIntrinsic,
			bodyless: rule.Definition().Kind() ==
				identity.DefinitionBodylessDecl,
		}) {
			return true
		}
	}
	return false
}

func automaticPossible(
	selected contract.Contract,
	base ruleCandidate,
) bool {
	for _, candidateRule := range selected.Rules() {
		if candidateRule.Provider() !=
			contract.ProviderAutomaticTranslation ||
			!planningSelectorMatches(candidateRule, base) ||
			!planningConditionCanMatch(candidateRule, base) {
			continue
		}
		candidate := base
		candidate.facts = map[contract.SelectionFactKind]bool{}
		if candidateRule.Condition() == contract.ConditionFactTrue {
			candidate.facts[candidateRule.FactKind()] = true
		}
		bestTier := int(^uint(0) >> 1)
		provider := contract.ProviderInvalid
		disagreement := false
		for _, rule := range selected.Rules() {
			if !planningRuleMatches(rule, candidate) {
				continue
			}
			tier := planningRuleTier(rule)
			if tier < bestTier {
				bestTier = tier
				provider = rule.Provider()
				disagreement = false
			}
			if tier == bestTier && rule.Provider() != provider {
				disagreement = true
			}
		}
		if !disagreement &&
			provider == contract.ProviderAutomaticTranslation {
			return true
		}
	}
	return false
}

func planningSelectorMatches(
	rule contract.Rule,
	candidate ruleCandidate,
) bool {
	switch rule.Selector() {
	case contract.SelectorExactDefinition:
		return !candidate.definition.IsZero() &&
			rule.Definition() == candidate.definition
	case contract.SelectorExactPackage:
		return rule.Package() == candidate.pkg
	case contract.SelectorNamespace:
		return rule.Namespace() == candidate.pkg.Owner().Class()
	default:
		return false
	}
}

func planningConditionCanMatch(
	rule contract.Rule,
	candidate ruleCandidate,
) bool {
	switch rule.Condition() {
	case contract.ConditionAlways, contract.ConditionFactTrue:
		return true
	case contract.ConditionBodyless:
		return candidate.bodyless
	case contract.ConditionIntrinsic:
		return candidate.intrinsic
	case contract.ConditionSynthetic:
		return candidate.synthetic
	default:
		return false
	}
}

func planningRuleMatches(
	rule contract.Rule,
	candidate ruleCandidate,
) bool {
	if !planningSelectorMatches(rule, candidate) {
		return false
	}
	switch rule.Condition() {
	case contract.ConditionAlways:
		return true
	case contract.ConditionFactTrue:
		return candidate.facts[rule.FactKind()]
	case contract.ConditionBodyless:
		return candidate.bodyless
	case contract.ConditionIntrinsic:
		return candidate.intrinsic
	case contract.ConditionSynthetic:
		return candidate.synthetic
	default:
		return false
	}
}

func planningRuleTier(rule contract.Rule) int {
	selector := map[contract.Selector]int{
		contract.SelectorExactDefinition: 0,
		contract.SelectorExactPackage:    1,
		contract.SelectorNamespace:       2,
	}[rule.Selector()]
	condition := 0
	if rule.Condition() == contract.ConditionAlways {
		condition = 1
	}
	return selector*2 + condition
}
