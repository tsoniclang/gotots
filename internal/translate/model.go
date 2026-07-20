// Translation result model: proof records, the per-unit support
// ledger, withheld output, and run options.
package translate

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/tsoniclang/gotots/internal/emit"
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
	// LoweredHash is the sha256 of the body's retained analysis artifact
	// (the exact emitted fragment under analysis/bodies/, spec 01) —
	// present for every lowered body, withheld packages included.
	LoweredHash string `json:"loweredHash,omitempty"`
	// InitHash is a package variable's initializer source hash — joined
	// against the census's identity-bearing initializer evidence.
	InitHash string `json:"initHash,omitempty"`
	// TypeShapeHash is a named type declaration's complete shape hash
	// (census.TypeShapeSignature over the translator's own loaded type) —
	// joined against the census type shape so kind, underlying, type
	// parameters, fields, tags, embeds, interface methods, and declared
	// methods cannot drift between the two pipelines.
	TypeShapeHash string `json:"typeShapeHash,omitempty"`
	// AliasShapeHash is a type alias declaration's shape hash
	// (census.AliasShapeSignature over the translator's own loaded alias).
	AliasShapeHash string `json:"aliasShapeHash,omitempty"`
	// VarTypeHash is a package variable's declared-TYPE hash (sha256 of
	// the canonical type spelling the translator lowered).
	VarTypeHash string `json:"varTypeHash,omitempty"`
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

// KeyUnreachableClaims — see Generated.
//
// BodySupport is one implementation unit's support-state record: every
// unsupported operation site is retained, and an unimplemented unit
// never has an emitted body.
type BodySupport struct {
	ID      string `json:"id"`
	Package string `json:"package"`
	// Kind is the typed entity class of this record: "body" for
	// function/method bodies, "initializer" for package-variable
	// initializers, "declaration" for declaration-level blockers.
	// Producers set it; no consumer may infer it from ID spelling.
	Kind  string               `json:"kind"`
	State ir.SupportState      `json:"state"`
	Sites []ir.UnsupportedSite `json:"sites,omitempty"`
}

// ImplementationArtifact is one implementation's artifact record
// (ADR-0010): exactly one per ImplementationID, hard-failed on
// duplicates at emission.
type ImplementationArtifact struct {
	ImplementationID string `json:"implementationId"`
	SourceID         string `json:"sourceId"`
	Package          string `json:"package"`
	// ArtifactPath is the dump-relative analysis artifact file — the
	// one spelling, recorded by the writer that owns the sanitizer.
	ArtifactPath string `json:"artifactPath"`
	Sha256       string `json:"sha256"`
}

// ModuleDisposition is one selected package's typed module outcome,
// recorded by the emission driver at the decision site.
type ModuleDisposition struct {
	Package string `json:"package"`
	// State: "emitted-runtime" | "no-runtime-output" | "not-materialized".
	State  string `json:"state"`
	Module string `json:"module,omitempty"`
	Reason string `json:"reason,omitempty"`
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
	// KeyUnreachableClaims are the union-$key member identities whose
	// encoder branches claim value-box unreachability — verified against
	// the corpus-complete value-box log at finalize (a violated claim is
	// an emitter defect).
	KeyUnreachableClaims map[string]bool
	// Support is the per-unit implementation support ledger.
	Support []BodySupport
	// Emissions is the emitter-owned emission-event ledger, collected
	// from every module in emission order: one event per emitted body,
	// placeholder, or initializer slot. Multiplicity is evidence.
	Emissions []emit.EmissionEvent
	// ExternSymbols is the emitter-owned exported-symbol ledger of
	// external-contract modules: stub symbols carry their thrown
	// obligation identity; support definitions carry none.
	ExternSymbols []emit.ExternSymbolRecord
	// ImplementationArtifacts is the one-artifact-per-implementation
	// ledger (ADR-0010); duplicates fail generation.
	ImplementationArtifacts []ImplementationArtifact
	// ModuleDispositions is the typed per-package module outcome: every
	// selected package records exactly one of emitted-runtime,
	// no-runtime-output, or not-materialized. No consumer may infer a
	// package's module state from file presence.
	ModuleDispositions []ModuleDisposition
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

// shapeHash digests one canonical declaration-shape join string.
func shapeHash(signature string) string {
	digest := sha256.Sum256([]byte(signature))
	return hex.EncodeToString(digest[:])
}
