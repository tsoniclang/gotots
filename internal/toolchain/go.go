package toolchain

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/version"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
)

type Go struct {
	executable    Executable
	cacheRoot     string
	temporaryRoot string
	root          string
	toolDirectory string
	version       string
	defaultGOOS   string
	defaultGOARCH string
	rootContract  rootContract
}

type GoIdentity struct {
	version          string
	executableDigest string
	rootDigest       string
}

type GoError struct {
	Operation string
	Path      string
	Reason    string
}

const (
	PackagesDriverModeEnvironment     = "GOTOTS_EXACT_PACKAGES_DRIVER"
	PackagesDriverVersionEnvironment  = "GOTOTS_EXACT_PACKAGES_GO_VERSION"
	PackagesDriverArchEnvironment     = "GOTOTS_EXACT_PACKAGES_GO_ARCH"
	PackagesDriverEvidenceEnvironment = "GOTOTS_EXACT_PACKAGES_EVIDENCE"
)

func (e *GoError) Error() string {
	if e.Path == "" {
		return "resolve Go tool " + e.Operation + ": " + e.Reason
	}
	return fmt.Sprintf("resolve Go tool %s %q: %s", e.Operation, e.Path, e.Reason)
}

func ResolveGo(selection string, cacheRoot string) (Go, error) {
	cacheRoot, err := filepath.Abs(cacheRoot)
	if err != nil {
		return Go{}, goError("inspect cache", cacheRoot, err.Error())
	}
	if err := os.MkdirAll(cacheRoot, 0o755); err != nil {
		return Go{}, goError("inspect cache", cacheRoot, err.Error())
	}
	cacheRoot, err = canonicalDirectory(cacheRoot)
	if err != nil {
		return Go{}, goError("inspect cache", cacheRoot, err.Error())
	}
	temporaryRoot := filepath.Join(cacheRoot, "process-temp")
	if err := os.MkdirAll(temporaryRoot, 0o755); err != nil {
		return Go{}, goError("inspect cache", cacheRoot, err.Error())
	}
	executable, err := SelectExecutable(selection, "go", goExecutableName(), cacheRoot)
	if err != nil {
		return Go{}, err
	}
	payload, err := executable.SelectedOutput(
		context.Background(),
		identityEnvironment(executable.Directory(), temporaryRoot),
		"",
		"env", "-json", "GOROOT", "GOVERSION", "GOOS", "GOARCH", "GOTOOLDIR",
	)
	if err != nil {
		return Go{}, err
	}
	var identity struct {
		Root          string `json:"GOROOT"`
		Version       string `json:"GOVERSION"`
		GOOS          string `json:"GOOS"`
		GOARCH        string `json:"GOARCH"`
		ToolDirectory string `json:"GOTOOLDIR"`
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&identity); err != nil {
		return Go{}, goError("inspect", executable.Path(), err.Error())
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("identity contains multiple JSON values")
		}
		return Go{}, goError("inspect", executable.Path(), err.Error())
	}
	root, err := canonicalDirectory(identity.Root)
	if err != nil || identity.Root == "" || !version.IsValid(identity.Version) ||
		!validWord(identity.GOOS) || !validWord(identity.GOARCH) || identity.ToolDirectory == "" {
		if err != nil {
			return Go{}, goError("inspect", executable.Path(), err.Error())
		}
		return Go{}, goError("inspect", executable.Path(), "reported identity is invalid")
	}
	if identity.Version != runtime.Version() {
		return Go{}, goError(
			"select frontend",
			executable.Path(),
			fmt.Sprintf(
				"tool version %s differs from compiler frontend version %s",
				identity.Version,
				runtime.Version(),
			),
		)
	}
	toolDirectory, err := canonicalDirectory(identity.ToolDirectory)
	if err != nil || !withinRoot(root, toolDirectory) {
		if err == nil {
			err = fmt.Errorf("tool directory is outside GOROOT")
		}
		return Go{}, goError("inspect GOTOOLDIR", identity.ToolDirectory, err.Error())
	}
	for index, required := range []string{root, filepath.Join(root, "src"), filepath.Join(root, "src", "go.mod")} {
		info, statErr := os.Stat(required)
		wantDirectory := index < 2
		if statErr != nil || info.IsDir() != wantDirectory {
			if statErr == nil {
				statErr = fmt.Errorf("kind is invalid")
			}
			return Go{}, goError("inspect GOROOT", required, statErr.Error())
		}
	}
	contract, err := sealRootContract(root, toolDirectory, cacheRoot)
	if err != nil {
		return Go{}, goError("inspect GOROOT", root, err.Error())
	}
	resolved := Go{
		executable: executable, cacheRoot: filepath.Clean(cacheRoot),
		temporaryRoot: temporaryRoot,
		root:          contract.root, version: identity.Version,
		toolDirectory: contract.toolDirectory,
		defaultGOOS:   identity.GOOS, defaultGOARCH: identity.GOARCH,
		rootContract: contract,
	}
	if err := resolved.Verify(); err != nil {
		return Go{}, err
	}
	return resolved, nil
}

