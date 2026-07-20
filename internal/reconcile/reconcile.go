// Package reconcile produces the canonical identity-join report: every
// corpus count with a NAMED denominator, every pair of related identity
// sets joined with multiplicity preserved, and every one-sided
// difference carrying an explicit typed disposition. Every fact is read
// from a typed producer ledger (census, support, emission, extern
// symbol, module disposition); no fact is recovered from generated
// text, identity spelling, or disposition prose. The report is
// versioned machine evidence — Markdown summaries render FROM it.
package reconcile

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/census"
	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/translate"
)

const SchemaVersion = 2

// Report is the versioned reconciliation evidence.
type Report struct {
	SchemaVersion int    `json:"schemaVersion"`
	Head          string `json:"head"`
	// InputDigests binds every identity set to the artifact it was read
	// from (ledger/census digests supplied by the caller).
	InputDigests map[string]string `json:"inputDigests,omitempty"`
	Denominators []Denominator     `json:"denominators"`
	Joins        []Join            `json:"joins"`
	// Duplicates lists every identity that appears more than once in a
	// ledger whose identities must be unique. Each is a reconciliation
	// defect: the collapse that a set/map would silently perform is
	// recorded instead.
	Duplicates []Duplicate `json:"duplicates,omitempty"`
	// VariantReemissions lists body-placeholder identities emitted more
	// than once across core modules (family-variant copies of a shared
	// source ID, distinct once ImplementationID lands) with copy counts.
	VariantReemissions []VariantReemission `json:"variantReemissions,omitempty"`
	// Packages is the per-package materialization/publication ledger.
	Packages []PackageState `json:"packages"`
}

// Duplicate is one identity recorded more than once in a
// must-be-unique ledger.
type Duplicate struct {
	Ledger string `json:"ledger"`
	ID     string `json:"id"`
	Count  int    `json:"count"`
}

// VariantReemission is one shared-ID placeholder emitted in multiple
// family-variant module copies.
type VariantReemission struct {
	ID         string `json:"id"`
	CoreCopies int    `json:"coreCopies"`
}

// History carries prior-baseline identity sets so historical
// disagreements are joined mechanically, never narrated.
type History struct {
	Source           string   `json:"source"`
	PriorBodies      []string `json:"priorBodies"`
	PriorBodiesFrom  string   `json:"priorBodiesFrom"`
	PriorPublished   []string `json:"priorPublished"`
	PriorPublishedAt string   `json:"priorPublishedAt"`
}

// Denominator is one named count with its exact definition.
type Denominator struct {
	Name       string `json:"name"`
	Count      int    `json:"count"`
	Definition string `json:"definition"`
}

// Join relates two identity multisets; every one-sided identity carries
// a typed disposition.
type Join struct {
	Name  string `json:"name"`
	Left  string `json:"left"`
	Right string `json:"right"`
	// LeftSetSha256/RightSetSha256 digest each side's sorted identity
	// list (repeated identities repeated), so equal large sets are
	// verifiable without repeating them.
	LeftSetSha256  string  `json:"leftSetSha256"`
	RightSetSha256 string  `json:"rightSetSha256"`
	Both           int     `json:"both"`
	OnlyLeft       []Delta `json:"onlyLeft,omitempty"`
	OnlyRight      []Delta `json:"onlyRight,omitempty"`
}

// Delta is one identity present on one side only. Defect marks it a
// reconciliation defect — the TYPED verdict; disposition text is for
// humans and carries no authority.
type Delta struct {
	ID          string `json:"id"`
	Disposition string `json:"disposition"`
	Defect      bool   `json:"defect,omitempty"`
}

// PackageState records one owned production package's pipeline state.
type PackageState struct {
	Path         string `json:"path"`
	Materialized bool   `json:"materialized"`
	Published    bool   `json:"published"`
	WithheldWhy  string `json:"withheldWhy,omitempty"`
}

// multiset is an identity multiset: counts preserved, never collapsed.
type multiset map[string]int

func (m multiset) add(id string) { m[id]++ }

