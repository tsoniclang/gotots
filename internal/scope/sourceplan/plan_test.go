package sourceplan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/source"
)

func TestDefaultPlanSeparatesLocalAndCertifiedSource(t *testing.T) {
	universe := resolvePlanFixture(t)
	selected, err := contract.Default()
	if err != nil {
		t.Fatal(err)
	}
	audit, err := BuildForAudit(universe, selected)
	if err != nil {
		t.Fatal(err)
	}
	if audit.Purpose() != PurposeProviderProduction {
		t.Fatalf("audit purpose = %s", audit.Purpose())
	}
	certified := certifiedFromPlan(audit, "certified-digest")
	plan, err := Build(universe, selected, certified)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Purpose() != PurposeCompilation ||
		plan.Fingerprint() == "" {
		t.Fatalf("invalid compilation plan purpose=%s fingerprint=%q", plan.Purpose(), plan.Fingerprint())
	}

	var local, provider int
	for _, decision := range plan.Files() {
		pkg := loadedPackageForFile(t, universe, decision.ID())
		switch decision.Kind() {
		case KindLocalSyntax:
			local++
			if decision.ArtifactDigest() != "" {
				t.Errorf("local file %s carries provider authority", decision.ID())
			}
			if pkg.ID().Owner().Class() == identity.OwnerStandardLibrary &&
				pkg.Disposition() != source.DispositionUnsafeIntrinsic {
				t.Errorf("ordinary std file %s was selected locally", decision.ID())
			}
		case KindCertifiedGraph:
			provider++
			if decision.ArtifactDigest() != certified.Digest {
				t.Errorf("certified file %s has digest %q", decision.ID(), decision.ArtifactDigest())
			}
			if pkg.ID().Owner().Class() == identity.OwnerModule {
				t.Errorf("module file %s was delegated to a provider", decision.ID())
			}
		default:
			t.Errorf("file %s has invalid decision %s", decision.ID(), decision.Kind())
		}
		if decision.ContractDigest() != selected.Fingerprint() {
			t.Errorf("file %s has wrong contract digest", decision.ID())
		}
	}
	if local == 0 || provider == 0 {
		t.Fatalf("plan partition local=%d provider=%d is vacuous", local, provider)
	}
	if len(plan.LocalFileIDs()) != local {
		t.Fatalf("LocalFileIDs=%d, local decisions=%d", len(plan.LocalFileIDs()), local)
	}
}

func TestCompilationPlanFailsClosedWithoutCertifiedAuthority(t *testing.T) {
	universe := resolvePlanFixture(t)
	selected, err := contract.Default()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Build(universe, selected, CertifiedInput{}); err == nil {
		t.Fatal("compilation admitted a certified decision without an artifact")
	}
	audit, err := BuildForAudit(universe, selected)
	if err != nil {
		t.Fatal(err)
	}
	certified := certifiedFromPlan(audit, "certified-digest")
	for file := range certified.Files {
		delete(certified.Files, file)
		break
	}
	if _, err := Build(universe, selected, certified); err == nil {
		t.Fatal("compilation admitted an artifact with a missing file")
	}
}

