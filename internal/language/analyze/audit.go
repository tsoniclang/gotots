// Catalog-coverage audit: a versioned, fingerprinted gate artifact proving
// the construct catalog is total over the non-full-semantic closure (standard
// library and other contract-depth source) for one exact toolchain contract,
// and carrying the provider unit manifest ordinary compilation consumes. The
// audit STREAMS: each file's selected bytes are read, digest-verified,
// parsed, classified fail-closed, counted, and dropped — it never enlarges a
// compilation's retained semantic model. Ordinary compilation verifies and
// consumes the artifact; it does not rescan provider interiors.
package analyze

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/source"
)

// AuditArtifactVersion is the audit artifact schema version.
const AuditArtifactVersion = 4

// ManifestUnit is one provider-owned unit's manifest record: identity
// components, selected-span content address, display name, and typed cgo
// evidence, captured from the production census. It is a derived census
// projection of the definition/reference graph, never the topology authority.
type ManifestUnit struct {
	Unit       string `json:"unit"` // canonical SourceUnitID string
	Kind       uint8  `json:"kind"` // pinned identity.UnitKind value
	Start      int    `json:"start"`
	End        int    `json:"end"`
	Name       string `json:"name,omitempty"` // display only
	Hash       string `json:"hash"`           // SourceSpanHash hex
	CDependent bool   `json:"cDependent,omitempty"`
}

// ManifestDefinition is one provider unit's definition in the artifact graph:
// its canonical identity and kind. Definitions are the census projection's
// authority — the flat unit list derives from them.
type ManifestDefinition struct {
	Unit string `json:"unit"`
	Kind uint8  `json:"kind"`
}

// ManifestReference is one nested implementation reference in the artifact
// graph: the parent owner, the enclosing occurrence, the child unit, the exact
// grammatical edge, the source anchor, and the source ordinal.
type ManifestReference struct {
	Parent  string `json:"parent"`  // "decl:<fileID>" or "unit:<unitID>"
	Occ     string `json:"occ"`     // parent occurrence identity
	Child   string `json:"child"`   // child unit identity
	Edge    string `json:"edge"`    // grammatical edge
	Ordinal int    `json:"ordinal"` // source order within the parent region
	Anchor  string `json:"anchor"`  // child root span identity
}

// AuditFile is one audited file's content-addressed record: the identity is
// bound to the selected-byte digest captured during resolution, with exact
// per-file evidence and the complete provider unit manifest (top-level and
// interior units).
type AuditFile struct {
	File        string               `json:"file"`   // canonical FileID string
	Digest      string               `json:"digest"` // sha256 of selected bytes
	Occurrences int                  `json:"occurrences"`
	Directives  int                  `json:"directives"`
	Units       []ManifestUnit       `json:"units,omitempty"`
	Definitions []ManifestDefinition `json:"definitions,omitempty"`
	References  []ManifestReference  `json:"references,omitempty"`
}

// AuditMeta binds the artifact to its complete production context: the
// selected toolchain binary and its resolved build-selection environment, the
// catalog, and the provider contract. BuildFlags and OverlayDigest record the
// production request's own selection inputs; they are sealed provenance,
// while cross-request consumption exactness rests on the per-file
// selected-byte digests.
type AuditMeta struct {
	ToolchainVersion      string `json:"toolchainVersion"`
	ToolchainBinaryDigest string `json:"toolchainBinaryDigest"`
	GOOS                  string `json:"goos"`
	GOARCH                string `json:"goarch"`
	Experiments           string `json:"experiments"`
	GoFlags               string `json:"goflags"`
	CgoEnabled            string `json:"cgoEnabled"`
	CatalogVersion        string `json:"catalogVersion"`
	CatalogDigest         string `json:"catalogDigest"`
	ContractID            string `json:"contractID"`
	ContractFingerprint   string `json:"contractFingerprint"`
	BuildFlags            string `json:"buildFlags"`
	OverlayDigest         string `json:"overlayDigest"`
}

