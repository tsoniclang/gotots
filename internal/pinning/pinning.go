// Package pinning defines the immutable source pin contract and verifies
// that a checkout and toolchain exactly match it before any extraction runs.
//
// A pin identifies both the source revision and the complete toolchain
// identity: version string, target platform, the go executable digest, and a
// digest of the GOROOT source tree. Two different toolchains reporting the
// same version string do not pass. The candidate go executable's digest is
// checked before the binary is ever executed, and the Go frontend compiled
// into the gotots binary itself must match the pinned version, because
// go/parser and go/types run in-process.
//
// Source identity is stronger than cleanliness: every analyzed file must be
// reconciled against the pinned commit tree, so an ignored-but-present file
// cannot be admitted while git status stays clean.
package pinning

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
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

func isLowerHex(value string, length int) bool {
	if len(value) != length {
		return false
	}
	for _, c := range value {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// decodeStrict parses exactly one JSON document with no unknown fields and
// no trailing content.
func decodeStrict(data []byte, value any, what string) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return fmt.Errorf("parse %s: %w", what, err)
	}
	if decoder.More() {
		return fmt.Errorf("parse %s: trailing content after JSON document", what)
	}
	return nil
}

// Load reads and validates a pin file.
func Load(path string) (*Pin, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read pin: %w", err)
	}
	var pin Pin
	if err := decodeStrict(data, &pin, "pin "+path); err != nil {
		return nil, err
	}
	if pin.SchemaVersion != 2 {
		return nil, fmt.Errorf("pin %s: unsupported schemaVersion %d", path, pin.SchemaVersion)
	}
	if pin.GoModule == "" || !isLowerHex(pin.Revision, 40) {
		return nil, fmt.Errorf("pin %s: goModule and a 40-hex-digit revision are required", path)
	}
	t := pin.Toolchain
	if t.Version == "" || t.GOOS == "" || t.GOARCH == "" ||
		!isLowerHex(t.GoExecutableSha256, 64) || !isLowerHex(t.GorootSrcDigest, 64) {
		return nil, fmt.Errorf("pin %s: complete toolchain identity (version, goos, goarch, hex executable sha256, hex GOROOT src digest) is required", path)
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
	FrontendVersion    string `json:"frontendVersion"` // Go release compiled into gotots
	GoExecutableSha256 string `json:"goExecutableSha256"`
	GorootSrcDigest    string `json:"gorootSrcDigest"`
	TrackedFiles       int    `json:"trackedFiles"`
	Submodules         int    `json:"submodules"`
	CleanBeforeLoad    bool   `json:"cleanBeforeLoad"`
	CleanAfterLoad     bool   `json:"cleanAfterLoad"`
}

func git(dir string, args ...string) (string, error) {
	command := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var out bytes.Buffer
	var errOut bytes.Buffer
	command.Stdout = &out
	command.Stderr = &errOut
	if err := command.Run(); err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, errOut.String())
	}
	return strings.TrimRight(out.String(), "\n"), nil
}

// VerifyToolchain locates the go executable, verifies its digest against
// the pin BEFORE executing it, then bootstraps it and verifies the full
// toolchain identity including the GOROOT source digest and the Go frontend
// compiled into this process.
func VerifyToolchain(pin *Pin) (*goenv.Resolved, error) {
	if frontend := runtime.Version(); frontend != pin.Toolchain.Version {
		return nil, fmt.Errorf("gotots was compiled with Go frontend %s but the pin requires %s; in-process go/parser and go/types semantics would not match", frontend, pin.Toolchain.Version)
	}

	goExecutable, err := goenv.Locate()
	if err != nil {
		return nil, err
	}
	digest, err := fileSHA256(goExecutable)
	if err != nil {
		return nil, err
	}
	if digest != pin.Toolchain.GoExecutableSha256 {
		return nil, fmt.Errorf("go executable %s digest %s does not match pinned %s; refusing to execute an unverified toolchain",
			goExecutable, digest, pin.Toolchain.GoExecutableSha256)
	}

	resolved, err := goenv.Bootstrap(goExecutable)
	if err != nil {
		return nil, err
	}
	identity, err := measureIdentity(resolved)
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
	if identity.GorootSrcDigest != pin.Toolchain.GorootSrcDigest {
		return nil, fmt.Errorf("GOROOT src digest %s does not match pinned %s",
			identity.GorootSrcDigest, pin.Toolchain.GorootSrcDigest)
	}
	return resolved, nil
}

// ToolchainIdentity measures the identity of an already-bootstrapped
// toolchain. It exists for the toolchain-id bootstrap command, which by
// definition runs before a pin exists; census verification must use
// VerifyToolchain instead.
func ToolchainIdentity(resolved *goenv.Resolved) (*Toolchain, error) {
	identity, err := measureIdentity(resolved)
	if err != nil {
		return nil, err
	}
	return identity, nil
}

