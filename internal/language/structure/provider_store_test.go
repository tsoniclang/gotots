package structure

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/scope/contract"
)

type providerStoreFixture struct {
	path       string
	digest     string
	pkg        identity.PackageID
	definition identity.DefinitionID
	context    providerContextRecord
	manifest   providerManifest
	shard      []byte
}

func TestProviderArtifactIsDeterministicLazyAndRelocatable(t *testing.T) {
	first := writeProviderStoreFixture(t, filepath.Join(t.TempDir(), "first.gotots"))
	second := writeProviderStoreFixture(t, filepath.Join(t.TempDir(), "second.gotots"))
	firstBytes, err := os.ReadFile(first.path)
	if err != nil {
		t.Fatal(err)
	}
	secondBytes, err := os.ReadFile(second.path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstBytes, secondBytes) || first.digest != second.digest {
		t.Fatal("identical provider evidence did not produce identical bytes")
	}

	originalDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(filepath.Dir(first.path))
	artifact, err := DecodeProviderArtifact(filepath.Base(first.path), first.digest)
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(originalDirectory)
	if len(artifact.fileGraphs) != 0 ||
		len(artifact.packageGraphs) != 0 ||
		len(artifact.packageCensus) != 1 ||
		len(artifact.factsByPackage) != 1 {
		t.Fatal("manifest admission eagerly retained a package payload")
	}
	if artifact.PackageContextCount() != 1 ||
		artifact.SyntheticPackageCount() != 1 ||
		artifact.FactCount() != 1 {
		t.Fatalf(
			"manifest denominators packages=%d synthetic=%d facts=%d",
			artifact.PackageContextCount(),
			artifact.SyntheticPackageCount(),
			artifact.FactCount(),
		)
	}
	stats := artifact.ManifestStats()
	if stats.PackageContexts != 1 ||
		stats.Files != 0 ||
		stats.SyntheticPackages != 1 ||
		stats.Definitions != 1 ||
		stats.HeaderOccurrences != 0 ||
		stats.BoundaryEntries != 0 ||
		stats.SelectionFacts != 1 ||
		stats.LargestShardBytes != int64(len(first.shard)) {
		t.Fatalf("manifest stats = %+v", stats)
	}
	if artifact.storage.loadedArtifact != nil {
		t.Fatal("manifest stats decoded a detailed package shard")
	}

	var wait sync.WaitGroup
	failures := make(chan error, 64)
	for range 32 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			graph, present, err := artifact.SyntheticPackageGraph(first.pkg)
			if err != nil {
				failures <- err
				return
			}
			if !present || len(graph.Definitions()) != 1 ||
				graph.Definitions()[0].ID() != first.definition {
				failures <- fmt.Errorf("concurrent graph projection is incoherent")
				return
			}
			facts := artifact.CertifiedFactsForPackage(first.pkg)
			if len(facts) != 1 ||
				facts[0].Definition() != first.definition ||
				facts[0].Kind() != contract.SelectionFactCDependent {
				failures <- fmt.Errorf("concurrent fact projection is incoherent")
			}
		}()
	}
	wait.Wait()
	close(failures)
	for failure := range failures {
		t.Error(failure)
	}
	graph, _, err := artifact.SyntheticPackageGraph(first.pkg)
	if err != nil {
		t.Fatal(err)
	}
	graphDefinitions := graph.Definitions()
	graphDefinitions[0].name = "mutated-copy"
	graph, _, err = artifact.SyntheticPackageGraph(first.pkg)
	if err != nil {
		t.Fatal(err)
	}
	if graph.Definitions()[0].Name() == "mutated-copy" {
		t.Fatal("package graph projection exposed canonical backing storage")
	}
	facts := artifact.CertifiedFactsForPackage(first.pkg)
	facts[0] = CertifiedFact{}
	if artifact.CertifiedFactsForPackage(first.pkg)[0].Definition().IsZero() {
		t.Fatal("selection-fact projection exposed canonical backing storage")
	}
	census, present := artifact.PackageCensus(first.pkg)
	if !present || len(census.Definitions()) != 1 {
		t.Fatal("provider definition census is absent")
	}
	censusDefinitions := census.Definitions()
	censusDefinitions[0] = identity.DefinitionID{}
	census, _ = artifact.PackageCensus(first.pkg)
	if census.Definitions()[0].IsZero() {
		t.Fatal("provider census exposed canonical backing storage")
	}
}