func TestExactDefinitionRuleChangesOnlyItsOwningFile(t *testing.T) {
	universe := resolvePlanFixture(t)
	base, err := contract.Default()
	if err != nil {
		t.Fatal(err)
	}
	var target identity.FileID
	for _, pkg := range universe.Packages() {
		if pkg.ID().Owner().Class() == identity.OwnerStandardLibrary &&
			pkg.Disposition() == source.DispositionOrdinarySource &&
			len(pkg.Files()) > 1 {
			target = pkg.Files()[0].ID()
			break
		}
	}
	if target.IsZero() {
		t.Fatal("fixture closure has no multi-file ordinary std package")
	}
	span, err := identity.NewSpanID(target, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	root, err := identity.NewOccurrenceID(span, 47)
	if err != nil {
		t.Fatal(err)
	}
	definition, err := identity.NewSourceDefinitionID(
		root, identity.DefinitionFuncDecl,
	)
	if err != nil {
		t.Fatal(err)
	}
	rule, err := contract.NewDefinitionRule(
		definition,
		contract.ConditionAlways,
		contract.SelectionFactInvalid,
		contract.ProviderAutomaticTranslation,
	)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := contract.New(
		"exact-file@v1", append(base.Rules(), rule),
	)
	if err != nil {
		t.Fatal(err)
	}
	audit, err := BuildForAudit(universe, selected)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := Build(
		universe,
		selected,
		certifiedFromPlan(audit, "certified-digest"),
	)
	if err != nil {
		t.Fatal(err)
	}
	decision, present := plan.For(target)
	if !present || decision.Kind() != KindLocalSyntax {
		t.Fatalf("exact-definition file decision = %+v, present=%t", decision, present)
	}
	for _, file := range loadedPackageForFile(t, universe, target).Files() {
		if file.ID() == target {
			continue
		}
		sibling, present := plan.For(file.ID())
		if !present || sibling.Kind() != KindCertifiedGraph {
			t.Fatalf("sibling %s decision = %+v, present=%t", file.ID(), sibling, present)
		}
	}
}

func TestPlanIsDeterministicAndCollectionsAreIsolated(t *testing.T) {
	universe := resolvePlanFixture(t)
	selected, err := contract.Default()
	if err != nil {
		t.Fatal(err)
	}
	audit, err := BuildForAudit(universe, selected)
	if err != nil {
		t.Fatal(err)
	}
	certified := certifiedFromPlan(audit, "certified-digest")
	first, err := Build(universe, selected, certified)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Build(universe, selected, certified)
	if err != nil {
		t.Fatal(err)
	}
	if first.Fingerprint() != second.Fingerprint() {
		t.Fatalf("identical plans have fingerprints %s and %s", first.Fingerprint(), second.Fingerprint())
	}
	files := first.Files()
	original := files[0]
	files[0] = File{}
	indexed, present := first.For(original.ID())
	if !present || indexed != original {
		t.Fatal("Plan.Files exposes canonical backing storage")
	}
	local := first.LocalFileIDs()
	if len(local) == 0 {
		t.Fatal("fixture has no local files")
	}
	local[0] = identity.FileID{}
	if first.LocalFileIDs()[0].IsZero() {
		t.Fatal("Plan.LocalFileIDs exposes backing storage")
	}
}

func TestPlanAdmissionRejectsInvalidRecords(t *testing.T) {
	universe := resolvePlanFixture(t)
	selected, err := contract.Default()
	if err != nil {
		t.Fatal(err)
	}
	audit, err := BuildForAudit(universe, selected)
	if err != nil {
		t.Fatal(err)
	}
	valid := audit.Files()[0]
	tests := map[string]*Plan{
		"invalid purpose": {
			files: []File{valid}, byID: map[identity.FileID]*File{},
			syntheticBy: map[identity.PackageID]*SyntheticOwner{},
		},
		"local with artifact": {
			purpose: PurposeCompilation,
			files: []File{{
				id: valid.ID(), kind: KindLocalSyntax,
				contractDigest: "contract", artifactDigest: "artifact",
			}},
			byID:        map[identity.FileID]*File{},
			syntheticBy: map[identity.PackageID]*SyntheticOwner{},
		},
		"certified without artifact": {
			purpose: PurposeCompilation,
			files: []File{{
				id: valid.ID(), kind: KindCertifiedGraph,
				contractDigest: "contract",
			}},
			byID:        map[identity.FileID]*File{},
			syntheticBy: map[identity.PackageID]*SyntheticOwner{},
		},
		"duplicate file": {
			purpose:     PurposeProviderProduction,
			files:       []File{valid, valid},
			byID:        map[identity.FileID]*File{},
			syntheticBy: map[identity.PackageID]*SyntheticOwner{},
		},
	}
	for name, candidate := range tests {
		t.Run(name, func(t *testing.T) {
			if err := finishPlan(candidate); err == nil {
				t.Fatal("invalid plan passed admission")
			}
		})
	}
}

func resolvePlanFixture(t *testing.T) *source.Universe {
	t.Helper()
	dir := t.TempDir()
	writePlanFile(t, dir, "go.mod", "module example.com/app\n\ngo 1.26.0\n")
	writePlanFile(
		t,
		dir,
		"main.go",
		"package main\n\nimport \"errors\"\n\nfunc main() { _ = errors.New(\"fixture\") }\n",
	)
	universe, err := source.ResolveUniverse(source.Request{
		Dir: dir, Patterns: []string{"."},
	})
	if err != nil {
		t.Fatal(err)
	}
	return universe
}

func writePlanFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func certifiedFromPlan(plan *Plan, digest string) CertifiedInput {
	out := CertifiedInput{
		Digest:   digest,
		Files:    map[identity.FileID]bool{},
		Packages: map[identity.PackageID]bool{},
	}
	for _, file := range plan.Files() {
		if file.Kind() == KindCertifiedGraph {
			out.Files[file.ID()] = true
		}
	}
	for _, owner := range plan.SyntheticOwners() {
		if owner.Kind() == KindCertifiedGraph {
			out.Packages[owner.Package()] = true
		}
	}
	return out
}

func loadedPackageForFile(
	t *testing.T,
	universe *source.Universe,
	file identity.FileID,
) *source.LoadedPackage {
	t.Helper()
	for _, pkg := range universe.Packages() {
		for _, candidate := range pkg.Files() {
			if candidate.ID() == file {
				return pkg
			}
		}
	}
	t.Fatalf("file %s has no loaded package", file)
	return nil
}
