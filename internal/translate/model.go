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
	// Withheld maps package path -> reason runnable output is withheld.
	Withheld map[string]string
	// ModuleImports maps each package path to the co-generated packages
	// its emitted module actually imports (symbol references, including
	// interface-dispatch branch targets, plus init edges). Withholding
	// cascades over these real edges.
	ModuleImports map[string][]string
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
