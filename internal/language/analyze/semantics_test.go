package analyze

import (
	"errors"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/source"
)

// TestVariantCoverageIsTotal proves the variant catalog and resolver are total
// against real syntax: every catalog variant (including VariantNone) appears
// in the fixture inventory, and every variant-bearing occurrence resolved.
// A deleted resolver arm or catalog member fails here.
func TestVariantCoverageIsTotal(t *testing.T) {
	_, inv := fixtureFile(t)
	observed := map[catalog.Variant]int{}
	for _, occurrence := range inv.Occurrences() {
		if catalog.VariantBearing(occurrence.Kind()) && !occurrence.Variant().Valid() {
			t.Errorf("variant-bearing %s at %s unresolved", occurrence.Kind(), occurrence.ID())
		}
		observed[occurrence.Variant()]++
	}
	for _, variant := range catalog.AllVariants() {
		if observed[variant] == 0 {
			t.Errorf("variant %s not exercised by the fixture", variant)
		}
	}
}

// TestImplicitOperationCoverage proves every inventory-owned implicit
// operation is detected at least once in the fixture, and no semantic-model-
// owned member is produced by the inventory.
func TestImplicitOperationCoverage(t *testing.T) {
	_, inv := fixtureFile(t)
	observed := map[catalog.ImplicitOp]int{}
	for _, occurrence := range inv.Occurrences() {
		for _, op := range occurrence.Implicit() {
			observed[op]++
		}
	}
	for _, op := range catalog.AllImplicitOps() {
		switch op.Owner() {
		case catalog.ImplicitOwnerInventory:
			if observed[op] == 0 {
				t.Errorf("inventory-owned implicit op %s not detected in the fixture", op)
			}
		case catalog.ImplicitOwnerSemanticModel:
			if observed[op] != 0 {
				t.Errorf("semantic-model-owned op %s produced by the inventory", op)
			}
		default:
			t.Errorf("implicit op %s has invalid owner", op)
		}
	}
}

// TestDirectiveInventory proves directives join the closed catalog: the go
// directive, the external tool directive, and their dispositions.
func TestDirectiveInventory(t *testing.T) {
	_, inv := fixtureFile(t)
	kinds := map[catalog.DirectiveKind]DirectiveRecord{}
	for _, directive := range inv.Directives() {
		kinds[directive.Kind()] = directive
	}
	generate, ok := kinds[catalog.DirectiveGoGenerate]
	if !ok {
		t.Fatal("//go:generate not inventoried")
	}
	if generate.Tool() != "go" || generate.Name() != "generate" {
		t.Errorf("generate record tool/name = %s/%s", generate.Tool(), generate.Name())
	}
	if generate.Kind().Disposition() != catalog.DirectiveToolingOnly {
		t.Errorf("generate disposition = %s", generate.Kind().Disposition())
	}
	external, ok := kinds[catalog.DirectiveExternalTool]
	if !ok {
		t.Fatal("//custom:note not inventoried as external tool directive")
	}
	if external.Tool() != "custom" {
		t.Errorf("external record tool = %s", external.Tool())
	}
}

// TestUnknownGoDirectiveFailsClosed proves an unknown go:* directive aborts
// the inventory with a typed error.
func TestUnknownGoDirectiveFailsClosed(t *testing.T) {
	dir := t.TempDir()
	for rel, content := range map[string]string{
		"go.mod":  "module fixture.example/unknown\n\ngo 1.26\n",
		"main.go": "package p\n\n//go:frobnicate all\nfunc F() {}\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(content), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	ws, err := source.LoadWorkspace(source.Request{Dir: dir})
	if err != nil {
		t.Fatalf("LoadWorkspace: %v", err)
	}
	_, err = BuildWorkspaceInventory(ws)
	if err == nil {
		t.Fatal("unknown go directive was admitted")
	}
	var unknown *UnknownDirectiveError
	if !errors.As(err, &unknown) {
		t.Fatalf("error = %T, want *UnknownDirectiveError", err)
	}
	if unknown.Tool() != "go" || unknown.Name() != "frobnicate" {
		t.Errorf("unknown directive tool/name = %s/%s", unknown.Tool(), unknown.Name())
	}
}

// TestLegacyBuildTagInventoried proves the legacy // +build form joins its
// catalog member (classified directly; a real fixture file with the tag would
// change build selection).
func TestLegacyBuildTagInventoried(t *testing.T) {
	fset := token.NewFileSet()
	base := fset.AddFile("synthetic.go", -1, 100)
	fileID, err := identity.NewFileID(mustModuleID(t), "synthetic.go")
	if err != nil {
		t.Fatalf("NewFileID: %v", err)
	}
	b := &builder{fset: fset, file: fileID}
	record, ok, err := b.directiveOf(&ast.Comment{Slash: base.Pos(0), Text: "// +build linux"})
	if err != nil || !ok {
		t.Fatalf("legacy build tag not classified: ok=%v err=%v", ok, err)
	}
	if record.Kind() != catalog.DirectiveLegacyBuildTag {
		t.Errorf("kind = %s, want legacy build tag", record.Kind())
	}
	if record.Kind().Disposition() != catalog.DirectiveLoaderOwned {
		t.Errorf("disposition = %s, want loader-owned", record.Kind().Disposition())
	}
}

// TestVisitorRejectsNonActiveConstructs proves the catalog disposition is the
// single admission policy: deprecated and recovery kinds produce a typed
// ConstructError carrying the catalog's own disposition, and nothing is
// recorded.
func TestVisitorRejectsNonActiveConstructs(t *testing.T) {
	fileID, err := identity.NewFileID(mustModuleID(t), "synthetic.go")
	if err != nil {
		t.Fatalf("NewFileID: %v", err)
	}
	fset := token.NewFileSet()
	base := fset.AddFile("synthetic.go", -1, 100)
	pos, end := base.Pos(0), base.Pos(10)
	cases := []ast.Node{
		&ast.Package{},
		&ast.BadExpr{From: pos, To: end},
		&ast.BadStmt{From: pos, To: end},
		&ast.BadDecl{From: pos, To: end},
	}
	for _, node := range cases {
		b := &builder{fset: fset, file: fileID}
		err := b.visit(node, -1, 0)
		if err == nil {
			t.Errorf("%T was admitted", node)
			continue
		}
		var rejected *ConstructError
		if !errors.As(err, &rejected) {
			t.Errorf("%T error = %T, want *ConstructError", node, err)
			continue
		}
		if rejected.Disposition() != rejected.Kind().Disposition() {
			t.Errorf("%T rejection carries disposition %s, catalog says %s",
				node, rejected.Disposition(), rejected.Kind().Disposition())
		}
		if len(b.occurrences) != 0 {
			t.Errorf("%T was recorded despite rejection", node)
		}
	}
}

func mustModuleID(t *testing.T) identity.Owner {
	t.Helper()
	m, err := identity.NewModuleID("fixture.example/synthetic", "")
	if err != nil {
		t.Fatalf("NewModuleID: %v", err)
	}
	owner, err := identity.NewModuleOwner(m)
	if err != nil {
		t.Fatalf("NewModuleOwner: %v", err)
	}
	return owner
}
