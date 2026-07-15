// Package productinputs closes the non-Go half of the input contract:
// the TypeScript compiler, JavaScript runtime, module resolver, strict
// TypeScript configuration, and generated-helper-runtime identities.
// Every identity is declared in a committed pin and verified against
// its materialization before a run's evidence may claim attestation.
//
// The pinned strict configuration deliberately sets
// noFallthroughCasesInSwitch and noImplicitReturns to false: generated
// switch fallthrough is Go semantics, and Go's termination analysis
// (not TypeScript's) proves result coverage.
package productinputs

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/abi"
	"github.com/tsoniclang/gotots/internal/oracle"
)

// Pin is the committed identity set for every non-Go product input.
type Pin struct {
	SchemaVersion      int              `json:"schemaVersion"`
	TypescriptCompiler CompilerIdentity `json:"typescriptCompiler"`
	JavascriptRuntime  RuntimeIdentity  `json:"javascriptRuntime"`
	ModuleResolver     ResolverIdentity `json:"moduleResolver"`
	StrictConfig       FileIdentity     `json:"strictTypeScriptConfig"`
	HelperRuntime      HelperRuntimePin `json:"generatedHelperRuntime"`
	Verified           *VerifiedInputs  `json:"-"`
}

// CompilerIdentity names the exact TypeScript compiler revision the
// product validates against.
type CompilerIdentity struct {
	Package string `json:"package"`
	Version string `json:"version"`
	// TscJsSha256 is the digest of the materialized compiler's lib/tsc.js,
	// verified before the strict typecheck stage runs it.
	TscJsSha256 string `json:"tscJsSha256"`
}

// RuntimeIdentity names the exact JavaScript runtime revision.
type RuntimeIdentity struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ResolverIdentity names the module-resolution policy and digests the
// documented host-normalization loader.
type ResolverIdentity struct {
	Policy       string `json:"policy"`
	LoaderSha256 string `json:"loaderSha256"`
}

// FileIdentity pins one committed file by repository-relative path and
// content digest.
type FileIdentity struct {
	Path   string `json:"path"`
	Sha256 string `json:"sha256"`
}

// HelperRuntimePin pins the generated language-ABI revision: the ABI
// version constant plus a digest over every ABI module's exact source.
type HelperRuntimePin struct {
	AbiVersion int    `json:"abiVersion"`
	Sha256     string `json:"sha256"`
}

// VerifiedInputs records the measured identities of one verification.
type VerifiedInputs struct {
	NodeExecutableSha256 string
	NodeVersion          string
}

// Load reads and structurally validates the pin.
func Load(pinPath string) (*Pin, error) {
	data, err := os.ReadFile(pinPath)
	if err != nil {
		return nil, fmt.Errorf("read product-toolchain pin: %w", err)
	}
	var pin Pin
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&pin); err != nil {
		return nil, fmt.Errorf("parse product-toolchain pin %s: %w", pinPath, err)
	}
	if decoder.More() {
		return nil, fmt.Errorf("product-toolchain pin %s: trailing content", pinPath)
	}
	if pin.SchemaVersion != 1 {
		return nil, fmt.Errorf("product-toolchain pin %s: unsupported schemaVersion %d", pinPath, pin.SchemaVersion)
	}
	for name, value := range map[string]string{
		"typescriptCompiler.package":     pin.TypescriptCompiler.Package,
		"typescriptCompiler.version":     pin.TypescriptCompiler.Version,
		"typescriptCompiler.tscJsSha256": pin.TypescriptCompiler.TscJsSha256,
		"javascriptRuntime.name":         pin.JavascriptRuntime.Name,
		"javascriptRuntime.version":      pin.JavascriptRuntime.Version,
		"moduleResolver.policy":          pin.ModuleResolver.Policy,
		"moduleResolver.loaderSha256":    pin.ModuleResolver.LoaderSha256,
		"strictTypeScriptConfig.path":    pin.StrictConfig.Path,
		"strictTypeScriptConfig.sha256":  pin.StrictConfig.Sha256,
		"generatedHelperRuntime.sha256":  pin.HelperRuntime.Sha256,
	} {
		if value == "" {
			return nil, fmt.Errorf("product-toolchain pin %s: %s is required", pinPath, name)
		}
	}
	return &pin, nil
}

// HelperRuntimeDigest computes the canonical digest over every generated
// language-ABI module: sorted by name, each name and exact source.
func HelperRuntimeDigest() string {
	files := abi.Files()
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	hash := sha256.New()
	for _, name := range names {
		hash.Write([]byte(name))
		hash.Write([]byte{0})
		hash.Write([]byte(files[name]))
		hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// LoaderDigest computes the module-resolver loader identity.
func LoaderDigest() string {
	digest := sha256.Sum256([]byte(oracle.LoaderSource))
	return hex.EncodeToString(digest[:])
}

// Verify checks every pinned identity against its materialization.
// repoDir anchors the strict-config path. The JavaScript runtime must be
// materialized (the differential oracles execute under it); its measured
// executable digest is recorded as run evidence.
func (p *Pin) Verify(repoDir string) error {
	if p.HelperRuntime.AbiVersion != abi.Version {
		return fmt.Errorf("generatedHelperRuntime.abiVersion %d does not match the implementation ABI version %d",
			p.HelperRuntime.AbiVersion, abi.Version)
	}
	if measured := HelperRuntimeDigest(); measured != p.HelperRuntime.Sha256 {
		return fmt.Errorf("generatedHelperRuntime digest mismatch: pin %s, implementation %s (update the pin with the ABI change)",
			p.HelperRuntime.Sha256, measured)
	}
	if measured := LoaderDigest(); measured != p.ModuleResolver.LoaderSha256 {
		return fmt.Errorf("moduleResolver loader digest mismatch: pin %s, implementation %s",
			p.ModuleResolver.LoaderSha256, measured)
	}

	configPath := filepath.Join(repoDir, filepath.FromSlash(p.StrictConfig.Path))
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("strict TypeScript configuration: %w", err)
	}
	configDigest := sha256.Sum256(configBytes)
	if measured := hex.EncodeToString(configDigest[:]); measured != p.StrictConfig.Sha256 {
		return fmt.Errorf("strictTypeScriptConfig digest mismatch: pin %s, file %s", p.StrictConfig.Sha256, measured)
	}

	nodePath, err := exec.LookPath(p.JavascriptRuntime.Name)
	if err != nil {
		return fmt.Errorf("javascript runtime %q is not materialized: %w", p.JavascriptRuntime.Name, err)
	}
	versionOut, err := exec.Command(nodePath, "--version").Output()
	if err != nil {
		return fmt.Errorf("javascript runtime version: %w", err)
	}
	version := strings.TrimSpace(string(versionOut))
	if version != p.JavascriptRuntime.Version {
		return fmt.Errorf("javascript runtime version %s does not match pinned %s", version, p.JavascriptRuntime.Version)
	}
	nodeBytes, err := os.ReadFile(nodePath)
	if err != nil {
		return fmt.Errorf("javascript runtime digest: %w", err)
	}
	nodeDigest := sha256.Sum256(nodeBytes)
	p.Verified = &VerifiedInputs{
		NodeExecutableSha256: hex.EncodeToString(nodeDigest[:]),
		NodeVersion:          version,
	}
	return nil
}
