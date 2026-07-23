package structure

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/scope/sourceplan"
	"github.com/tsoniclang/gotots/internal/source"
)

// ProduceProviderPackageArtifact seals one independently verified all-local
// package graph for the disk-backed catalog writer. It never carries another
// package's records.
func ProduceProviderPackageArtifact(
	universe *source.Universe,
	selected contract.Contract,
	plan *sourceplan.Plan,
	packageID identity.PackageID,
	graph *Graph,
	facts []CertifiedFact,
) (*ProviderArtifact, error) {
	if universe == nil || graph == nil || plan == nil ||
		plan.Purpose() != sourceplan.PurposeProviderProduction ||
		packageID.IsZero() {
		return nil, fmt.Errorf(
			"provider package artifact requires universe, package, and graph",
		)
	}
	var loaded *source.LoadedPackage
	for _, pkg := range universe.Packages() {
		if pkg.ID() == packageID {
			loaded = pkg
			break
		}
	}
	if loaded == nil {
		return nil, fmt.Errorf(
			"provider package %s is absent from source", packageID,
		)
	}
	graphPackages := graph.Packages()
	if len(graphPackages) != 1 ||
		graphPackages[0].ID() != packageID {
		return nil, fmt.Errorf(
			"provider package graph is not exactly %s", packageID,
		)
	}
	context := providerContextRecord{
		Version:              ProviderArtifactVersion,
		ToolchainFingerprint: toolchainFingerprint(universe.Toolchain()),
		CatalogFingerprint:   catalog.StructureDigest(),
		BuildFlagsFingerprint: buildFlagsFingerprint(
			universe.Request().BuildFlags,
		),
		ContractID: selected.ID(), ContractFingerprint: selected.Fingerprint(),
	}
	admission, err := newProviderAdmission(context)
	if err != nil {
		return nil, err
	}
	fileGraphs := map[identity.FileID]FileGraph{}
	for _, fileGraph := range graphPackages[0].Files() {
		fileGraphs[fileGraph.Owner().ID().File()] = fileGraph
	}
	record := artifactPackage{Package: packageID.String()}
	syntheticCertified := false
	if decision, present := plan.SyntheticFor(packageID); present &&
		decision.Kind() == sourceplan.KindCertifiedGraph {
		record = encodeSyntheticPackage(graphPackages[0])
		syntheticCertified = true
	}
	record.InputDigest = loaded.ProviderInputFingerprint()
	certifiedFiles := map[identity.FileID]bool{}
	for _, file := range loaded.Files() {
		decision, present := plan.For(file.ID())
		if !present {
			return nil, fmt.Errorf(
				"provider plan omits %s", file.ID(),
			)
		}
		if decision.Kind() != sourceplan.KindCertifiedGraph {
			continue
		}
		fileGraph, present := fileGraphs[file.ID()]
		if !present {
			return nil, fmt.Errorf(
				"provider package graph omits %s", file.ID(),
			)
		}
		encoded := encodeFileGraph(
			fileGraph, file.ByteDigest().String(),
		)
		canonicalizeArtifactFile(&encoded)
		if err := admission.addFile(encoded); err != nil {
			return nil, err
		}
		record.Files = append(record.Files, file.ID().String())
		certifiedFiles[file.ID()] = true
	}
	if len(certifiedFiles) == 0 && !syntheticCertified {
		return nil, fmt.Errorf(
			"provider package %s has no certified records", packageID,
		)
	}
	canonicalizeArtifactPackage(&record)
	if err := admission.addPackage(record); err != nil {
		return nil, err
	}
	for _, fact := range facts {
		definition := fact.Definition()
		belongs := definition.SyntheticRole().Valid() &&
			definition.Package() == packageID &&
			syntheticCertified
		if !definition.File().IsZero() {
			belongs = certifiedFiles[definition.File()]
		}
		if !belongs {
			continue
		}
		if err := admission.addFact(artifactFact{
			Definition:     fact.Definition().String(),
			Kind:           uint8(fact.Kind()),
			Value:          fact.Value(),
			ProducerDigest: fact.ProducerDigest(),
			EvidenceDigest: fact.EvidenceDigest(),
		}); err != nil {
			return nil, err
		}
	}
	return admission.finish()
}

func encodeSyntheticPackage(graph PackageGraph) artifactPackage {
	record := artifactPackage{Package: graph.id.String()}
	owners := map[OwnerRegionID]bool{}
	for _, definition := range graph.ownedDefinitions {
		if !definition.id.SyntheticRole().Valid() {
			continue
		}
		owners[definition.owner] = true
		record.Definitions = append(record.Definitions, artifactDefinition{
			ID: definition.id.String(), Owner: definition.owner.String(),
			Header: definition.header.String(), Boundary: definition.boundary.String(),
			Name: definition.name,
		})
	}
	for owner := range owners {
		record.Owners = append(record.Owners, owner.String())
	}
	sort.Strings(record.Owners)
	for _, site := range graph.ownedSites {
		if site.definition.SyntheticRole().Valid() {
			record.Sites = append(record.Sites, artifactSite{
				Kind:       uint8(site.kind),
				Definition: site.definition.String(), Owner: site.owner.String(),
			})
		}
	}
	for _, header := range graph.ownedHeaders {
		if header.id.Definition().SyntheticRole().Valid() {
			record.Headers = append(record.Headers, artifactHeader{
				ID: header.id.String(), Digest: header.digest,
				Members: occurrenceStrings(header.members),
			})
		}
	}
	for _, boundary := range graph.ownedBoundaries {
		if !boundary.id.Definition().SyntheticRole().Valid() {
			continue
		}
		encoded := artifactBoundary{
			ID: boundary.id.String(), Kind: uint8(boundary.kind),
			CombinedDigest: boundary.combinedDigest,
			Implicit:       uint8(boundary.implicit), Synthetic: uint8(boundary.synthetic),
		}
		record.Boundaries = append(record.Boundaries, encoded)
	}
	return record
}

// VerifyProviderArtifactContext binds an artifact to the exact request.
func VerifyProviderArtifactContext(
	artifact *ProviderArtifact,
	universe *source.Universe,
	selected contract.Contract,
) error {
	if artifact == nil {
		return fmt.Errorf("provider artifact is absent")
	}
	if artifact.toolchainFingerprint != toolchainFingerprint(universe.Toolchain()) ||
		artifact.catalogFingerprint != catalog.StructureDigest() ||
		artifact.buildFlagsFingerprint != buildFlagsFingerprint(
			universe.Request().BuildFlags,
		) ||
		artifact.contractID != selected.ID() ||
		artifact.contractFingerprint != selected.Fingerprint() {
		return fmt.Errorf("provider artifact context does not match the compilation request")
	}
	return nil
}

func toolchainFingerprint(toolchain source.Toolchain) string {
	material := toolchain.BinaryDigest() + "|" + toolchain.Version() + "|" +
		toolchain.GOOS() + "|" + toolchain.GOARCH() + "|" +
		toolchain.Experiments() + "|" + toolchain.GoFlags() + "|" +
		toolchain.CgoEnabled() + "|" +
		toolchain.BuildConfigurationDigest()
	return fmt.Sprintf("%x", sha256.Sum256([]byte(material)))
}

func buildFlagsFingerprint(flags []string) string {
	canonical := strings.Join(flags, "\x00")
	return fmt.Sprintf("%x", sha256.Sum256([]byte(canonical)))
}
