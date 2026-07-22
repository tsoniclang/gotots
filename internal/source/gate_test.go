package source

import (
	"crypto/sha256"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
)

// TestPackageRecordConstructorRejectsIncoherence proves admission is a total
// validating gate: incoherent owner/provenance/acquisition/disposition
// combinations, missing evidence, and structural-evidence/depth divergence
// never enter a workspace. The valid case carries genuine full evidence — a
// typeless or evidence-free record is never a valid fixture.
func TestPackageRecordConstructorRejectsIncoherence(t *testing.T) {
	module, err := identity.NewModuleID("gate.example/m", "")
	if err != nil {
		t.Fatal(err)
	}
	owner, err := identity.NewModuleOwner(module)
	if err != nil {
		t.Fatal(err)
	}
	modPkg, err := identity.NewPackageID(owner, "gate.example/m")
	if err != nil {
		t.Fatal(err)
	}
	stdPkg, err := identity.NewPackageID(identity.StandardLibraryOwner(), "fmt")
	if err != nil {
		t.Fatal(err)
	}
	fileID, err := identity.NewFileID(owner, "m.go")
	if err != nil {
		t.Fatal(err)
	}
	const src = "package m\n\nfunc F() {}\n"
	fset := token.NewFileSet()
	syntax, err := parser.ParseFile(fset, "m.go", src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	fn := syntax.Decls[0]
	bodyStart := fset.PositionFor(fn.Pos(), false).Offset
	bodyEnd := fset.PositionFor(fn.End(), false).Offset
	spanID, err := identity.NewSpanID(fileID, bodyStart, bodyEnd)
	if err != nil {
		t.Fatal(err)
	}
	unitID, err := identity.NewSourceUnitID(spanID, identity.UnitFuncBody)
	if err != nil {
		t.Fatal(err)
	}
	fullUnit := SourceUnit{
		id: unitID, name: "F",
		span:  Span{Start: Position{Offset: bodyStart}, End: Position{Offset: bodyEnd}},
		hash:  sha256.Sum256([]byte(src[bodyStart:bodyEnd])),
		depth: DepthFullSemantic,
	}
	evidencedFile := func(depth EvidenceDepth, evidence FileEvidence) *File {
		unit := fullUnit
		unit.depth = depth
		return &File{path: "m.go", id: fileID, fset: fset, evidence: evidence, units: []SourceUnit{unit}}
	}
	tp := types.NewPackage("gate.example/m", "m")
	cases := []struct {
		name   string
		record *Package
	}{
		{"zero identity", &Package{provenance: ProvenanceWorkspaceModule, acquisition: AcquisitionWorkspace, disposition: DispositionOrdinarySource}},
		{"invalid disposition (zero)", &Package{id: modPkg, provenance: ProvenanceWorkspaceModule, acquisition: AcquisitionWorkspace}},
		{"std owner with workspace provenance", &Package{id: stdPkg, provenance: ProvenanceWorkspaceModule, acquisition: AcquisitionWorkspace, disposition: DispositionOrdinarySource}},
		{"module owner with goroot acquisition", &Package{id: modPkg, provenance: ProvenanceWorkspaceModule, acquisition: AcquisitionGOROOT, disposition: DispositionOrdinarySource}},
		{"dependency with workspace acquisition", &Package{id: modPkg, provenance: ProvenanceModuleDependency, acquisition: AcquisitionWorkspace, disposition: DispositionOrdinarySource}},
		{"std owner with module go directive", &Package{id: stdPkg, provenance: ProvenanceStandardLibrary, acquisition: AcquisitionGOROOT, disposition: DispositionOrdinarySource, moduleGoVersion: "1.26"}},
		{"source record without types", &Package{id: modPkg, provenance: ProvenanceWorkspaceModule, acquisition: AcquisitionWorkspace, disposition: DispositionOrdinarySource,
			files: []*File{evidencedFile(DepthFullSemantic, FullSyntax{Syntax: syntax})}}},
		{"full unit without type information", &Package{id: modPkg, provenance: ProvenanceWorkspaceModule, acquisition: AcquisitionWorkspace, disposition: DispositionOrdinarySource,
			types: tp, files: []*File{evidencedFile(DepthFullSemantic, FullSyntax{Syntax: syntax})}}},
		{"unselected depth", &Package{id: modPkg, provenance: ProvenanceWorkspaceModule, acquisition: AcquisitionWorkspace, disposition: DispositionOrdinarySource,
			types: tp, typesInfo: &types.Info{}, files: []*File{evidencedFile(DepthInvalid, FullSyntax{Syntax: syntax})}}},
		{"full syntax retained over non-full unit", &Package{id: modPkg, provenance: ProvenanceWorkspaceModule, acquisition: AcquisitionWorkspace, disposition: DispositionOrdinarySource,
			types: tp, files: []*File{evidencedFile(DepthDeclarationContract, FullSyntax{Syntax: syntax})}}},
		{"contract file dropping a full unit", &Package{id: modPkg, provenance: ProvenanceWorkspaceModule, acquisition: AcquisitionWorkspace, disposition: DispositionOrdinarySource,
			types: tp, typesInfo: &types.Info{}, files: []*File{evidencedFile(DepthFullSemantic, ContractOnly{})}}},
		{"type info without any full unit", &Package{id: modPkg, provenance: ProvenanceWorkspaceModule, acquisition: AcquisitionWorkspace, disposition: DispositionOrdinarySource,
			types: tp, typesInfo: &types.Info{}, files: []*File{evidencedFile(DepthDeclarationContract, ContractOnly{})}}},
		{"ordinary package without implicit initialization unit", &Package{id: modPkg, provenance: ProvenanceWorkspaceModule, acquisition: AcquisitionWorkspace, disposition: DispositionOrdinarySource,
			types: tp, typesInfo: &types.Info{}, files: []*File{evidencedFile(DepthFullSemantic, FullSyntax{Syntax: syntax})}}},
		{"implicit unit without selected depth", &Package{id: modPkg, provenance: ProvenanceWorkspaceModule, acquisition: AcquisitionWorkspace, disposition: DispositionOrdinarySource,
			types: tp, typesInfo: &types.Info{}, files: []*File{evidencedFile(DepthFullSemantic, FullSyntax{Syntax: syntax})},
			implicitUnits: []ImplicitUnit{{id: mustImplicit(t, modPkg)}}}},
	}
	for _, c := range cases {
		ws := &Workspace{}
		if err := ws.admit(c.record); err == nil {
			t.Errorf("%s: admitted into the workspace", c.name)
		}
		if len(ws.Packages()) != 0 {
			t.Errorf("%s: rejected record still entered the workspace", c.name)
		}
	}
	ws := &Workspace{}
	valid := &Package{
		id: modPkg, provenance: ProvenanceWorkspaceModule, acquisition: AcquisitionWorkspace,
		disposition: DispositionOrdinarySource, types: tp, typesInfo: &types.Info{},
		files:         []*File{evidencedFile(DepthFullSemantic, FullSyntax{Syntax: syntax})},
		implicitUnits: []ImplicitUnit{{id: mustImplicit(t, modPkg), depth: DepthFullSemantic}},
	}
	if err := ws.admit(valid); err != nil {
		t.Errorf("coherent evidenced record rejected: %v", err)
	}
	if len(ws.Packages()) != 1 {
		t.Error("coherent evidenced record not admitted")
	}
}

func mustImplicit(t *testing.T, pkg identity.PackageID) identity.ImplicitUnitID {
	t.Helper()
	id, err := identity.NewImplicitUnitID(pkg, identity.ImplicitOpPackageInit)
	if err != nil {
		t.Fatal(err)
	}
	return id
}
