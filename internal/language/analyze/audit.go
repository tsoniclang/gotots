// Catalog-coverage audit: a versioned, fingerprinted gate artifact proving
// the construct catalog is total over the non-full-semantic closure (standard
// library and other contract-depth source) for one exact toolchain contract.
// The audit STREAMS: each file is parsed, classified fail-closed, counted,
// and dropped — it never enlarges a compilation's retained semantic model.
// Ordinary compilation verifies and consumes the artifact; it does not rescan.
package analyze

import (
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
const AuditArtifactVersion = 2

// ManifestUnit is one provider-owned unit's manifest record.
type ManifestUnit struct {
	Unit string `json:"unit"` // canonical SourceUnitID string
	Hash string `json:"hash"` // SourceSpanHash hex
}

// AuditFile is one audited file's content-addressed record: the identity is
// bound to the selected-byte digest captured during resolution, with exact
// per-file evidence and the provider unit manifest.
type AuditFile struct {
	File        string         `json:"file"`   // canonical FileID string
	Digest      string         `json:"digest"` // sha256 of selected bytes
	Occurrences int            `json:"occurrences"`
	Directives  int            `json:"directives"`
	Units       []ManifestUnit `json:"units,omitempty"`
}

// AuditMeta binds the artifact to its complete production context.
type AuditMeta struct {
	ToolchainVersion    string `json:"toolchainVersion"`
	GOOS                string `json:"goos"`
	GOARCH              string `json:"goarch"`
	CatalogVersion      string `json:"catalogVersion"`
	CatalogDigest       string `json:"catalogDigest"`
	ContractID          string `json:"contractID"`
	ContractFingerprint string `json:"contractFingerprint"`
	BuildFlags          string `json:"buildFlags"`
	OverlayDigest       string `json:"overlayDigest"`
}

// AuditArtifact is the content-addressed catalog-coverage artifact: exact
// per-file evidence bound to selected-byte digests and the complete
// production context, sealed by a canonical artifact digest. Consumers
// exact-join everything; counts are never trusted without content binding.
type AuditArtifact struct {
	SchemaVersion  int         `json:"schemaVersion"`
	Meta           AuditMeta   `json:"meta"`
	Files          []AuditFile `json:"files"`
	Occurrences    int         `json:"occurrences"`
	Directives     int         `json:"directives"`
	ArtifactDigest string      `json:"artifactDigest"`
}

// canonicalDigest computes the artifact's canonical self-digest over the
// complete content (meta, every record, aggregates).
func (a *AuditArtifact) canonicalDigest() string {
	var b strings.Builder
	fmt.Fprintf(&b, "v%d|%s|%s|%s|%s|%s|%s|%s|%s|%s|%d|%d",
		a.SchemaVersion, a.Meta.ToolchainVersion, a.Meta.GOOS, a.Meta.GOARCH,
		a.Meta.CatalogVersion, a.Meta.CatalogDigest, a.Meta.ContractID,
		a.Meta.ContractFingerprint, a.Meta.BuildFlags, a.Meta.OverlayDigest,
		a.Occurrences, a.Directives)
	for _, file := range a.Files {
		fmt.Fprintf(&b, "|%s#%s#%d#%d", file.File, file.Digest, file.Occurrences, file.Directives)
		for _, unit := range file.Units {
			fmt.Fprintf(&b, "~%s~%s", unit.Unit, unit.Hash)
		}
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(b.String())))
}

// AuditError is the typed failure of an audit run or artifact verification.
type AuditError struct{ Reason string }

func (e *AuditError) Error() string { return "GOTOTS_CATALOG_AUDIT: " + e.Reason }

