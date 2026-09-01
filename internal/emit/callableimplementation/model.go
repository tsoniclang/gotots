package callableimplementation

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/types"
	"path/filepath"
	"slices"
	"strings"

	implementationcontract "github.com/tsoniclang/gotots/internal/contracts/implementation"
	"github.com/tsoniclang/gotots/internal/load"
)

const SchemaVersion = 3

type Variant string

const (
	VariantInvalid Variant = ""
	VariantSource  Variant = "source"
	VariantKernel  Variant = "kernel"
)

func (v Variant) Valid() bool {
	return v == VariantSource || v == VariantKernel
}

type Document struct {
	SchemaVersion        int                             `json:"schemaVersion"`
	SourceProgramDigest  string                          `json:"sourceProgramDigest"`
	Package              PackageDocument                 `json:"package"`
	Build                BuildDocument                   `json:"build"`
	Compilation          CompilationDocument             `json:"compilation"`
	Source               string                          `json:"source"`
	Output               string                          `json:"output"`
	CertificationSources []string                        `json:"certificationSources,omitempty"`
	Envelope             implementationcontract.Envelope `json:"equivalenceEnvelope"`
	Callables            []CallableDocument              `json:"callables"`
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

type CallableDocument struct {
	SourceIdentity   string  `json:"sourceIdentity"`
	SourceSignature  string  `json:"sourceSignature"`
	SourceBodyDigest string  `json:"sourceBodyDigest"`
	Variant          Variant `json:"variant"`
	Export           string  `json:"export"`
}

type Module struct {
	sourceProgramDigest  string
	packagePath          string
	modulePath           string
	moduleVersion        string
	sourcePath           string
	outputPath           string
	sourceDigest         string
	digest               string
	envelope             implementationcontract.Envelope
	callableClaims       []CallableDocument
	certificationSources []CertificationSource
}

func (m Module) PackagePath() string                           { return m.packagePath }
func (m Module) ModulePath() string                            { return m.modulePath }
func (m Module) ModuleVersion() string                         { return m.moduleVersion }
func (m Module) SourcePath() string                            { return m.sourcePath }
func (m Module) OutputPath() string                            { return m.outputPath }
func (m Module) SourceDigest() string                          { return m.sourceDigest }
func (m Module) Digest() string                                { return m.digest }
func (m Module) Envelope() implementationcontract.EnvelopeKind { return m.envelope.Kind }
func (m Module) EquivalenceEnvelope() implementationcontract.Envelope {
	result := m.envelope
	result.PreservedObservables = slices.Clone(result.PreservedObservables)
	result.Evidence = slices.Clone(result.Evidence)
	return result
}
func (m Module) CallableClaims() []CallableDocument {
	return slices.Clone(m.callableClaims)
}
func (m Module) CertificationSources() []CertificationSource {
	return slices.Clone(m.certificationSources)
}

type CertificationSource struct {
	sourcePath   string
	sourceDigest string
}

func NewCertificationSource(
	sourcePath string,
	sourceDigest string,
) (CertificationSource, error) {
	decoded, err := hex.DecodeString(sourceDigest)
	if !filepath.IsAbs(sourcePath) || filepath.Clean(sourcePath) != sourcePath ||
		!strings.HasSuffix(sourcePath, ".d.ts") || err != nil || len(decoded) != sha256.Size {
		return CertificationSource{}, &Error{
			Operation: "admit certification source",
			Subject:   sourcePath,
			Reason:    "source evidence is invalid",
		}
	}
	return CertificationSource{
		sourcePath: sourcePath, sourceDigest: sourceDigest,
	}, nil
}

func (s CertificationSource) SourcePath() string   { return s.sourcePath }
func (s CertificationSource) SourceDigest() string { return s.sourceDigest }

func (s CertificationSource) Valid() bool {
	decoded, err := hex.DecodeString(s.sourceDigest)
	return filepath.IsAbs(s.sourcePath) && filepath.Clean(s.sourcePath) == s.sourcePath &&
		strings.HasSuffix(s.sourcePath, ".d.ts") && err == nil && len(decoded) == sha256.Size
}

type Implementation struct {
	function         *types.Func
	sourceIdentity   string
	sourceSignature  string
	sourceBodyDigest string
	variant          Variant
	export           string
	module           Module
}

func (i Implementation) Function() *types.Func    { return i.function }
func (i Implementation) SourceIdentity() string   { return i.sourceIdentity }
func (i Implementation) SourceSignature() string  { return i.sourceSignature }
func (i Implementation) SourceBodyDigest() string { return i.sourceBodyDigest }
func (i Implementation) Variant() Variant         { return i.variant }
func (i Implementation) Export() string           { return i.export }
func (i Implementation) Module() Module           { return i.module }

type Prepared struct {
	buildProfile        load.BuildProfile
	compilation         CompilationDocument
	sourceProgramDigest string
	modules             []Module
	digest              string
}

func (p *Prepared) SourceProgramDigest() string {
	if p == nil {
		return ""
	}
	return p.sourceProgramDigest
}

func (p *Prepared) Modules() []Module {
	if p == nil {
		return nil
	}
	return slicesCloneModules(p.modules)
}

type Certificate struct {
	buildProfile        load.BuildProfile
	compilation         CompilationDocument
	sourceProgramDigest string
	byFunction          map[*types.Func]Implementation
	byIdentity          map[string]Implementation
	modules             []Module
	digest              string
}

func (c *Certificate) Valid() bool {
	return c != nil && c.buildProfile.Valid() && c.compilation.valid() &&
		validSHA256(c.sourceProgramDigest) &&
		len(c.byFunction) != 0 && len(c.byIdentity) == len(c.byFunction) &&
		len(c.modules) != 0 && c.digest != ""
}

func (c *Certificate) SupportsCompilation(integers string, evaluationOrder string) bool {
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

func (c *Certificate) ForFunction(function *types.Func) (Implementation, bool) {
	if !c.Valid() || function == nil {
		return Implementation{}, false
	}
	selected, ok := c.byFunction[function.Origin()]
	return selected, ok
}

func (c *Certificate) ImplementationByIdentity(
	sourceIdentity string,
) (Implementation, bool) {
	if !c.Valid() || sourceIdentity == "" {
		return Implementation{}, false
	}
	selected, ok := c.byIdentity[sourceIdentity]
	return selected, ok
}

func (c *Certificate) Implementations() []Implementation {
	if !c.Valid() {
		return nil
	}
	result := make([]Implementation, 0, len(c.byIdentity))
	for _, selected := range c.byIdentity {
		result = append(result, selected)
	}
	slices.SortFunc(result, func(left, right Implementation) int {
		if left.sourceIdentity < right.sourceIdentity {
			return -1
		}
		if left.sourceIdentity > right.sourceIdentity {
			return 1
		}
		return 0
	})
	return result
}

func (c *Certificate) Modules() []Module {
	if !c.Valid() {
		return nil
	}
	return slices.Clone(c.modules)
}

type Error struct {
	Operation string
	Subject   string
	Reason    string
}

func (e *Error) Error() string {
	if e.Subject == "" {
		return fmt.Sprintf("certify callable implementation %s: %s", e.Operation, e.Reason)
	}
	return fmt.Sprintf(
		"certify callable implementation %s %q: %s",
		e.Operation,
		e.Subject,
		e.Reason,
	)
}
