package load

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestDriverEvidenceExactJoinRejectsDroppedAndTrailingRecords(t *testing.T) {
	module := &packages.Module{Path: "example.com/evidence", Version: "v1.2.3"}
	dependency := &packages.Package{
		ID: "example.com/evidence/dependency", Name: "dependency",
		PkgPath: "example.com/evidence/dependency",
		Dir:     filepath.Join(t.TempDir(), "dependency"), Module: module,
	}
	root := &packages.Package{
		ID: "example.com/evidence", Name: "evidence",
		PkgPath: "example.com/evidence", Dir: t.TempDir(), Module: module,
		Imports: map[string]*packages.Package{
			"example.com/evidence/dependency": dependency,
		},
	}
	path := filepath.Join(t.TempDir(), "evidence.json")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeDriverEvidence(
		path,
		[]string{root.ID},
		[]*packages.Package{dependency, root},
	); err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(payload, []byte(" trailing")...), 0o600); err != nil {
		t.Fatal(err)
	}
	responseDependency := &packages.Package{
		ID: dependency.ID, Name: dependency.Name, PkgPath: dependency.PkgPath,
	}
	responseRoot := &packages.Package{
		ID: root.ID, Name: root.Name, PkgPath: root.PkgPath,
		Imports: map[string]*packages.Package{
			"example.com/evidence/dependency": responseDependency,
		},
	}
	if err := joinDriverEvidence(path, []*packages.Package{responseRoot}); err == nil {
		t.Fatal("trailing driver evidence was accepted")
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	responseRoot.Imports = nil
	if err := joinDriverEvidence(path, []*packages.Package{responseRoot}); err == nil {
		t.Fatal("dropped driver package was accepted")
	}
}
