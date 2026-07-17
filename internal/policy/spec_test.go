package policy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

type specificationManifest struct {
	SchemaVersion int      `json:"schemaVersion"`
	Authority     string   `json:"authority"`
	Files         []string `json:"files"`
}

func TestCanonicalSpecificationPresent(t *testing.T) {
	root := repositoryRoot(t)
	specDir := filepath.Join(root, "docs", "spec")
	manifest := readSpecificationManifest(t, specDir)

	if manifest.SchemaVersion != 1 {
		t.Errorf("manifest schemaVersion = %d; want 1", manifest.SchemaVersion)
	}
	if manifest.Authority != "docs/spec" {
		t.Errorf("manifest authority = %q; want docs/spec", manifest.Authority)
	}
	if len(manifest.Files) == 0 || manifest.Files[0] != "README.md" {
		t.Fatalf("manifest must begin with README.md")
	}

	seen := map[string]bool{}
	for _, name := range manifest.Files {
		if seen[name] {
			t.Errorf("manifest contains duplicate %s", name)
		}
		seen[name] = true
		if filepath.Base(name) != name || !strings.HasSuffix(name, ".md") {
			t.Errorf("manifest entry %q must be one Markdown basename", name)
		}
	}

	entries, err := os.ReadDir(specDir)
	if err != nil {
		t.Fatal(err)
	}
	actual := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		actual = append(actual, name)
	}
	expected := append([]string(nil), manifest.Files...)
	expected = append(expected, "manifest.json")
	sort.Strings(actual)
	sort.Strings(expected)
	if strings.Join(actual, "\n") != strings.Join(expected, "\n") {
		t.Fatalf("canonical specification inventory mismatch\nactual:\n%s\nexpected:\n%s",
			strings.Join(actual, "\n"), strings.Join(expected, "\n"))
	}

	index := readSpecFile(t, specDir, "README.md")
	for _, name := range manifest.Files {
		content := readSpecFile(t, specDir, name)
		if !strings.HasPrefix(content, "# ") {
			t.Errorf("%s must begin with one H1", name)
		}
		if h1Count(content) != 1 {
			t.Errorf("%s has %d H1 headings; want 1", name, h1Count(content))
		}
		lines := countPhysicalLines([]byte(content))
		if lines > MaxSourceFileLines {
			t.Errorf("%s has %d lines; maximum is %d", name, lines, MaxSourceFileLines)
		}
		if name != "README.md" && strings.Count(index, "`"+name+"`") != 1 {
			t.Errorf("README must list %s exactly once", name)
		}
	}
	readingOrderPattern := regexp.MustCompile(`(?m)^\d+\. \x60([^\x60]+\.md)\x60$`)
	readingOrderMatches := readingOrderPattern.FindAllStringSubmatch(index, -1)
	readingOrder := make([]string, 0, len(readingOrderMatches))
	for _, match := range readingOrderMatches {
		readingOrder = append(readingOrder, match[1])
	}
	wantReadingOrder := manifest.Files[1:]
	if strings.Join(readingOrder, "\n") != strings.Join(wantReadingOrder, "\n") {
		t.Fatalf("README reading order does not match manifest\nREADME:\n%s\nmanifest:\n%s",
			strings.Join(readingOrder, "\n"), strings.Join(wantReadingOrder, "\n"))
	}
}

func TestCanonicalSpecificationReferencesResolve(t *testing.T) {
	root := repositoryRoot(t)
	specDir := filepath.Join(root, "docs", "spec")
	manifest := readSpecificationManifest(t, specDir)
	reference := regexp.MustCompile("`((?:[0-9][^`/]*)|README)\\.md`")
	for _, name := range manifest.Files {
		content := readSpecFile(t, specDir, name)
		for _, match := range reference.FindAllStringSubmatch(content, -1) {
			target := match[1] + ".md"
			if _, err := os.Stat(filepath.Join(specDir, target)); err != nil {
				t.Errorf("%s references missing canonical spec %s", name, target)
			}
		}
	}
}

func TestMaintainedSpecificationReferencesResolve(t *testing.T) {
	root := repositoryRoot(t)
	files, err := MaintainedFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	reference := regexp.MustCompile(`docs/spec/[A-Za-z0-9._-]+\.(?:md|json)`)
	for _, relative := range files {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		content := string(data)
		if strings.HasSuffix(relative, ".go") {
			parsed, parseErr := parser.ParseFile(token.NewFileSet(), relative, data, parser.ParseComments)
			if parseErr != nil {
				t.Fatalf("parse maintained Go source %s: %v", relative, parseErr)
			}
			var comments strings.Builder
			for _, group := range parsed.Comments {
				comments.WriteString(group.Text())
				comments.WriteByte('\n')
			}
			content = comments.String()
		}
		for _, target := range reference.FindAllString(content, -1) {
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(target))); err != nil {
				t.Errorf("%s references missing specification path %s", relative, target)
			}
		}
	}
}

