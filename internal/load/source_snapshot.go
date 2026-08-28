package load

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"go/format"
	"hash"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/toolchain"
)

const sourceSnapshotSchema = "gotots-selected-source-v1"

type sourceSnapshotFile struct {
	identity string
	payload  []byte
	version  string
}

func sealSourceSnapshot(
	pattern string,
	selectedPackages []*packages.Package,
	profile BuildProfile,
	selectedGo toolchain.Go,
) (string, error) {
	if len(selectedPackages) == 0 || !profile.Valid() || !selectedGo.Valid() {
		return "", &Error{Pattern: pattern, Reason: "source snapshot input is incomplete"}
	}
	digest := sha256.New()
	writeSnapshotString(digest, "schema", sourceSnapshotSchema)
	writeSnapshotString(digest, "toolchain", selectedGo.Identity().String())
	writeSnapshotString(digest, "go-version", profile.ToolchainVersion())
	writeSnapshotString(digest, "goos", profile.GOOS())
	writeSnapshotString(digest, "goarch", profile.GOARCH())
	writeSnapshotString(digest, "cgo", fmt.Sprintf("%t", profile.CgoEnabled()))
	writeSnapshotStrings(digest, "tags", profile.Tags())
	writeSnapshotCount(digest, "packages", len(selectedPackages))
	for _, selected := range selectedPackages {
		if err := writePackageSnapshot(digest, pattern, selected); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func writePackageSnapshot(
	digest hash.Hash,
	pattern string,
	selected *packages.Package,
) error {
	if selected == nil || selected.PkgPath == "" || selected.Name == "" ||
		selected.Fset == nil || selected.TypesInfo == nil || selected.Dir == "" {
		return &Error{Pattern: pattern, Reason: "source snapshot package is incomplete"}
	}
	modulePath := ""
	moduleVersion := ""
	moduleGoVersion := ""
	if selected.Module != nil {
		modulePath = selected.Module.Path
		moduleVersion = selected.Module.Version
		moduleGoVersion = selected.Module.GoVersion
	}
	writeSnapshotString(digest, "package-path", selected.PkgPath)
	writeSnapshotString(digest, "package-name", selected.Name)
	writeSnapshotString(digest, "module-path", modulePath)
	writeSnapshotString(digest, "module-version", moduleVersion)
	writeSnapshotString(digest, "module-go-version", moduleGoVersion)

	goInputs, err := readSnapshotFiles(selected, "go-input", selected.GoFiles, false)
	if err != nil {
		return &Error{Pattern: pattern, Reason: selected.PkgPath + ": " + err.Error()}
	}
	writeSnapshotFiles(digest, "go-inputs", goInputs)

	checkedSyntax, err := checkedSyntaxSnapshot(selected)
	if err != nil {
		return &Error{Pattern: pattern, Reason: selected.PkgPath + ": " + err.Error()}
	}
	writeSnapshotFiles(digest, "checked-syntax", checkedSyntax)

	otherInputs, err := readSnapshotFiles(selected, "other-input", selected.OtherFiles, false)
	if err != nil {
		return &Error{Pattern: pattern, Reason: selected.PkgPath + ": " + err.Error()}
	}
	writeSnapshotFiles(digest, "other-inputs", otherInputs)

	embedInputs, err := readSnapshotFiles(selected, "embed-input", selected.EmbedFiles, false)
	if err != nil {
		return &Error{Pattern: pattern, Reason: selected.PkgPath + ": " + err.Error()}
	}
	writeSnapshotFiles(digest, "embed-inputs", embedInputs)
	return nil
}

func checkedSyntaxSnapshot(selected *packages.Package) ([]sourceSnapshotFile, error) {
	if len(selected.Syntax) != len(selected.CompiledGoFiles) {
		return nil, fmt.Errorf(
			"checked syntax has %d trees for %d files",
			len(selected.Syntax),
			len(selected.CompiledGoFiles),
		)
	}
	result := make([]sourceSnapshotFile, len(selected.Syntax))
	seen := make(map[string]struct{}, len(result))
	for index, syntax := range selected.Syntax {
		if syntax == nil {
			return nil, fmt.Errorf("checked syntax %d is absent", index)
		}
		identity, err := snapshotFileIdentity(
			selected,
			"checked-syntax",
			selected.CompiledGoFiles[index],
			true,
		)
		if err != nil {
			return nil, err
		}
		if _, duplicate := seen[identity]; duplicate {
			return nil, fmt.Errorf("duplicate checked syntax identity %q", identity)
		}
		seen[identity] = struct{}{}
		version, ok := selected.TypesInfo.FileVersions[syntax]
		if !ok || version == "" {
			return nil, fmt.Errorf("checked syntax %q lacks an effective Go version", identity)
		}
		var canonical bytes.Buffer
		if err := format.Node(&canonical, selected.Fset, syntax); err != nil {
			return nil, fmt.Errorf("canonicalize checked syntax %q: %w", identity, err)
		}
		result[index] = sourceSnapshotFile{
			identity: identity,
			payload:  canonical.Bytes(),
			version:  version,
		}
	}
	sort.Slice(result, func(left, right int) bool {
		return result[left].identity < result[right].identity
	})
	return result, nil
}

func readSnapshotFiles(
	selected *packages.Package,
	category string,
	paths []string,
	allowGenerated bool,
) ([]sourceSnapshotFile, error) {
	result := make([]sourceSnapshotFile, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for index, path := range paths {
		identity, err := snapshotFileIdentity(selected, category, path, allowGenerated)
		if err != nil {
			return nil, err
		}
		if _, duplicate := seen[identity]; duplicate {
			return nil, fmt.Errorf("duplicate %s identity %q", category, identity)
		}
		seen[identity] = struct{}{}
		payload, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s %q: %w", category, identity, err)
		}
		result[index] = sourceSnapshotFile{identity: identity, payload: payload}
	}
	sort.Slice(result, func(left, right int) bool {
		return result[left].identity < result[right].identity
	})
	return result, nil
}

func snapshotFileIdentity(
	selected *packages.Package,
	category string,
	path string,
	allowGenerated bool,
) (string, error) {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return "", fmt.Errorf("%s path %q is not canonical and absolute", category, path)
	}
	relative, err := filepath.Rel(selected.Dir, path)
	if err == nil {
		logical := filepath.ToSlash(relative)
		if logical != "." && logical != ".." && !strings.HasPrefix(logical, "../") {
			return category + ":" + logical, nil
		}
	}
	if allowGenerated {
		base := filepath.Base(path)
		if base != "." && base != string(filepath.Separator) && base != "" {
			return category + ":@generated/" + base, nil
		}
	}
	return "", fmt.Errorf("%s path %q is outside package directory", category, path)
}

func writeSnapshotFiles(digest hash.Hash, label string, files []sourceSnapshotFile) {
	writeSnapshotCount(digest, label, len(files))
	for _, file := range files {
		writeSnapshotString(digest, label+"-identity", file.identity)
		writeSnapshotString(digest, label+"-version", file.version)
		writeSnapshotBytes(digest, label+"-payload", file.payload)
	}
}

func writeSnapshotStrings(digest hash.Hash, label string, values []string) {
	selected := slices.Clone(values)
	slices.Sort(selected)
	writeSnapshotCount(digest, label, len(selected))
	for _, value := range selected {
		writeSnapshotString(digest, label+"-value", value)
	}
}

func writeSnapshotCount(digest hash.Hash, label string, count int) {
	var encoded [binary.MaxVarintLen64]byte
	length := binary.PutUvarint(encoded[:], uint64(count))
	writeSnapshotBytes(digest, label, encoded[:length])
}

func writeSnapshotString(digest hash.Hash, label string, value string) {
	writeSnapshotBytes(digest, label, []byte(value))
}

func writeSnapshotBytes(digest hash.Hash, label string, value []byte) {
	var encoded [binary.MaxVarintLen64]byte
	labelLength := binary.PutUvarint(encoded[:], uint64(len(label)))
	_, _ = digest.Write(encoded[:labelLength])
	_, _ = digest.Write([]byte(label))
	valueLength := binary.PutUvarint(encoded[:], uint64(len(value)))
	_, _ = digest.Write(encoded[:valueLength])
	_, _ = digest.Write(value)
}
