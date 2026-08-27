package callableimplementation

import (
	"path/filepath"
	"slices"
	"sort"
)

type GeneratedTargetKind uint8

const (
	GeneratedTargetInvalid GeneratedTargetKind = iota
	GeneratedTargetModuleFunction
	GeneratedTargetStaticMethod
)

func (k GeneratedTargetKind) Valid() bool {
	return k == GeneratedTargetModuleFunction ||
		k == GeneratedTargetStaticMethod
}

type GeneratedTarget struct {
	sourceIdentity string
	variant        Variant
	outputPath     string
	kind           GeneratedTargetKind
	export         string
	className      string
	memberName     string
}

func NewGeneratedModuleTarget(
	sourceIdentity string,
	variant Variant,
	outputPath string,
	export string,
) (GeneratedTarget, error) {
	target := GeneratedTarget{
		sourceIdentity: sourceIdentity,
		variant:        variant,
		outputPath:     outputPath,
		kind:           GeneratedTargetModuleFunction,
		export:         export,
	}
	if !target.Valid() {
		return GeneratedTarget{}, &Error{
			Operation: "plan generated target",
			Subject:   sourceIdentity,
			Reason:    "module target is invalid",
		}
	}
	return target, nil
}

func NewGeneratedStaticMethodTarget(
	sourceIdentity string,
	variant Variant,
	outputPath string,
	className string,
	memberName string,
) (GeneratedTarget, error) {
	target := GeneratedTarget{
		sourceIdentity: sourceIdentity,
		variant:        variant,
		outputPath:     outputPath,
		kind:           GeneratedTargetStaticMethod,
		className:      className,
		memberName:     memberName,
	}
	if !target.Valid() {
		return GeneratedTarget{}, &Error{
			Operation: "plan generated target",
			Subject:   sourceIdentity,
			Reason:    "static-method target is invalid",
		}
	}
	return target, nil
}

func (t GeneratedTarget) Valid() bool {
	if t.sourceIdentity == "" || !t.variant.Valid() ||
		!validTargetPath(t.outputPath) || !t.kind.Valid() {
		return false
	}
	if t.kind == GeneratedTargetModuleFunction {
		return t.export != "" && t.className == "" && t.memberName == ""
	}
	return t.export == "" && t.className != "" && t.memberName != ""
}

func (t GeneratedTarget) SourceIdentity() string    { return t.sourceIdentity }
func (t GeneratedTarget) Variant() Variant          { return t.variant }
func (t GeneratedTarget) OutputPath() string        { return t.outputPath }
func (t GeneratedTarget) Kind() GeneratedTargetKind { return t.kind }
func (t GeneratedTarget) Export() string            { return t.export }
func (t GeneratedTarget) ClassName() string         { return t.className }
func (t GeneratedTarget) MemberName() string        { return t.memberName }

type GeneratedContractPlan struct {
	modules []Module
	targets []GeneratedTarget
}

func (p GeneratedContractPlan) Valid() bool {
	if len(p.modules) == 0 || len(p.targets) == 0 {
		return false
	}
	identities := make(map[string]struct{}, len(p.targets))
	for _, target := range p.targets {
		if !target.Valid() {
			return false
		}
		if _, duplicate := identities[target.sourceIdentity]; duplicate {
			return false
		}
		identities[target.sourceIdentity] = struct{}{}
	}
	claims := 0
	outputs := make(map[string]struct{}, len(p.modules))
	for _, module := range p.modules {
		if module.outputPath == "" || module.sourcePath == "" ||
			module.sourceDigest == "" || module.digest == "" ||
			len(module.callableClaims) == 0 {
			return false
		}
		if _, duplicate := outputs[module.outputPath]; duplicate {
			return false
		}
		outputs[module.outputPath] = struct{}{}
		claims += len(module.callableClaims)
	}
	return claims == len(p.targets)
}

func (p GeneratedContractPlan) Modules() []Module {
	return slicesCloneModules(p.modules)
}

func (p GeneratedContractPlan) Targets() []GeneratedTarget {
	return slices.Clone(p.targets)
}

func (c *Certificate) PlanGeneratedContracts(
	targets []GeneratedTarget,
) (GeneratedContractPlan, error) {
	if !c.Valid() || len(targets) != len(c.byIdentity) {
		return GeneratedContractPlan{}, &Error{
			Operation: "plan generated contract",
			Reason:    "generated target evidence is incomplete",
		}
	}
	selected := slices.Clone(targets)
	sort.Slice(selected, func(left, right int) bool {
		return selected[left].sourceIdentity < selected[right].sourceIdentity
	})
	seen := make(map[string]struct{}, len(selected))
	for _, target := range selected {
		implementation, ok := c.byIdentity[target.sourceIdentity]
		if !ok || !target.Valid() || target.variant != implementation.variant {
			return GeneratedContractPlan{}, &Error{
				Operation: "join generated target",
				Subject:   target.sourceIdentity,
				Reason:    "generated target differs from its selected callable",
			}
		}
		if _, duplicate := seen[target.sourceIdentity]; duplicate {
			return GeneratedContractPlan{}, &Error{
				Operation: "join generated target",
				Subject:   target.sourceIdentity,
				Reason:    "generated target is duplicated",
			}
		}
		seen[target.sourceIdentity] = struct{}{}
	}
	plan := GeneratedContractPlan{
		modules: slicesCloneModules(c.modules),
		targets: selected,
	}
	if !plan.Valid() {
		return GeneratedContractPlan{}, &Error{
			Operation: "plan generated contract",
			Reason:    "generated contract plan is invalid",
		}
	}
	return plan, nil
}

func validTargetPath(path string) bool {
	return path != "" && filepath.IsLocal(filepath.FromSlash(path)) &&
		filepath.ToSlash(filepath.Clean(filepath.FromSlash(path))) == path &&
		filepath.Ext(path) == ".ts"
}