func (g Go) Path() string          { return g.executable.Path() }
func (g Go) CacheRoot() string     { return g.cacheRoot }
func (g Go) Root() string          { return g.root }
func (g Go) Version() string       { return g.version }
func (g Go) DefaultGOOS() string   { return g.defaultGOOS }
func (g Go) DefaultGOARCH() string { return g.defaultGOARCH }
func (g Go) Identity() GoIdentity {
	return GoIdentity{
		version: g.version, executableDigest: g.executable.Digest(),
		rootDigest: g.rootContract.rootDigest,
	}
}

func (g Go) Valid() bool {
	return g.executable.Valid() && g.cacheRoot != "" && g.temporaryRoot != "" &&
		g.root != "" &&
		g.toolDirectory != "" && g.rootContract.Valid() && version.IsValid(g.version) &&
		validWord(g.defaultGOOS) && validWord(g.defaultGOARCH)
}

func (i GoIdentity) Version() string          { return i.version }
func (i GoIdentity) ExecutableDigest() string { return i.executableDigest }
func (i GoIdentity) RootDigest() string       { return i.rootDigest }
func (i GoIdentity) Valid() bool {
	return version.IsValid(i.version) && len(i.executableDigest) == 64 &&
		len(i.rootDigest) == 64
}
func (i GoIdentity) String() string {
	if !i.Valid() {
		return ""
	}
	return "go:" + i.version + ":executable=" + i.executableDigest +
		":root=" + i.rootDigest
}

func (g Go) Verify() error {
	if !g.Valid() {
		return goError("verify", g.Path(), "selection is invalid")
	}
	if info, err := os.Stat(g.temporaryRoot); err != nil || !info.IsDir() {
		if err == nil {
			err = fmt.Errorf("not a directory")
		}
		return goError("verify cache", g.temporaryRoot, err.Error())
	}
	if err := g.executable.Verify(); err != nil {
		return err
	}
	if err := g.rootContract.VerifyHandle(); err != nil {
		return goError("verify GOROOT", g.root, err.Error())
	}
	return nil
}

func (g Go) VerifyComplete() error {
	if err := g.Verify(); err != nil {
		return err
	}
	if err := g.rootContract.VerifyComplete(); err != nil {
		return goError("verify complete GOROOT", g.root, err.Error())
	}
	return nil
}

func (g Go) FullRootInspectionCount() uint64 {
	return g.rootContract.FullWalkCount()
}

func (g Go) ValidateProfile(profile environmentcontract.BuildProfile) error {
	if !g.Valid() || !profile.Valid() || profile.ToolchainVersion() != g.version {
		return goError("select profile", g.Path(), "tool and build profile differ or are invalid")
	}
	if profile.CgoEnabled() {
		return goError(
			"select profile",
			g.Path(),
			"cgo requires an explicitly selected external-tool contract",
		)
	}
	return nil
}

func (g Go) Output(
	ctx context.Context,
	directory string,
	profile environmentcontract.BuildProfile,
	arguments ...string,
) ([]byte, error) {
	if err := g.ValidateProfile(profile); err != nil {
		return nil, err
	}
	if err := g.Verify(); err != nil {
		return nil, err
	}
	command, err := g.executable.CommandContext(ctx, arguments...)
	if err != nil {
		return nil, err
	}
	command.Dir = directory
	command.Env = g.Environment(profile)
	payload, runErr := command.Output()
	verifyErr := g.Verify()
	if runErr != nil || verifyErr != nil {
		var executionErr error
		if runErr != nil {
			executionErr = goError("execute", g.Path(), commandFailure(runErr))
		}
		return nil, errors.Join(executionErr, verifyErr)
	}
	return payload, nil
}

