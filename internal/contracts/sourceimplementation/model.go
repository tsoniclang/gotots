package sourceimplementation

import (
	"fmt"
	"slices"

	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const SchemaVersion = 1

type EnvelopeKind string

const (
	EnvelopeInvalid           EnvelopeKind = ""
	EnvelopeExact             EnvelopeKind = "exact"
	EnvelopeInternalAlgorithm EnvelopeKind = "internal-algorithm"
)

func (k EnvelopeKind) Valid() bool {
	return k == EnvelopeExact || k == EnvelopeInternalAlgorithm
}

type Document struct {
	SchemaVersion int              `json:"schemaVersion"`
	Package       PackageDocument  `json:"package"`
	Build         BuildDocument    `json:"build"`
	Source        string           `json:"source"`
	TSConfig      string           `json:"tsconfig"`
	Envelope      EnvelopeDocument `json:"equivalenceEnvelope"`
	Exports       []string         `json:"exports"`
}

type PackageDocument struct {
	ImportPath    string `json:"importPath"`
	ModulePath    string `json:"modulePath"`
	ModuleVersion string `json:"moduleVersion"`
}

type BuildDocument struct {
	GoVersion  string   `json:"goVersion"`
	GOOS       string   `json:"goos"`
	GOARCH     string   `json:"goarch"`
	CGOEnabled bool     `json:"cgoEnabled"`
	BuildTags  []string `json:"buildTags"`
}

type EnvelopeDocument struct {
	Kind                 EnvelopeKind `json:"kind"`
	RelaxedBehavior      string       `json:"relaxedBehavior,omitempty"`
	PreservedObservables []string     `json:"preservedObservables,omitempty"`
	Evidence             []string     `json:"evidence,omitempty"`
}

type Export struct {
	name        string
	typeString  string
	fingerprint string
}

func (e Export) Name() string        { return e.name }
func (e Export) TypeString() string  { return e.typeString }
func (e Export) Fingerprint() string { return e.fingerprint }

type Implementation struct {
	packagePath   string
	modulePath    string
	moduleVersion string
	sourcePath    string
	digest        string
	sourceDigest  string
	envelope      EnvelopeKind
	exports       []Export
	sourceFile    tsgo.SourceFile
}

func (i Implementation) PackagePath() string         { return i.packagePath }
func (i Implementation) ModulePath() string          { return i.modulePath }
func (i Implementation) ModuleVersion() string       { return i.moduleVersion }
func (i Implementation) SourcePath() string          { return i.sourcePath }
func (i Implementation) Digest() string              { return i.digest }
func (i Implementation) SourceDigest() string        { return i.sourceDigest }
func (i Implementation) Envelope() EnvelopeKind      { return i.envelope }
func (i Implementation) Exports() []Export           { return slices.Clone(i.exports) }
func (i Implementation) SourceFile() tsgo.SourceFile { return i.sourceFile }

type Certificate struct {
	byPath map[string]Implementation
	digest string
}

func (c *Certificate) Valid() bool {
	return c != nil && c.digest != "" && len(c.byPath) != 0
}

func (c *Certificate) Digest() string {
	if c == nil {
		return ""
	}
	return c.digest
}

func (c *Certificate) ForPackage(source *load.Package) (Implementation, bool) {
	if !c.Valid() || source == nil {
		return Implementation{}, false
	}
	selected, ok := c.byPath[source.Path()]
	if !ok || selected.modulePath != source.ModulePath() ||
		selected.moduleVersion != source.ModuleVersion() {
		return Implementation{}, false
	}
	return selected, true
}

func (c *Certificate) Implementations() []Implementation {
	if !c.Valid() {
		return nil
	}
	result := make([]Implementation, 0, len(c.byPath))
	for _, implementation := range c.byPath {
		result = append(result, implementation)
	}
	slices.SortFunc(result, func(left, right Implementation) int {
		if left.packagePath < right.packagePath {
			return -1
		}
		if left.packagePath > right.packagePath {
			return 1
		}
		return 0
	})
	return result
}

func (c *Certificate) add(implementation Implementation) error {
	if c.byPath == nil {
		c.byPath = make(map[string]Implementation)
	}
	if _, duplicate := c.byPath[implementation.packagePath]; duplicate {
		return &Error{
			Operation: "admit",
			Subject:   implementation.packagePath,
			Reason:    "package has multiple implementation owners",
		}
	}
	if implementation.packagePath == "" || implementation.digest == "" ||
		implementation.sourceFile == nil {
		return &Error{Operation: "admit", Reason: "implementation evidence is incomplete"}
	}
	c.byPath[implementation.packagePath] = implementation
	return nil
}

type Error struct {
	Operation string
	Subject   string
	Reason    string
}

func (e *Error) Error() string {
	if e.Subject == "" {
		return fmt.Sprintf("certify source implementation %s: %s", e.Operation, e.Reason)
	}
	return fmt.Sprintf(
		"certify source implementation %s %q: %s",
		e.Operation,
		e.Subject,
		e.Reason,
	)
}
