// Package profile defines the explicit project profile that selects owned
// package roots, hard exclusions, and build profiles for one product.
package profile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/pinning"
)

// BuildProfile is one explicit, complete source-selection input set. Every
// input that affects file selection is stated here; nothing relies on
// ambient or undocumented defaults.
type BuildProfile struct {
	Name       string   `json:"name"`
	GOOS       string   `json:"goos"`
	GOARCH     string   `json:"goarch"`
	GOAMD64    string   `json:"goamd64,omitempty"` // required when goarch is amd64
	GOARM64    string   `json:"goarm64,omitempty"` // required when goarch is arm64
	CgoEnabled bool     `json:"cgoEnabled"`
	Tags       []string `json:"tags"`
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
	// ToolingRoots name directory trees (relative to the checkout root, not
	// package paths) that hold generator/tooling source such as nested tool
	// modules. Their files are classified tooling in the tracked-source
	// universe and never enter translation denominators.
	ToolingRoots []string `json:"toolingRoots"`
	Notes        []string `json:"notes"`

	// Pin is resolved from PinPath at load time.
	Pin *pinning.Pin `json:"-"`
}

// Load reads a profile and its referenced pin. PinPath is resolved relative
// to the directory containing the profile file — one explicit, deterministic
// base; no parent-directory searching.
func Load(profilePath string) (*Profile, error) {
	data, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("read profile: %w", err)
	}
	var p Profile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&p); err != nil {
		return nil, fmt.Errorf("parse profile %s: %w", profilePath, err)
	}
	if decoder.More() {
		return nil, fmt.Errorf("parse profile %s: trailing content after JSON document", profilePath)
	}
	if p.SchemaVersion != 1 {
		return nil, fmt.Errorf("profile %s: unsupported schemaVersion %d", profilePath, p.SchemaVersion)
	}
	if p.GoModule == "" || len(p.OwnedRoots) == 0 || len(p.BuildProfiles) == 0 || p.PinPath == "" {
		return nil, fmt.Errorf("profile %s: goModule, pin, ownedRoots, and buildProfiles are required", profilePath)
	}
	if err := p.validate(); err != nil {
		return nil, fmt.Errorf("profile %s: %w", profilePath, err)
	}

	pinPath := p.PinPath
	if !filepath.IsAbs(pinPath) {
		pinPath = filepath.Join(filepath.Dir(profilePath), filepath.FromSlash(pinPath))
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

// validate enforces normalized roots, unique names, disjoint root lists, and
// reachable nesting. It sorts every list for deterministic iteration.
func (p *Profile) validate() error {
	names := map[string]bool{}
	for _, build := range p.BuildProfiles {
		if build.Name == "" || build.GOOS == "" || build.GOARCH == "" {
			return fmt.Errorf("build profile %q: name, goos, and goarch are required", build.Name)
		}
		if names[build.Name] {
			return fmt.Errorf("duplicate build profile name %q", build.Name)
		}
		names[build.Name] = true
		// Unsupported source-selection inputs are rejected before loading
		// rather than silently producing a wrong file selection. Every
		// supported architecture's feature variable must be explicit;
		// architectures without a complete modeled profile fail closed.
		switch build.GOARCH {
		case "amd64":
			if build.GOAMD64 == "" {
				return fmt.Errorf("build profile %q: goamd64 is required when goarch is amd64", build.Name)
			}
			if build.GOARM64 != "" {
				return fmt.Errorf("build profile %q: goarm64 is only valid when goarch is arm64", build.Name)
			}
		case "arm64":
			if build.GOARM64 == "" {
				return fmt.Errorf("build profile %q: goarm64 is required when goarch is arm64", build.Name)
			}
			if build.GOAMD64 != "" {
				return fmt.Errorf("build profile %q: goamd64 is only valid when goarch is amd64", build.Name)
			}
		default:
			return fmt.Errorf("build profile %q: goarch %q has no complete modeled feature profile yet", build.Name, build.GOARCH)
		}
		if build.CgoEnabled {
			return fmt.Errorf("build profile %q: cgoEnabled=true is not supported yet", build.Name)
		}
		if len(build.Tags) > 0 {
			return fmt.Errorf("build profile %q: build tags are not supported yet", build.Name)
		}
	}

	seen := map[string]string{} // root -> list it came from
	checkRoots := func(listName string, roots []string) error {
		for _, root := range roots {
			if root == "" || path.Clean(root) != root || path.IsAbs(root) ||
				strings.HasPrefix(root, "./") || strings.HasPrefix(root, "../") || root == "." {
				return fmt.Errorf("%s: root %q is not a normalized module-relative path", listName, root)
			}
			if previous, ok := seen[root]; ok {
				return fmt.Errorf("root %q appears in both %s and %s", root, previous, listName)
			}
			seen[root] = listName
		}
		return nil
	}
	if err := checkRoots("ownedRoots", p.OwnedRoots); err != nil {
		return err
	}
	if err := checkRoots("testOnlyRoots", p.TestOnlyRoots); err != nil {
		return err
	}
	categories := make([]string, 0, len(p.HardExcludedRoots))
	for category := range p.HardExcludedRoots {
		categories = append(categories, category)
	}
	sort.Strings(categories)
	for _, category := range categories {
		if err := checkRoots("hardExcludedRoots."+category, p.HardExcludedRoots[category]); err != nil {
			return err
		}
	}

	for _, root := range p.ToolingRoots {
		if root == "" || path.Clean(root) != root || path.IsAbs(root) ||
			strings.HasPrefix(root, "./") || strings.HasPrefix(root, "../") || root == "." {
			return fmt.Errorf("toolingRoots: root %q is not a normalized checkout-relative path", root)
		}
	}
	sort.Strings(p.ToolingRoots)

	sort.Strings(p.OwnedRoots)
	sort.Strings(p.TestOnlyRoots)
	for _, roots := range p.HardExcludedRoots {
		sort.Strings(roots)
	}
	if problem := p.invalidRootNesting(); problem != "" {
		return fmt.Errorf("invalid root nesting: %s", problem)
	}
	return nil
}

// invalidRootNesting rejects root layouts that would make some root
// unreachable. Hard-excluded carve-outs inside owned or test-only roots are
// legal because exclusion is matched first; the reverse nesting is not.
func (p *Profile) invalidRootNesting() string {
	under := func(inner, outer string) bool {
		return inner == outer || strings.HasPrefix(inner, outer+"/")
	}
	for categoryName, category := range p.HardExcludedRoots {
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
			// Nesting across exclusion categories would make classification
			// depend on category iteration order; reject it outright.
			for otherName, other := range p.HardExcludedRoots {
				if otherName == categoryName {
					continue
				}
				for _, otherRoot := range other {
					if under(otherRoot, excluded) {
						return "hard-excluded root " + otherRoot + " (" + otherName + ") is nested inside " + excluded + " (" + categoryName + ")"
					}
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

// BuildProfileByName returns the named build profile.
func (p *Profile) BuildProfileByName(name string) (*BuildProfile, error) {
	for i := range p.BuildProfiles {
		if p.BuildProfiles[i].Name == name {
			return &p.BuildProfiles[i], nil
		}
	}
	return nil, fmt.Errorf("build profile %q is not defined in the project profile", name)
}

// PackageClass classifies one package path under this profile.
type PackageClass string

const (
	ClassOwned        PackageClass = "owned"
	ClassTestOnly     PackageClass = "owned-test-support"
	ClassHardExcluded PackageClass = "hard-excluded"
	ClassUnselected   PackageClass = "unselected"
	// ClassExternal covers every package outside the owned module. The
	// standard-library versus third-party split is not decided here: it
	// requires loader evidence (go list Standard/Module), never path shape.
	ClassExternal PackageClass = "external"
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
		categories := make([]string, 0, len(p.HardExcludedRoots))
		for category := range p.HardExcludedRoots {
			categories = append(categories, category)
		}
		sort.Strings(categories)
		for _, category := range categories {
			if matchRoot(relative, p.HardExcludedRoots[category]) {
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
	return ClassExternal, ""
}
