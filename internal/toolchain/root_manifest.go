package toolchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	pathpkg "path"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
)

type rootEntryKind string

const (
	rootEntryDirectory rootEntryKind = "directory"
	rootEntryFile      rootEntryKind = "file"
	rootEntrySymlink   rootEntryKind = "symlink"
)

type rootEntry struct {
	Path   string        `json:"path"`
	Kind   rootEntryKind `json:"kind"`
	Mode   uint32        `json:"mode"`
	Size   int64         `json:"size,omitempty"`
	Digest string        `json:"digest,omitempty"`
	Target string        `json:"target,omitempty"`
}

type rootManifest struct {
	Schema        int         `json:"schema"`
	ToolDirectory string      `json:"toolDirectory"`
	Entries       []rootEntry `json:"entries"`
}

type rootManifestDocument struct {
	RootDigest string       `json:"rootDigest"`
	Manifest   rootManifest `json:"manifest"`
}

func inspectRootManifest(
	root string,
	toolDirectory string,
	census *rootInspectionCensus,
) (rootManifest, error) {
	if census != nil {
		census.fullWalks.Add(1)
	}
	root, err := canonicalDirectory(root)
	if err != nil {
		return rootManifest{}, err
	}
	manifest := rootManifest{
		Schema: rootManifestSchema, ToolDirectory: filepath.ToSlash(toolDirectory),
	}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		record := rootEntry{Path: filepath.ToSlash(relative)}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		record.Mode = uint32(info.Mode().Perm())
		switch {
		case info.IsDir():
			record.Kind = rootEntryDirectory
		case info.Mode()&os.ModeSymlink != 0:
			target, err := normalizedSymlinkTarget(root, path)
			if err != nil {
				return err
			}
			record.Kind = rootEntrySymlink
			record.Target = target
		case info.Mode().IsRegular():
			digest, err := fileDigest(path)
			if err != nil {
				return err
			}
			record.Kind = rootEntryFile
			record.Size = info.Size()
			record.Digest = digest
		default:
			return fmt.Errorf("special entry %s is outside the sealed Go root contract", path)
		}
		manifest.Entries = append(manifest.Entries, record)
		return nil
	})
	if err != nil {
		return rootManifest{}, err
	}
	sort.Slice(manifest.Entries, func(left int, right int) bool {
		return manifest.Entries[left].Path < manifest.Entries[right].Path
	})
	if err := validateRootManifest(manifest); err != nil {
		return rootManifest{}, err
	}
	return manifest, nil
}

func normalizedSymlinkTarget(root string, path string) (string, error) {
	target, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return "", err
	}
	if !withinRoot(root, target) {
		return "", fmt.Errorf("symbolic link %s escapes the selected Go root", path)
	}
	relative, err := filepath.Rel(root, target)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(relative), nil
}

func validateRootManifest(manifest rootManifest) error {
	if manifest.Schema != rootManifestSchema || !validRootPath(manifest.ToolDirectory, false) ||
		len(manifest.Entries) == 0 {
		return fmt.Errorf("Go root manifest header is invalid")
	}
	seen := make(map[string]rootEntry, len(manifest.Entries))
	previous := ""
	for index, entry := range manifest.Entries {
		if !validRootPath(entry.Path, true) || index > 0 && entry.Path <= previous {
			return fmt.Errorf("Go root manifest path %q is invalid or unordered", entry.Path)
		}
		previous = entry.Path
		if _, duplicate := seen[entry.Path]; duplicate {
			return fmt.Errorf("Go root manifest path %q is duplicated", entry.Path)
		}
		if entry.Mode&^uint32(0o777) != 0 {
			return fmt.Errorf("Go root manifest path %q has an invalid mode", entry.Path)
		}
		switch entry.Kind {
		case rootEntryDirectory:
			if entry.Size != 0 || entry.Digest != "" || entry.Target != "" {
				return fmt.Errorf("Go root directory %q has file evidence", entry.Path)
			}
		case rootEntryFile:
			if entry.Size < 0 || len(entry.Digest) != sha256.Size*2 || entry.Target != "" {
				return fmt.Errorf("Go root file %q has invalid evidence", entry.Path)
			}
		case rootEntrySymlink:
			if entry.Size != 0 || entry.Digest != "" || !validRootPath(entry.Target, true) {
				return fmt.Errorf("Go root symbolic link %q has invalid evidence", entry.Path)
			}
		default:
			return fmt.Errorf("Go root entry %q has invalid kind %q", entry.Path, entry.Kind)
		}
		seen[entry.Path] = entry
	}
	root, ok := seen["."]
	if !ok || root.Kind != rootEntryDirectory {
		return fmt.Errorf("Go root manifest has no root directory")
	}
	toolDirectory, ok := seen[manifest.ToolDirectory]
	if !ok || toolDirectory.Kind != rootEntryDirectory {
		return fmt.Errorf("Go root manifest tool directory is absent")
	}
	for _, name := range []string{"compile", "link"} {
		path := pathpkg.Join(manifest.ToolDirectory, executableFileName(name))
		entry, ok := seen[path]
		if !ok || entry.Kind != rootEntryFile || runtime.GOOS != "windows" && entry.Mode&0o111 == 0 {
			return fmt.Errorf("Go root tool %q is not an executable regular file", path)
		}
	}
	return nil
}

func validRootPath(value string, allowRoot bool) bool {
	if value == "." {
		return allowRoot
	}
	return value != "" && !strings.ContainsAny(value, "\\\x00") &&
		!strings.HasPrefix(value, "/") && pathpkg.Clean(value) == value &&
		value != ".." && !strings.HasPrefix(value, "../")
}

func rootManifestDigest(manifest rootManifest) (string, error) {
	if err := validateRootManifest(manifest); err != nil {
		return "", err
	}
	payload, err := json.Marshal(manifest)
	if err != nil {
		return "", err
	}
	return digestBytes(payload), nil
}

func encodeRootManifest(manifest rootManifest) ([]byte, string, string, error) {
	rootDigest, err := rootManifestDigest(manifest)
	if err != nil {
		return nil, "", "", err
	}
	payload, err := json.Marshal(rootManifestDocument{RootDigest: rootDigest, Manifest: manifest})
	if err != nil {
		return nil, "", "", err
	}
	payload = append(payload, '\n')
	return payload, rootDigest, digestBytes(payload), nil
}

func decodeRootManifest(payload []byte) (rootManifestDocument, error) {
	var document rootManifestDocument
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return rootManifestDocument{}, err
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return rootManifestDocument{}, err
	}
	digest, err := rootManifestDigest(document.Manifest)
	if err != nil {
		return rootManifestDocument{}, err
	}
	if document.RootDigest != digest {
		return rootManifestDocument{}, fmt.Errorf("Go root manifest digest differs")
	}
	return document, nil
}

func equalRootManifests(left rootManifest, right rootManifest) bool {
	return left.Schema == right.Schema && left.ToolDirectory == right.ToolDirectory &&
		slices.Equal(left.Entries, right.Entries)
}
