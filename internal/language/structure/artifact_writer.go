package structure

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/source"
)

// ProviderWriteResult reports exact physical and logical artifact
// denominators after atomic publication.
type ProviderWriteResult struct {
	Digest            string
	EncodedBytes      int64
	PackageContexts   int
	Files             int
	SyntheticPackages int
	Facts             int
}

// ProviderArtifactWriter owns one disk-backed package-sharded publication.
// Append order is canonical package identity order; at most one package graph
// is resident outside the caller at any time.
type ProviderArtifactWriter struct {
	path      string
	spool     *os.File
	spoolPath string
	manifest  providerManifest
	previous  string
	closed    bool
	result    ProviderWriteResult
}

func NewProviderArtifactWriter(
	universe *source.Universe,
	selected contract.Contract,
	path string,
) (*ProviderArtifactWriter, error) {
	if universe == nil || selected.ID() == "" || path == "" {
		return nil, providerArtifactError(
			"provider writer requires universe, contract, and output",
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
	if err := validateProviderContext(context); err != nil {
		return nil, err
	}
	spool, err := os.CreateTemp(
		filepath.Dir(path), ".gotots-provider-shards-*",
	)
	if err != nil {
		return nil, fmt.Errorf("provider shard spool: %w", err)
	}
	return &ProviderArtifactWriter{
		path: path, spool: spool, spoolPath: spool.Name(),
		manifest: providerManifest{
			Version: ProviderArtifactVersion,
			Context: context,
		},
	}, nil
}

func (w *ProviderArtifactWriter) Append(
	artifact *ProviderArtifact,
) error {
	if w == nil || w.closed || w.spool == nil {
		return providerArtifactError("provider writer is closed")
	}
	if artifact == nil ||
		artifactContext(artifact) != w.manifest.Context ||
		len(artifact.packageDigests) != 1 {
		return providerArtifactError(
			"provider writer requires one context-matched package shard",
		)
	}
	var packageID identity.PackageID
	for candidate := range artifact.packageDigests {
		packageID = candidate
	}
	if packageID.String() <= w.previous {
		return providerArtifactError(
			"provider packages are not appended in canonical order",
		)
	}
	start, err := w.spool.Seek(0, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("provider shard offset: %w", err)
	}
	shardHash := sha256.New()
	if err := encodeProviderArtifact(
		io.MultiWriter(w.spool, shardHash), artifact,
	); err != nil {
		return err
	}
	end, err := w.spool.Seek(0, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("provider shard size: %w", err)
	}
	entry := providerManifestPackage{
		Package:     packageID.String(),
		InputDigest: artifact.packageDigests[packageID],
		Synthetic:   artifact.syntheticPackages[packageID],
		FactCount:   artifact.FactCount(),
		ShardBytes:  end - start,
		ShardDigest: fmt.Sprintf("%x", shardHash.Sum(nil)),
	}
	for file, owner := range artifact.filePackages {
		if owner != packageID {
			return providerArtifactError(
				"provider shard contains a foreign package file",
			)
		}
		entry.Files = append(entry.Files, file.String())
	}
	sort.Strings(entry.Files)
	w.manifest.Packages = append(w.manifest.Packages, entry)
	w.previous = packageID.String()
	w.result.PackageContexts++
	w.result.Files += len(entry.Files)
	if entry.Synthetic {
		w.result.SyntheticPackages++
	}
	w.result.Facts += entry.FactCount
	return nil
}

func (w *ProviderArtifactWriter) Finish() (
	ProviderWriteResult,
	error,
) {
	if w == nil || w.closed || w.spool == nil {
		return ProviderWriteResult{}, providerArtifactError(
			"provider writer is closed",
		)
	}
	manifest, err := json.Marshal(w.manifest)
	if err != nil {
		return ProviderWriteResult{}, fmt.Errorf(
			"provider manifest encoding failed: %w", err,
		)
	}
	encodedBytes := int64(providerContainerHeaderBytes + len(manifest))
	for _, entry := range w.manifest.Packages {
		encodedBytes += entry.ShardBytes
	}
	digest, err := writeProviderContainer(
		w.path, w.spool, w.manifest,
	)
	if err != nil {
		return ProviderWriteResult{}, err
	}
	w.result.Digest = digest
	w.result.EncodedBytes = encodedBytes
	w.closed = true
	_ = w.spool.Close()
	_ = os.Remove(w.spoolPath)
	w.spool = nil
	return w.result, nil
}

// ManifestArtifact returns the immutable logical index accumulated so far
// without reading or publishing shard bytes. It exists for the independent
// pre-publication manifest exact-join.
func (w *ProviderArtifactWriter) ManifestArtifact() (
	*ProviderArtifact,
	error,
) {
	if w == nil || w.closed {
		return nil, providerArtifactError("provider writer is closed")
	}
	artifact, _, err := admitProviderManifest(w.manifest)
	if err != nil {
		return nil, err
	}
	return artifact, nil
}

func (w *ProviderArtifactWriter) Abort() {
	if w == nil || w.closed {
		return
	}
	w.closed = true
	if w.spool != nil {
		_ = w.spool.Close()
	}
	if w.spoolPath != "" {
		_ = os.Remove(w.spoolPath)
	}
	w.spool = nil
}
