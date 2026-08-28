package load

import (
	"context"
	"crypto/sha256"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestSourceSnapshotPinsEffectiveFileVersion(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "source.go")
	fileSet := token.NewFileSet()
	syntax, err := parser.ParseFile(fileSet, path, "package snapshot\n", parser.AllErrors)
	if err != nil {
		t.Fatal(err)
	}
	selected := &packages.Package{
		PkgPath:         "example.test/snapshot",
		Name:            "snapshot",
		Dir:             root,
		Fset:            fileSet,
		CompiledGoFiles: []string{path},
		Syntax:          []*ast.File{syntax},
		TypesInfo: &types.Info{FileVersions: map[*ast.File]string{
			syntax: "go1.25",
		}},
	}
	firstFiles, err := checkedSyntaxSnapshot(selected)
	if err != nil {
		t.Fatal(err)
	}
	first := sha256.New()
	writeSnapshotFiles(first, "checked-syntax", firstFiles)
	selected.TypesInfo.FileVersions[syntax] = "go1.26"
	secondFiles, err := checkedSyntaxSnapshot(selected)
	if err != nil {
		t.Fatal(err)
	}
	second := sha256.New()
	writeSnapshotFiles(second, "checked-syntax", secondFiles)
	if string(first.Sum(nil)) == string(second.Sum(nil)) {
		t.Fatal("effective file-version mutation did not change source evidence")
	}
}

func TestSourceSnapshotIsRelocationStable(t *testing.T) {
	left := filepath.Join(t.TempDir(), "left")
	right := filepath.Join(t.TempDir(), "right")
	writeSourceSnapshotFixture(t, left)
	writeSourceSnapshotFixture(t, right)

	leftDigest := loadSourceSnapshotFixture(t, left).SourceDigest()
	rightDigest := loadSourceSnapshotFixture(t, right).SourceDigest()
	if leftDigest == "" || leftDigest != rightDigest {
		t.Fatalf("relocated source digests differ: %q != %q", leftDigest, rightDigest)
	}
}

func TestSourceSnapshotChangesForEverySemanticInputClass(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		path    string
		content string
	}{
		{
			name: "referenced global",
			path: "source.go",
			content: `package snapshot

import (
	_ "embed"
	"example.test/snapshot/dependency"
)

const bias = 2

//go:embed payload.txt
var payload string

func helper(value int) int { return value + dependency.Value }
func Score(value int) int { return helper(value) + bias + len(payload) }
`,
		},
		{
			name: "referenced helper",
			path: "source.go",
			content: `package snapshot

import (
	_ "embed"
	"example.test/snapshot/dependency"
)

const bias = 1

//go:embed payload.txt
var payload string

func helper(value int) int { return value + dependency.Value + 1 }
func Score(value int) int { return helper(value) + bias + len(payload) }
`,
		},
		{
			name:    "selected dependency",
			path:    "dependency/dependency.go",
			content: "package dependency\n\nconst Value = 2\n",
		},
		{
			name:    "module Go version",
			path:    "go.mod",
			content: "module example.test/snapshot\n\ngo 1.26.3\n",
		},
		{
			name:    "non-Go input",
			path:    "source.s",
			content: "// changed selected assembly input\n",
		},
		{
			name:    "embedded bytes",
			path:    "payload.txt",
			content: "changed payload\n",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			root := t.TempDir()
			writeSourceSnapshotFixture(t, root)
			before := loadSourceSnapshotFixture(t, root).SourceDigest()
			writeSourceSnapshotFile(t, root, testCase.path, testCase.content)
			after := loadSourceSnapshotFixture(t, root).SourceDigest()
			if before == "" || after == "" || before == after {
				t.Fatalf("source digest did not change for %s: %q", testCase.name, before)
			}
		})
	}
}

func writeSourceSnapshotFixture(t *testing.T, root string) {
	t.Helper()
	for name, content := range map[string]string{
		"go.mod": "module example.test/snapshot\n\ngo 1.26.4\n",
		"source.go": `package snapshot

import (
	_ "embed"
	"example.test/snapshot/dependency"
)

const bias = 1

//go:embed payload.txt
var payload string

func helper(value int) int { return value + dependency.Value }
func Score(value int) int { return helper(value) + bias + len(payload) }
`,
		"dependency/dependency.go": "package dependency\n\nconst Value = 1\n",
		"source.s":                 "// selected assembly input\n",
		"payload.txt":              "selected payload\n",
	} {
		writeSourceSnapshotFile(t, root, name, content)
	}
}

func writeSourceSnapshotFile(t *testing.T, root string, name string, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func loadSourceSnapshotFixture(t *testing.T, root string) *Program {
	t.Helper()
	program, err := Load(context.Background(), Request{
		Directory: root,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return program
}