func TestProviderArtifactRejectsContainerAndShardMutations(t *testing.T) {
	fixture := writeProviderStoreFixture(
		t, filepath.Join(t.TempDir(), "provider.gotots"),
	)
	original, err := os.ReadFile(fixture.path)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("externally-selected-digest", func(t *testing.T) {
		mutated := append([]byte(nil), original...)
		mutated[len(mutated)-1] ^= 0xff
		path := filepath.Join(t.TempDir(), "mutated.gotots")
		if err := os.WriteFile(path, mutated, 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := DecodeProviderArtifact(path, fixture.digest); err == nil {
			t.Fatal("changed container passed its externally selected digest")
		}
	})

	t.Run("package-shard-digest", func(t *testing.T) {
		mutated := append([]byte(nil), original...)
		mutated[len(mutated)-1] ^= 0xff
		path := filepath.Join(t.TempDir(), "mutated.gotots")
		if err := os.WriteFile(path, mutated, 0o644); err != nil {
			t.Fatal(err)
		}
		artifact, err := DecodeProviderArtifact(path, "")
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := artifact.SyntheticPackageGraph(fixture.pkg); err == nil {
			t.Fatal("changed package shard passed its manifest digest")
		}
	})

	t.Run("truncation", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "truncated.gotots")
		if err := os.WriteFile(path, original[:len(original)-1], 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := DecodeProviderArtifact(path, ""); err == nil {
			t.Fatal("truncated provider container was admitted")
		}
	})

	t.Run("manifest-fact-duplicate", func(t *testing.T) {
		manifest := fixture.manifest
		manifest.Packages = append(
			[]providerManifestPackage(nil), manifest.Packages...,
		)
		manifest.Packages[0].Facts = append(
			[]artifactFact(nil), manifest.Packages[0].Facts...,
		)
		manifest.Packages[0].Facts = append(
			manifest.Packages[0].Facts,
			manifest.Packages[0].Facts[0],
		)
		if _, err := decodeMutatedManifestResult(
			t, fixture, manifest,
		); err == nil {
			t.Fatal("duplicate manifest fact survived admission")
		}
	})

	t.Run("manifest-definition-duplicate", func(t *testing.T) {
		manifest := fixture.manifest
		manifest.Packages = append(
			[]providerManifestPackage(nil), manifest.Packages...,
		)
		manifest.Packages[0].Definitions = append(
			[]string(nil), manifest.Packages[0].Definitions...,
		)
		manifest.Packages[0].Definitions = append(
			manifest.Packages[0].Definitions,
			manifest.Packages[0].Definitions[0],
		)
		if _, err := decodeMutatedManifestResult(
			t, fixture, manifest,
		); err == nil {
			t.Fatal("duplicate manifest definition survived admission")
		}
	})

	t.Run("manifest-definition-omission", func(t *testing.T) {
		manifest := fixture.manifest
		manifest.Packages = append(
			[]providerManifestPackage(nil), manifest.Packages...,
		)
		manifest.Packages[0].Definitions = nil
		if _, err := decodeMutatedManifestResult(
			t, fixture, manifest,
		); err == nil {
			t.Fatal("fact without a manifest definition survived admission")
		}
	})

	t.Run("manifest-input-digest", func(t *testing.T) {
		manifest := fixture.manifest
		manifest.Packages = append(
			[]providerManifestPackage(nil), manifest.Packages...,
		)
		manifest.Packages[0].InputDigest = testProviderDigest("wrong-input")
		artifact := decodeMutatedManifest(t, fixture, manifest)
		if _, _, err := artifact.SyntheticPackageGraph(fixture.pkg); err == nil {
			t.Fatal("manifest input-digest mutation survived lazy admission")
		}
	})

	t.Run("manifest-definition-drift", func(t *testing.T) {
		manifest := fixture.manifest
		manifest.Packages = append(
			[]providerManifestPackage(nil), manifest.Packages...,
		)
		replacement, err := identity.NewSyntheticDefinitionID(
			fixture.pkg,
			identity.SyntheticDefinitionAdapter,
			"differentAdapter",
		)
		if err != nil {
			t.Fatal(err)
		}
		manifest.Packages[0].Definitions = []string{
			replacement.String(),
		}
		manifest.Packages[0].Facts = append(
			[]artifactFact(nil), manifest.Packages[0].Facts...,
		)
		manifest.Packages[0].Facts[0].Definition =
			replacement.String()
		artifact := decodeMutatedManifest(t, fixture, manifest)
		if _, _, err := artifact.SyntheticPackageGraph(
			fixture.pkg,
		); err == nil {
			t.Fatal("manifest definition drift exact-joined its shard")
		}
	})

	t.Run("manifest-file-membership", func(t *testing.T) {
		manifest := fixture.manifest
		manifest.Packages = append(
			[]providerManifestPackage(nil), manifest.Packages...,
		)
		file, err := identity.NewFileID(
			fixture.pkg.Owner(),
			"provider.go",
		)
		if err != nil {
			t.Fatal(err)
		}
		manifest.Packages[0].Files = []string{file.String()}
		artifact := decodeMutatedManifest(t, fixture, manifest)
		if _, _, err := artifact.SyntheticPackageGraph(
			fixture.pkg,
		); err == nil {
			t.Fatal("changed file membership exact-joined its shard")
		}
	})

	t.Run("manifest-package-membership", func(t *testing.T) {
		manifest := fixture.manifest
		manifest.Packages = append(
			append(
				[]providerManifestPackage(nil),
				manifest.Packages...,
			),
			manifest.Packages[0],
		)
		if _, err := decodeMutatedManifestResult(
			t,
			fixture,
			manifest,
		); err == nil {
			t.Fatal("duplicate package membership survived admission")
		}
	})
}

