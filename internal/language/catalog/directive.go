package catalog

import "fmt"

// DirectiveKind is the closed catalog of comment directives. Every go:*
// directive must be a member — an unknown go:* form fails inventory closed —
// while non-go tool directives fall under the single external-tool member and
// are inventoried with their tool recorded.
type DirectiveKind uint16

// DirectiveDisposition is the closed statement of which owner handles one
// directive family.
type DirectiveDisposition uint8

const (
	DirectiveDispositionInvalid DirectiveDisposition = iota
	// DirectiveLoaderOwned: honored by the workspace loader / build selection.
	DirectiveLoaderOwned
	// DirectiveDisplayOwned: affects display positions only.
	DirectiveDisplayOwned
	// DirectiveToolingOnly: consumed by external tooling, no translation effect.
	DirectiveToolingOnly
	// DirectivePerformanceHint: a compiler performance pragma with no
	// observable semantics.
	DirectivePerformanceHint
	// DirectiveSemanticObligation: carries translation semantics that a later
	// phase must implement or explicitly block on.
	DirectiveSemanticObligation
	// DirectiveUnsupportedPragma: a low-level pragma outside the product's
	// semantic surface; reachable occurrences must be explicitly dispositioned.
	DirectiveUnsupportedPragma
	// DirectiveExternalObligation: binds host/external behavior (cgo family).
	DirectiveExternalObligation
	// DirectiveRuntimeConfiguration: configures the runtime (go:debug).
	DirectiveRuntimeConfiguration

	numDirectiveDispositions
)

var directiveDispositionNames = [numDirectiveDispositions]string{
	DirectiveLoaderOwned: "loader-owned", DirectiveDisplayOwned: "display-owned",
	DirectiveToolingOnly: "tooling-only", DirectivePerformanceHint: "performance-hint",
	DirectiveSemanticObligation: "semantic-obligation", DirectiveUnsupportedPragma: "unsupported-pragma",
	DirectiveExternalObligation: "external-obligation", DirectiveRuntimeConfiguration: "runtime-configuration",
}

// Valid reports whether d names a disposition.
func (d DirectiveDisposition) Valid() bool {
	return d > DirectiveDispositionInvalid && d < numDirectiveDispositions
}

// String renders d for reports.
func (d DirectiveDisposition) String() string {
	if d.Valid() {
		return directiveDispositionNames[d]
	}
	return fmt.Sprintf("catalog.DirectiveDisposition(%d)", uint8(d))
}

// Explicit, permanent directive identities. Do not renumber; append only.
const (
	DirectiveInvalid DirectiveKind = 0

	// Structural members.
	DirectiveLine           DirectiveKind = 1 // //line file:line display adjustment
	DirectiveLegacyBuildTag DirectiveKind = 2 // // +build (legacy constraint form)
	DirectiveExternalTool   DirectiveKind = 3 // //tool:name for tool != go

	// go:* members.
	DirectiveGoBuild              DirectiveKind = 4
	DirectiveGoEmbed              DirectiveKind = 5
	DirectiveGoGenerate           DirectiveKind = 6
	DirectiveGoLinkname           DirectiveKind = 7
	DirectiveGoNoinline           DirectiveKind = 8
	DirectiveGoNosplit            DirectiveKind = 9
	DirectiveGoNoescape           DirectiveKind = 10
	DirectiveGoNorace             DirectiveKind = 11
	DirectiveGoNocheckptr         DirectiveKind = 12
	DirectiveGoUintptrescapes     DirectiveKind = 13
	DirectiveGoUintptrkeepalive   DirectiveKind = 14
	DirectiveGoSystemstack        DirectiveKind = 15
	DirectiveGoNowritebarrier     DirectiveKind = 16
	DirectiveGoNowritebarrierrec  DirectiveKind = 17
	DirectiveGoYeswritebarrierrec DirectiveKind = 18
	DirectiveGoNointerface        DirectiveKind = 19
	DirectiveGoDebug              DirectiveKind = 20
	DirectiveGoCgoImportDynamic   DirectiveKind = 21
	DirectiveGoCgoExportStatic    DirectiveKind = 22
	DirectiveGoCgoExportDynamic   DirectiveKind = 23
	DirectiveGoCgoImportStatic    DirectiveKind = 24
	DirectiveGoCgoDynamicLinker   DirectiveKind = 25
	DirectiveGoCgoLdflag          DirectiveKind = 26
	DirectiveGoCgoUnsafeArgs      DirectiveKind = 27
	DirectiveGoNotinheap          DirectiveKind = 28
	DirectiveGoFix                DirectiveKind = 29

	// directiveCount is the highest assigned identity; append-only.
	directiveCount = 29
)

