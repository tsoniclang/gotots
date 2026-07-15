// Package pinning defines the immutable source pin contract and verifies
// that a checkout and toolchain exactly match it before any extraction runs.
//
// A pin identifies both the source revision and the complete toolchain
// identity: version string, target platform, the go executable digest, and a
// digest of the GOROOT source tree. Two different toolchains reporting the
// same version string do not pass.
package pinning

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/goenv"
)

// Toolchain identifies the exact Go toolchain a pin was created with.
type Toolchain struct {
	Version string `json:"version"`
	GOOS    string `json:"goos"`
	GOARCH  string `json:"goarch"`
	// GoExecutableSha256 pins the exact go command binary.
	GoExecutableSha256 string `json:"goExecutableSha256"`
	// GorootSrcDigest pins the toolchain's GOROOT/src tree plus its VERSION
	// file: a running sha256 over sorted relative paths and file contents.
	GorootSrcDigest string `json:"gorootSrcDigest"`
}

// Pin is the immutable identity of one upstream source revision.
type Pin struct {
	SchemaVersion int       `json:"schemaVersion"`
	Upstream      string    `json:"upstream"`
	GoModule      string    `json:"goModule"`
	Revision      string    `json:"revision"`
	Toolchain     Toolchain `json:"toolchain"`
}

// Load reads and validates a pin file.
func Load(path string) (*Pin, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read pin: %w", err)
	}
	var pin Pin
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&pin); err != nil {
		return nil, fmt.Errorf("parse pin %s: %w", path, err)
	}
	if pin.SchemaVersion != 2 {
		return nil, fmt.Errorf("pin %s: unsupported schemaVersion %d", path, pin.SchemaVersion)
	}
	if pin.GoModule == "" || len(pin.Revision) != 40 {
		return nil, fmt.Errorf("pin %s: goModule and 40-hex revision are required", path)
	}
	t := pin.Toolchain
	if t.Version == "" || t.GOOS == "" || t.GOARCH == "" || len(t.GoExecutableSha256) != 64 || len(t.GorootSrcDigest) != 64 {
		return nil, fmt.Errorf("pin %s: complete toolchain identity (version, goos, goarch, executable sha256, GOROOT src digest) is required", path)
	}
	return &pin, nil
}

// VerifiedSource is the deterministic evidence produced by verifying a
// checkout against a pin. It contains no machine-specific paths; those
// belong in the environment evidence.
type VerifiedSource struct {
	Revision           string `json:"revision"`
	GoModule           string `json:"goModule"`
	ToolchainVersion   string `json:"toolchainVersion"`
	GoExecutableSha256 string `json:"goExecutableSha256"`
	GorootSrcDigest    string `json:"gorootSrcDigest"`
	CleanBeforeLoad    bool   `json:"cleanBeforeLoad"`
	CleanAfterLoad     bool   `json:"cleanAfterLoad"`
}

func git(dir string, args ...string) (string, error) {
	command := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var out bytes.Buffer
	command.Stdout = &out
	command.Stderr = &out
	if err := command.Run(); err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, out.String())
	}
	return strings.TrimSpace(out.String()), nil
}

// ToolchainIdentity measures the identity of the resolved toolchain.
func ToolchainIdentity(resolved *goenv.Resolved) (*Toolchain, error) {
	command := exec.Command(resolved.GoExecutable, "env", "-json", "GOVERSION", "GOHOSTOS", "GOHOSTARCH")
	command.Env = resolved.Environ("", "", false)
	var out bytes.Buffer
	command.Stdout = &out
	command.Stderr = &out
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("go env: %v: %s", err, out.String())
	}
	var values struct {
		GOVERSION  string
		GOHOSTOS   string
		GOHOSTARCH string
	}
	if err := json.Unmarshal(out.Bytes(), &values); err != nil {
		return nil, fmt.Errorf("parse go env output: %w", err)
	}
	executableDigest, err := fileSHA256(resolved.GoExecutable)
	if err != nil {
		return nil, err
	}
	srcDigest, err := gorootSrcDigest(resolved.GOROOT)
	if err != nil {
		return nil, err
	}
	return &Toolchain{
		Version:            values.GOVERSION,
		GOOS:               values.GOHOSTOS,
		GOARCH:             values.GOHOSTARCH,
		GoExecutableSha256: executableDigest,
		GorootSrcDigest:    srcDigest,
	}, nil
}