func TestLogicalGraphUsesManifestCensusBeforeShardProjection(t *testing.T) {
	fixture := writeProviderStoreFixture(
		t, filepath.Join(t.TempDir(), "provider.gotots"),
	)
	artifact, err := DecodeProviderArtifact(
		fixture.path,
		fixture.digest,
	)
	if err != nil {
		t.Fatal(err)
	}
	graph := &Graph{
		version: ArtifactVersion,
		packages: []PackageGraph{{
			id: fixture.pkg,
		}},
		projections: []packageProjection{{
			id:                 fixture.pkg,
			certifiedSynthetic: true,
		}},
		provider: artifact,
	}
	if err := sealGraph(graph); err != nil {
		t.Fatal(err)
	}
	if err := sealDefinitionCensus(graph); err != nil {
		t.Fatal(err)
	}
	if artifact.storage.loadedArtifact != nil {
		t.Fatal("definition census loaded a detailed provider shard")
	}
	if len(graph.ResidentDefinitions()) != 0 ||
		len(graph.DefinitionCensus()) != 1 ||
		graph.DefinitionCensus()[0].ID() != fixture.definition {
		t.Fatal("logical graph did not separate resident detail from census")
	}
	if err := Validate(graph); err != nil {
		t.Fatal(err)
	}
	if artifact.storage.loadedArtifact != nil {
		t.Fatal("logical graph validation loaded a detailed provider shard")
	}
	if err := graph.VisitPackages(func(pkg PackageGraph) error {
		if len(pkg.Definitions()) != 1 ||
			pkg.Definitions()[0].ID() != fixture.definition {
			t.Fatal("package projection lost certified detail")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if artifact.storage.loadedPackage != fixture.pkg ||
		artifact.storage.loadedArtifact == nil {
		t.Fatal("package projection did not use the bounded provider cache")
	}
}

func writeProviderStoreFixture(
	t *testing.T,
	path string,
) providerStoreFixture {
	t.Helper()
	module, err := identity.NewModuleID("example.com/provider", "v1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	owner, err := identity.NewModuleOwner(module)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := identity.NewPackageID(owner, "example.com/provider")
	if err != nil {
		t.Fatal(err)
	}
	ownerID, err := SyntheticOwner(pkg, SyntheticOwnerCgoGenerated)
	if err != nil {
		t.Fatal(err)
	}
	definition, err := identity.NewSyntheticDefinitionID(
		pkg, identity.SyntheticDefinitionAdapter, "providerAdapter",
	)
	if err != nil {
		t.Fatal(err)
	}
	header, err := identity.NewHeaderRegionID(definition)
	if err != nil {
		t.Fatal(err)
	}
	boundary, err := identity.NewExecutionBoundaryID(definition)
	if err != nil {
		t.Fatal(err)
	}
	graph := PackageGraph{
		id:        pkg,
		synthetic: []OwnerRegion{{id: ownerID}},
		ownedDefinitions: []ImplementationDefinition{{
			id: definition, owner: ownerID, header: header,
			boundary: boundary, name: "providerAdapter",
		}},
		ownedSites: []DefinitionSite{{
			kind: DefinitionSiteSynthetic, definition: definition,
			owner: ownerID,
		}},
		ownedHeaders: []HeaderRegion{{
			id: header, digest: digestStrings(definition.String(), "header"),
		}},
		ownedBoundaries: []ExecutionBoundary{{
			id: boundary, kind: BoundaryImplicit,
			combinedDigest: digestStrings(definition.String(), "execution"),
			synthetic:      identity.SyntheticDefinitionAdapter,
		}},
	}
	if err := validatePackageGraph(graph); err != nil {
		t.Fatal(err)
	}
	context := providerContextRecord{
		Version:               ProviderArtifactVersion,
		ToolchainFingerprint:  testProviderDigest("toolchain"),
		CatalogFingerprint:    testProviderDigest("catalog"),
		BuildFlagsFingerprint: testProviderDigest("flags"),
		ContractID:            "test@v1",
		ContractFingerprint:   testProviderDigest("contract"),
	}
	admission, err := newProviderAdmission(context)
	if err != nil {
		t.Fatal(err)
	}
	record := encodeSyntheticPackage(graph)
	record.InputDigest = testProviderDigest("package-input")
	canonicalizeArtifactPackage(&record)
	if err := admission.addPackage(record); err != nil {
		t.Fatal(err)
	}
	if err := admission.addFact(artifactFact{
		Definition:     definition.String(),
		Kind:           uint8(contract.SelectionFactCDependent),
		Value:          true,
		ProducerDigest: testProviderDigest("producer"),
		EvidenceDigest: testProviderDigest("evidence"),
	}); err != nil {
		t.Fatal(err)
	}
	artifact, err := admission.finish()
	if err != nil {
		t.Fatal(err)
	}
	shardArtifact := *artifact
	shardArtifact.factsByPackage = map[identity.PackageID][]CertifiedFact{}
	shardArtifact.factCount = 0
	var shard bytes.Buffer
	if err := encodeProviderArtifact(&shard, &shardArtifact); err != nil {
		t.Fatal(err)
	}
	shardBytes := shard.Bytes()
	manifest := providerManifest{
		Version: ProviderArtifactVersion,
		Context: context,
		Packages: []providerManifestPackage{{
			Package: pkg.String(), InputDigest: record.InputDigest,
			Synthetic: true,
			Definitions: []string{
				definition.String(),
			},
			Facts: []artifactFact{{
				Definition:     definition.String(),
				Kind:           uint8(contract.SelectionFactCDependent),
				Value:          true,
				ProducerDigest: testProviderDigest("producer"),
				EvidenceDigest: testProviderDigest("evidence"),
			}},
			ShardBytes:  int64(len(shardBytes)),
			ShardDigest: fmt.Sprintf("%x", sha256.Sum256(shardBytes)),
		}},
	}
	spool, err := os.CreateTemp(t.TempDir(), "provider-shard-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = spool.Close()
		_ = os.Remove(spool.Name())
	})
	if _, err := spool.Write(shardBytes); err != nil {
		t.Fatal(err)
	}
	digest, err := writeProviderContainer(path, spool, manifest)
	if err != nil {
		t.Fatal(err)
	}
	return providerStoreFixture{
		path: path, digest: digest, pkg: pkg, definition: definition,
		context: context, manifest: manifest,
		shard: append([]byte(nil), shardBytes...),
	}
}

func decodeMutatedManifest(
	t *testing.T,
	fixture providerStoreFixture,
	manifest providerManifest,
) *ProviderArtifact {
	t.Helper()
	artifact, err := decodeMutatedManifestResult(
		t, fixture, manifest,
	)
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}

func decodeMutatedManifestResult(
	t *testing.T,
	fixture providerStoreFixture,
	manifest providerManifest,
) (*ProviderArtifact, error) {
	t.Helper()
	spool, err := os.CreateTemp(t.TempDir(), "mutated-shard-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() {
		_ = spool.Close()
		_ = os.Remove(spool.Name())
	})
	if _, err := spool.Write(fixture.shard); err != nil {
		return nil, err
	}
	path := filepath.Join(t.TempDir(), "mutated-manifest.gotots")
	digest, err := writeProviderContainer(path, spool, manifest)
	if err != nil {
		return nil, err
	}
	return DecodeProviderArtifact(path, digest)
}

func testProviderDigest(value string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(value)))
}
