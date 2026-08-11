package toolchain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
)

type rootPublicationHook func(candidateRoot string) error

func sealRootContract(
	sourceRoot string,
	sourceToolDirectory string,
	cacheRoot string,
) (rootContract, error) {
	return sealRootContractWithHook(sourceRoot, sourceToolDirectory, cacheRoot, nil)
}

func sealRootContractWithHook(
	sourceRoot string,
	sourceToolDirectory string,
	cacheRoot string,
	hook rootPublicationHook,
) (rootContract, error) {
	census := &rootInspectionCensus{}
	toolRelative, err := filepath.Rel(sourceRoot, sourceToolDirectory)
	if err != nil || !validRootPath(filepath.ToSlash(toolRelative), false) {
		return rootContract{}, fmt.Errorf("selected Go tool directory is outside GOROOT")
	}
	if withinRoot(sourceRoot, cacheRoot) {
		return rootContract{}, fmt.Errorf("selected tool cache is inside GOROOT")
	}
	manifest, err := inspectRootManifest(sourceRoot, toolRelative, census)
	if err != nil {
		return rootContract{}, err
	}
	manifestPayload, rootDigest, manifestDigest, err := encodeRootManifest(manifest)
	if err != nil {
		return rootContract{}, err
	}
	rootStore := filepath.Join(cacheRoot, "go-roots")
	if err := os.MkdirAll(rootStore, 0o755); err != nil {
		return rootContract{}, err
	}
	target := filepath.Join(rootStore, rootDigest)
	if _, err := os.Stat(target); err == nil {
		return openRootContract(target, &manifest, rootDigest, census)
	} else if !os.IsNotExist(err) {
		return rootContract{}, err
	}
	candidate, err := os.MkdirTemp(rootStore, ".candidate-")
	if err != nil {
		return rootContract{}, err
	}
	published := false
	defer func() {
		if !published {
			_ = os.RemoveAll(candidate)
		}
	}()
	candidateRoot := filepath.Join(candidate, "root")
	if err := copyRootSnapshot(sourceRoot, candidateRoot, manifest); err != nil {
		return rootContract{}, err
	}
	afterCopy, err := inspectRootManifest(sourceRoot, toolRelative, census)
	if err != nil {
		return rootContract{}, err
	}
	if !equalRootManifests(manifest, afterCopy) {
		return rootContract{}, fmt.Errorf("selected Go root changed while sealing")
	}
	if err := writeSnapshotMetadata(candidate, manifestPayload, rootDigest, manifestDigest); err != nil {
		return rootContract{}, err
	}
	if hook != nil {
		if err := hook(candidateRoot); err != nil {
			return rootContract{}, err
		}
	}
	if _, _, err := verifyRootSnapshot(candidate, &manifest, rootDigest, census); err != nil {
		return rootContract{}, fmt.Errorf("verify Go root candidate: %w", err)
	}
	if err := os.Rename(candidate, target); err != nil {
		if _, statErr := os.Stat(target); statErr != nil {
			return rootContract{}, errors.Join(err, statErr)
		}
		return openRootContract(target, &manifest, rootDigest, census)
	}
	published = true
	if err := syncDirectory(rootStore); err != nil {
		return rootContract{}, err
	}
	return openRootContract(target, &manifest, rootDigest, census)
}

func openRootContract(
	container string,
	expected *rootManifest,
	rootDigest string,
	census *rootInspectionCensus,
) (rootContract, error) {
	document, manifestDigest, err := verifyRootSnapshot(container, expected, rootDigest, census)
	if err != nil {
		return rootContract{}, err
	}
	root := filepath.Join(container, "root")
	rootInfo, err := os.Stat(root)
	if err != nil {
		return rootContract{}, err
	}
	sealInfo, err := os.Stat(filepath.Join(container, "seal.json"))
	if err != nil {
		return rootContract{}, err
	}
	contract := rootContract{
		root: root, toolDirectory: filepath.Join(root, filepath.FromSlash(document.Manifest.ToolDirectory)),
		container: container, rootDigest: rootDigest, manifestDigest: manifestDigest,
		rootInfo: rootInfo, sealInfo: sealInfo, census: census,
	}
	if err := contract.VerifyHandle(); err != nil {
		return rootContract{}, err
	}
	return contract, nil
}

