package structure

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/scope/contract"
)

type providerAdmission struct {
	artifact    *ProviderArtifact
	definitions map[identity.DefinitionID]bool
}

func newProviderAdmission(
	context providerContextRecord,
) (*providerAdmission, error) {
	if err := validateProviderContext(context); err != nil {
		return nil, err
	}
	return &providerAdmission{
		artifact: &ProviderArtifact{
			version:               context.Version,
			toolchainFingerprint:  context.ToolchainFingerprint,
			catalogFingerprint:    context.CatalogFingerprint,
			buildFlagsFingerprint: context.BuildFlagsFingerprint,
			contractID:            context.ContractID,
			contractFingerprint:   context.ContractFingerprint,
			fileDigests:           map[identity.FileID]string{},
			fileGraphs:            map[identity.FileID]FileGraph{},
			filePackages:          map[identity.FileID]identity.PackageID{},
			packageDigests:        map[identity.PackageID]string{},
			packageGraphs:         map[identity.PackageID]PackageGraph{},
			syntheticPackages:     map[identity.PackageID]bool{},
			factsByID:             map[certifiedFactID]CertifiedFact{},
		},
		definitions: map[identity.DefinitionID]bool{},
	}, nil
}

func validateProviderContext(context providerContextRecord) error {
	if context.Version != ProviderArtifactVersion ||
		!validSHA256(context.ToolchainFingerprint) ||
		!validSHA256(context.CatalogFingerprint) ||
		!validSHA256(context.BuildFlagsFingerprint) ||
		context.ContractID == "" ||
		!validSHA256(context.ContractFingerprint) {
		return providerArtifactError(
			"artifact version or context is invalid",
		)
	}
	return nil
}

func (a *providerAdmission) addFile(record artifactFile) error {
	if record.File == "" ||
		!validSHA256(record.ByteDigest) ||
		!artifactFileIsCanonical(record) {
		return providerArtifactError(
			"artifact has an invalid or noncanonical file " + record.File,
		)
	}
	file, err := identity.ParseFileID(record.File)
	if err != nil {
		return err
	}
	if _, duplicate := a.artifact.fileGraphs[file]; duplicate {
		return providerArtifactError(
			"artifact duplicates file " + record.File,
		)
	}
	graph, err := decodeFileGraph(record)
	if err != nil {
		return fmt.Errorf("%s: %w", record.File, err)
	}
	for _, definition := range graph.definitions {
		if a.definitions[definition.id] {
			return providerArtifactError(
				"definition is stored by multiple graph records " +
					definition.id.String(),
			)
		}
		a.definitions[definition.id] = true
	}
	a.artifact.fileDigests[file] = record.ByteDigest
	a.artifact.fileGraphs[file] = graph
	return nil
}

func (a *providerAdmission) addPackage(
	record artifactPackage,
) error {
	if !validSHA256(record.InputDigest) ||
		!artifactPackageIsCanonical(record) {
		return providerArtifactError(
			"artifact has a noncanonical synthetic package " +
				record.Package,
		)
	}
	pkg, err := identity.ParsePackageID(record.Package)
	if err != nil {
		return err
	}
	if _, duplicate := a.artifact.packageDigests[pkg]; duplicate {
		return providerArtifactError(
			"artifact duplicates package context " + record.Package,
		)
	}
	if len(a.artifact.packageDigests) != 0 {
		return providerArtifactError(
			"provider shard contains multiple package contexts",
		)
	}
	a.artifact.packageDigests[pkg] = record.InputDigest
	for _, fileText := range record.Files {
		file, err := identity.ParseFileID(fileText)
		if err != nil {
			return err
		}
		if _, present := a.artifact.fileGraphs[file]; !present {
			return providerArtifactError(
				"package context names absent file " + fileText,
			)
		}
		if prior, duplicate := a.artifact.filePackages[file]; duplicate {
			return providerArtifactError(
				"file belongs to provider packages " +
					prior.String() + " and " + pkg.String(),
			)
		}
		a.artifact.filePackages[file] = pkg
	}
	if len(record.Definitions) == 0 {
		if len(record.Owners) != 0 ||
			len(record.Sites) != 0 ||
			len(record.Headers) != 0 ||
			len(record.Boundaries) != 0 {
			return providerArtifactError(
				"artifact package context has a partial synthetic graph " +
					record.Package,
			)
		}
		return nil
	}
	graph, err := decodeSyntheticPackage(pkg, record)
	if err != nil {
		return err
	}
	for _, definition := range graph.ownedDefinitions {
		if a.definitions[definition.id] {
			return providerArtifactError(
				"definition is stored by multiple graph records " +
					definition.id.String(),
			)
		}
		a.definitions[definition.id] = true
	}
	a.artifact.packageGraphs[pkg] = graph
	a.artifact.syntheticPackages[pkg] = true
	return nil
}

