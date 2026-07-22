// Catalog-coverage audit: a versioned, fingerprinted gate artifact proving
// the construct catalog is total over the non-full-semantic closure (standard
// library and other contract-depth source) for one exact toolchain contract.
// The audit STREAMS: each file is parsed, classified fail-closed, counted,
// and dropped — it never enlarges a compilation's retained semantic model.
// Ordinary compilation verifies and consumes the artifact; it does not rescan.
package analyze

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"

	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/source"
)

// AuditArtifactVersion is the audit artifact schema version.
const AuditArtifactVersion = 1

// AuditFile is one audited file's exact record.
type AuditFile struct {
	File        string `json:"file"` // canonical FileID string
	Occurrences int    `json:"occurrences"`
	Directives  int    `json:"directives"`
}

// AuditArtifact is the versioned, fingerprinted catalog-coverage artifact.
// Membership is an exact identity set; consumers exact-join it, never bound
// it.
type AuditArtifact struct {
	SchemaVersion    int         `json:"schemaVersion"`
	ToolchainVersion string      `json:"toolchainVersion"`
	CatalogVersion   string      `json:"catalogVersion"`
	Files            []AuditFile `json:"files"`
	Occurrences      int         `json:"occurrences"`
	Directives       int         `json:"directives"`
}

// AuditError is the typed failure of an audit run or artifact verification.
type AuditError struct{ Reason string }

func (e *AuditError) Error() string { return "GOTOTS_CATALOG_AUDIT: " + e.Reason }

// AuditCatalog streams the catalog-coverage audit over every non-full file of
// the finalized universe (declaration-contract and external-boundary source,
// including cgo originals). Any unknown construct or unknown go directive
// fails closed with its identity.
func AuditCatalog(ws *source.Workspace) (*AuditArtifact, error) {
	artifact := &AuditArtifact{
		SchemaVersion:    AuditArtifactVersion,
		ToolchainVersion: ws.Toolchain().Version(),
		CatalogVersion:   catalog.SelectedGoVersion,
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
	record := AuditFile{File: file.ID().String()}
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
// universe: schema version, toolchain fingerprint, catalog version, and the
// exact non-full file identity set — both one-sided lists, never a bound.
func VerifyAuditArtifact(ws *source.Workspace, path string) error {
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
	if artifact.ToolchainVersion != ws.Toolchain().Version() {
		return &AuditError{Reason: "artifact toolchain " + artifact.ToolchainVersion +
			" does not match selected " + ws.Toolchain().Version()}
	}
	if artifact.CatalogVersion != catalog.SelectedGoVersion {
		return &AuditError{Reason: "artifact catalog version mismatch"}
	}
	audited := map[string]bool{}
	for _, file := range artifact.Files {
		audited[file.File] = true
	}
	var missing, extra []string
	current := map[string]bool{}
	for _, pkg := range ws.Packages() {
		for _, file := range pkg.Files() {
			if !auditMember(file) {
				continue
			}
			id := file.ID().String()
			current[id] = true
			if !audited[id] {
				missing = append(missing, id)
			}
		}
	}
	for id := range audited {
		if !current[id] {
			extra = append(extra, id)
		}
	}
	if len(missing)+len(extra) > 0 {
		sort.Strings(missing)
		sort.Strings(extra)
		return &AuditError{Reason: "audit membership join failed; unaudited=" +
			join(missing) + " stale=" + join(extra)}
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
