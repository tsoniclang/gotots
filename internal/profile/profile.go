// Package profile defines the explicit project profile that selects owned
// package roots, hard exclusions, and build profiles for one product.
package profile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/pinning"
)

// BuildProfile is one explicit GOOS/GOARCH/tags selection.
type BuildProfile struct {
	Name   string   `json:"name"`
	GOOS   string   `json:"goos"`
	GOARCH string   `json:"goarch"`
	Tags   []string `json:"tags"`
}

// Profile is the explicit, reviewed project scope for one product.
type Profile struct {
	SchemaVersion int            `json:"schemaVersion"`
	Product       string         `json:"product"`
	GoModule      string         `json:"goModule"`
	PinPath       string         `json:"pin"`
	BuildProfiles []BuildProfile `json:"buildProfiles"`
	OwnedRoots    []string       `json:"ownedRoots"`
	// TestOnlyRoots are owned test-support roots: their source is analyzed
	// under the owned-test scope and may be imported only from test scope.
	TestOnlyRoots     []string            `json:"testOnlyRoots"`
	HardExcludedRoots map[string][]string `json:"hardExcludedRoots"`
	Notes             []string            `json:"notes"`

	// Pin is resolved from PinPath at load time.
	Pin *pinning.Pin `json:"-"`
}

// Load reads a profile and its referenced pin. Paths inside the profile are
// resolved relative to the profile file's directory's repository root, i.e.
// relative to the directory that contains the profile path's first segment.
func Load(path string) (*Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read profile: %w", err)
	}
	var p Profile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&p); err != nil {
		return nil, fmt.Errorf("parse profile %s: %w", path, err)
	}
	if p.SchemaVersion != 1 {
		return nil, fmt.Errorf("profile %s: unsupported schemaVersion %d", path, p.SchemaVersion)
	}
	if p.GoModule == "" || len(p.OwnedRoots) == 0 || len(p.BuildProfiles) == 0 {
		return nil, fmt.Errorf("profile %s: goModule, ownedRoots, and buildProfiles are required", path)
	}
	sort.Strings(p.OwnedRoots)
	sort.Strings(p.TestOnlyRoots)
	for _, roots := range p.HardExcludedRoots {
		sort.Strings(roots)
	}
	if problem := p.invalidRootNesting(); problem != "" {
		return nil, fmt.Errorf("profile %s: invalid root nesting: %s", path, problem)
	}

	pinPath := p.PinPath
	if !filepath.IsAbs(pinPath) {
		// Profile-relative paths resolve against the repository root, which is
		// assumed to be the parent of the profiles/ directory the profile
		// lives in. Fall back to profile-file-relative resolution.
		base := filepath.Dir(path)
		for dir := base; ; dir = filepath.Dir(dir) {
			candidate := filepath.Join(dir, pinPath)
			if _, err := os.Stat(candidate); err == nil {
				pinPath = candidate
				break
			}
			if dir == filepath.Dir(dir) {
				return nil, fmt.Errorf("profile %s: pin file %q not found relative to any parent directory", path, p.PinPath)
			}
		}
	}
	pin, err := pinning.Load(pinPath)
	if err != nil {
		return nil, err
	}
	if pin.GoModule != p.GoModule {
		return nil, fmt.Errorf("profile module %q does not match pin module %q", p.GoModule, pin.GoModule)
	}
	p.Pin = pin
	return &p, nil
}

// invalidRootNesting rejects root layouts that would make some root
// unreachable. Hard-excluded carve-outs inside owned or test-only roots are
// legal because exclusion is matched first; the reverse nesting is not.
func (p *Profile) invalidRootNesting() string {
	under := func(inner, outer string) bool {
		return inner == outer || strings.HasPrefix(inner, outer+"/")
	}
	for _, category := range p.HardExcludedRoots {
		for _, excluded := range category {
			for _, owned := range p.OwnedRoots {
				if under(owned, excluded) {
					return "owned root " + owned + " is inside hard-excluded root " + excluded
				}
			}
			for _, testOnly := range p.TestOnlyRoots {
				if under(testOnly, excluded) {
					return "test-only root " + testOnly + " is inside hard-excluded root " + excluded
				}
			}
		}
	}
	for _, testOnly := range p.TestOnlyRoots {
		for _, owned := range p.OwnedRoots {
			if under(owned, testOnly) {
				return "owned root " + owned + " is inside test-only root " + testOnly
			}
		}
	}
	return ""
}

// PackageClass classifies one package path under this profile.
type PackageClass string

const (
	ClassOwned        PackageClass = "owned"
	ClassTestOnly     PackageClass = "owned-test-support"
	ClassHardExcluded PackageClass = "hard-excluded"
	ClassUnselected   PackageClass = "unselected"
	ClassExternalStd  PackageClass = "external-std"
	ClassExternalMod  PackageClass = "external-module"
)

func matchRoot(relative string, roots []string) bool {
	for _, root := range roots {
		if relative == root || strings.HasPrefix(relative, root+"/") {
			return true
		}
	}
	return false
}

// Classify assigns exactly one class to a package import path.
// The second result names the hard-exclusion category when applicable.
func (p *Profile) Classify(pkgPath string) (PackageClass, string) {
	if pkgPath == p.GoModule || strings.HasPrefix(pkgPath, p.GoModule+"/") {
		relative := strings.TrimPrefix(strings.TrimPrefix(pkgPath, p.GoModule), "/")
		for category, roots := range p.HardExcludedRoots {
			if matchRoot(relative, roots) {
				return ClassHardExcluded, category
			}
		}
		if matchRoot(relative, p.TestOnlyRoots) {
			return ClassTestOnly, ""
		}
		if matchRoot(relative, p.OwnedRoots) {
			return ClassOwned, ""
		}
		return ClassUnselected, ""
	}
	firstSegment, _, _ := strings.Cut(pkgPath, "/")
	if strings.Contains(firstSegment, ".") {
		return ClassExternalMod, ""
	}
	return ClassExternalStd, ""
}
