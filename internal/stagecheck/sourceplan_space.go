package stagecheck

import (
	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/source"
)

type independentRuleCandidate struct {
	definition identity.DefinitionID
	pkg        identity.PackageID
	intrinsic  bool
	bodyless   bool
	facts      map[contract.SelectionFactKind]bool
}

func independentPackageMayTranslate(
	pkg *source.LoadedPackage,
	selected contract.Contract,
) bool {
	for _, bodyless := range []bool{false, true} {
		if independentAutomaticPossible(
			selected,
			independentRuleCandidate{
				pkg: pkg.ID(),
				intrinsic: pkg.Disposition() ==
					source.DispositionUnsafeIntrinsic,
				bodyless: bodyless,
			},
		) {
			return true
		}
	}
	return false
}

func independentExactMayTranslate(
	file identity.FileID,
	pkg *source.LoadedPackage,
	selected contract.Contract,
) bool {
	seen := map[identity.DefinitionID]bool{}
	for _, rule := range selected.Rules() {
		definition := rule.Definition()
		if rule.Selector() != contract.SelectorExactDefinition ||
			definition.File() != file ||
			seen[definition] {
			continue
		}
		seen[definition] = true
		if independentAutomaticPossible(
			selected,
			independentRuleCandidate{
				definition: definition,
				pkg:        pkg.ID(),
				intrinsic: pkg.Disposition() ==
					source.DispositionUnsafeIntrinsic,
				bodyless: definition.Kind() ==
					identity.DefinitionBodylessDecl,
			},
		) {
			return true
		}
	}
	return false
}

func independentAutomaticPossible(
	selected contract.Contract,
	base independentRuleCandidate,
) bool {
	for _, witness := range selected.Rules() {
		if witness.Provider() !=
			contract.ProviderAutomaticTranslation ||
			!independentSelectorMatches(witness, base) ||
			!independentConditionPossible(witness, base) {
			continue
		}
		candidate := base
		candidate.facts = map[contract.SelectionFactKind]bool{}
		if witness.Condition() == contract.ConditionFactTrue {
			candidate.facts[witness.FactKind()] = true
		}
		best := int(^uint(0) >> 1)
		provider := contract.ProviderInvalid
		conflict := false
		for _, rule := range selected.Rules() {
			if !independentRuleMatches(rule, candidate) {
				continue
			}
			tier := independentRuleTier(rule)
			if tier < best {
				best = tier
				provider = rule.Provider()
				conflict = false
			}
			if tier == best && provider != rule.Provider() {
				conflict = true
			}
		}
		if !conflict &&
			provider == contract.ProviderAutomaticTranslation {
			return true
		}
	}
	return false
}

func independentSelectorMatches(
	rule contract.Rule,
	candidate independentRuleCandidate,
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

func independentConditionPossible(
	rule contract.Rule,
	candidate independentRuleCandidate,
) bool {
	switch rule.Condition() {
	case contract.ConditionAlways, contract.ConditionFactTrue:
		return true
	case contract.ConditionBodyless:
		return candidate.bodyless
	case contract.ConditionIntrinsic:
		return candidate.intrinsic
	case contract.ConditionSynthetic:
		return false
	default:
		return false
	}
}

func independentRuleMatches(
	rule contract.Rule,
	candidate independentRuleCandidate,
) bool {
	if !independentSelectorMatches(rule, candidate) {
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
		return false
	default:
		return false
	}
}

func independentRuleTier(rule contract.Rule) int {
	tier := 0
	switch rule.Selector() {
	case contract.SelectorExactPackage:
		tier = 2
	case contract.SelectorNamespace:
		tier = 4
	}
	if rule.Condition() == contract.ConditionAlways {
		tier++
	}
	return tier
}
