package structure

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/tsoniclang/gotots/internal/identity"
)

var providerContainerMagic = [8]byte{'G', 'O', 'T', 'O', 'T', 'S', 'P', '1'}

const providerContainerHeaderBytes = 16

func writeProviderContainer(
	path string,
	spool *os.File,
	manifest providerManifest,
) (string, error) {
	manifestRaw, err := json.Marshal(manifest)
	if err != nil {
		return "", err
	}
	if err := spool.Sync(); err != nil {
		return "", fmt.Errorf("provider shard sync: %w", err)
	}
	if _, err := spool.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("provider shard rewind: %w", err)
	}
	temp, err := os.CreateTemp(
		filepath.Dir(path), ".gotots-provider-*",
	)
	if err != nil {
		return "", fmt.Errorf("provider artifact temp file: %w", err)
	}
	tempPath := temp.Name()
	cleanup := func() {
		_ = temp.Close()
		_ = os.Remove(tempPath)
	}
	fileHash := sha256.New()
	output := io.MultiWriter(temp, fileHash)
	if _, err := output.Write(providerContainerMagic[:]); err != nil {
		cleanup()
		return "", err
	}
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(manifestRaw)))
	if _, err := output.Write(length[:]); err != nil {
		cleanup()
		return "", err
	}
	if _, err := output.Write(manifestRaw); err != nil {
		cleanup()
		return "", err
	}
	if _, err := io.Copy(output, spool); err != nil {
		cleanup()
		return "", fmt.Errorf("provider shard copy: %w", err)
	}
	if err := syncAndReplace(temp, tempPath, path); err != nil {
		cleanup()
		return "", err
	}
	return fmt.Sprintf("%x", fileHash.Sum(nil)), nil
}

// DecodeProviderArtifact validates the externally selected container and
// admits only its canonical manifest. Package shards remain disk-backed until
// the source plan requests them.
func DecodeProviderArtifact(
	path string,
	expectedDigest string,
) (*ProviderArtifact, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("provider artifact path unresolvable: %w", err)
	}
	file, err := os.Open(absolutePath)
	if err != nil {
		return nil, fmt.Errorf("provider artifact unreadable: %w", err)
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("provider artifact stat failed: %w", err)
	}
	if expectedDigest != "" {
		hash := sha256.New()
		if _, err := io.Copy(hash, file); err != nil {
			return nil, fmt.Errorf("provider artifact digest failed: %w", err)
		}
		if fmt.Sprintf("%x", hash.Sum(nil)) != expectedDigest {
			return nil, providerArtifactError(
				"artifact file digest does not match selected evidence",
			)
		}
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("provider artifact rewind failed: %w", err)
		}
	}
	manifest, manifestBytes, err := readProviderManifest(file, stat.Size())
	if err != nil {
		return nil, err
	}
	artifact, admitted, err := admitProviderManifest(manifest)
	if err != nil {
		return nil, err
	}
	artifact.storage = &providerStorage{
		path: absolutePath, shards: map[identity.PackageID]providerShard{},
		loaded: map[identity.PackageID]*ProviderArtifact{},
	}
	offset := int64(providerContainerHeaderBytes) + manifestBytes
	for _, record := range admitted {
		entry := record.entry
		if entry.ShardBytes > stat.Size()-offset {
			return nil, providerArtifactError(
				"provider shard extends beyond its container",
			)
		}
		artifact.storage.shards[record.id] = providerShard{
			offset: offset, bytes: entry.ShardBytes,
			digest: entry.ShardDigest, factCount: entry.FactCount,
			synthetic: entry.Synthetic, files: record.files,
			inputDigest: entry.InputDigest,
		}
		artifact.storage.factCount += entry.FactCount
		offset += entry.ShardBytes
	}
	if offset != stat.Size() {
		return nil, providerArtifactError(
			"provider container size disagrees with its manifest",
		)
	}
	return artifact, nil
}

func readProviderManifest(
	input io.Reader,
	fileBytes int64,
) (providerManifest, int64, error) {
	var header [providerContainerHeaderBytes]byte
	if _, err := io.ReadFull(input, header[:]); err != nil {
		return providerManifest{}, 0, providerArtifactError(
			"provider container header is truncated",
		)
	}
	if !bytes.Equal(header[:8], providerContainerMagic[:]) {
		return providerManifest{}, 0, providerArtifactError(
			"provider container magic is invalid",
		)
	}
	length := binary.BigEndian.Uint64(header[8:])
	if length == 0 ||
		length > uint64(fileBytes-providerContainerHeaderBytes) ||
		length > uint64(^uint(0)>>1) {
		return providerManifest{}, 0, providerArtifactError(
			"provider manifest length is invalid",
		)
	}
	raw := make([]byte, int(length))
	if _, err := io.ReadFull(input, raw); err != nil {
		return providerManifest{}, 0, providerArtifactError(
			"provider manifest is truncated",
		)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var manifest providerManifest
	if err := decoder.Decode(&manifest); err != nil {
		return providerManifest{}, 0, providerArtifactError(
			"provider manifest is undecodable",
		)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return providerManifest{}, 0, providerArtifactError(
			"provider manifest has trailing JSON",
		)
	}
	canonical, err := json.Marshal(manifest)
	if err != nil || !bytes.Equal(raw, canonical) {
		return providerManifest{}, 0, providerArtifactError(
			"provider manifest is not canonical JSON",
		)
	}
	return manifest, int64(length), nil
}

func syncAndReplace(
	temp *os.File,
	tempPath string,
	path string,
) error {
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("provider artifact sync failed: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("provider artifact close failed: %w", err)
	}
	if err := os.Chmod(tempPath, 0o644); err != nil {
		return fmt.Errorf("provider artifact mode failed: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("provider artifact replace failed: %w", err)
	}
	directory, err := os.Open(filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("provider artifact directory open failed: %w", err)
	}
	if err := directory.Sync(); err != nil {
		_ = directory.Close()
		return fmt.Errorf("provider artifact directory sync failed: %w", err)
	}
	return directory.Close()
}