func verifyRootSnapshot(
	container string,
	expected *rootManifest,
	rootDigest string,
	census *rootInspectionCensus,
) (rootManifestDocument, string, error) {
	seal, err := readRootSeal(filepath.Join(container, "seal.json"))
	if err != nil {
		return rootManifestDocument{}, "", err
	}
	payload, err := os.ReadFile(filepath.Join(container, "manifest.json"))
	if err != nil {
		return rootManifestDocument{}, "", err
	}
	manifestDigest := digestBytes(payload)
	if seal.RootDigest != rootDigest || seal.ManifestDigest != manifestDigest {
		return rootManifestDocument{}, "", fmt.Errorf("sealed Go root metadata differs")
	}
	document, err := decodeRootManifest(payload)
	if err != nil {
		return rootManifestDocument{}, "", err
	}
	if document.RootDigest != rootDigest || expected != nil && !equalRootManifests(*expected, document.Manifest) {
		return rootManifestDocument{}, "", fmt.Errorf("sealed Go root manifest differs")
	}
	actual, err := inspectRootManifest(
		filepath.Join(container, "root"),
		filepath.FromSlash(document.Manifest.ToolDirectory),
		census,
	)
	if err != nil {
		return rootManifestDocument{}, "", err
	}
	if !equalRootManifests(document.Manifest, actual) {
		return rootManifestDocument{}, "", fmt.Errorf("sealed Go root members differ from manifest")
	}
	return document, manifestDigest, nil
}

func copyRootSnapshot(sourceRoot string, targetRoot string, manifest rootManifest) error {
	if err := os.Mkdir(targetRoot, 0o755); err != nil {
		return err
	}
	var directories []rootEntry
	for _, entry := range manifest.Entries {
		target := filepath.Join(targetRoot, filepath.FromSlash(entry.Path))
		source := filepath.Join(sourceRoot, filepath.FromSlash(entry.Path))
		switch entry.Kind {
		case rootEntryDirectory:
			directories = append(directories, entry)
			if entry.Path != "." {
				if err := os.Mkdir(target, 0o755); err != nil {
					return err
				}
			}
		case rootEntryFile:
			if err := copyRootFile(source, target, entry); err != nil {
				return err
			}
		case rootEntrySymlink:
			actualTarget, err := normalizedSymlinkTarget(sourceRoot, source)
			if err != nil || actualTarget != entry.Target {
				if err == nil {
					err = fmt.Errorf("symbolic link target changed")
				}
				return fmt.Errorf("copy Go root symbolic link %s: %w", source, err)
			}
			targetPath := filepath.Join(targetRoot, filepath.FromSlash(entry.Target))
			linkTarget, err := filepath.Rel(filepath.Dir(target), targetPath)
			if err != nil {
				return err
			}
			if err := os.Symlink(linkTarget, target); err != nil {
				return err
			}
		}
	}
	sort.Slice(directories, func(left int, right int) bool {
		return len(directories[left].Path) > len(directories[right].Path)
	})
	for _, entry := range directories {
		if err := os.Chmod(
			filepath.Join(targetRoot, filepath.FromSlash(entry.Path)),
			os.FileMode(entry.Mode),
		); err != nil {
			return err
		}
	}
	return nil
}

func copyRootFile(source string, target string, expected rootEntry) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || uint32(info.Mode().Perm()) != expected.Mode || info.Size() != expected.Size {
		return fmt.Errorf("Go root source file %s changed before copy", source)
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		_ = input.Close()
		return err
	}
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(output, hash), input)
	inputCloseErr := input.Close()
	closeErr := output.Close()
	if err := errors.Join(copyErr, inputCloseErr, closeErr); err != nil {
		return err
	}
	if written != expected.Size || hex.EncodeToString(hash.Sum(nil)) != expected.Digest {
		return fmt.Errorf("Go root source file %s changed while copying", source)
	}
	return os.Chmod(target, os.FileMode(expected.Mode))
}

func writeSnapshotMetadata(
	container string,
	manifest []byte,
	rootDigest string,
	manifestDigest string,
) error {
	if err := writeSyncedFile(filepath.Join(container, "manifest.json"), manifest, 0o444); err != nil {
		return err
	}
	seal, err := json.Marshal(rootSeal{
		Schema: rootManifestSchema, RootDigest: rootDigest, ManifestDigest: manifestDigest,
	})
	if err != nil {
		return err
	}
	seal = append(seal, '\n')
	if err := writeSyncedFile(filepath.Join(container, "seal.json"), seal, 0o444); err != nil {
		return err
	}
	return syncDirectory(container)
}

func writeSyncedFile(path string, payload []byte, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(payload)
	syncErr := file.Sync()
	closeErr := file.Close()
	return errors.Join(writeErr, syncErr, closeErr)
}

func syncDirectory(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	syncErr := directory.Sync()
	closeErr := directory.Close()
	return errors.Join(syncErr, closeErr)
}