// Verify confirms that dir is a clean checkout of exactly pin.Revision, that
// its module identity matches, and that the resolved toolchain matches the
// pinned toolchain identity byte-for-byte. It fails closed on any mismatch.
func Verify(pin *Pin, dir string, resolved *goenv.Resolved) (*VerifiedSource, error) {
	revision, err := git(dir, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	if revision != pin.Revision {
		return nil, fmt.Errorf("source revision %s does not match pinned revision %s", revision, pin.Revision)
	}
	clean, err := CheckClean(dir)
	if err != nil {
		return nil, err
	}
	if !clean {
		return nil, fmt.Errorf("source checkout %s is not clean before load", dir)
	}

	moduleData, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return nil, fmt.Errorf("read source go.mod: %w", err)
	}
	moduleLine := ""
	for line := range strings.Lines(string(moduleData)) {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			moduleLine = strings.TrimSpace(rest)
			break
		}
	}
	if moduleLine != pin.GoModule {
		return nil, fmt.Errorf("source module %q does not match pinned module %q", moduleLine, pin.GoModule)
	}

	identity, err := ToolchainIdentity(resolved)
	if err != nil {
		return nil, err
	}
	if identity.Version != pin.Toolchain.Version ||
		identity.GOOS != pin.Toolchain.GOOS ||
		identity.GOARCH != pin.Toolchain.GOARCH {
		return nil, fmt.Errorf("active toolchain %s %s/%s does not match pinned %s %s/%s",
			identity.Version, identity.GOOS, identity.GOARCH,
			pin.Toolchain.Version, pin.Toolchain.GOOS, pin.Toolchain.GOARCH)
	}
	if identity.GoExecutableSha256 != pin.Toolchain.GoExecutableSha256 {
		return nil, fmt.Errorf("go executable digest %s does not match pinned %s",
			identity.GoExecutableSha256, pin.Toolchain.GoExecutableSha256)
	}
	if identity.GorootSrcDigest != pin.Toolchain.GorootSrcDigest {
		return nil, fmt.Errorf("GOROOT src digest %s does not match pinned %s",
			identity.GorootSrcDigest, pin.Toolchain.GorootSrcDigest)
	}

	return &VerifiedSource{
		Revision:           revision,
		GoModule:           moduleLine,
		ToolchainVersion:   identity.Version,
		GoExecutableSha256: identity.GoExecutableSha256,
		GorootSrcDigest:    identity.GorootSrcDigest,
		CleanBeforeLoad:    true,
	}, nil
}

// CheckClean reports whether the checkout has no modified or untracked
// files. It is re-run after loading to prove extraction mutated nothing.
func CheckClean(dir string) (bool, error) {
	status, err := git(dir, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return status == "", nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// gorootSrcDigest computes one sha256 over the GOROOT VERSION file plus the
// complete GOROOT/src tree: for each regular file in sorted relative-path
// order, the path, a NUL separator, the content, and a NUL separator are
// folded into the running hash.
func gorootSrcDigest(goroot string) (string, error) {
	hash := sha256.New()

	version, err := os.ReadFile(filepath.Join(goroot, "VERSION"))
	if err != nil {
		return "", fmt.Errorf("read GOROOT VERSION: %w", err)
	}
	hash.Write([]byte("VERSION\x00"))
	hash.Write(version)
	hash.Write([]byte{0})

	root := filepath.Join(goroot, "src")
	var files []string
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type().IsRegular() {
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files = append(files, filepath.ToSlash(relative))
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walk GOROOT/src: %w", err)
	}
	sort.Strings(files)
	for _, relative := range files {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			return "", err
		}
		hash.Write([]byte(relative))
		hash.Write([]byte{0})
		hash.Write(data)
		hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
