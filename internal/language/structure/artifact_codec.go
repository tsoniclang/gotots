package structure

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/tsoniclang/gotots/internal/identity"
)

const (
	providerRecordContext = "context"
	providerRecordFile    = "file"
	providerRecordPackage = "package"
	providerRecordFact    = "selection-fact"
	providerRecordSeal    = "seal"
)

// encodeProviderArtifact writes one admitted artifact as deterministic
// gzip-compressed canonical JSON lines. It retains at most one transport
// record; the compressed artifact is never accumulated in memory.
func encodeProviderArtifact(
	output io.Writer,
	artifact *ProviderArtifact,
) error {
	if artifact == nil {
		return providerArtifactError("artifact is absent")
	}
	if len(artifact.packageDigests) != 1 {
		return providerArtifactError(
			"provider shard must contain exactly one package context",
		)
	}
	context := artifactContext(artifact)
	if err := validateProviderContext(context); err != nil {
		return err
	}
	compressed := gzip.NewWriter(output)
	compressed.Header.ModTime = time.Unix(0, 0)
	compressed.Header.OS = 255
	contentHash := sha256.New()
	write := func(record providerStreamRecord, sealed bool) error {
		raw, err := json.Marshal(record)
		if err != nil {
			return err
		}
		raw = append(raw, '\n')
		if !sealed {
			if _, err := contentHash.Write(raw); err != nil {
				return err
			}
		}
		_, err = compressed.Write(raw)
		return err
	}
	if err := write(providerStreamRecord{
		Record: providerRecordContext, Context: &context,
	}, false); err != nil {
		return err
	}
	for _, file := range orderedArtifactFiles(artifact) {
		record := encodeFileGraph(
			artifact.fileGraphs[file], artifact.fileDigests[file],
		)
		canonicalizeArtifactFile(&record)
		if err := write(providerStreamRecord{
			Record: providerRecordFile, File: &record,
		}, false); err != nil {
			return err
		}
	}
	for _, pkg := range orderedArtifactPackages(artifact) {
		record := artifactPackage{Package: pkg.String()}
		if graph, present := artifact.packageGraphs[pkg]; present {
			record = encodeSyntheticPackage(graph)
		}
		record.InputDigest = artifact.packageDigests[pkg]
		for file, owner := range artifact.filePackages {
			if owner == pkg {
				record.Files = append(record.Files, file.String())
			}
		}
		canonicalizeArtifactPackage(&record)
		if err := write(providerStreamRecord{
			Record: providerRecordPackage, Package: &record,
		}, false); err != nil {
			return err
		}
	}
	for _, fact := range orderedCertifiedFacts(artifact) {
		record := artifactFact{
			Definition: fact.definition.String(),
			Kind:       uint8(fact.kind), Value: fact.value,
			ProducerDigest: fact.producerDigest,
			EvidenceDigest: fact.evidenceDigest,
		}
		if err := write(providerStreamRecord{
			Record: providerRecordFact, Fact: &record,
		}, false); err != nil {
			return err
		}
	}
	if err := write(providerStreamRecord{
		Record: providerRecordSeal,
		Seal:   fmt.Sprintf("%x", contentHash.Sum(nil)),
	}, true); err != nil {
		return err
	}
	return compressed.Close()
}

