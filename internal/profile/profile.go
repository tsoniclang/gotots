// Package profile defines the explicit project profile that selects owned
// package roots, declares completely outside-universe roots (filtered
// before census), and fixes build profiles for one product.
package profile

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	// SourceUniverse is the schema-2 typed package-disposition contract:
	// every in-checkout package classifies through exactly one winning
	// rule (explicit override edges; no array-order or prefix authority).
	SourceUniverse SourceUniverse `json:"sourceUniverse"`
	Notes          []string       `json:"notes"`

	// Pin is resolved from PinPath at load time.
	Pin *pinning.Pin `json:"-"`
	// Hash is the sha256 of the profile file's exact bytes, computed at
	// load time; it attests the corpus boundary in every published report.
	Hash string `json:"-"`
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
	digest := sha256.Sum256(data)
	p.Hash = hex.EncodeToString(digest[:])
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&p); err != nil {
		return nil, fmt.Errorf("parse profile %s: %w", profilePath, err)
	}
	if decoder.More() {
		return nil, fmt.Errorf("parse profile %s: trailing content after JSON document", profilePath)
	}
	if p.SchemaVersion != 2 {
		return nil, fmt.Errorf("profile %s: schema version %d is not supported (schema 2 required; no compatibility reader exists)", profilePath, p.SchemaVersion)
	}
	if p.GoModule == "" || len(p.SourceUniverse.PackageRules) == 0 || len(p.BuildProfiles) == 0 || p.PinPath == "" {
		return nil, fmt.Errorf("profile %s: goModule, pin, sourceUniverse.packageRules, and buildProfiles are required", profilePath)
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

// validate enforces build-profile completeness and the schema-2
// source-universe contract (typed rules, explicit override containment,
// unambiguous total classification semantics).
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
		if build.GOARCH == "amd64" && build.GOAMD64 == "" {
			return fmt.Errorf("build profile %q: goamd64 is required for amd64", build.Name)
		}
		if build.GOARCH == "arm64" && build.GOARM64 == "" {
			return fmt.Errorf("build profile %q: goarm64 is required for arm64", build.Name)
		}
	}
	return p.SourceUniverse.Validate()
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
	ClassOwned           PackageClass = "selected-owned"
	ClassTestOnly        PackageClass = "selected-test-support"
	ClassOutsideUniverse PackageClass = "outside-universe"
	ClassTooling         PackageClass = "tooling"
	ClassPolicyExcluded  PackageClass = "product-policy-excluded"
	// ClassUnclassified marks a module-internal package no rule matched
	// or an ambiguous overlap — always a scope defect under the schema-2
	// total-classification contract, never a silent skip.
	ClassUnclassified PackageClass = "unclassified"
	// ClassExternal covers every package outside the owned module. The
	// external-contract subsystem, not the profile, owns its handling.
	ClassExternal PackageClass = "external"
)

// Classify assigns exactly one scope class to a package import path via
// the schema-2 rule contract. The second result is the winning rule's
// category (or the classification error text for ClassUnclassified).
func (p *Profile) Classify(pkgPath string) (PackageClass, string) {
	if pkgPath != p.GoModule && !strings.HasPrefix(pkgPath, p.GoModule+"/") {
		return ClassExternal, ""
	}
	relative := strings.TrimPrefix(strings.TrimPrefix(pkgPath, p.GoModule), "/")
	rule, err := p.SourceUniverse.Classify(relative)
	if err != nil {
		return ClassUnclassified, err.Error()
	}
	switch rule.Disposition {
	case DispositionSelected:
		return ClassOwned, rule.Category
	case DispositionTestOnly:
		return ClassTestOnly, rule.Category
	case DispositionOutside:
		return ClassOutsideUniverse, rule.Category
	case DispositionTooling:
		return ClassTooling, rule.Category
	case DispositionPolicyExcluded:
		return ClassPolicyExcluded, rule.Category
	}
	return ClassUnclassified, "unknown disposition " + string(rule.Disposition)
}

// ClassifyRule resolves the winning rule itself (provenance consumers).
func (p *Profile) ClassifyRule(pkgPath string) (*PackageRule, error) {
	if pkgPath != p.GoModule && !strings.HasPrefix(pkgPath, p.GoModule+"/") {
		return nil, fmt.Errorf("package %s is outside module %s", pkgPath, p.GoModule)
	}
	relative := strings.TrimPrefix(strings.TrimPrefix(pkgPath, p.GoModule), "/")
	return p.SourceUniverse.Classify(relative)
}