func (a *providerAdmission) addFact(record artifactFact) error {
	definition, err := identity.ParseDefinitionID(record.Definition)
	if err != nil {
		return err
	}
	kind := contract.SelectionFactKind(record.Kind)
	fact, err := NewCertifiedFact(
		definition,
		kind,
		record.Value,
		record.ProducerDigest,
		record.EvidenceDigest,
	)
	if err != nil {
		return err
	}
	id := certifiedFactID{definition: definition, kind: kind}
	if !a.definitions[definition] {
		return providerArtifactError(
			"selection fact has no certified definition " +
				definition.String(),
		)
	}
	if _, duplicate := a.artifact.factsByID[id]; duplicate {
		return providerArtifactError(
			"artifact duplicates selection fact " +
				definition.String() + "/" + kind.String(),
		)
	}
	a.artifact.factsByID[id] = fact
	return nil
}

func (a *providerAdmission) finish() (*ProviderArtifact, error) {
	if len(a.artifact.filePackages) != len(a.artifact.fileGraphs) {
		return nil, providerArtifactError(
			"one or more files have no package context",
		)
	}
	return a.artifact, nil
}

func validSHA256(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}

type admittedManifestPackage struct {
	id    identity.PackageID
	entry providerManifestPackage
	files []identity.FileID
}

func admitProviderManifest(
	manifest providerManifest,
) (*ProviderArtifact, []admittedManifestPackage, error) {
	if manifest.Version != ProviderArtifactVersion ||
		manifest.Context.Version != ProviderArtifactVersion ||
		(len(manifest.Packages) == 0 && manifest.Packages != nil) {
		return nil, nil, providerArtifactError(
			"provider manifest version or package list is noncanonical",
		)
	}
	admission, err := newProviderAdmission(manifest.Context)
	if err != nil {
		return nil, nil, err
	}
	artifact := admission.artifact
	var admitted []admittedManifestPackage
	previousPackage := ""
	for _, entry := range manifest.Packages {
		if entry.Package <= previousPackage ||
			!validSHA256(entry.InputDigest) ||
			!validSHA256(entry.ShardDigest) ||
			entry.ShardBytes <= 0 ||
			entry.FactCount < 0 ||
			(len(entry.Files) == 0 && entry.Files != nil) ||
			(len(entry.Files) == 0 && !entry.Synthetic) ||
			!sort.StringsAreSorted(entry.Files) {
			return nil, nil, providerArtifactError(
				"provider manifest has a noncanonical package entry",
			)
		}
		packageID, err := identity.ParsePackageID(entry.Package)
		if err != nil {
			return nil, nil, err
		}
		if _, duplicate := artifact.packageDigests[packageID]; duplicate {
			return nil, nil, providerArtifactError(
				"provider manifest duplicates package " + entry.Package,
			)
		}
		artifact.packageDigests[packageID] = entry.InputDigest
		if entry.Synthetic {
			artifact.syntheticPackages[packageID] = true
		}
		record := admittedManifestPackage{id: packageID, entry: entry}
		previousFile := ""
		for _, fileText := range entry.Files {
			if fileText <= previousFile {
				return nil, nil, providerArtifactError(
					"provider manifest has duplicate file " + fileText,
				)
			}
			fileID, err := identity.ParseFileID(fileText)
			if err != nil {
				return nil, nil, err
			}
			if prior, duplicate := artifact.filePackages[fileID]; duplicate {
				return nil, nil, providerArtifactError(
					"provider file belongs to packages " +
						prior.String() + " and " + entry.Package,
				)
			}
			artifact.filePackages[fileID] = packageID
			record.files = append(record.files, fileID)
			previousFile = fileText
		}
		artifact.manifestFacts += entry.FactCount
		admitted = append(admitted, record)
		previousPackage = entry.Package
	}
	return artifact, admitted, nil
}