// decodeProviderStream validates one package shard's compression, exact
// record schema/order, canonical JSON, stream seal, and typed records while
// retaining at most one transport record.
func decodeProviderStream(
	input io.Reader,
) (*ProviderArtifact, error) {
	compressed, err := gzip.NewReader(input)
	if err != nil {
		return nil, fmt.Errorf(
			"provider artifact compression is invalid: %w", err,
		)
	}
	defer compressed.Close()
	reader := bufio.NewReader(compressed)
	contentHash := sha256.New()
	var admission *providerAdmission
	phase := 0
	previous := ""
	sealed := false
	for {
		line, readErr := reader.ReadBytes('\n')
		if readErr == io.EOF && len(line) == 0 {
			break
		}
		if readErr != nil {
			return nil, providerArtifactError(
				"artifact has a truncated stream record",
			)
		}
		record, err := decodeProviderRecord(line)
		if err != nil {
			return nil, err
		}
		if sealed {
			return nil, providerArtifactError(
				"artifact has records after its seal",
			)
		}
		switch record.Record {
		case providerRecordContext:
			if phase != 0 || !contextRecordShape(record) {
				return nil, invalidProviderRecord(record.Record)
			}
			admission, err = newProviderAdmission(*record.Context)
			phase, previous = 1, ""
		case providerRecordFile:
			if phase < 1 || phase > 1 ||
				!fileRecordShape(record) ||
				record.File.File <= previous {
				return nil, invalidProviderRecord(record.Record)
			}
			err = admission.addFile(*record.File)
			previous = record.File.File
		case providerRecordPackage:
			if phase < 1 || phase > 2 ||
				!packageRecordShape(record) {
				return nil, invalidProviderRecord(record.Record)
			}
			if phase == 1 {
				phase, previous = 2, ""
			}
			if record.Package.Package <= previous {
				return nil, invalidProviderRecord(record.Record)
			}
			err = admission.addPackage(*record.Package)
			previous = record.Package.Package
		case providerRecordFact:
			if phase < 1 || phase > 3 ||
				!factRecordShape(record) {
				return nil, invalidProviderRecord(record.Record)
			}
			if phase < 3 {
				phase, previous = 3, ""
			}
			key := fmt.Sprintf(
				"%s/%03d",
				record.Fact.Definition,
				record.Fact.Kind,
			)
			if key <= previous {
				return nil, invalidProviderRecord(record.Record)
			}
			err = admission.addFact(*record.Fact)
			previous = key
		case providerRecordSeal:
			if phase < 1 || !sealRecordShape(record) ||
				!validSHA256(record.Seal) ||
				record.Seal != fmt.Sprintf(
					"%x", contentHash.Sum(nil),
				) {
				return nil, invalidProviderRecord(record.Record)
			}
			sealed = true
		default:
			return nil, invalidProviderRecord(record.Record)
		}
		if err != nil {
			return nil, err
		}
		if record.Record != providerRecordSeal {
			if _, err := contentHash.Write(line); err != nil {
				return nil, err
			}
		}
	}
	if !sealed || admission == nil {
		return nil, providerArtifactError(
			"artifact has no canonical context/seal",
		)
	}
	return admission.finish()
}

func decodeProviderRecord(
	line []byte,
) (providerStreamRecord, error) {
	decoder := json.NewDecoder(bytes.NewReader(line))
	decoder.DisallowUnknownFields()
	var record providerStreamRecord
	if err := decoder.Decode(&record); err != nil {
		return providerStreamRecord{}, fmt.Errorf(
			"provider artifact record undecodable: %w", err,
		)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return providerStreamRecord{}, providerArtifactError(
			"artifact record has trailing JSON",
		)
	}
	canonical, err := json.Marshal(record)
	if err != nil {
		return providerStreamRecord{}, err
	}
	canonical = append(canonical, '\n')
	if !bytes.Equal(line, canonical) {
		return providerStreamRecord{}, providerArtifactError(
			"artifact record is not canonical JSON",
		)
	}
	return record, nil
}

func contextRecordShape(record providerStreamRecord) bool {
	return record.Context != nil &&
		record.File == nil &&
		record.Package == nil &&
		record.Fact == nil &&
		record.Seal == ""
}

func fileRecordShape(record providerStreamRecord) bool {
	return record.Context == nil &&
		record.File != nil &&
		record.Package == nil &&
		record.Fact == nil &&
		record.Seal == ""
}

func packageRecordShape(record providerStreamRecord) bool {
	return record.Context == nil &&
		record.File == nil &&
		record.Package != nil &&
		record.Fact == nil &&
		record.Seal == ""
}

func factRecordShape(record providerStreamRecord) bool {
	return record.Context == nil &&
		record.File == nil &&
		record.Package == nil &&
		record.Fact != nil &&
		record.Seal == ""
}

func sealRecordShape(record providerStreamRecord) bool {
	return record.Context == nil &&
		record.File == nil &&
		record.Package == nil &&
		record.Fact == nil &&
		record.Seal != ""
}

func invalidProviderRecord(kind string) error {
	return providerArtifactError(
		"artifact record order or shape is invalid at " + kind,
	)
}

func artifactContext(
	artifact *ProviderArtifact,
) providerContextRecord {
	return providerContextRecord{
		Version:               artifact.version,
		ToolchainFingerprint:  artifact.toolchainFingerprint,
		CatalogFingerprint:    artifact.catalogFingerprint,
		BuildFlagsFingerprint: artifact.buildFlagsFingerprint,
		ContractID:            artifact.contractID,
		ContractFingerprint:   artifact.contractFingerprint,
	}
}

