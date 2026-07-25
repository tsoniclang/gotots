package verify

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStage2VerificationUsesOneSemanticRecordPassPerClass(
	t *testing.T,
) {
	root := filepath.Join(repoRoot(t), "internal", "stagecheck")
	counts, err := semanticVerificationPassCounts(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"VisitResolutions",
		"VisitOperations",
	} {
		if counts[name] != 1 {
			t.Errorf(
				"stagecheck has %d %s passes, want exactly one",
				counts[name],
				name,
			)
		}
	}
}

func TestStage2VerificationPassWallRejectsDuplicateWalk(
	t *testing.T,
) {
	directory := t.TempDir()
	path := filepath.Join(directory, "duplicate.go")
	source := `package stagecheck
func duplicate(pkg Package) {
	_ = pkg.VisitResolutions(nil)
	_ = pkg.VisitResolutions(nil)
	_ = pkg.VisitOperations(nil)
	_ = pkg.VisitOperations(nil)
}
`
	if err := writeTestFile(path, source); err != nil {
		t.Fatal(err)
	}
	counts, err := semanticVerificationPassCounts(directory)
	if err != nil {
		t.Fatal(err)
	}
	if counts["VisitResolutions"] != 2 ||
		counts["VisitOperations"] != 2 {
		t.Fatalf("duplicate-pass control was not detected: %v", counts)
	}
}

func semanticVerificationPassCounts(
	directory string,
) (map[string]int, error) {
	counts := map[string]int{
		"VisitResolutions": 0,
		"VisitOperations":  0,
	}
	fset := token.NewFileSet()
	err := filepath.WalkDir(
		directory,
		func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() ||
				!strings.HasSuffix(path, ".go") ||
				strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return err
			}
			ast.Inspect(file, func(node ast.Node) bool {
				selector, ok := node.(*ast.SelectorExpr)
				if ok {
					if _, tracked := counts[selector.Sel.Name]; tracked {
						counts[selector.Sel.Name]++
					}
				}
				return true
			})
			return nil
		},
	)
	return counts, err
}

func writeTestFile(path string, source string) error {
	return os.WriteFile(path, []byte(source), 0o600)
}
