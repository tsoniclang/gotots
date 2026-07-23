package semantic

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

var providerArtifactMagic = [8]byte{
	'G', 'T', 'S', 'S', 'E', 'M', '0', '1',
}

const providerArtifactHeaderBytes = 16

type ProviderArtifactContext struct {
	ToolchainDigest     string
	ConfigurationDigest string
	ContractID          string
	ContractFingerprint string
}

type ProviderWriteResult struct {
	Path                  string
	Digest                string
	EncodedBytes          int64
	Packages              int
	Definitions           int
	Resolutions           int
	Declarations          int
	Bindings              int
	Types                 int
	Operations            int
	Unsupported           int
	TypeClosureDuplicates int
	LargestShardBytes     int64
	LargestPackageRecords int
	metrics               Metrics
}

func (result ProviderWriteResult) Metrics() Metrics {
	return result.metrics.clone()
}

type ProviderArtifactWriter struct {
	path       string
	spool      *os.File
	spoolPath  string
	context    providerContext
	manifest   providerManifest
	previous   string
	closed     bool
	result     ProviderWriteResult
	metrics    Metrics
	typeOwners map[identity.SemanticTypeID]int
}

func NewProviderArtifactWriter(
	context ProviderArtifactContext,
	path string,
) (*ProviderArtifactWriter, error) {
	if path == "" ||
		!fullDigest(context.ToolchainDigest) ||
		!fullDigest(context.ConfigurationDigest) ||
		context.ContractID == "" ||
		!fullDigest(context.ContractFingerprint) {
		return nil, &artifactError{
			reason: "writer requires output and exact semantic context",
		}
	}
	spool, err := os.CreateTemp(
		filepath.Dir(path), ".gotots-semantic-shards-*",
	)
	if err != nil {
		return nil, fmt.Errorf(
			"semantic provider shard spool: %w", err,
		)
	}
	wireContext := providerContext{
		Version:             ProviderArtifactVersion,
		ToolchainDigest:     context.ToolchainDigest,
		ConfigurationDigest: context.ConfigurationDigest,
		ContractID:          context.ContractID,
		ContractFingerprint: context.ContractFingerprint,
	}
	return &ProviderArtifactWriter{
		path:       path,
		spool:      spool,
		spoolPath:  spool.Name(),
		context:    wireContext,
		manifest:   providerManifest{Context: wireContext},
		typeOwners: map[identity.SemanticTypeID]int{},
		result:     ProviderWriteResult{Path: path},
	}, nil
}

func (writer *ProviderArtifactWriter) Append(pkg Package) error {
	if writer == nil || writer.closed || writer.spool == nil {
		return &artifactError{reason: "semantic provider writer is closed"}
	}
	if pkg.ID().String() <= writer.previous {
		return &artifactError{
			reason: "semantic provider packages are not canonical",
		}
	}
	authority, err := checkerPackageAuthority(pkg)
	if err != nil {
		return err
	}
	if authority.ToolchainDigest() !=
		writer.context.ToolchainDigest ||
		authority.Configuration() !=
			writer.context.ConfigurationDigest {
		return &artifactError{
			reason: "semantic package checker context disagrees with writer",
		}
	}
	offset, err := writer.spool.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	hash := sha256.New()
	encodedBytes, err := writeProviderShard(
		io.MultiWriter(writer.spool, hash), pkg,
	)
	if err != nil {
		_ = writer.spool.Truncate(offset)
		_, _ = writer.spool.Seek(offset, io.SeekStart)
		return err
	}
	var packageMetrics Metrics
	if err := packageMetrics.addMeasuredPackage(
		pkg, encodedBytes,
	); err != nil {
		_ = writer.spool.Truncate(offset)
		_, _ = writer.spool.Seek(offset, io.SeekStart)
		return err
	}
	entry := providerManifestPackage{
		Package:          pkg.ID().String(),
		Provenance:       uint8(pkg.Provenance()),
		PackageInput:     authority.PackageInput(),
		Structure:        authority.StructureDigest(),
		Selection:        authority.SelectionDigest(),
		ShardOffset:      offset,
		ShardBytes:       encodedBytes,
		ShardDigest:      fmt.Sprintf("%x", hash.Sum(nil)),
		DefinitionCount:  len(pkg.definitions),
		ResolutionCount:  len(pkg.resolutions),
		DeclarationCount: len(pkg.declarations),
		BindingCount:     len(pkg.bindings),
		TypeCount:        len(pkg.types),
		OperationCount:   len(pkg.operations),
		UnsupportedCount: len(pkg.unsupported),
	}
	for _, definition := range pkg.definitions {
		entry.Definitions = append(
			entry.Definitions, definition.Definition().String(),
		)
	}
	for _, declaration := range pkg.declarations {
		entry.Declarations = append(
			entry.Declarations, declaration.ID().String(),
		)
	}
	sort.Strings(entry.Definitions)
	sort.Strings(entry.Declarations)
	writer.manifest.Packages = append(
		writer.manifest.Packages, entry,
	)
	writer.previous = pkg.ID().String()
	writer.metrics.merge(packageMetrics)
	for _, record := range pkg.types {
		writer.typeOwners[record.ID()]++
	}
	return nil
}

