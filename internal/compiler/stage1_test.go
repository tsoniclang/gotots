package compiler

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/source"
)

func TestInspectConstructsCanonicalShapesAndContext(t *testing.T) {
	dir := t.TempDir()
	writeCompilerFile(t, dir, "go.mod", "module example.com/shapes\n\ngo 1.26.0\n")
	writeCompilerFile(t, dir, "shapes.go", `package shapes

var X = makeValue()

func makeValue() int { return 1 }

func Outer() func(int) int {
	local := X
	println(local)
	return func(value int) int { return local + value }
}

func Read(buffer []byte) (count int, err error)
`)
	writeCompilerFile(t, dir, "read_amd64.s", `#include "textflag.h"

TEXT ·Read(SB), NOSPLIT, $0-48
	MOVQ $0, count+24(FP)
	MOVQ $0, err+32(FP)
	MOVQ $0, err+40(FP)
	RET
`)
	inspection, err := inspectConstructsForTest(t, source.Request{
		Dir: dir, Patterns: []string{"."},
		ProviderContract: contract.DefaultID,
	})
	if err != nil {
		t.Fatal(err)
	}
	graph := inspection.Structure()
	definitions := graph.ResidentDefinitions()
	if len(definitions) != 6 {
		t.Fatalf("definitions = %d, want 6", len(definitions))
	}
	kinds := map[identity.DefinitionKind]int{}
	names := map[string]identity.DefinitionID{}
	for _, definition := range definitions {
		kinds[definition.Kind()]++
		names[definition.Name()] = definition.ID()
	}
	wantKinds := map[identity.DefinitionKind]int{
		identity.DefinitionFuncDecl:           2,
		identity.DefinitionFuncLit:            1,
		identity.DefinitionPackageInitializer: 1,
		identity.DefinitionBodylessDecl:       1,
		identity.DefinitionImplicit:           1,
	}
	for kind, want := range wantKinds {
		if kinds[kind] != want {
			t.Errorf("definitions of kind %s = %d, want %d", kind, kinds[kind], want)
		}
	}
	if names["X"].IsZero() || names["Outer"].IsZero() ||
		names["Read"].IsZero() || names["func literal"].IsZero() ||
		names["package initialization"].IsZero() {
		t.Fatalf("definition names = %v", names)
	}

	full, external := 0, 0
	for _, selection := range inspection.Selections().Records() {
		switch selection.Depth() {
		case contract.DepthFullSemantic:
			full++
		case contract.DepthExternalBoundary:
			external++
		default:
			t.Errorf("unexpected selection depth %s for %s", selection.Depth(), selection.Definition())
		}
	}
	if full != 5 || external != 1 {
		t.Fatalf("selection depths full=%d external=%d, want 5/1", full, external)
	}
	regions := inspection.Executable().Regions()
	if len(regions) != full {
		t.Fatalf("executable regions=%d, full definitions=%d", len(regions), full)
	}
	literal := names["func literal"]
	outer := names["Outer"]
	literalSite := findSite(t, graph, literal)
	if literalSite.ParentDefinition() != outer {
		t.Fatalf("literal parent = %s, want Outer %s", literalSite.ParentDefinition(), outer)
	}
	outerRegion, present := inspection.Executable().For(outer)
	if !present || len(outerRegion.References()) != 1 ||
		outerRegion.References()[0].Child() != literal {
		t.Fatalf("Outer references = %+v, present=%t", outerRegion.References(), present)
	}
	if _, present := inspection.Executable().For(names["Read"]); present {
		t.Fatal("bodyless external obligation owns an executable region")
	}

	roles := map[catalog.Role]int{}
	seen := map[identity.OccurrenceID]structure.Occurrence{}
	for _, occurrence := range graph.ResidentOccurrences() {
		seen[occurrence.ID()] = occurrence
		if occurrence.Edge().Valid() {
			roles[occurrence.Role()]++
		}
	}
	for _, occurrence := range inspection.Executable().AdditionalOccurrences() {
		if _, duplicate := seen[occurrence.ID()]; duplicate {
			t.Errorf("occurrence %s payload is stored in both artifacts", occurrence.ID())
		}
		seen[occurrence.ID()] = occurrence
		if occurrence.Edge().Valid() {
			roles[occurrence.Role()]++
		}
	}
	for _, role := range []catalog.Role{
		catalog.RoleAssignmentTarget,
		catalog.RoleAssignedValue,
		catalog.RoleCallee,
		catalog.RoleCallArgument,
		catalog.RoleReturnValue,
	} {
		if roles[role] == 0 {
			t.Errorf("context-assigned role %s has no occurrence", role)
		}
	}
	for _, region := range regions {
		for _, member := range region.Members() {
			if _, present := seen[member]; !present {
				t.Errorf("region %s references absent canonical occurrence %s", region.ID(), member)
			}
		}
	}

	if inspection.Hydration().SemanticPackages != 1 ||
		inspection.Hydration().LocalFiles != 1 {
		t.Fatalf("hydration = %+v, want one package/file", inspection.Hydration())
	}
	var requestedRoot bool
	for _, pkg := range inspection.Workspace().Packages() {
		requestedRoot = requestedRoot || pkg.RequestedRoot()
	}
	if !requestedRoot {
		t.Fatal("finalized package lost requested-root evidence")
	}
}

