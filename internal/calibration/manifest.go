// Package calibration derives the revision-bound fixture manifest
// MECHANICALLY: declaration byte spans and hashes from the pinned Go
// source, baseline artifact hashes from the generation's analysis
// artifacts. Nothing is copied from prose; a missing, moved, or
// hash-mismatched fixture fails closed.
package calibration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

const SchemaVersion = 1

// Seed is one reviewed fixture selection (identity and anchor only —
// spans and hashes are derived, never declared).
type Seed struct {
	FixtureID        string   `json:"fixtureId"`
	Origin           string   `json:"origin"` // tsgo | neutral-oracle
	SourceIdentity   string   `json:"sourceIdentity"`
	SourceFile       string   `json:"sourceFile"`
	DeclName         string   `json:"declName"`
	RecvType         string   `json:"recvType,omitempty"`
	DeclLine         int      `json:"declLine,omitempty"` // repeatable decls (init)
	SemanticFamilies []string `json:"semanticFamilies"`
	CandidateVerdict string   `json:"candidateVerdict"` // ordinary | specialized | exception | manual-required
	// CallSites anchor instantiation-sensitive fixtures (class C) to
	// exact corpus call expressions; Derive resolves the spans.
	CallSites []SeedCallSite `json:"callSites,omitempty"`
}

// SeedCallSite is one reviewed call-site anchor (file + line only —
// the exact expression span is derived, never declared).
type SeedCallSite struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

// Seeds is the checked selection document.
type Seeds struct {
	SchemaVersion  int    `json:"schemaVersion"`
	SourceRevision string `json:"sourceRevision"`
	Seeds          []Seed `json:"seeds"`
}

// Fixture is one derived manifest entry.
type Fixture struct {
	FixtureID               string   `json:"fixtureId"`
	Origin                  string   `json:"origin"`
	SourceRevision          string   `json:"sourceRevision"`
	SourceIdentity          string   `json:"sourceIdentity"`
	SourceFile              string   `json:"sourceFile"`
	StartByte               int      `json:"startByte"`
	EndByte                 int      `json:"endByte"`
	SourceBodySha256        string   `json:"sourceBodySha256"`
	SemanticFamilies        []string `json:"semanticFamilies"`
	BaselineImplementations []string `json:"baselineImplementationIds,omitempty"`
	BaselineArtifactSha256  []string `json:"baselineArtifactSha256,omitempty"`
	BaselineArtifactBytes   int      `json:"baselineArtifactBytes,omitempty"`
	CandidateVerdict        string   `json:"candidateVerdict"`
	// CallSites are the resolved instantiation records for class-C
	// fixtures: exact call-expression spans and hashes plus the
	// enclosing declaration.
	CallSites           []CallSiteRecord `json:"callSites,omitempty"`
	HandPortOwner       string           `json:"handPortOwner"`
	IndependentReviewer string           `json:"independentReviewer"`
	GoBytes             int              `json:"goBytes"`
}

// Manifest is the derived, revision-bound calibration manifest.
type Manifest struct {
	SchemaVersion  int       `json:"schemaVersion"`
	SourceRevision string    `json:"sourceRevision"`
	Fixtures       []Fixture `json:"fixtures"`
}

// locateDecl finds the seed's declaration in its file and returns the
// exact byte span [start, end).
func locateDecl(sourcePath string, seed Seed) (start, end int, err error) {
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, sourcePath, nil, parser.ParseComments)
	if err != nil {
		return 0, 0, err
	}
	var matches []*ast.FuncDecl
	for _, decl := range parsed.Decls {
		fn, isFunc := decl.(*ast.FuncDecl)
		if !isFunc || fn.Name.Name != seed.DeclName {
			continue
		}
		if seed.RecvType != "" {
			if fn.Recv == nil || len(fn.Recv.List) == 0 {
				continue
			}
			recv := recvTypeName(fn.Recv.List[0].Type)
			if recv != seed.RecvType {
				continue
			}
		} else if fn.Recv != nil {
			continue
		}
		if seed.DeclLine != 0 && fset.Position(fn.Pos()).Line != seed.DeclLine {
			continue
		}
		matches = append(matches, fn)
	}
	if len(matches) != 1 {
		return 0, 0, fmt.Errorf("fixture %s: %d declarations match %s in %s (exactly one required)",
			seed.FixtureID, len(matches), seed.DeclName, seed.SourceFile)
	}
	fn := matches[0]
	pos := fn.Pos()
	if fn.Doc != nil {
		pos = fn.Doc.Pos()
	}
	return fset.Position(pos).Offset, fset.Position(fn.End()).Offset, nil
}

func recvTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return recvTypeName(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return recvTypeName(t.X)
	case *ast.IndexListExpr:
		return recvTypeName(t.X)
	}
	return ""
}

// Derive builds the manifest from seeds, the pinned corpus, and a
// baseline generation dump. repoDir anchors neutral fixture files.
// CallSiteRecord is one resolved call-site span.
type CallSiteRecord struct {
	File          string `json:"file"`
	Line          int    `json:"line"`
	StartByte     int    `json:"startByte"`
	EndByte       int    `json:"endByte"`
	Sha256        string `json:"sha256"`
	EnclosingDecl string `json:"enclosingDecl"`
}

// resolveCallSite finds the call expression that STARTS on the anchored
// line and records its exact span, hash, and enclosing declaration.
// Zero or several starting calls on that line fail closed: the anchor
// must be unambiguous.
func resolveCallSite(corpusDir string, site SeedCallSite) (CallSiteRecord, error) {
	full := filepath.Join(corpusDir, filepath.FromSlash(site.File))
	data, err := os.ReadFile(full)
	if err != nil {
		return CallSiteRecord{}, err
	}
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, full, data, parser.ParseComments)
	if err != nil {
		return CallSiteRecord{}, err
	}
	var matches []*ast.CallExpr
	var enclosing []string
	ast.Inspect(parsed, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if fset.Position(call.Pos()).Line == site.Line {
			matches = append(matches, call)
			name := ""
			for _, decl := range parsed.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok && fn.Pos() <= call.Pos() && call.End() <= fn.End() {
					name = fn.Name.Name
				}
			}
			enclosing = append(enclosing, name)
		}
		return true
	})
	// Keep only OUTERMOST calls starting on the line (an argument list
	// often contains nested calls on the same line).
	outer := make([]*ast.CallExpr, 0, len(matches))
	outerNames := make([]string, 0, len(matches))
	for i, call := range matches {
		contained := false
		for j, other := range matches {
			if i != j && other.Pos() <= call.Pos() && call.End() <= other.End() {
				contained = true
			}
		}
		if !contained {
			outer = append(outer, call)
			outerNames = append(outerNames, enclosing[i])
		}
	}
	if len(outer) != 1 {
		return CallSiteRecord{}, fmt.Errorf("call-site anchor %s:%d matches %d outermost calls; must be exactly 1", site.File, site.Line, len(outer))
	}
	start := fset.Position(outer[0].Pos()).Offset
	end := fset.Position(outer[0].End()).Offset
	digest := sha256.Sum256(data[start:end])
	return CallSiteRecord{
		File: site.File, Line: site.Line, StartByte: start, EndByte: end,
		Sha256: hex.EncodeToString(digest[:]), EnclosingDecl: outerNames[0],
	}, nil
}

// LedgerArtifact mirrors the dump ledger record (translate owns the
// producer; this is the read-side shape).
type LedgerArtifact struct {
	ImplementationID string `json:"implementationId"`
	SourceID         string `json:"sourceId"`
	Package          string `json:"package"`
	ArtifactPath     string `json:"artifactPath"`
	Sha256           string `json:"sha256"`
}

func loadArtifactLedger(dumpDir string) (map[string][]LedgerArtifact, error) {
	if dumpDir == "" {
		return nil, nil
	}
	data, err := os.ReadFile(filepath.Join(dumpDir, "ledgers", "implementation-artifacts.json"))
	if err != nil {
		return nil, fmt.Errorf("implementation-artifact ledger: %w", err)
	}
	var records []LedgerArtifact
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, err
	}
	out := map[string][]LedgerArtifact{}
	for _, record := range records {
		out[record.SourceID] = append(out[record.SourceID], record)
	}
	return out, nil
}

