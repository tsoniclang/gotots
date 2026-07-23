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
	facts       map[certifiedFactID]bool
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
			packageFiles:          map[identity.PackageID][]identity.FileID{},
			packageDigests:        map[identity.PackageID]string{},
			packageGraphs:         map[identity.PackageID]PackageGraph{},
			packageCensus:         map[identity.PackageID]ProviderPackageCensus{},
			syntheticPackages:     map[identity.PackageID]bool{},
			factsByPackage:        map[identity.PackageID][]CertifiedFact{},
		},
		definitions: map[identity.DefinitionID]bool{},
		facts:       map[certifiedFactID]bool{},
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
		a.artifact.packageFiles[pkg] = append(
			a.artifact.packageFiles[pkg],
			file,
		)
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
	if a.facts[id] {
		return providerArtifactError(
			"artifact duplicates selection fact " +
				definition.String() + "/" + kind.String(),
		)
	}
	packageID := a.artifact.packageForDefinition(definition)
	if packageID.IsZero() {
		return providerArtifactError(
			"selection fact has no package " + definition.String(),
		)
	}
	a.facts[id] = true
	a.artifact.factsByPackage[packageID] = append(
		a.artifact.factsByPackage[packageID],
		fact,
	)
	a.artifact.factCount++
	return nil
}

func (a *providerAdmission) finish() (*ProviderArtifact, error) {
	if len(a.artifact.packageDigests) != 1 {
		return nil, providerArtifactError(
			"provider shard must contain exactly one package context",
		)
	}
	if len(a.artifact.filePackages) != len(a.artifact.fileGraphs) {
		return nil, providerArtifactError(
			"one or more files have no package context",
		)
	}
	if err := sealProviderPackageCensus(a.artifact); err != nil {
		return nil, err
	}
	pkg, err := admittedProviderPackage(a.artifact)
	if err != nil {
		return nil, err
	}
	if err := validateCompletePackage(pkg); err != nil {
		return nil, providerArtifactError(
			"package graph is structurally invalid: " + err.Error(),
		)
	}
	return a.artifact, nil
}

func admittedProviderPackage(
	artifact *ProviderArtifact,
) (PackageGraph, error) {
	var packageID identity.PackageID
	for candidate := range artifact.packageDigests {
		packageID = candidate
	}
	pkg := PackageGraph{id: packageID}
	for _, fileID := range artifact.packageFiles[packageID] {
		file, present := artifact.fileGraphs[fileID]
		if !present {
			return PackageGraph{}, providerArtifactError(
				"provider package omits file graph " + fileID.String(),
			)
		}
		pkg.files = append(pkg.files, file)
	}
	if synthetic, present := artifact.packageGraphs[packageID]; present {
		pkg.synthetic = append(pkg.synthetic, synthetic.synthetic...)
		pkg.ownedDefinitions = append(
			pkg.ownedDefinitions,
			synthetic.ownedDefinitions...,
		)
		pkg.ownedSites = append(pkg.ownedSites, synthetic.ownedSites...)
		pkg.ownedHeaders = append(
			pkg.ownedHeaders,
			synthetic.ownedHeaders...,
		)
		pkg.ownedBoundaries = append(
			pkg.ownedBoundaries,
			synthetic.ownedBoundaries...,
		)
	}
	if len(pkg.files) == 0 && len(pkg.synthetic) == 0 {
		return PackageGraph{}, providerArtifactError(
			"provider package has no structural owner " + packageID.String(),
		)
	}
	return pkg, nil
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
			entry.HeaderOccurrences < 0 ||
			entry.BoundaryEntries < 0 ||
			(len(entry.Files) == 0 && entry.Files != nil) ||
			(len(entry.Definitions) == 0 && entry.Definitions != nil) ||
			(len(entry.Facts) == 0 && entry.Facts != nil) ||
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
			artifact.packageFiles[packageID] = append(
				artifact.packageFiles[packageID],
				fileID,
			)
			record.files = append(record.files, fileID)
			previousFile = fileText
		}
		definitions := make(
			[]identity.DefinitionID,
			0,
			len(entry.Definitions),
		)
		previousDefinition := ""
		for _, definitionText := range entry.Definitions {
			if definitionText <= previousDefinition {
				return nil, nil, providerArtifactError(
					"provider manifest definitions are noncanonical",
				)
			}
			definition, err := identity.ParseDefinitionID(
				definitionText,
			)
			if err != nil {
				return nil, nil, err
			}
			belongs := !definition.File().IsZero() &&
				artifact.filePackages[definition.File()] == packageID
			if definition.SyntheticRole().Valid() {
				belongs = definition.Package() == packageID &&
					entry.Synthetic
			}
			if !belongs {
				return nil, nil, providerArtifactError(
					"provider manifest definition has no package authority " +
						definition.String(),
				)
			}
			if admission.definitions[definition] {
				return nil, nil, providerArtifactError(
					"provider manifest duplicates definition " +
						definition.String(),
				)
			}
			admission.definitions[definition] = true
			definitions = append(definitions, definition)
			previousDefinition = definitionText
		}
		census, err := newProviderPackageCensus(
			packageID,
			definitions,
			entry.HeaderOccurrences,
			entry.BoundaryEntries,
		)
		if err != nil {
			return nil, nil, err
		}
		artifact.packageCensus[packageID] = census
		previousFact := ""
		for _, encoded := range entry.Facts {
			definition, err := identity.ParseDefinitionID(
				encoded.Definition,
			)
			if err != nil {
				return nil, nil, err
			}
			kind := contract.SelectionFactKind(encoded.Kind)
			fact, err := NewCertifiedFact(
				definition,
				kind,
				encoded.Value,
				encoded.ProducerDigest,
				encoded.EvidenceDigest,
			)
			if err != nil {
				return nil, nil, err
			}
			key := fmt.Sprintf(
				"%s/%03d", definition, uint8(kind),
			)
			if key <= previousFact {
				return nil, nil, providerArtifactError(
					"provider manifest facts are noncanonical",
				)
			}
			previousFact = key
			if !admission.definitions[definition] {
				return nil, nil, providerArtifactError(
					"provider manifest fact has no definition " +
						definition.String(),
				)
			}
			id := certifiedFactID{
				definition: definition,
				kind:       kind,
			}
			if admission.facts[id] {
				return nil, nil, providerArtifactError(
					"provider manifest duplicates fact " + key,
				)
			}
			admission.facts[id] = true
			artifact.factsByPackage[packageID] = append(
				artifact.factsByPackage[packageID],
				fact,
			)
			artifact.factCount++
		}
		admitted = append(admitted, record)
		previousPackage = entry.Package
	}
	return artifact, admitted, nil
}