func (writer *ProviderArtifactWriter) Finish(
	structuralArtifactDigest string,
) (ProviderWriteResult, error) {
	if writer == nil || writer.closed || writer.spool == nil {
		return ProviderWriteResult{}, &artifactError{
			reason: "semantic provider writer is closed",
		}
	}
	if !fullDigest(structuralArtifactDigest) {
		return ProviderWriteResult{}, &artifactError{
			reason: "semantic provider requires structural artifact digest",
		}
	}
	writer.context.StructuralArtifactDigest =
		structuralArtifactDigest
	writer.manifest.Context = writer.context
	manifestBytes, err := json.Marshal(writer.manifest)
	if err != nil {
		return ProviderWriteResult{}, err
	}
	if err := writer.spool.Sync(); err != nil {
		return ProviderWriteResult{}, err
	}
	if _, err := writer.spool.Seek(0, io.SeekStart); err != nil {
		return ProviderWriteResult{}, err
	}
	output, err := os.CreateTemp(
		filepath.Dir(writer.path), ".gotots-semantic-artifact-*",
	)
	if err != nil {
		return ProviderWriteResult{}, err
	}
	outputPath := output.Name()
	published := false
	defer func() {
		_ = output.Close()
		if !published {
			_ = os.Remove(outputPath)
		}
	}()
	hash := sha256.New()
	target := io.MultiWriter(output, hash)
	header := make([]byte, providerArtifactHeaderBytes)
	copy(header, providerArtifactMagic[:])
	binary.BigEndian.PutUint64(
		header[8:], uint64(len(manifestBytes)),
	)
	if _, err := target.Write(header); err != nil {
		return ProviderWriteResult{}, err
	}
	if _, err := target.Write(manifestBytes); err != nil {
		return ProviderWriteResult{}, err
	}
	if _, err := io.Copy(target, writer.spool); err != nil {
		return ProviderWriteResult{}, err
	}
	if err := output.Sync(); err != nil {
		return ProviderWriteResult{}, err
	}
	if err := output.Close(); err != nil {
		return ProviderWriteResult{}, err
	}
	if err := os.Rename(outputPath, writer.path); err != nil {
		return ProviderWriteResult{}, err
	}
	published = true
	writer.result.Digest = fmt.Sprintf("%x", hash.Sum(nil))
	writer.result.EncodedBytes =
		int64(providerArtifactHeaderBytes + len(manifestBytes))
	for _, entry := range writer.manifest.Packages {
		writer.result.EncodedBytes += entry.ShardBytes
	}
	for _, count := range writer.typeOwners {
		if count > 1 {
			writer.result.TypeClosureDuplicates += count - 1
		}
	}
	writer.result.Packages = writer.metrics.Packages()
	writer.result.Definitions = writer.metrics.Definitions()
	writer.result.Resolutions = writer.metrics.Resolutions()
	writer.result.Declarations = writer.metrics.Declarations()
	writer.result.Bindings = writer.metrics.Bindings()
	writer.result.Types = writer.metrics.Types()
	writer.result.Operations = writer.metrics.Operations()
	writer.result.Unsupported = writer.metrics.Unsupported()
	writer.result.metrics = writer.metrics.clone()
	if packages := writer.metrics.LargestPackages(); len(packages) != 0 {
		writer.result.LargestShardBytes =
			packages[0].EncodedBytes
	}
	writer.result.LargestPackageRecords =
		writer.metrics.LargestPackageRecords()
	writer.closed = true
	_ = writer.spool.Close()
	_ = os.Remove(writer.spoolPath)
	writer.spool = nil
	return writer.result, nil
}

func (writer *ProviderArtifactWriter) Abort() {
	if writer == nil || writer.closed {
		return
	}
	writer.closed = true
	if writer.spool != nil {
		_ = writer.spool.Close()
		_ = os.Remove(writer.spoolPath)
		writer.spool = nil
	}
}

func checkerPackageAuthority(pkg Package) (Authority, error) {
	var selected Authority
	admit := func(authority Authority) error {
		if authority.Kind() != AuthorityChecker {
			return &artifactError{
				reason: "semantic production package is not checker-owned",
			}
		}
		if !selected.Valid() {
			selected = authority
			return nil
		}
		if selected != authority {
			return &artifactError{
				reason: "semantic package has multiple checker authorities",
			}
		}
		return nil
	}
	for _, record := range pkg.definitions {
		if err := admit(record.Authority()); err != nil {
			return Authority{}, err
		}
	}
	for _, record := range pkg.declarations {
		if err := admit(record.Authority()); err != nil {
			return Authority{}, err
		}
	}
	for _, record := range pkg.bindings {
		if err := admit(record.Authority()); err != nil {
			return Authority{}, err
		}
	}
	for _, record := range pkg.typeWitnesses {
		if err := admit(record.Authority()); err != nil {
			return Authority{}, err
		}
	}
	for _, record := range pkg.unsupported {
		if err := admit(record.Authority()); err != nil {
			return Authority{}, err
		}
	}
	if !selected.Valid() {
		return Authority{}, &artifactError{
			reason: "semantic package carries no checker authority",
		}
	}
	return selected, nil
}