type directiveDescriptor struct {
	name        string // go:* name for go members; symbolic name otherwise
	disposition DirectiveDisposition
}

var directiveTable = [directiveCount + 1]directiveDescriptor{
	DirectiveLine:           {"line", DirectiveDisplayOwned},
	DirectiveLegacyBuildTag: {"+build", DirectiveLoaderOwned},
	DirectiveExternalTool:   {"external-tool", DirectiveToolingOnly},

	DirectiveGoBuild:              {"build", DirectiveLoaderOwned},
	DirectiveGoEmbed:              {"embed", DirectiveSemanticObligation},
	DirectiveGoGenerate:           {"generate", DirectiveToolingOnly},
	DirectiveGoLinkname:           {"linkname", DirectiveUnsupportedPragma},
	DirectiveGoNoinline:           {"noinline", DirectivePerformanceHint},
	DirectiveGoNosplit:            {"nosplit", DirectivePerformanceHint},
	DirectiveGoNoescape:           {"noescape", DirectiveUnsupportedPragma},
	DirectiveGoNorace:             {"norace", DirectivePerformanceHint},
	DirectiveGoNocheckptr:         {"nocheckptr", DirectivePerformanceHint},
	DirectiveGoUintptrescapes:     {"uintptrescapes", DirectiveUnsupportedPragma},
	DirectiveGoUintptrkeepalive:   {"uintptrkeepalive", DirectiveUnsupportedPragma},
	DirectiveGoSystemstack:        {"systemstack", DirectiveUnsupportedPragma},
	DirectiveGoNowritebarrier:     {"nowritebarrier", DirectiveUnsupportedPragma},
	DirectiveGoNowritebarrierrec:  {"nowritebarrierrec", DirectiveUnsupportedPragma},
	DirectiveGoYeswritebarrierrec: {"yeswritebarrierrec", DirectiveUnsupportedPragma},
	DirectiveGoNointerface:        {"nointerface", DirectiveUnsupportedPragma},
	DirectiveGoDebug:              {"debug", DirectiveRuntimeConfiguration},
	DirectiveGoCgoImportDynamic:   {"cgo_import_dynamic", DirectiveExternalObligation},
	DirectiveGoCgoExportStatic:    {"cgo_export_static", DirectiveExternalObligation},
	DirectiveGoCgoExportDynamic:   {"cgo_export_dynamic", DirectiveExternalObligation},
	DirectiveGoCgoImportStatic:    {"cgo_import_static", DirectiveExternalObligation},
	DirectiveGoCgoDynamicLinker:   {"cgo_dynamic_linker", DirectiveExternalObligation},
	DirectiveGoCgoLdflag:          {"cgo_ldflag", DirectiveExternalObligation},
	DirectiveGoCgoUnsafeArgs:      {"cgo_unsafe_args", DirectiveExternalObligation},
	DirectiveGoNotinheap:          {"notinheap", DirectiveUnsupportedPragma},
	DirectiveGoFix:                {"fix", DirectiveToolingOnly},
}

// Valid reports whether k names a directive member.
func (k DirectiveKind) Valid() bool { return k >= 1 && k <= directiveCount }

// Name is the member's directive name (go:<name> for go members).
func (k DirectiveKind) Name() string {
	if !k.Valid() {
		return ""
	}
	return directiveTable[k].name
}

// Disposition is the owning disposition.
func (k DirectiveKind) Disposition() DirectiveDisposition {
	if !k.Valid() {
		return DirectiveDispositionInvalid
	}
	return directiveTable[k].disposition
}

// String renders k for reports.
func (k DirectiveKind) String() string {
	if name := k.Name(); name != "" {
		return name
	}
	return fmt.Sprintf("catalog.DirectiveKind(%d)", uint16(k))
}

// AllDirectives returns every directive member in ascending identity order.
func AllDirectives() []DirectiveKind {
	out := make([]DirectiveKind, 0, directiveCount)
	for id := 1; id <= directiveCount; id++ {
		out = append(out, DirectiveKind(id))
	}
	return out
}

// goDirectiveByName indexes the go:* members by name once.
var goDirectiveByName = func() map[string]DirectiveKind {
	m := map[string]DirectiveKind{}
	for id := DirectiveGoBuild; id <= directiveCount; id++ {
		m[directiveTable[id].name] = id
	}
	return m
}()

// GoDirectiveByName resolves a go:<name> directive to its member, or
// DirectiveInvalid when the name is unknown (which fails inventory closed).
func GoDirectiveByName(name string) DirectiveKind { return goDirectiveByName[name] }
