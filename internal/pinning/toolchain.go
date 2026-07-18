package pinning

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/tsoniclang/gotots/internal/goenv"
)

// VerifyToolchain locates the go executable, verifies its digest against
// the pin BEFORE executing it, then bootstraps it and verifies the full
// toolchain identity — including the GOROOT source and tool-binary digests
// and the Go frontend compiled into this process.
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
	if identity.GorootToolDigest != pin.Toolchain.GorootToolDigest {
		return nil, fmt.Errorf("GOROOT tool digest %s does not match pinned %s",
			identity.GorootToolDigest, pin.Toolchain.GorootToolDigest)
	}
	return resolved, nil
}

// ToolchainIdentity measures the identity of an already-bootstrapped
// toolchain. It exists for the toolchain-id bootstrap command, which by
// definition runs before a pin exists; census verification must use
// VerifyToolchain instead.
func ToolchainIdentity(resolved *goenv.Resolved) (*Toolchain, error) {
	return measureIdentity(resolved)
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
	srcDigest, err := treeDigest(filepath.Join(resolved.GOROOT, "src"), filepath.Join(resolved.GOROOT, "VERSION"))
	if err != nil {
		return nil, fmt.Errorf("GOROOT src digest: %w", err)
	}
	toolDigest, err := treeDigest(filepath.Join(resolved.GOROOT, "pkg", "tool", values.GOHOSTOS+"_"+values.GOHOSTARCH), "")
	if err != nil {
		return nil, fmt.Errorf("GOROOT tool digest: %w", err)
	}
	return &Toolchain{
		Version:            values.GOVERSION,
		GOOS:               values.GOHOSTOS,
		GOARCH:             values.GOHOSTARCH,
		GoExecutableSha256: executableDigest,
		GorootSrcDigest:    srcDigest,
		GorootToolDigest:   toolDigest,
	}, nil
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

// treeDigest computes one sha256 over an optional prologue file plus a
// complete file tree: for each regular file in sorted relative-path order,
// the path, a NUL separator, the content, and a NUL separator are folded
// into the running hash.
func treeDigest(root, prologuePath string) (string, error) {
	hash := sha256.New()
	if prologuePath != "" {
		prologue, err := os.ReadFile(prologuePath)
		if err != nil {
			return "", err
		}
		hash.Write([]byte(filepath.Base(prologuePath)))
		hash.Write([]byte{0})
		hash.Write(prologue)
		hash.Write([]byte{0})
	}

	var files []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
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
		return "", err
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