func (m multiset) digest() string {
	ids := make([]string, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	h := sha256.New()
	for _, id := range ids {
		for i := 0; i < m[id]; i++ {
			h.Write([]byte(id))
			h.Write([]byte{0})
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (m multiset) total() int {
	n := 0
	for _, c := range m {
		n += c
	}
	return n
}

// recordDuplicates appends a typed Duplicate for every identity with
// count > 1 in a must-be-unique ledger.
func (r *Report) recordDuplicates(ledger string, m multiset) {
	for id, count := range m {
		if count > 1 {
			r.Duplicates = append(r.Duplicates, Duplicate{Ledger: ledger, ID: id, Count: count})
		}
	}
}

// joinSets joins two identity multisets. Matching consumes counts
// pairwise; the surplus side carries the given disposition/defect.
func joinSets(name, left, right string, l, r multiset, onlyLeft, onlyRight string, leftDefect, rightDefect bool) Join {
	j := Join{Name: name, Left: left, Right: right, LeftSetSha256: l.digest(), RightSetSha256: r.digest()}
	for id, lc := range l {
		rc := r[id]
		if rc > lc {
			rc = lc
		}
		j.Both += rc
		for i := 0; i < lc-rc; i++ {
			j.OnlyLeft = append(j.OnlyLeft, Delta{ID: id, Disposition: onlyLeft, Defect: leftDefect})
		}
	}
	for id, rc := range r {
		lc := l[id]
		if lc > rc {
			lc = rc
		}
		for i := 0; i < rc-lc; i++ {
			j.OnlyRight = append(j.OnlyRight, Delta{ID: id, Disposition: onlyRight, Defect: rightDefect})
		}
	}
	sortDeltas(j.OnlyLeft)
	sortDeltas(j.OnlyRight)
	return j
}

// Build computes the report from one census run and one generation,
// consuming only typed producer ledgers.
func Build(head string, run *census.Result, generated *translate.Generated) *Report {
	out := &Report{SchemaVersion: SchemaVersion, Head: head}

	// --- census (typed declaration records) ---
	censusBodies := multiset{}
	censusBodiless := 0
	censusPackages := multiset{}
	censusPackageSet := map[string]bool{}
	for _, decl := range run.Report.Declarations {
		if decl.Scope != "production" {
			continue
		}
		if !censusPackageSet[decl.Package] {
			censusPackageSet[decl.Package] = true
			censusPackages.add(decl.Package)
		}
		if decl.Kind == "func" || decl.Kind == "method" {
			if decl.HasBody {
				censusBodies.add(decl.ID)
			} else {
				censusBodiless++
			}
		}
	}
	out.recordDuplicates("census-production-bodies", censusBodies)

	// --- support ledger (typed Kind; spelling carries no authority) ---
	supportAll := multiset{}
	supportByKind := map[string]multiset{"body": {}, "initializer": {}, "declaration": {}}
	supportUnimplementedBodies := multiset{}
	supportUnimplementedInits := multiset{}
	kindUnknown := []Delta{}
	for _, s := range generated.Support {
		supportAll.add(s.ID)
		kind, known := supportByKind[s.Kind]
		if !known {
			kindUnknown = append(kindUnknown, Delta{ID: s.ID,
				Disposition: "support record with unknown kind " + s.Kind, Defect: true})
			continue
		}
		kind.add(s.ID)
		if string(s.State) == "unimplemented" {
			switch s.Kind {
			case "body":
				supportUnimplementedBodies.add(s.ID)
			case "initializer":
				supportUnimplementedInits.add(s.ID)
			}
		}
	}
	out.recordDuplicates("support-records", supportAll)

	// --- emission ledger (typed events; multiplicity preserved) ---
	bodyEmissions := multiset{}
	bodyPlaceholderEmissions := multiset{}
	initEmissions := multiset{}
	initPlaceholderEmissions := multiset{}
	for _, event := range generated.Emissions {
		switch event.Kind {
		case emit.EmissionBody:
			bodyEmissions.add(event.ID)
		case emit.EmissionBodyPlaceholder:
			bodyPlaceholderEmissions.add(event.ID)
		case emit.EmissionInitializer:
			initEmissions.add(event.ID)
		case emit.EmissionInitializerPlaceholder:
			initPlaceholderEmissions.add(event.ID)
		}
	}
	for id, count := range bodyPlaceholderEmissions {
		if count > 1 {
			out.VariantReemissions = append(out.VariantReemissions, VariantReemission{ID: id, CoreCopies: count})
		}
	}
	sort.Slice(out.VariantReemissions, func(i, j int) bool { return out.VariantReemissions[i].ID < out.VariantReemissions[j].ID })

	// --- function-literal ledger ---
	funcLits := multiset{}
	for _, lit := range generated.FuncLits {
		funcLits.add(lit.ID)
	}
	out.recordDuplicates("function-literals", funcLits)

	// --- extern symbol ledger (typed records from the stub printer) ---
	externSymbols := multiset{}
	externThrowing := multiset{}
	externDirect := multiset{}
	externObligations := multiset{}
	supportDefinitionCopies := multiset{}
	for _, record := range generated.ExternSymbols {
		qualified := record.Module + "::" + record.Symbol
		externSymbols.add(qualified)
		if record.Obligation == "" {
			externDirect.add(qualified)
			// Cross-module copies of one support definition (union
			// equality/key encoders re-emitted per module) are the
			// measured alias-duplication class: typed evidence here,
			// deleted by the interface-ownership wave.
			supportDefinitionCopies.add(record.Symbol)
		} else {
			externThrowing.add(qualified)
			externObligations.add(record.Obligation)
		}
	}
	out.recordDuplicates("extern-symbols", externSymbols)
	out.recordDuplicates("extern-obligations", externObligations)
	out.recordDuplicates("extern-support-definition-copies", supportDefinitionCopies)

	// --- extern obligation contract: stub proofs (typed records whose
	// generated file the ownership ledger marks external) ---
	externProofObligations := multiset{}
	proofIDs := multiset{}
	noOutput := 0
	effectOnly := 0
	proofsWithLoweredHash := multiset{}
	for _, p := range generated.Proofs {
		proofIDs.add(p.ID)
		if p.NoOutput {
			noOutput++
		}
		if p.EffectOnly {
			effectOnly++
		}
		if p.LoweredHash != "" {
			proofsWithLoweredHash.add(p.ID)
		}
		if generated.Ownership[p.GeneratedFile] == "generated-external-contracts" {
			externProofObligations.add(p.ID)
		}
	}
	out.recordDuplicates("proofs", proofIDs)

	// --- implementation artifacts (ADR-0010 one-to-one ledger) ---
	implementationArtifacts := multiset{}
	for _, artifact := range generated.ImplementationArtifacts {
		implementationArtifacts.add(artifact.ImplementationID)
	}
	out.recordDuplicates("implementation-artifacts", implementationArtifacts)

	// --- module dispositions (typed; cascade-blocked packages carry
	// their typed reason in NotMaterialized) ---
	dispositionByPackage := multiset{}
	dispositionStates := map[string]string{}
	for _, d := range generated.ModuleDispositions {
		dispositionByPackage.add(d.Package)
		dispositionStates[d.Package] = d.State
	}
	for pkg, reason := range generated.NotMaterialized {
		if dispositionByPackage[pkg] == 0 {
			dispositionByPackage.add(pkg)
			dispositionStates[pkg] = "not-materialized"
			_ = reason
		}
	}
	out.recordDuplicates("module-dispositions", dispositionByPackage)

	// --- file ownership ledger (typed map; content never scanned) ---
	coreModules := 0
	externModules := 0
	artifactFiles := 0
	for path := range generated.Files {
		switch generated.Ownership[path] {
		case "generated-core":
			coreModules++
		case "generated-external-contracts":
			externModules++
		case "analysis-body":
			artifactFiles++
		}
	}

	den := func(name string, count int, definition string) {
		out.Denominators = append(out.Denominators, Denominator{Name: name, Count: count, Definition: definition})
	}
	den("census-production-bodies", censusBodies.total(), "census production func/method declarations with a body")
	den("census-production-bodiless-funcs", censusBodiless, "census production func/method declarations without a body")
	den("census-owned-production-packages", censusPackages.total(), "distinct packages of production declarations")
	den("support-records-total", supportAll.total(), "all support ledger records")
	den("support-body-records", supportByKind["body"].total(), "support records of kind body")
	den("support-initializer-records", supportByKind["initializer"].total(), "support records of kind initializer")
	den("support-declaration-records", supportByKind["declaration"].total(), "support records of kind declaration (declaration-level blockers)")
	den("support-unimplemented-bodies", supportUnimplementedBodies.total(), "body support records in state unimplemented")
	den("support-unimplemented-initializers", supportUnimplementedInits.total(), "initializer support records in state unimplemented")
	den("function-literals", funcLits.total(), "function-literal ledger records")
	den("proofs", proofIDs.total(), "proof records")
	den("proofs-no-output", noOutput, "proofs with an exact no-output lowering")
	den("proofs-effect-only", effectOnly, "proofs retained as ordered effects without a symbol")
	den("emission-bodies", bodyEmissions.total(), "body emission events")
	den("emission-body-placeholders", bodyPlaceholderEmissions.total(), "body-placeholder emission events")
	den("emission-initializers", initEmissions.total(), "initializer emission events")
	den("emission-initializer-placeholders", initPlaceholderEmissions.total(), "initializer-placeholder emission events")
	den("core-modules", coreModules, "emitted generated-core module files")
	den("extern-modules", externModules, "emitted external-contract module files")
	den("analysis-artifacts", artifactFiles, "per-body analysis artifact files")
	den("extern-symbols", externSymbols.total(), "exported symbols recorded by external-contract modules")
	den("extern-throwing-symbols", externThrowing.total(), "extern symbols whose stub body throws an obligation")
	den("extern-direct-symbols", externDirect.total(), "extern symbols implemented directly (support definitions)")
	den("extern-obligations", externObligations.total(), "obligation identities thrown by stub bodies")
	den("extern-obligation-proofs", externProofObligations.total(), "proof records whose generated file is an external-contract module")
	den("module-dispositions", dispositionByPackage.total(), "typed per-package module dispositions")
	den("implementation-artifacts", implementationArtifacts.total(), "per-implementation artifact records (ADR-0010)")

	// Join: census bodies vs body support records.
	out.Joins = append(out.Joins, joinSets("bodies-vs-body-support",
		"census-production-bodies", "support-body-records", censusBodies, supportByKind["body"],
		"census body without a support record", "body support record without a census production body", true, true))
	// Join: unimplemented body support vs body-placeholder emissions in
	// materialized packages. A placeholder event without an unimplemented
	// support record is untracked emission; extra placeholder copies of
	// one record are the variant re-emissions (typed above) and stay
	// visible here as surplus.
	placeholderJoin := joinSets("unimplemented-bodies-vs-placeholder-emissions",
		"support-unimplemented-bodies", "emission-body-placeholders", supportUnimplementedBodies, bodyPlaceholderEmissions,
		"unimplemented body without a placeholder emission (package not materialized or missing event)",
		"body-placeholder emission without an unimplemented support record", false, true)
	// Surplus placeholder copies whose SOURCE identity has an
	// unimplemented record are family-variant implementations (distinct
	// ADR-0010 keys, listed in variantReemissions): explained evidence,
	// not unknown-emission defects. Only a placeholder for an identity
	// with NO record at all stays a defect.
	for i, delta := range placeholderJoin.OnlyRight {
		if supportUnimplementedBodies[delta.ID] > 0 {
			placeholderJoin.OnlyRight[i].Defect = false
			placeholderJoin.OnlyRight[i].Disposition = "family-variant implementation copy of a recorded unimplemented body (see variantReemissions)"
		}
	}
	out.Joins = append(out.Joins, placeholderJoin)
	// Join: unimplemented initializer support vs initializer-placeholder
	// emissions.
	out.Joins = append(out.Joins, joinSets("unimplemented-initializers-vs-placeholder-emissions",
		"support-unimplemented-initializers", "emission-initializer-placeholders", supportUnimplementedInits, initPlaceholderEmissions,
		"unimplemented initializer without a placeholder emission (package not materialized or missing event)",
		"initializer-placeholder emission without an unimplemented support record", false, true))
	// Join: extern obligations thrown vs extern stub proof records.
	out.Joins = append(out.Joins, joinSets("extern-obligations-vs-stub-proofs",
		"extern-obligations", "extern-obligation-proofs", externObligations, externProofObligations,
		"thrown obligation without a stub proof record", "stub proof record whose obligation is never thrown (direct implementation or missing event)", true, false))
	// Join: analysis artifacts (proof lowered hashes) vs body emissions.
	bodyEmissionAll := multiset{}
	for id, c := range bodyEmissions {
		bodyEmissionAll[id] += c
	}
	for id, c := range bodyPlaceholderEmissions {
		bodyEmissionAll[id] += c
	}
	out.Joins = append(out.Joins, joinSets("lowered-proofs-vs-body-emissions",
		"proofs-with-lowered-hash", "emission-bodies-and-placeholders", proofsWithLoweredHash, bodyEmissionAll,
		"proof with a lowered hash but no emission event", "body emission event without a lowered-hash proof (placeholder, effect slot, or artifact-identity collision)", true, false))
	// Join: census packages vs typed module dispositions.
	out.Joins = append(out.Joins, joinSets("packages-vs-module-dispositions",
		"census-owned-production-packages", "module-dispositions", censusPackages, dispositionByPackage,
		"census package without a typed module disposition", "module disposition without a census package", true, true))

	// Support records with an unknown kind are defects surfaced on the
	// kind join directly.
	if len(kindUnknown) > 0 {
		j := Join{Name: "support-kind-totality", Left: "support-records-total", Right: "typed-kinds"}
		j.OnlyLeft = kindUnknown
		sortDeltas(j.OnlyLeft)
		out.Joins = append(out.Joins, j)
	}

	// Package pipeline states from the typed disposition + withholding
	// ledgers.
	packages := make([]string, 0, len(censusPackageSet))
	for pkg := range censusPackageSet {
		packages = append(packages, pkg)
	}
	sort.Strings(packages)
	for _, pkg := range packages {
		state := dispositionStates[pkg]
		why, withheld := generated.Withheld[pkg]
		out.Packages = append(out.Packages, PackageState{
			Path:         pkg,
			Materialized: state == "emitted-runtime" || state == "no-runtime-output",
			Published:    state == "emitted-runtime" && !withheld,
			WithheldWhy:  why,
		})
	}
	sort.Slice(out.Duplicates, func(i, j int) bool {
		if out.Duplicates[i].Ledger != out.Duplicates[j].Ledger {
			return out.Duplicates[i].Ledger < out.Duplicates[j].Ledger
		}
		return out.Duplicates[i].ID < out.Duplicates[j].ID
	})
	return out
}

func sortDeltas(deltas []Delta) {
	sort.Slice(deltas, func(i, j int) bool { return deltas[i].ID < deltas[j].ID })
}

// BuildWithHistory extends Build with mechanical joins against a prior
// baseline's identity sets.
func BuildWithHistory(head string, run *census.Result, generated *translate.Generated, history *History, inputDigests map[string]string) *Report {
	out := Build(head, run, generated)
	out.InputDigests = inputDigests
	if history == nil {
		return out
	}
	current := multiset{}
	for _, decl := range run.Report.Declarations {
		if decl.Scope == "production" && (decl.Kind == "func" || decl.Kind == "method") && decl.HasBody {
			current.add(decl.ID)
		}
	}
	prior := multiset{}
	for _, id := range history.PriorBodies {
		prior.add(id)
	}
	out.Joins = append(out.Joins, joinSets("bodies-vs-prior-baseline",
		"prior-production-bodies("+history.PriorBodiesFrom+")", "census-production-bodies", prior, current,
		"left production scope after the prior baseline (scope change; classify against the profile delta)",
		"entered production scope after the prior baseline (scope change; classify against the profile delta)", false, false))

	published := multiset{}
	for _, pkg := range out.Packages {
		if pkg.Published {
			published.add(pkg.Path)
		}
	}
	priorPublished := multiset{}
	for _, pkg := range history.PriorPublished {
		priorPublished.add(pkg)
	}
	pubJoin := joinSets("published-vs-prior",
		"prior-published("+history.PriorPublishedAt+")", "published-now", priorPublished, published,
		"withheld now", "newly published since the prior baseline", false, false)
	for i, d := range pubJoin.OnlyLeft {
		for _, state := range out.Packages {
			if state.Path == d.ID && state.WithheldWhy != "" {
				pubJoin.OnlyLeft[i].Disposition = "withheld now: " + state.WithheldWhy
			}
		}
	}
	out.Joins = append(out.Joins, pubJoin)
	return out
}

// Render produces the human summary FROM the report — never beside it.
func (r *Report) Render() string {
	var b strings.Builder
	b.WriteString("# Count Reconciliation — " + r.Head + "\n\n")
	b.WriteString("| Denominator | Count | Definition |\n|---|---:|---|\n")
	for _, d := range r.Denominators {
		b.WriteString("| " + d.Name + " | " + itoa(d.Count) + " | " + d.Definition + " |\n")
	}
	for _, j := range r.Joins {
		b.WriteString("\n## Join " + j.Name + " (" + j.Left + " vs " + j.Right + ")\n")
		b.WriteString("both=" + itoa(j.Both) + " onlyLeft=" + itoa(len(j.OnlyLeft)) + " onlyRight=" + itoa(len(j.OnlyRight)) + "\n")
		b.WriteString("leftSetSha256=" + j.LeftSetSha256 + "\nrightSetSha256=" + j.RightSetSha256 + "\n")
		for _, d := range j.OnlyLeft {
			b.WriteString("- L " + d.ID + " → " + d.Disposition + defectTag(d) + "\n")
		}
		for _, d := range j.OnlyRight {
			b.WriteString("- R " + d.ID + " → " + d.Disposition + defectTag(d) + "\n")
		}
	}
	if len(r.Duplicates) > 0 {
		b.WriteString("\n## Duplicate identities (defects)\n")
		for _, dup := range r.Duplicates {
			b.WriteString("- " + dup.Ledger + ": " + dup.ID + " ×" + itoa(dup.Count) + "\n")
		}
	}
	if len(r.VariantReemissions) > 0 {
		b.WriteString("\n## Family-variant placeholder re-emissions\n")
		for _, v := range r.VariantReemissions {
			b.WriteString("- " + v.ID + " ×" + itoa(v.CoreCopies) + " core copies\n")
		}
	}
	if len(r.InputDigests) > 0 {
		b.WriteString("\n## Input digests\n")
		names := make([]string, 0, len(r.InputDigests))
		for name := range r.InputDigests {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			b.WriteString("- " + name + ": " + r.InputDigests[name] + "\n")
		}
	}
	return b.String()
}

func defectTag(d Delta) string {
	if d.Defect {
		return " [DEFECT]"
	}
	return ""
}

// Defects returns the typed gate-facing verdict: every delta marked
// Defect plus every duplicate identity.
func (r *Report) Defects() []Delta {
	var out []Delta
	for _, j := range r.Joins {
		for _, d := range append(append([]Delta{}, j.OnlyLeft...), j.OnlyRight...) {
			if d.Defect {
				out = append(out, d)
			}
		}
	}
	for _, dup := range r.Duplicates {
		out = append(out, Delta{ID: dup.ID,
			Disposition: "duplicate identity in ledger " + dup.Ledger + " (×" + itoa(dup.Count) + ")", Defect: true})
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