func TestSpecificationContractMarkers(t *testing.T) {
	root := repositoryRoot(t)
	specDir := filepath.Join(root, "docs", "spec")
	requirements := map[string][]string{
		"README.md": {
			"Central Contract",
			"Honest Incompleteness",
			"One Implementation",
		},
		"00-authority-scope.md": {
			"completely outside",
			"Simplest Exact Output",
			"unimplemented",
		},
		"01-input-census-publication.md": {
			"Attest Before Execute",
			"Scope Filter Before Census",
			"Unimplemented Publication Rule",
		},
		"02-compiler-semantic-ir.md": {
			"The body IR describes Go semantics, not JavaScript syntax",
			"Constraint Propagation",
			"Emission consumes this plan without semantic rediscovery",
		},
		"03-declarations-bodies-control.md": {
			"readonly tuple ABI",
			"GoPanic<P>",
			"Unimplemented Bodies",
		},
		"04-types-values-pointers.md": {
			"Numeric Callable ABI",
			"canonical nil storage",
			"storage-root identity",
		},
		"05-collections-strings.md": {
			"Slice Representation Candidates",
			"Map Representation Candidates",
			"String Representation Candidates",
		},
		"06-interfaces-generics-functions.md": {
			"Interface Representation Candidates",
			"Generic Lowering Candidates",
			"statically typed readonly tuple",
		},
		"07-packages-concurrency.md": {
			"Concurrency Lowering Candidates",
			"uninitialized/running/done",
			"Promise-returning TypeScript boundary",
		},
		"08-externals-manual-extensions.md": {
			"Generated Baselines And Body Hashes",
			"Automatic Dependency And Reachability Graph",
			"Reset, Acceptance And Pruning",
			"Mid-Body Seams",
		},
		"09-representation-output.md": {
			"Custom-Mechanism Necessity",
			"simplest ordinary TypeScript",
			"uses no LLM",
		},
		"10-machine-contracts-diagnostics.md": {
			"No LLM Dependency",
			"Necessity Record",
			"GOTOTS_UNIMPLEMENTED_*",
		},
		"11-testing-acceptance.md": {
			"Semantic-Class Oracles",
			"Unimplemented Tests",
			"Compiler Differential",
		},
		"12-performance.md": {
			"Default Budget Matrix",
			"at least forty independent measured samples",
			"Asymptotic complexity",
		},
		"13-governance-upgrades.md": {
			"Diffusion Workflow",
			"600 physical lines",
			"one active feature branch",
		},
	}
	for name, markers := range requirements {
		content := readSpecFile(t, specDir, name)
		for _, marker := range markers {
			if !strings.Contains(content, marker) {
				t.Errorf("%s does not contain required marker %q", name, marker)
			}
		}
	}
}

func TestSpecificationOmitsTargetSpecificAcceptance(t *testing.T) {
	root := repositoryRoot(t)
	specDir := filepath.Join(root, "docs", "spec")
	manifest := readSpecificationManifest(t, specDir)
	for _, name := range manifest.Files {
		content := readSpecFile(t, specDir, name)
		for _, forbidden := range []string{"C#", "Tsonic Rust", "Tsonic CSharp"} {
			if strings.Contains(content, forbidden) {
				t.Errorf("%s contains target-specific acceptance term %q", name, forbidden)
			}
		}
	}
}

func TestRetiredSpecificationDirectoriesAreAbsent(t *testing.T) {
	root := repositoryRoot(t)
	for _, relative := range []string{
		".analysis/scope",
		".analysis/go-semantics-policy",
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(relative))); !os.IsNotExist(err) {
			t.Errorf("%s must not exist; docs/spec is the only governing specification directory", relative)
		}
	}
}

func readSpecificationManifest(t *testing.T, specDir string) specificationManifest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(specDir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var manifest specificationManifest
	if err := decoder.Decode(&manifest); err != nil {
		t.Fatalf("decode specification manifest: %v", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		t.Fatalf("specification manifest has trailing JSON")
	}
	return manifest
}

func readSpecFile(t *testing.T, specDir, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(specDir, name))
	if err != nil {
		t.Fatalf("read canonical specification %s: %v", name, err)
	}
	return string(data)
}

func h1Count(content string) int {
	count := 0
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "# ") {
			count++
		}
	}
	return count
}