// contextEqual compares the consumption-binding context: toolchain identity,
// the complete build-selection environment including build flags, catalog
// structure, and provider contract. The artifact is produced per build
// configuration (spec: toolchain audit runs per selected contract and build
// configuration). Only OverlayDigest is production provenance: an overlay
// that actually changes selected bytes is caught exactly by the per-file
// digest joins, while an unrelated module-file overlay never invalidates the
// toolchain artifact.
func contextEqual(a, b AuditMeta) bool {
	a.OverlayDigest, b.OverlayDigest = "", ""
	return a == b
}

// AuditArtifact is the content-addressed catalog-coverage and unit-manifest
// artifact: exact per-file evidence bound to selected-byte digests and the
// complete production context, sealed by a canonical artifact digest.
// Consumers exact-join everything; counts are never trusted without content
// binding.
type AuditArtifact struct {
	SchemaVersion  int         `json:"schemaVersion"`
	Meta           AuditMeta   `json:"meta"`
	Files          []AuditFile `json:"files"`
	Occurrences    int         `json:"occurrences"`
	Directives     int         `json:"directives"`
	ArtifactDigest string      `json:"artifactDigest"`
}

// canonicalDigest computes the artifact's canonical self-digest over the
// complete content (meta, every record and unit, aggregates).
func (a *AuditArtifact) canonicalDigest() string {
	var b strings.Builder
	fmt.Fprintf(&b, "v%d|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%d|%d",
		a.SchemaVersion, a.Meta.ToolchainVersion, a.Meta.ToolchainBinaryDigest,
		a.Meta.GOOS, a.Meta.GOARCH, a.Meta.Experiments, a.Meta.GoFlags, a.Meta.CgoEnabled,
		a.Meta.CatalogVersion, a.Meta.CatalogDigest, a.Meta.ContractID,
		a.Meta.ContractFingerprint, a.Meta.BuildFlags, a.Meta.OverlayDigest,
		a.Occurrences, a.Directives)
	for _, file := range a.Files {
		fmt.Fprintf(&b, "|%s#%s#%d#%d", file.File, file.Digest, file.Occurrences, file.Directives)
		for _, unit := range file.Units {
			fmt.Fprintf(&b, "~%s~%d~%d~%d~%s~%s~%t", unit.Unit, unit.Kind, unit.Start, unit.End, unit.Name, unit.Hash, unit.CDependent)
		}
		for _, def := range file.Definitions {
			fmt.Fprintf(&b, "@%s~%d", def.Unit, def.Kind)
		}
		for _, ref := range file.References {
			fmt.Fprintf(&b, ">%s~%s~%s~%s~%d~%s", ref.Parent, ref.Occ, ref.Child, ref.Edge, ref.Ordinal, ref.Anchor)
		}
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(b.String())))
}

// AuditError is the typed failure of an audit run or artifact verification.
type AuditError struct{ Reason string }

func (e *AuditError) Error() string { return "GOTOTS_CATALOG_AUDIT: " + e.Reason }

// AuditCatalog streams the catalog-coverage audit over every non-full file of
// the finalized universe (declaration-contract and external-boundary source,
// including cgo originals). The universe must come from the manifest-producing
// acquisition policy, so every file's census — including provider interiors —
// is complete and enters the manifest. Any unknown construct or unknown go
// directive fails closed with its identity.
func AuditCatalog(ws *source.Workspace, meta AuditMeta, overlay map[string][]byte, ordinary source.AcquisitionPolicy, graph map[string]FileGraph) (*AuditArtifact, error) {
	meta = boundMeta(ws, meta)
	artifact := &AuditArtifact{
		SchemaVersion: AuditArtifactVersion,
		Meta:          meta,
	}
	for _, pkg := range ws.Packages() {
		mode, err := ordinary.ModeFor(pkg.ID())
		if err != nil {
			return nil, &AuditError{Reason: err.Error()}
		}
		for _, file := range pkg.Files() {
			if !auditMember(file, mode) {
				continue
			}
			record, err := auditFile(file, overlay)
			if err != nil {
				return nil, err
			}
			embedGraph(&record, graph[file.ID().String()])
			artifact.Files = append(artifact.Files, record)
			artifact.Occurrences += record.Occurrences
			artifact.Directives += record.Directives
		}
	}
	sort.Slice(artifact.Files, func(i, j int) bool { return artifact.Files[i].File < artifact.Files[j].File })
	artifact.seal()
	return artifact, nil
}

