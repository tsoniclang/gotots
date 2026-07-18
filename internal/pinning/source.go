package pinning

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/mod/modfile"
)

// VerifiedSource is the deterministic evidence produced by verifying a
// checkout against a pin. It contains no machine-specific paths; those
// belong in the environment evidence.
type VerifiedSource struct {
	Revision           string `json:"revision"`
	GoModule           string `json:"goModule"`
	GoLanguageVersion  string `json:"goLanguageVersion"` // module go directive
	ToolchainVersion   string `json:"toolchainVersion"`
	FrontendVersion    string `json:"frontendVersion"` // Go release compiled into gotots
	GoExecutableSha256 string `json:"goExecutableSha256"`
	GorootSrcDigest    string `json:"gorootSrcDigest"`
	GorootToolDigest   string `json:"gorootToolDigest"`
	TrackedFiles       int    `json:"trackedFiles"`
	Submodules         int    `json:"submodules"`
	CleanBeforeLoad    bool   `json:"cleanBeforeLoad"`
	CleanAfterLoad     bool   `json:"cleanAfterLoad"`
}

// VerifiedCheckout bundles the source verification evidence with the
// tracked tree used for per-file reconciliation.
type VerifiedCheckout struct {
	Source *VerifiedSource
	Tree   *Tree
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

// VerifySource confirms that dir is a clean checkout of exactly
// pin.Revision with matching module identity (parsed with the canonical Go
// module parser), and loads its complete tracked tree for file
// reconciliation.
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

	modulePath := filepath.Join(dir, "go.mod")
	moduleData, err := os.ReadFile(modulePath)
	if err != nil {
		return nil, fmt.Errorf("read source go.mod: %w", err)
	}
	moduleFile, err := modfile.Parse(modulePath, moduleData, nil)
	if err != nil {
		return nil, fmt.Errorf("parse source go.mod: %w", err)
	}
	if moduleFile.Module == nil || moduleFile.Module.Mod.Path != pin.GoModule {
		declared := "<none>"
		if moduleFile.Module != nil {
			declared = moduleFile.Module.Mod.Path
		}
		return nil, fmt.Errorf("source module %q does not match pinned module %q", declared, pin.GoModule)
	}
	languageVersion := ""
	if moduleFile.Go != nil {
		languageVersion = moduleFile.Go.Version
	}

	tree, err := loadTree(dir)
	if err != nil {
		return nil, err
	}

	return &VerifiedCheckout{
		Source: &VerifiedSource{
			Revision:          revision,
			GoModule:          pin.GoModule,
			GoLanguageVersion: languageVersion,
			FrontendVersion:   runtime.Version(),
			TrackedFiles:      len(tree.Entries),
			Submodules:        len(tree.Submodules),
			CleanBeforeLoad:   true,
		},
		Tree: tree,
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