func measureIdentity(resolved *goenv.Resolved) (*Toolchain, error) {
	command := exec.Command(resolved.GoExecutable, "env", "-json", "GOVERSION", "GOHOSTOS", "GOHOSTARCH")
	command.Env = goenv.BootstrapEnviron()
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
	if err := decodeStrict(out.Bytes(), &values, "go env output"); err != nil {
		return nil, err
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

// TreeEntry is one blob in the pinned commit tree.
type TreeEntry struct {
	OID  string // object ID in the repository's object format
	Path string // slash-separated, relative to the checkout root
}

// Tree is the complete tracked-file manifest of the pinned commit.
type Tree struct {
	dir     string
	Entries map[string]TreeEntry // path -> entry (blobs only)
	// Submodules maps gitlink paths to their pinned commit IDs.
	Submodules map[string]string
	// objectFormat is "sha1" or "sha256".
	objectFormat string
}

// VerifiedCheckout bundles the source verification evidence with the
// tracked tree used for per-file reconciliation.
type VerifiedCheckout struct {
	Source *VerifiedSource
	Tree   *Tree
}

// VerifySource confirms that dir is a clean checkout of exactly
// pin.Revision with matching module identity, and loads its complete
// tracked tree for file reconciliation.
func VerifySource(pin *Pin, dir string) (*VerifiedCheckout, error) {
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

	tree, err := loadTree(dir)
	if err != nil {
		return nil, err
	}

	return &VerifiedCheckout{
		Source: &VerifiedSource{
			Revision:        revision,
			GoModule:        moduleLine,
			FrontendVersion: runtime.Version(),
			TrackedFiles:    len(tree.Entries),
			Submodules:      len(tree.Submodules),
			CleanBeforeLoad: true,
		},
		Tree: tree,
	}, nil
}

func loadTree(dir string) (*Tree, error) {
	objectFormat, err := git(dir, "rev-parse", "--show-object-format")
	if err != nil {
		return nil, err
	}
	if objectFormat != "sha1" && objectFormat != "sha256" {
		return nil, fmt.Errorf("unsupported git object format %q", objectFormat)
	}
	output, err := git(dir, "ls-tree", "-r", "-z", "HEAD")
	if err != nil {
		return nil, err
	}
	tree := &Tree{
		dir:          dir,
		Entries:      map[string]TreeEntry{},
		Submodules:   map[string]string{},
		objectFormat: objectFormat,
	}
	for _, record := range strings.Split(output, "\x00") {
		if record == "" {
			continue
		}
		// Format: <mode> <type> <oid>\t<path>
		meta, path, ok := strings.Cut(record, "\t")
		if !ok {
			return nil, fmt.Errorf("unexpected ls-tree record %q", record)
		}
		fields := strings.Fields(meta)
		if len(fields) != 3 {
			return nil, fmt.Errorf("unexpected ls-tree record %q", record)
		}
		mode, objectType, oid := fields[0], fields[1], fields[2]
		switch objectType {
		case "blob":
			tree.Entries[path] = TreeEntry{OID: oid, Path: path}
		case "commit":
			tree.Submodules[path] = oid
		default:
			return nil, fmt.Errorf("unexpected ls-tree object type %q for %s (mode %s)", objectType, path, mode)
		}
	}
	return tree, nil
}

// VerifyFile proves that the given content is byte-identical to the blob
// recorded for path in the pinned commit tree. This catches files that git
// status does not report, such as ignored files injected into package
// directories.
func (t *Tree) VerifyFile(path string, content []byte) error {
	entry, ok := t.Entries[path]
	if !ok {
		return fmt.Errorf("file %s is not tracked by the pinned commit; refusing untracked source", path)
	}
	var h hash.Hash
	switch t.objectFormat {
	case "sha1":
		h = sha1.New()
	case "sha256":
		h = sha256.New()
	}
	h.Write([]byte("blob " + strconv.Itoa(len(content))))
	h.Write([]byte{0})
	h.Write(content)
	actual := hex.EncodeToString(h.Sum(nil))
	if actual != entry.OID {
		return fmt.Errorf("file %s content (blob %s) does not match the pinned commit blob %s", path, actual, entry.OID)
	}
	return nil
}

// Has reports whether path is a tracked blob in the pinned commit.
func (t *Tree) Has(path string) bool {
	_, ok := t.Entries[path]
	return ok
}

// GoFiles returns every tracked path ending in .go, sorted.
func (t *Tree) GoFiles() []string {
	var files []string
	for path := range t.Entries {
		if strings.HasSuffix(path, ".go") {
			files = append(files, path)
		}
	}
	sort.Strings(files)
	return files
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
