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
	GeneratedFile   string            `json:"generatedFile"`
	GeneratedSymbol string            `json:"generatedSymbol"`
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
}

// Options carries provenance inputs for product runs.
type Options struct {
	SourceRevision string
	ProfileHash    string
}

// LoweringPlanV1 names the conservative vertical-slice plan: every value
// uses the exact conservative carrier of its Go type; no direct-form
// optimizations are selected.
const LoweringPlanV1 = "conservative-v1"

// abiDir is the bundle directory of the language-ABI modules.
const abiDir = "language-abi"

// supportRegistry is the reviewed semantic-class support registry; a
// generated body whose operation census contains an unregistered class
// fails the run — support is explicit, never implicit.
var supportRegistry = sync.OnceValues(contracts.LoadRegistry)