func orderedArtifactFiles(
	artifact *ProviderArtifact,
) []identity.FileID {
	out := make([]identity.FileID, 0, len(artifact.fileGraphs))
	for file := range artifact.fileGraphs {
		out = append(out, file)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Compare(out[j]) < 0
	})
	return out
}

func orderedArtifactPackages(
	artifact *ProviderArtifact,
) []identity.PackageID {
	out := make(
		[]identity.PackageID, 0, len(artifact.packageDigests),
	)
	for pkg := range artifact.packageDigests {
		out = append(out, pkg)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Compare(out[j]) < 0
	})
	return out
}

func (a *ProviderArtifact) FileIDs() map[identity.FileID]bool {
	out := make(map[identity.FileID]bool, len(a.filePackages))
	for file := range a.filePackages {
		out[file] = true
	}
	return out
}

func (a *ProviderArtifact) PackageIDs() map[identity.PackageID]bool {
	out := make(map[identity.PackageID]bool, len(a.syntheticPackages))
	for pkg := range a.syntheticPackages {
		out[pkg] = true
	}
	return out
}

func (a *ProviderArtifact) PackageFileIDs(
	pkg identity.PackageID,
) map[identity.FileID]bool {
	out := map[identity.FileID]bool{}
	if a == nil {
		return out
	}
	for _, file := range a.packageFiles[pkg] {
		out[file] = true
	}
	return out
}

func (a *ProviderArtifact) HasPackageFile(
	pkg identity.PackageID,
	file identity.FileID,
) bool {
	return a != nil && a.filePackages[file] == pkg
}

func (a *ProviderArtifact) PackageFileCount(
	pkg identity.PackageID,
) int {
	if a == nil {
		return 0
	}
	return len(a.packageFiles[pkg])
}

func (a *ProviderArtifact) HasSyntheticPackage(
	pkg identity.PackageID,
) bool {
	return a != nil && a.syntheticPackages[pkg]
}

func (a *ProviderArtifact) PackageInputDigest(
	pkg identity.PackageID,
) (string, bool) {
	if a == nil {
		return "", false
	}
	digest, present := a.packageDigests[pkg]
	return digest, present
}

func (a *ProviderArtifact) ContextPackageIDs() map[identity.PackageID]bool {
	out := make(map[identity.PackageID]bool, len(a.packageDigests))
	for pkg := range a.packageDigests {
		out[pkg] = true
	}
	return out
}

func orderedCertifiedFacts(
	a *ProviderArtifact,
) []CertifiedFact {
	if a == nil {
		return nil
	}
	out := make([]CertifiedFact, 0, a.factCount)
	for _, facts := range a.factsByPackage {
		out = append(out, facts...)
	}
	sort.Slice(out, func(i, j int) bool {
		return compareCertifiedFactID(
			certifiedFactID{
				definition: out[i].definition,
				kind:       out[i].kind,
			},
			certifiedFactID{
				definition: out[j].definition,
				kind:       out[j].kind,
			},
		) < 0
	})
	return out
}

func (a *ProviderArtifact) SyntheticPackageGraph(
	pkg identity.PackageID,
) (PackageGraph, bool, error) {
	if a == nil {
		return PackageGraph{}, false, nil
	}
	shard, err := a.packageArtifact(pkg)
	if err != nil {
		return PackageGraph{}, false, err
	}
	graph, present := shard.packageGraphs[pkg]
	return graph, present, nil
}

func (a *ProviderArtifact) FileGraph(
	file identity.FileID,
) (FileGraph, string, bool, error) {
	if a == nil {
		return FileGraph{}, "", false, nil
	}
	pkg, known := a.filePackages[file]
	if !known {
		return FileGraph{}, "", false, nil
	}
	shard, err := a.packageArtifact(pkg)
	if err != nil {
		return FileGraph{}, "", false, err
	}
	graph, present := shard.fileGraphs[file]
	if !present {
		return FileGraph{}, "", false, nil
	}
	return graph, shard.fileDigests[file], true, nil
}

func providerArtifactError(reason string) error {
	return fmt.Errorf("GOTOTS_PROVIDER_ARTIFACT: %s", reason)
}
