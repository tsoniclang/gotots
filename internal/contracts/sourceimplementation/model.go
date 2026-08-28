package sourceimplementation

import (
	"fmt"
	"slices"

	implementationcontract "github.com/tsoniclang/gotots/internal/contracts/implementation"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const SchemaVersion = 2

type EnvelopeKind = implementationcontract.EnvelopeKind

const (
	EnvelopeInvalid           = implementationcontract.EnvelopeInvalid
	EnvelopeExact             = implementationcontract.EnvelopeExact
	EnvelopeInternalAlgorithm = implementationcontract.EnvelopeInternalAlgorithm
)

type Document struct {
	SchemaVersion        int                     `json:"schemaVersion"`
	Package              PackageDocument         `json:"package"`
	Build                BuildDocument           `json:"build"`
	Compilation          CompilationDocument     `json:"compilation"`
	Source               string                  `json:"source"`
	TSConfig             string                  `json:"tsconfig"`
	CertificationSources []string                `json:"certificationSources,omitempty"`
	Envelope             EnvelopeDocument        `json:"equivalenceEnvelope"`
	Exports              []string                `json:"exports"`
	PrivateModules       []PrivateModuleDocument `json:"privateModules,omitempty"`
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

type CompilationDocument struct {
	Integers        string `json:"integers"`
	EvaluationOrder string `json:"evaluationOrder"`
}

func (d CompilationDocument) valid() bool {
	return d.Integers != "" && d.EvaluationOrder != ""
}

type PrivateModuleDocument struct {
	GoFile  string   `json:"goFile"`
	Source  string   `json:"source"`
	Exports []string `json:"exports"`
}

type EnvelopeDocument = implementationcontract.Envelope

type Export struct {
	name        string
	typeString  string
	fingerprint string
}

func (e Export) Name() string        { return e.name }
func (e Export) TypeString() string  { return e.typeString }
func (e Export) Fingerprint() string { return e.fingerprint }

type PrivateModule struct {
	goFile       string
	sourcePath   string
	sourceDigest string
	exports      []Export
	sourceFile   tsgo.SourceFile
}

func (m PrivateModule) GoFile() string              { return m.goFile }
func (m PrivateModule) SourcePath() string          { return m.sourcePath }
func (m PrivateModule) SourceDigest() string        { return m.sourceDigest }
func (m PrivateModule) Exports() []Export           { return slices.Clone(m.exports) }
func (m PrivateModule) SourceFile() tsgo.SourceFile { return m.sourceFile }

type Implementation struct {
	packagePath    string
	modulePath     string
	moduleVersion  string
	sourcePath     string
	digest         string
	sourceDigest   string
	envelope       EnvelopeKind
	exports        []Export
	sourceFile     tsgo.SourceFile
	privateModules []PrivateModule
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
func (i Implementation) PrivateModules() []PrivateModule {
	return slices.Clone(i.privateModules)
}

type Certificate struct {
	byPath      map[string]Implementation
	compilation CompilationDocument
	digest      string
}

type Prepared struct {
	buildProfile load.BuildProfile
	certificate  Certificate
}

func (c *Certificate) Valid() bool {
	return c != nil && c.compilation.valid() && c.digest != "" &&
		len(c.byPath) != 0
}

func (c *Certificate) SupportsCompilation(
	integers string,
	evaluationOrder string,
) bool {
	return c != nil && c.compilation == (CompilationDocument{
		Integers: integers, EvaluationOrder: evaluationOrder,
	})
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

func (c *Certificate) bindCompilation(selected CompilationDocument) error {
	if !selected.valid() {
		return &Error{Operation: "admit", Reason: "compilation profile is incomplete"}
	}
	if !c.compilation.valid() {
		c.compilation = selected
		return nil
	}
	if c.compilation != selected {
		return &Error{Operation: "admit", Reason: "compilation profiles differ"}
	}
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
