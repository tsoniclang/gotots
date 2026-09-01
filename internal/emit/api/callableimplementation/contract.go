package callableimplementation

import (
	"fmt"
	"slices"

	implementationcontract "github.com/tsoniclang/gotots/internal/contracts/implementation"
)

type Variant uint8

const (
	VariantInvalid Variant = iota
	VariantSource
	VariantKernel
)

func (v Variant) Valid() bool {
	return v == VariantSource || v == VariantKernel
}

type Selection struct {
	sourceIdentity string
	outputPath     string
	export         string
	variant        Variant
	packagePath    string
	modulePath     string
	moduleVersion  string
	moduleDigest   string
	sourceDigest   string
	envelope       implementationcontract.Envelope
}

func NewSelection(
	sourceIdentity string,
	outputPath string,
	export string,
	variant Variant,
	packagePath string,
	modulePath string,
	moduleVersion string,
	moduleDigest string,
	sourceDigest string,
	envelope implementationcontract.Envelope,
) (Selection, error) {
	selection := Selection{
		sourceIdentity: sourceIdentity,
		outputPath:     outputPath,
		export:         export,
		variant:        variant,
		packagePath:    packagePath,
		modulePath:     modulePath,
		moduleVersion:  moduleVersion,
		moduleDigest:   moduleDigest,
		sourceDigest:   sourceDigest,
		envelope:       cloneEnvelope(envelope),
	}
	if !selection.Valid() {
		return Selection{}, &Error{Reason: "selection is incomplete"}
	}
	return selection, nil
}

func (s Selection) Valid() bool {
	return s.sourceIdentity != "" && s.outputPath != "" &&
		s.export != "" && s.variant.Valid() && s.packagePath != "" &&
		s.moduleDigest != "" && s.sourceDigest != "" && s.envelope.Valid()
}

func (s Selection) SourceIdentity() string { return s.sourceIdentity }
func (s Selection) OutputPath() string     { return s.outputPath }
func (s Selection) Export() string         { return s.export }
func (s Selection) Variant() Variant       { return s.variant }
func (s Selection) PackagePath() string    { return s.packagePath }
func (s Selection) ModulePath() string     { return s.modulePath }
func (s Selection) ModuleVersion() string  { return s.moduleVersion }
func (s Selection) ModuleDigest() string   { return s.moduleDigest }
func (s Selection) SourceDigest() string   { return s.sourceDigest }
func (s Selection) EquivalenceEnvelope() implementationcontract.Envelope {
	return cloneEnvelope(s.envelope)
}

func cloneEnvelope(source implementationcontract.Envelope) implementationcontract.Envelope {
	result := source
	result.PreservedObservables = slices.Clone(source.PreservedObservables)
	result.Evidence = slices.Clone(source.Evidence)
	return result
}

type TargetKind uint8

const (
	TargetInvalid TargetKind = iota
	TargetModuleFunction
	TargetStaticMethod
)

func (k TargetKind) Valid() bool {
	return k == TargetModuleFunction || k == TargetStaticMethod
}

type Target struct {
	outputPath string
	kind       TargetKind
	export     string
	className  string
	memberName string
}

func NewModuleTarget(outputPath string, export string) (Target, error) {
	target := Target{
		outputPath: outputPath,
		kind:       TargetModuleFunction,
		export:     export,
	}
	if !target.Valid() {
		return Target{}, &Error{Reason: "module target is incomplete"}
	}
	return target, nil
}

func NewStaticMethodTarget(
	outputPath string,
	className string,
	memberName string,
) (Target, error) {
	target := Target{
		outputPath: outputPath,
		kind:       TargetStaticMethod,
		className:  className,
		memberName: memberName,
	}
	if !target.Valid() {
		return Target{}, &Error{Reason: "static-method target is incomplete"}
	}
	return target, nil
}

func (t Target) Valid() bool {
	if t.outputPath == "" || !t.kind.Valid() {
		return false
	}
	if t.kind == TargetModuleFunction {
		return t.export != "" && t.className == "" && t.memberName == ""
	}
	return t.export == "" && t.className != "" && t.memberName != ""
}

func (t Target) OutputPath() string { return t.outputPath }
func (t Target) Kind() TargetKind   { return t.kind }
func (t Target) Export() string     { return t.export }
func (t Target) ClassName() string  { return t.className }
func (t Target) MemberName() string { return t.memberName }

type Error struct {
	Reason string
}

func (e *Error) Error() string {
	return fmt.Sprintf("validate callable implementation contract: %s", e.Reason)
}