// embedGraph serializes one file's definition/reference graph into its audit
// record. Definitions are the topology authority; the flat unit list is the
// derived census projection.
func embedGraph(record *AuditFile, g FileGraph) {
	for _, def := range g.Definitions {
		record.Definitions = append(record.Definitions, ManifestDefinition{
			Unit: def.Unit().String(), Kind: uint8(def.Kind()),
		})
	}
	for _, ref := range g.References {
		record.References = append(record.References, ManifestReference{
			Parent:  ref.Parent().String(),
			Occ:     ref.ParentOccurrence().String(),
			Child:   ref.Child().String(),
			Edge:    ref.Edge().String(),
			Ordinal: ref.Ordinal(),
			Anchor:  ref.Anchor().String(),
		})
	}
}

// seal stamps the canonical self-digest. Sealing is producer-owned: only the
// audit producer seals, and consumers bind to the certified digest the gate
// run reported — a resealed artifact never gains authority from its own seal.
func (a *AuditArtifact) seal() { a.ArtifactDigest = a.canonicalDigest() }

// boundMeta stamps the workspace-resolved toolchain and catalog facts into
// the metadata.
func boundMeta(ws *source.Workspace, meta AuditMeta) AuditMeta {
	toolchain := ws.Toolchain()
	meta.ToolchainVersion = toolchain.Version()
	meta.ToolchainBinaryDigest = toolchain.BinaryDigest()
	meta.GOOS = toolchain.GOOS()
	meta.GOARCH = toolchain.GOARCH()
	meta.Experiments = toolchain.Experiments()
	meta.GoFlags = toolchain.GoFlags()
	meta.CgoEnabled = toolchain.CgoEnabled()
	meta.CatalogVersion = catalog.SelectedGoVersion
	meta.CatalogDigest = catalog.StructureDigest()
	return meta
}

// auditMember is the single membership rule shared by the audit producer and
// the artifact verifier: every file of a provider-owned (manifest-mode)
// package, and every non-full file (declaration-contract, external-boundary,
// and mixed files that carry at least one non-full unit) is audited for
// catalog coverage. An all-full file with units is part of the application
// model, not the audit closure.
func auditMember(file *source.File, mode source.CensusMode) bool {
	if mode == source.CensusManifest {
		return true
	}
	hasFull := false
	for _, unit := range file.Units() {
		if unit.Depth() == source.DepthFullSemantic {
			hasFull = true
		} else {
			return true // a non-full unit places the file in the audit closure
		}
	}
	return !hasFull
}

// auditFile reads one file's exact selected bytes, verifies them against the
// resolution-captured digest, parses and classifies every node and directive
// fail-closed, counts, records the file's complete unit manifest, and drops
// the tree.
func auditFile(file *source.File, overlay map[string][]byte) (AuditFile, error) {
	record := AuditFile{File: file.ID().String(), Digest: file.ByteDigest().String()}
	for _, unit := range file.Units() {
		record.Units = append(record.Units, ManifestUnit{
			Unit:       unit.ID().String(),
			Kind:       uint8(unit.Kind()),
			Start:      unit.Span().Start.Offset,
			End:        unit.Span().End.Offset,
			Name:       unit.Name(),
			Hash:       unit.Hash().String(),
			CDependent: unit.CDependent(),
		})
	}
	raw, err := source.SelectedFileBytes(file.Path(), overlay)
	if err != nil {
		return record, &AuditError{Reason: file.ID().String() + ": selected bytes unreadable: " + err.Error()}
	}
	if digest := fmt.Sprintf("%x", sha256.Sum256(raw)); digest != file.ByteDigest().String() {
		return record, &AuditError{Reason: file.ID().String() + ": selected bytes drifted since resolution: " +
			digest + " vs " + file.ByteDigest().String()}
	}
	fset := token.NewFileSet()
	syntax, err := parser.ParseFile(fset, file.Path(), raw, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return record, &AuditError{Reason: file.ID().String() + ": unparsable: " + err.Error()}
	}
	b := &builder{fset: fset, file: file.ID()}
	var classifyErr error
	ast.Inspect(syntax, func(n ast.Node) bool {
		if n == nil || classifyErr != nil {
			return false
		}
		if _, err := Classify(n); err != nil {
			classifyErr = err
			return false
		}
		record.Occurrences++
		return true
	})
	if classifyErr != nil {
		return record, classifyErr
	}
	for _, group := range syntax.Comments {
		for _, comment := range group.List {
			directiveRecord, isDirective, err := b.directiveOf(comment)
			if err != nil {
				return record, err
			}
			if isDirective {
				_ = directiveRecord
				record.Directives++
			}
		}
	}
	return record, nil
}