func (g Go) HostOutput(ctx context.Context, directory string, arguments ...string) ([]byte, error) {
	profile, err := environmentcontract.NewBuildProfileForToolchain(
		g.version,
		g.defaultGOOS,
		g.defaultGOARCH,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}
	return g.Output(ctx, directory, profile, arguments...)
}

func (g Go) Environment(profile environmentcontract.BuildProfile) []string {
	if g.ValidateProfile(profile) != nil {
		return nil
	}
	base := profile.Environment(os.Environ())
	result := make([]string, 0, len(base)+7)
	for _, entry := range base {
		if environmentKey(entry, "PATH") || environmentKey(entry, "GOROOT") ||
			environmentKey(entry, "TMPDIR") || environmentKey(entry, "TMP") ||
			environmentKey(entry, "TEMP") ||
			environmentKey(entry, "GOENV") || environmentKey(entry, "GOTOOLCHAIN") ||
			environmentKey(entry, "GOPACKAGESDRIVER") ||
			environmentKey(entry, PackagesDriverModeEnvironment) ||
			environmentKey(entry, PackagesDriverVersionEnvironment) ||
			environmentKey(entry, PackagesDriverArchEnvironment) ||
			environmentKey(entry, PackagesDriverEvidenceEnvironment) {
			continue
		}
		result = append(result, entry)
	}
	return append(
		result,
		"PATH="+g.executable.Directory(),
		"TMPDIR="+g.temporaryRoot,
		"TMP="+g.temporaryRoot,
		"TEMP="+g.temporaryRoot,
		"GOROOT="+g.root,
		"GOENV=off",
		"GOTOOLCHAIN=local",
	)
}

func identityEnvironment(executableDirectory string, temporaryRoot string) []string {
	result := make([]string, 0, len(os.Environ())+7)
	for _, entry := range os.Environ() {
		if environmentKey(entry, "GOOS") || environmentKey(entry, "GOARCH") ||
			environmentKey(entry, "CGO_ENABLED") || environmentKey(entry, "GOFLAGS") ||
			environmentKey(entry, "PATH") || environmentKey(entry, "TMPDIR") ||
			environmentKey(entry, "TMP") || environmentKey(entry, "TEMP") ||
			environmentKey(entry, "GOROOT") || environmentKey(entry, "GOENV") ||
			environmentKey(entry, "GOTOOLCHAIN") ||
			environmentKey(entry, "GOPACKAGESDRIVER") ||
			environmentKey(entry, PackagesDriverModeEnvironment) ||
			environmentKey(entry, PackagesDriverVersionEnvironment) ||
			environmentKey(entry, PackagesDriverArchEnvironment) ||
			environmentKey(entry, PackagesDriverEvidenceEnvironment) {
			continue
		}
		result = append(result, entry)
	}
	return append(
		result,
		"PATH="+executableDirectory,
		"TMPDIR="+temporaryRoot,
		"TMP="+temporaryRoot,
		"TEMP="+temporaryRoot,
		"GOFLAGS=",
		"GOENV=off",
		"GOTOOLCHAIN=local",
	)
}

func environmentKey(entry string, name string) bool {
	return strings.HasPrefix(entry, name+"=")
}

func validWord(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' ||
			character >= '0' && character <= '9' || character == '_' || character == '.' {
			continue
		}
		return false
	}
	return true
}

func commandFailure(err error) string {
	if exit, ok := err.(*exec.ExitError); ok {
		if stderr := strings.TrimSpace(string(exit.Stderr)); stderr != "" {
			return exit.Error() + ": " + stderr
		}
	}
	if exit, ok := err.(*os.PathError); ok {
		return exit.Error()
	}
	return err.Error()
}

func goExecutableName() string {
	if runtime.GOOS == "windows" {
		return "go.exe"
	}
	return "go"
}

func goError(operation string, path string, reason string) error {
	return &GoError{Operation: operation, Path: path, Reason: reason}
}