func Derive(seeds *Seeds, corpusDir, repoDir, dumpDir, owner, reviewer string) (*Manifest, error) {
	artifactLedger, err := loadArtifactLedger(dumpDir)
	if err != nil {
		return nil, err
	}
	out := &Manifest{SchemaVersion: SchemaVersion, SourceRevision: seeds.SourceRevision}
	for _, seed := range seeds.Seeds {
		base := corpusDir
		if seed.Origin == "neutral-oracle" {
			base = repoDir
		}
		sourcePath := filepath.Join(base, filepath.FromSlash(seed.SourceFile))
		start, end, err := locateDecl(sourcePath, seed)
		if err != nil {
			return nil, err
		}
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			return nil, err
		}
		if end > len(data) || start >= end {
			return nil, fmt.Errorf("fixture %s: span [%d,%d) outside file", seed.FixtureID, start, end)
		}
		body := data[start:end]
		digest := sha256.Sum256(body)
		fixture := Fixture{
			FixtureID:           seed.FixtureID,
			Origin:              seed.Origin,
			SourceRevision:      seeds.SourceRevision,
			SourceIdentity:      seed.SourceIdentity,
			SourceFile:          seed.SourceFile,
			StartByte:           start,
			EndByte:             end,
			SourceBodySha256:    hex.EncodeToString(digest[:]),
			SemanticFamilies:    seed.SemanticFamilies,
			CandidateVerdict:    seed.CandidateVerdict,
			HandPortOwner:       owner,
			IndependentReviewer: reviewer,
			GoBytes:             end - start,
		}
		for _, site := range seed.CallSites {
			record, err := resolveCallSite(corpusDir, site)
			if err != nil {
				return nil, fmt.Errorf("fixture %s: %w", seed.FixtureID, err)
			}
			fixture.CallSites = append(fixture.CallSites, record)
		}
		if seed.Origin == "tsgo" && dumpDir != "" {
			// Join through the dump's implementation-artifact ledger:
			// every recorded implementation of this source identity, its
			// recorded hash verified against the artifact file's actual
			// bytes. No identity or path is fabricated here.
			records := artifactLedger[seed.SourceIdentity]
			if len(records) == 0 {
				return nil, fmt.Errorf("fixture %s: no implementation artifact recorded for %s", seed.FixtureID, seed.SourceIdentity)
			}
			for _, record := range records {
				artifact, err := os.ReadFile(filepath.Join(dumpDir, filepath.FromSlash(record.ArtifactPath)))
				if err != nil {
					return nil, fmt.Errorf("fixture %s: ledger artifact: %w", seed.FixtureID, err)
				}
				artifactDigest := hex.EncodeToString(func() []byte { d := sha256.Sum256(artifact); return d[:] }())
				if artifactDigest != record.Sha256 {
					return nil, fmt.Errorf("fixture %s: artifact %s hash %s disagrees with ledger %s",
						seed.FixtureID, record.ArtifactPath, artifactDigest[:12], record.Sha256[:12])
				}
				fixture.BaselineImplementations = append(fixture.BaselineImplementations, record.ImplementationID)
				fixture.BaselineArtifactSha256 = append(fixture.BaselineArtifactSha256, record.Sha256)
				fixture.BaselineArtifactBytes += len(artifact)
			}
		}
		out.Fixtures = append(out.Fixtures, fixture)
	}
	return out, nil
}

// Render produces the review Markdown FROM the manifest.
func (m *Manifest) Render() string {
	var b strings.Builder
	b.WriteString("# Calibration Manifest (rendered)\n\n")
	b.WriteString("| ID | Verdict | Identity | Span | Go bytes | Baseline bytes | Source sha256 (12) |\n|---|---|---|---|---:|---:|---|\n")
	for _, f := range m.Fixtures {
		fmt.Fprintf(&b, "| %s | %s | %s | %s[%d:%d) | %d | %d | %s |\n",
			f.FixtureID, f.CandidateVerdict, f.SourceIdentity, f.SourceFile, f.StartByte, f.EndByte,
			f.GoBytes, f.BaselineArtifactBytes, f.SourceBodySha256[:12])
	}
	return b.String()
}

// Load reads and validates a seeds document.
func Load(path string) (*Seeds, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out Seeds
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	if out.SchemaVersion != 1 {
		return nil, fmt.Errorf("seeds %s: unsupported schema %d", path, out.SchemaVersion)
	}
	seen := map[string]bool{}
	for _, seed := range out.Seeds {
		if seen[seed.FixtureID] {
			return nil, fmt.Errorf("duplicate fixture id %s", seed.FixtureID)
		}
		seen[seed.FixtureID] = true
	}
	return &out, nil
}