// WriteAuditArtifact writes the artifact canonically.
func WriteAuditArtifact(artifact *AuditArtifact, path string) error {
	encoded, err := json.MarshalIndent(artifact, "", " ")
	if err != nil {
		return &AuditError{Reason: "artifact encoding: " + err.Error()}
	}
	return os.WriteFile(path, append(encoded, '\n'), 0o644)
}

// DecodeAuditArtifactBound reads one stored artifact for ordinary
// consumption. Authority is EXTERNAL: the recomputed content digest must
// equal the certified digest the request binds (the gate run's attested
// output) — before and independent of the artifact's own seal, so a resealed
// tamper (omitted interior, flipped evidence) never enters compilation.
func DecodeAuditArtifactBound(path, certifiedDigest string) (*AuditArtifact, error) {
	if certifiedDigest == "" {
		return nil, &AuditError{Reason: "the request supplies a manifest artifact without its certified digest"}
	}
	artifact, err := decodeAuditArtifact(path, certifiedDigest)
	if err != nil {
		return nil, err
	}
	return artifact, nil
}

// DecodeAuditArtifact reads one stored artifact for gate verification, which
// re-derives the complete universe independently instead of trusting a
// certified digest. Every artifact-internal invariant is verified: schema
// version, the sealed canonical digest, duplicate-record rejection, and
// aggregate/sum consistency.
func DecodeAuditArtifact(path string) (*AuditArtifact, error) {
	return decodeAuditArtifact(path, "")
}

func decodeAuditArtifact(path, certifiedDigest string) (*AuditArtifact, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, &AuditError{Reason: "artifact unreadable: " + err.Error()}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	var artifact AuditArtifact
	if err := decoder.Decode(&artifact); err != nil {
		return nil, &AuditError{Reason: "artifact undecodable: " + err.Error()}
	}
	if artifact.SchemaVersion != AuditArtifactVersion {
		return nil, &AuditError{Reason: "artifact schema version mismatch"}
	}
	content := artifact.canonicalDigest()
	if certifiedDigest != "" && content != certifiedDigest {
		return nil, &AuditError{Reason: "artifact content digest " + content +
			" does not match the request-bound certified digest " + certifiedDigest}
	}
	if artifact.ArtifactDigest != content {
		return nil, &AuditError{Reason: "artifact digest mismatch: content was altered after production"}
	}
	sumOccurrences, sumDirectives := 0, 0
	seen := map[string]bool{}
	for _, file := range artifact.Files {
		if seen[file.File] {
			return nil, &AuditError{Reason: "duplicate audit record " + file.File}
		}
		seen[file.File] = true
		sumOccurrences += file.Occurrences
		sumDirectives += file.Directives
	}
	if sumOccurrences != artifact.Occurrences || sumDirectives != artifact.Directives {
		return nil, &AuditError{Reason: fmt.Sprintf("aggregate counts %d/%d diverge from per-file sums %d/%d",
			artifact.Occurrences, artifact.Directives, sumOccurrences, sumDirectives)}
	}
	return &artifact, nil
}

