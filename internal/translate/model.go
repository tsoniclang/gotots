// Translation result model: proof records, the per-unit support
// ledger, withheld output, and run options.
package translate

import (
	"sync"

	"github.com/tsoniclang/gotots/contracts"
	"github.com/tsoniclang/gotots/internal/ir"
)

// Proof is the translation proof record of one declaration.
type Proof struct {
	ID              string            `json:"id"`
	SourceRevision  string            `json:"sourceRevision,omitempty"`
	Package         string            `json:"package"`
	File            string            `json:"file"`
	SignatureHash   string            `json:"signatureHash"`
	BodyHash        string            `json:"bodyHash"`
	Operations      []string          `json:"operations"`
	Representations map[string]string `json:"representations"`
	LoweringPlan    string            `json:"loweringPlan"`
	// GeneratedFile/GeneratedSymbol reference the retained module output;
	// withholding clears them (a proof must never reference an absent
	// file) and records the fact in ModuleRetained.
	GeneratedFile   string `json:"generatedFile,omitempty"`
	GeneratedSymbol string `json:"generatedSymbol,omitempty"`
	// ModuleRetained states whether the body survived into a runnable
	// module: the ir-admitted / module-retained evidence-stage split, per
	// proof rather than inferred from aggregates. It is true exactly when
	// the package is retained AND the generated file exists AND the
	// generated symbol exists in it — finalized once, after every proof
	// and file exists.
	ModuleRetained bool `json:"moduleRetained"`
	// InitHash is a package variable's initializer source hash — joined
	// against the census's identity-bearing initializer evidence.
	InitHash string `json:"initHash,omitempty"`
	// ConstHash is a package constant's declaration-shape hash
	// (census.ConstShapeSignature over the type the translator lowered and
	// the exact folded value) — joined against the census constant shape so
	// a constant's type or value cannot drift between the two pipelines.
	ConstHash string `json:"constHash,omitempty"`
	// EffectOnly marks the enumerated symbol-less retained forms: a
	// blank variable's ordered initializer effect, emitted inside the
	// module's initialization sequence with no named binding.
	EffectOnly bool `json:"effectOnly,omitempty"`
	// NoOutput marks a declaration whose exact lowering emits nothing (a
	// blank variable without initializer, a fold-at-use constant): it is
	// disposed but never counted as a retained body.
	NoOutput bool `json:"noOutput,omitempty"`
}

// BodySupport is one implementation unit's support-state record: every
// unsupported operation site is retained, and an unimplemented unit
// never has an emitted body.
type BodySupport struct {
	ID      string               `json:"id"`
	Package string               `json:"package"`
	State   ir.SupportState      `json:"state"`
	Sites   []ir.UnsupportedSite `json:"sites,omitempty"`
}

// FuncLitSupport is one function literal's independent disposition.
type FuncLitSupport struct {
	ID       string `json:"id"`
	Parent   string `json:"parent"`
	Package  string `json:"package"`
	BodyHash string `json:"bodyHash"`
	State    string `json:"state"`
}

// Generated is one deterministic translation result. Packages whose
// dependency closure contains an unimplemented unit are withheld from
// runnable output; their analysis records remain.
type Generated struct {
	// Files maps bundle-relative paths to complete file contents.
	Files map[string]string
	// Ownership maps every generated path to its ownership root.
	Ownership map[string]string
	Proofs    []Proof
	// Support is the per-unit implementation support ledger.
	Support []BodySupport
	// Withheld maps package path -> reason runnable output is WITHHELD FROM
	// PUBLICATION. A withheld package is still MATERIALIZED (its analyzable
	// TypeScript is emitted and retained) unless it is also NotMaterialized;
	// withholding governs the runnable product, not analysis output.
	Withheld map[string]string
	// NotMaterialized maps package path -> reason the package cannot produce
	// analyzable TypeScript at all: a declaration-level blocker leaves a
	// structural hole (a missing class or binding) that dependents
	// reference, and this propagates transitively over imports. A
	// NotMaterialized package emits no file; every other selected package is
	// materialized (with typed throwing placeholders for its unimplemented
	// bodies) so it can be independently typechecked and structurally
	// verified even while publication-withheld.
	NotMaterialized map[string]string
	// FuncLits is the independent function-literal ledger: each literal's
	// canonical identity with its own support disposition, derived from
	// whether any unsupported site falls within the literal's span, and
	// its withholding linkage through the parent's package.
	FuncLits []FuncLitSupport
	// ModuleImports maps each package path to the co-generated packages
	// its emitted module needs AT RUNTIME (value symbol references,
	// including interface-dispatch branch targets, plus init edges).
	// Publication withholding cascades over exactly these edges.
	ModuleImports map[string][]string
	// ModuleTypeImports maps each package path to its TYPE-ONLY imports:
	// erased by compilation, they are analysis edges — the target must be
	// materialized (its analyzable file exists) but need not be published,
	// and publication never cascades over them.
	ModuleTypeImports map[string][]string
	// EmitterDefects records every body (or package) whose emission failed
	// on a non-typed error: a compiler defect, never an ordinary
	// unsupported disposition. A diagnostic placeholder keeps the file
	// analyzable, but the acceptance gates HARD-FAIL while any defect is
	// on record — the honest state is "the compiler must be fixed", not a
	// placeholder statistic.
	EmitterDefects []EmitterDefect
}

// EmitterDefect is one emission failure with its exact identity.
type EmitterDefect struct {
	Package string `json:"package"`
	ID      string `json:"id"`
	Err     string `json:"error"`
}

// Options carries provenance inputs for product runs.
type Options struct {
	SourceRevision string
	ProfileHash    string
}

// LoweringPlanV2 names the fixed-point representation plan: each
// value-flow region is lowered to the least elaborate representation
// its observed requirements admit — the slice planner selects the
// native array for owner-only regions and the exact carrier otherwise,
// and interface dispatch is a closed discriminated token switch. The
// version advances when the planner's selection space changes.
const LoweringPlanV2 = "representation-fixedpoint-v2"

// abiDir is the bundle directory of the language-ABI modules.
const abiDir = "language-abi"

// supportRegistry is the reviewed semantic-class support registry; a
// generated body whose operation census contains an unregistered class
// fails the run — support is explicit, never implicit.
var supportRegistry = sync.OnceValues(contracts.LoadRegistry)