func TestProviderArtifactAuditVerifyAndRelocatedConsumption(t *testing.T) {
	first := writeProviderFixture(t, "example.com/first", "first")
	output := t.TempDir()
	structurePath := filepath.Join(output, "provider.structure.gotots")
	semanticPath := filepath.Join(output, "provider.semantic.gotots")
	request := source.Request{
		Dir: first, Patterns: []string{"."},
		ProviderContract: contract.DefaultID,
	}
	result, err := AuditCatalog(
		request, structurePath, semanticPath,
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Structure.Digest == "" ||
		result.Structure.EncodedBytes <= 0 ||
		result.Structure.PackageContexts == 0 ||
		result.Structure.Files == 0 ||
		result.Structure.Definitions == 0 ||
		result.Structure.LargestShardBytes == 0 ||
		result.Structure.LargestPackageRecords == 0 ||
		len(result.Structure.LargestPackages()) == 0 ||
		len(result.Structure.LargestPackages()) > 20 ||
		len(result.Structure.LargestHeaders()) == 0 ||
		len(result.Structure.LargestHeaders()) > 20 ||
		result.Semantic.Digest == "" ||
		result.Semantic.EncodedBytes <= 0 ||
		result.Semantic.Packages == 0 ||
		result.Semantic.Definitions == 0 ||
		result.Semantic.LargestShardBytes == 0 ||
		result.Semantic.LargestPackageRecords == 0 {
		t.Fatalf("provider result is vacuous: %+v", result)
	}
	productionMetrics := result.Semantic.Metrics()
	if productionMetrics.Packages() != result.Semantic.Packages ||
		productionMetrics.Definitions() != result.Semantic.Definitions ||
		productionMetrics.Resolutions() != result.Semantic.Resolutions ||
		productionMetrics.EncodedBytes() <= 0 ||
		len(productionMetrics.LargestPackages()) == 0 ||
		len(productionMetrics.LargestPackages()) > 20 ||
		len(productionMetrics.LargestDefinitions()) !=
			minimumInt(productionMetrics.Definitions(), 20) ||
		len(productionMetrics.LargestOperations()) !=
			minimumInt(productionMetrics.Operations(), 20) ||
		len(productionMetrics.LargestTypes()) !=
			minimumInt(productionMetrics.Types(), 20) {
		t.Fatalf(
			"provider semantic metrics disagree with production: %+v",
			productionMetrics,
		)
	}
	productionWork := result.SemanticWork
	if productionWork.Packages != result.Semantic.Packages ||
		productionWork.InputOccurrences == 0 ||
		productionWork.ContextAssignments !=
			productionWork.InputOccurrences ||
		productionWork.ResolutionVisits !=
			productionWork.InputOccurrences ||
		productionWork.OccurrenceResolutions !=
			productionWork.InputOccurrences ||
		productionWork.LinearOperations() == 0 {
		t.Fatalf(
			"provider semantic work is not conserved: %+v",
			productionWork,
		)
	}
	stat, err := os.Stat(structurePath)
	if err != nil {
		t.Fatal(err)
	}
	if stat.Size() != result.Structure.EncodedBytes {
		t.Fatalf(
			"provider structure size=%d, reported=%d",
			stat.Size(), result.Structure.EncodedBytes,
		)
	}
	semanticStat, err := os.Stat(semanticPath)
	if err != nil {
		t.Fatal(err)
	}
	if semanticStat.Size() != result.Semantic.EncodedBytes {
		t.Fatalf(
			"provider semantic size=%d, reported=%d",
			semanticStat.Size(), result.Semantic.EncodedBytes,
		)
	}
	if err := AuditVerify(
		request, structurePath, semanticPath,
	); err != nil {
		t.Fatal(err)
	}

	second := writeProviderFixture(t, "example.com/second", "second")
	inspection, err := inspectConstructsForTest(t, source.Request{
		Dir: second, Patterns: []string{"."},
		ProviderContract:          contract.DefaultID,
		ProviderStructureArtifact: structurePath,
		ProviderStructureDigest:   result.Structure.Digest,
		ProviderSemanticArtifact:  semanticPath,
		ProviderSemanticDigest:    result.Semantic.Digest,
	})
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Hydration().SemanticPackages != 2 ||
		inspection.Hydration().LocalFiles != 2 ||
		len(inspection.SourcePlan().LocalFileIDs()) != 2 {
		t.Fatalf("provider consumption widened local hydration: %+v", inspection.Hydration())
	}
	if len(inspection.Workspace().Packages()) <= 1 {
		t.Fatal("provider fixture has no imported closure")
	}
	manifestStats := inspection.Structure().ProviderManifestStats()
	projectionStats := inspection.Structure().ProviderProjectionStats()
	assertBoundedProviderAuthorities(
		t, inspection, manifestStats,
	)
	if manifestStats.PackageContexts <= 1 ||
		len(manifestStats.LargestShards()) == 0 ||
		projectionStats.ShardLoads != 0 ||
		projectionStats.ProjectedPackages != 0 ||
		projectionStats.MaxResidentPackages != 0 ||
		projectionStats.CacheHits != 0 ||
		projectionStats.LargestPackageBytes != 0 ||
		projectionStats.LargestPackageRecords != 0 ||
		len(inspection.Structure().LargestHeaderArtifacts()) == 0 ||
		len(inspection.Structure().LargestHeaderArtifacts()) > 20 {
		t.Fatalf(
			"provider manifest/projection stats = %+v / %+v",
			manifestStats,
			projectionStats,
		)
	}
	localMetrics := inspection.SemanticMetrics()
	semanticRead := inspection.Semantic().ProviderReadStats()
	providerMetrics := inspection.Semantic().ProviderManifestMetrics()
	consumedMetrics := semanticRead.Metrics()
	semanticProjection := inspection.Semantic().ProjectionStats()
	if localMetrics.Packages() == 0 ||
		localMetrics.Definitions() == 0 ||
		providerMetrics.Packages() != result.Semantic.Packages ||
		semanticProjection.CertifiedPackages != providerMetrics.Packages() ||
		semanticProjection.MixedPackages != 0 ||
		consumedMetrics.Packages() != 0 ||
		semanticRead.ShardLoads != 0 ||
		semanticRead.MaxProviderPackagesResident != 0 ||
		len(providerMetrics.LargestPackages()) == 0 ||
		len(providerMetrics.LargestPackages()) > 20 {
		t.Fatalf(
			"semantic production/manifest/consumption metrics local=%+v provider=%+v consumed=%+v projections=%+v reads=%+v",
			localMetrics,
			providerMetrics,
			consumedMetrics,
			semanticProjection,
			semanticRead,
		)
	}

	t.Chdir(filepath.Dir(structurePath))
	artifact, err := structure.DecodeProviderArtifact(
		filepath.Base(structurePath), result.Structure.Digest,
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	var fileID identity.FileID
	for id := range artifact.FileIDs() {
		fileID = id
		break
	}
	if fileID.IsZero() {
		t.Fatal("provider artifact contains no file graph")
	}
	if _, _, present, err := artifact.FileGraph(fileID); err != nil || !present {
		t.Fatalf("lazy graph after cwd change: present=%t err=%v", present, err)
	}
	assertProviderTamperRejected(
		t, structurePath, result.Structure.Digest,
	)
}

func minimumInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func TestCgoSelectionFactsAreScopeExact(t *testing.T) {
	if output, err := exec.Command("go", "env", "CGO_ENABLED").Output(); err != nil ||
		strings.TrimSpace(string(output)) != "1" {
		t.Skip("cgo is unavailable")
	}
	if _, err := exec.LookPath("cc"); err != nil {
		t.Skip("C compiler is unavailable")
	}
	dir := t.TempDir()
	writeCompilerFile(t, dir, "go.mod", "module example.com/cgo\n\ngo 1.26.0\n")
	writeCompilerFile(t, dir, "main.go", `package main

/*
static int add(int left, int right) { return left + right; }
*/
import "C"

func pure() int { return 1 }
func shadow() int {
	type C struct{}
	_ = C{}
	return 0
}
func external() int { return int(C.add(1, 2)) }
func parent() func() int {
	return func() int { return int(C.add(3, 4)) }
}
func cParent() func() int {
	_ = C.int(1)
	return func() int { return 5 }
}
func signature() func(C.int) int {
	return func(value C.int) int { return int(value) }
}
func main() { _, _, _, _, _, _ = pure(), shadow(), external(), parent(), cParent(), signature() }
`)
	request := source.Request{
		Dir: dir, Patterns: []string{"."},
		ProviderContract: contract.DefaultID,
		Env:              []string{"CGO_ENABLED=1"},
	}
	output := t.TempDir()
	structurePath := filepath.Join(output, "cgo-provider.structure.gotots")
	semanticPath := filepath.Join(output, "cgo-provider.semantic.gotots")
	provider, err := AuditCatalog(
		request, structurePath, semanticPath,
	)
	if err != nil {
		t.Fatal(err)
	}
	request.ProviderStructureArtifact = structurePath
	request.ProviderStructureDigest = provider.Structure.Digest
	request.ProviderSemanticArtifact = semanticPath
	request.ProviderSemanticDigest = provider.Semantic.Digest
	inspection, err := inspectConstructsForTest(t, request)
	if err != nil {
		t.Fatal(err)
	}
	var packageID identity.PackageID
	for _, pkg := range inspection.Workspace().Packages() {
		if pkg.RequestedRoot() {
			packageID = pkg.ID()
			break
		}
	}
	if packageID.IsZero() {
		t.Fatal("cgo fixture has no root package")
	}

	definitions := map[identity.DefinitionID]structure.ImplementationDefinition{}
	sites := map[identity.DefinitionID]structure.DefinitionSite{}
	if err := inspection.Structure().VisitPackages(func(
		pkg structure.PackageGraph,
	) error {
		if pkg.ID() != packageID {
			return nil
		}
		for _, definition := range pkg.Definitions() {
			definitions[definition.ID()] = definition
		}
		for _, site := range pkg.Sites() {
			sites[site.Definition()] = site
		}
		if len(pkg.CheckedMappings()) == 0 {
			t.Error("cgo package has no checked-definition mappings")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	expected := map[string]bool{
		"pure": false, "shadow": false, "external": true,
		"parent": false, "cParent": true, "signature": true,
		"main":                   false,
		"package initialization": false,
	}
	literalExpected := map[string]bool{
		"parent": true, "cParent": false, "signature": true,
	}
	for id, definition := range definitions {
		value, present := inspection.SelectionFacts().Value(
			id, contract.SelectionFactCDependent,
		)
		if !present {
			t.Errorf("definition %s has no C-dependence fact", id)
			continue
		}
		switch {
		case id.SyntheticRole().Valid():
			if !value {
				t.Errorf("synthetic cgo definition %s is not C-dependent", id)
			}
		case definition.Name() == "func literal":
			parent := definitions[sites[id].ParentDefinition()].Name()
			want, known := literalExpected[parent]
			if !known {
				t.Errorf("literal has unexpected parent %q", parent)
				continue
			}
			if value != want {
				t.Errorf("literal below %s C-dependent=%t, want %t", parent, value, want)
			}
		default:
			want, known := expected[definition.Name()]
			if known && value != want {
				t.Errorf("%s C-dependent=%t, want %t", definition.Name(), value, want)
			}
		}
	}
}

func findSite(
	t *testing.T,
	graph *structure.Graph,
	definition identity.DefinitionID,
) structure.DefinitionSite {
	t.Helper()
	var found structure.DefinitionSite
	if err := graph.VisitPackages(func(
		pkg structure.PackageGraph,
	) error {
		for _, site := range pkg.Sites() {
			if site.Definition() == definition {
				found = site
				return nil
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !found.Definition().IsZero() {
		return found
	}
	t.Fatalf("definition %s has no site", definition)
	return structure.DefinitionSite{}
}

func writeCompilerFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeProviderFixture(t *testing.T, module, marker string) string {
	t.Helper()
	dir := t.TempDir()
	writeCompilerFile(
		t, dir, "go.mod",
		"module "+module+"\n\ngo 1.26.0\n",
	)
	writeCompilerFile(
		t,
		dir,
		"main.go",
		"package main\n\nimport \"errors\"\n\nfunc main() { _ = errors.New(\""+marker+"\") }\n",
	)
	return dir
}

func assertProviderTamperRejected(
	t *testing.T,
	path string,
	originalDigest string,
) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	tampered := append([]byte(nil), raw...)
	tampered[len(tampered)-1] ^= 0xff
	tamperedPath := filepath.Join(t.TempDir(), "tampered.gotots")
	if err := os.WriteFile(tamperedPath, tampered, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := structure.DecodeProviderArtifact(
		tamperedPath, originalDigest,
	); err == nil {
		t.Fatal("external artifact digest failed to reject changed bytes")
	}
	sum := sha256.Sum256(tampered)
	rebound, err := structure.DecodeProviderArtifact(
		tamperedPath, hex.EncodeToString(sum[:]),
	)
	if err != nil {
		t.Fatalf("container admission rejected before shard check: %v", err)
	}
	shardRejected := false
	for id := range rebound.FileIDs() {
		if _, _, _, err := rebound.FileGraph(id); err != nil {
			shardRejected = true
			break
		}
	}
	if !shardRejected {
		t.Fatal("package-shard digest failed to reject rebound tampered bytes")
	}
	truncated := raw[:len(raw)-1]
	truncatedPath := filepath.Join(t.TempDir(), "truncated.gotots")
	if err := os.WriteFile(truncatedPath, truncated, 0o600); err != nil {
		t.Fatal(err)
	}
	truncatedSum := sha256.Sum256(truncated)
	if _, err := structure.DecodeProviderArtifact(
		truncatedPath, hex.EncodeToString(truncatedSum[:]),
	); err == nil {
		t.Fatal("truncated provider container was admitted")
	}
}