// VerifyAuditContext exact-joins a decoded artifact's production context
// against the current workspace and request context.
func VerifyAuditContext(ws *source.Workspace, meta AuditMeta, artifact *AuditArtifact) error {
	meta = boundMeta(ws, meta)
	if !contextEqual(artifact.Meta, meta) {
		return &AuditError{Reason: fmt.Sprintf("artifact context %+v does not match request context %+v", artifact.Meta, meta)}
	}
	return nil
}

// VerifyAuditCoverage exact-joins a decoded artifact's file membership
// against the current universe: the audited set and the universe's non-full
// set must be identical, with per-file selected-byte digests equal. Both
// one-sided lists and the drift list are reported. This is the gate check run
// against the artifact's own production patterns; ordinary consumption of a
// wider artifact joins through the census instead.
func VerifyAuditCoverage(ws *source.Workspace, artifact *AuditArtifact, ordinary source.AcquisitionPolicy) error {
	audited := map[string]AuditFile{}
	for _, file := range artifact.Files {
		audited[file.File] = file
	}
	var missing, extra, drifted []string
	current := map[string]bool{}
	for _, pkg := range ws.Packages() {
		mode, err := ordinary.ModeFor(pkg.ID())
		if err != nil {
			return &AuditError{Reason: err.Error()}
		}
		for _, file := range pkg.Files() {
			if !auditMember(file, mode) {
				continue
			}
			id := file.ID().String()
			current[id] = true
			record, exists := audited[id]
			if !exists {
				missing = append(missing, id)
				continue
			}
			if record.Digest != file.ByteDigest().String() {
				drifted = append(drifted, id)
			}
			drifted = append(drifted, joinUnitManifest(file, record)...)
		}
	}
	for id := range audited {
		if !current[id] {
			extra = append(extra, id)
		}
	}
	if len(missing)+len(extra)+len(drifted) > 0 {
		sort.Strings(missing)
		sort.Strings(extra)
		sort.Strings(drifted)
		return &AuditError{Reason: "audit join failed; unaudited=" + join(missing) +
			" stale=" + join(extra) + " byte-drift=" + join(drifted)}
	}
	return nil
}

// VerifyAuditArtifact is the complete gate verification of a stored artifact
// against one universe: internal invariants, production context, and exact
// bidirectional membership.
func VerifyAuditArtifact(ws *source.Workspace, meta AuditMeta, path string, ordinary source.AcquisitionPolicy) error {
	artifact, err := DecodeAuditArtifact(path)
	if err != nil {
		return err
	}
	if err := VerifyAuditContext(ws, meta, artifact); err != nil {
		return err
	}
	return VerifyAuditCoverage(ws, artifact, ordinary)
}

// joinUnitManifest exact-joins one file's manifest unit records against the
// finalized census, both ways: identity, hash, and cgo evidence. Against a
// recursively censused (gate) universe this proves the manifest total; a
// consumption universe derives its provider interiors from the manifest, so
// the join is exact there by construction.
func joinUnitManifest(file *source.File, record AuditFile) []string {
	var bad []string
	recorded := map[string]ManifestUnit{}
	for _, unit := range record.Units {
		if _, dup := recorded[unit.Unit]; dup {
			bad = append(bad, record.File+" duplicate manifest unit "+unit.Unit)
			continue
		}
		recorded[unit.Unit] = unit
	}
	seen := map[string]bool{}
	for _, unit := range file.Units() {
		id := unit.ID().String()
		seen[id] = true
		manifest, exists := recorded[id]
		if !exists {
			bad = append(bad, record.File+" manifest omits unit "+id)
			continue
		}
		if manifest.Hash != unit.Hash().String() || manifest.CDependent != unit.CDependent() {
			bad = append(bad, record.File+" manifest unit "+id+" evidence diverges")
		}
	}
	for id := range recorded {
		if !seen[id] {
			bad = append(bad, record.File+" manifest holds unit "+id+" the census does not")
		}
	}
	return bad
}

func join(values []string) string {
	out := "["
	for i, v := range values {
		if i > 0 {
			out += " "
		}
		out += v
	}
	return out + "]"
}
