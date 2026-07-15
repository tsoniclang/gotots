// Package pinning defines the immutable source pin contract and verifies
// that a checkout and toolchain exactly match it before any extraction runs.
package pinning

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Toolchain identifies the exact Go toolchain a pin was created with.
type Toolchain struct {
	Version string `json:"version"`
	GOOS    string `json:"goos"`
	GOARCH  string `json:"goarch"`
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
	if pin.SchemaVersion != 1 {
		return nil, fmt.Errorf("pin %s: unsupported schemaVersion %d", path, pin.SchemaVersion)
	}
	if pin.GoModule == "" || len(pin.Revision) != 40 {
		return nil, fmt.Errorf("pin %s: goModule and 40-hex revision are required", path)
	}
	return &pin, nil
}

// VerifiedSource is the evidence produced by verifying a checkout against a pin.
type VerifiedSource struct {
	Dir             string `json:"dir"`
	Revision        string `json:"revision"`
	Clean           bool   `json:"clean"`
	GoModule        string `json:"goModule"`
	ToolchainOutput string `json:"toolchainOutput"`
	GoExecutable    string `json:"goExecutable"`
	GoExecutableSHA string `json:"goExecutableSha256"`
	GOROOT          string `json:"goroot"`
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

func goTool(args ...string) (string, error) {
	command := exec.Command("go", args...)
	var out bytes.Buffer
	command.Stdout = &out
	command.Stderr = &out
	if err := command.Run(); err != nil {
		return "", fmt.Errorf("go %s: %v: %s", strings.Join(args, " "), err, out.String())
	}
	return strings.TrimSpace(out.String()), nil
}

// Verify confirms that dir is a clean checkout of exactly pin.Revision, that
// its module identity matches, and that the active Go toolchain matches the
// pinned toolchain identity. It fails closed on any mismatch.
func Verify(pin *Pin, dir string) (*VerifiedSource, error) {
	revision, err := git(dir, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	if revision != pin.Revision {
		return nil, fmt.Errorf("source revision %s does not match pinned revision %s", revision, pin.Revision)
	}
	status, err := git(dir, "status", "--porcelain")
	if err != nil {
		return nil, err
	}
	if status != "" {
		return nil, fmt.Errorf("source checkout %s is not clean:\n%s", dir, status)
	}

	moduleData, err := os.ReadFile(dir + "/go.mod")
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

	versionOutput, err := goTool("version")
	if err != nil {
		return nil, err
	}
	expected := fmt.Sprintf("%s %s/%s", pin.Toolchain.Version, pin.Toolchain.GOOS, pin.Toolchain.GOARCH)
	if !strings.Contains(versionOutput, pin.Toolchain.Version+" ") || !strings.HasSuffix(versionOutput, pin.Toolchain.GOOS+"/"+pin.Toolchain.GOARCH) {
		return nil, fmt.Errorf("active toolchain %q does not match pinned %q", versionOutput, expected)
	}

	goExecutable, err := exec.LookPath("go")
	if err != nil {
		return nil, fmt.Errorf("locate go executable: %w", err)
	}
	digest, err := fileSHA256(goExecutable)
	if err != nil {
		return nil, err
	}
	goroot, err := goTool("env", "GOROOT")
	if err != nil {
		return nil, err
	}

	return &VerifiedSource{
		Dir:             dir,
		Revision:        revision,
		Clean:           true,
		GoModule:        moduleLine,
		ToolchainOutput: versionOutput,
		GoExecutable:    goExecutable,
		GoExecutableSHA: digest,
		GOROOT:          goroot,
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
