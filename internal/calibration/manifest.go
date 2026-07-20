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
	CandidateVerdict string   `json:"candidateVerdict"` // ordinary | specialized | exception
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
	HandPortOwner           string   `json:"handPortOwner"`
	IndependentReviewer     string   `json:"independentReviewer"`
	GoBytes                 int      `json:"goBytes"`
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

// artifactPath resolves a source identity to its analysis-artifact path
// inside a generation dump (the current sanitizer's naming).
func artifactPath(dumpDir, identity string) string {
	pkg := identity[:strings.Index(identity, "::")]
	sanitized := strings.NewReplacer("/", "_", "::", "__", ":", "_", "@", "_").Replace(identity)
	return filepath.Join(dumpDir, "analysis", "bodies", filepath.FromSlash(pkg), sanitized+".ts.txt")
}

// Derive builds the manifest from seeds, the pinned corpus, and a
// baseline generation dump. repoDir anchors neutral fixture files.
func Derive(seeds *Seeds, corpusDir, repoDir, dumpDir, owner, reviewer string) (*Manifest, error) {
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
		if seed.Origin == "tsgo" && dumpDir != "" {
			path := artifactPath(dumpDir, seed.SourceIdentity)
			artifact, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("fixture %s: baseline artifact: %w", seed.FixtureID, err)
			}
			artifactDigest := sha256.Sum256(artifact)
			fixture.BaselineImplementations = []string{seed.SourceIdentity + "/default"}
			fixture.BaselineArtifactSha256 = []string{hex.EncodeToString(artifactDigest[:])}
			fixture.BaselineArtifactBytes = len(artifact)
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