// AuditCatalog streams the catalog-coverage audit over every non-full file of
// the finalized universe (declaration-contract and external-boundary source,
// including cgo originals). Any unknown construct or unknown go directive
// fails closed with its identity.
func AuditCatalog(ws *source.Workspace, meta AuditMeta) (*AuditArtifact, error) {
	meta.ToolchainVersion = ws.Toolchain().Version()
	meta.GOOS = ws.Toolchain().GOOS()
	meta.GOARCH = ws.Toolchain().GOARCH()
	meta.CatalogVersion = catalog.SelectedGoVersion
	meta.CatalogDigest = catalog.StructureDigest()
	artifact := &AuditArtifact{
		SchemaVersion: AuditArtifactVersion,
		Meta:          meta,
	}
	for _, pkg := range ws.Packages() {
		for _, file := range pkg.Files() {
			if !auditMember(file) {
				continue
			}
			record, err := auditFile(file)
			if err != nil {
				return nil, err
			}
			artifact.Files = append(artifact.Files, record)
			artifact.Occurrences += record.Occurrences
			artifact.Directives += record.Directives
		}
	}
	sort.Slice(artifact.Files, func(i, j int) bool { return artifact.Files[i].File < artifact.Files[j].File })
	artifact.ArtifactDigest = artifact.canonicalDigest()
	return artifact, nil
}

// auditMember is the single membership rule shared by the audit producer and
// the artifact verifier: every finalized file whose evidence is not full
// syntax (declaration-contract, external-boundary, and mixed files' source
// view) is audited for catalog coverage.
func auditMember(file *source.File) bool {
	_, full := file.Evidence().(source.FullSyntax)
	return !full
}

// auditFile parses one file transiently, classifies every node and directive
// fail-closed, counts, and drops the tree.
func auditFile(file *source.File) (AuditFile, error) {
	record := AuditFile{File: file.ID().String(), Digest: file.ByteDigest().String()}
	for _, unit := range file.Units() {
		record.Units = append(record.Units, ManifestUnit{Unit: unit.ID().String(), Hash: unit.Hash().String()})
	}
	fset := token.NewFileSet()
	syntax, err := parser.ParseFile(fset, file.Path(), nil, parser.ParseComments|parser.SkipObjectResolution)
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

// VerifyAuditArtifact exact-joins a stored artifact against the current
// universe: schema, complete production context (toolchain, GOOS/GOARCH,
// catalog structure, provider contract, build flags, overlays), the sealed
// artifact digest, per-file selected-byte digests (from resolution-captured
// hashes — no rescan), aggregate/sum consistency, duplicate rejection, and
// the exact membership set with both one-sided lists.
func VerifyAuditArtifact(ws *source.Workspace, meta AuditMeta, path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return &AuditError{Reason: "artifact unreadable: " + err.Error()}
	}
	var artifact AuditArtifact
	if err := json.Unmarshal(raw, &artifact); err != nil {
		return &AuditError{Reason: "artifact undecodable: " + err.Error()}
	}
	if artifact.SchemaVersion != AuditArtifactVersion {
		return &AuditError{Reason: "artifact schema version mismatch"}
	}
	if artifact.ArtifactDigest != artifact.canonicalDigest() {
		return &AuditError{Reason: "artifact digest mismatch: content was altered after production"}
	}
	meta.ToolchainVersion = ws.Toolchain().Version()
	meta.GOOS = ws.Toolchain().GOOS()
	meta.GOARCH = ws.Toolchain().GOARCH()
	meta.CatalogVersion = catalog.SelectedGoVersion
	meta.CatalogDigest = catalog.StructureDigest()
	if artifact.Meta != meta {
		return &AuditError{Reason: fmt.Sprintf("artifact context %+v does not match request context %+v", artifact.Meta, meta)}
	}
	sumOccurrences, sumDirectives := 0, 0
	audited := map[string]AuditFile{}
	for _, file := range artifact.Files {
		if _, dup := audited[file.File]; dup {
			return &AuditError{Reason: "duplicate audit record " + file.File}
		}
		audited[file.File] = file
		sumOccurrences += file.Occurrences
		sumDirectives += file.Directives
	}
	if sumOccurrences != artifact.Occurrences || sumDirectives != artifact.Directives {
		return &AuditError{Reason: fmt.Sprintf("aggregate counts %d/%d diverge from per-file sums %d/%d",
			artifact.Occurrences, artifact.Directives, sumOccurrences, sumDirectives)}
	}
	var missing, extra, drifted []string
	current := map[string]bool{}
	for _, pkg := range ws.Packages() {
		for _, file := range pkg.Files() {
			if !auditMember(file) {
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
