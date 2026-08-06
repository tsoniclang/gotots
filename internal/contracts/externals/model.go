package externals

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"slices"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
)

const (
	SchemaVersion = 2
	PackageName   = "@gotots/externals"
)

type TargetKind string

const (
	TargetInvalid TargetKind = ""
	TargetModule  TargetKind = "module"
	TargetSource  TargetKind = "source"
)

func (k TargetKind) Valid() bool {
	return k == TargetModule || k == TargetSource
}

type Document struct {
	SchemaVersion                 int               `json:"schemaVersion"`
	PackageName                   string            `json:"packageName"`
	PackageVersion                string            `json:"packageVersion"`
	Backend                       string            `json:"backend"`
	GoVersion                     string            `json:"goVersion"`
	GOOS                          string            `json:"goos"`
	GOARCH                        string            `json:"goarch"`
	CGOEnabled                    bool              `json:"cgoEnabled"`
	BuildTags                     []string          `json:"buildTags"`
	ProviderIntegerRepresentation string            `json:"providerIntegerRepresentation"`
	StandardLibraryDigest         string            `json:"standardLibraryDigest"`
	ProviderDigest                string            `json:"providerDigest"`
	Bindings                      []BindingDocument `json:"bindings"`
	ManifestDigest                string            `json:"manifestDigest,omitempty"`
}

type BindingDocument struct {
	SourceIdentity      string     `json:"sourceIdentity"`
	SourceSignature     string     `json:"sourceSignature"`
	SourceModulePath    string     `json:"sourceModulePath"`
	SourceModuleVersion string     `json:"sourceModuleVersion"`
	SourceLocation      string     `json:"sourceLocation"`
	TargetKind          TargetKind `json:"targetKind"`

	ModuleSpecifier     string `json:"moduleSpecifier,omitempty"`
	Export              string `json:"export,omitempty"`
	ImplementationOwner string `json:"implementationOwner,omitempty"`
	TargetFingerprint   string `json:"targetFingerprint,omitempty"`

	TargetIdentity  string `json:"targetIdentity,omitempty"`
	TargetSignature string `json:"targetSignature,omitempty"`
	TargetLocation  string `json:"targetLocation,omitempty"`
}

type Manifest struct {
	document Document
	payload  []byte
	bindings []Binding
}

type Binding struct {
	document BindingDocument
}

type ManifestError struct {
	Field  string
	Reason string
}

func (e *ManifestError) Error() string {
	if e.Field == "" {
		return "validate external-provider manifest: " + e.Reason
	}
	return fmt.Sprintf(
		"validate external-provider manifest field %q: %s",
		e.Field,
		e.Reason,
	)
}

func Seal(source Document) ([]byte, error) {
	document := cloneDocument(source)
	if document.ManifestDigest != "" {
		return nil, manifestError("manifestDigest", "must be empty before sealing")
	}
	if err := validateDocument(document, false); err != nil {
		return nil, err
	}
	digest, err := documentDigest(document)
	if err != nil {
		return nil, err
	}
	document.ManifestDigest = digest
	return encodeDocument(document)
}

func Parse(payload []byte) (Manifest, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var document Document
	if err := decoder.Decode(&document); err != nil {
		return Manifest{}, manifestError("", err.Error())
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return Manifest{}, manifestError("", err.Error())
	}
	if err := validateDocument(document, true); err != nil {
		return Manifest{}, err
	}
	want := document.ManifestDigest
	document.ManifestDigest = ""
	got, err := documentDigest(document)
	if err != nil {
		return Manifest{}, err
	}
	if got != want {
		return Manifest{}, manifestError(
			"manifestDigest",
			"content digest does not match payload",
		)
	}
	document.ManifestDigest = want
	canonical, err := encodeDocument(document)
	if err != nil {
		return Manifest{}, err
	}
	bindings := make([]Binding, len(document.Bindings))
	for index, binding := range document.Bindings {
		bindings[index] = Binding{document: binding}
	}
	return Manifest{
		document: cloneDocument(document),
		payload:  canonical,
		bindings: bindings,
	}, nil
}

func Encode(manifest Manifest) ([]byte, error) {
	if manifest.document.ManifestDigest == "" || len(manifest.payload) == 0 {
		return nil, manifestError("", "manifest is invalid")
	}
	return bytes.Clone(manifest.payload), nil
}

func (m Manifest) Digest() string {
	return m.document.ManifestDigest
}

func (m Manifest) PackageVersion() string {
	return m.document.PackageVersion
}

func (m Manifest) Backend() string {
	return m.document.Backend
}

func (m Manifest) ProviderDigest() string {
	return m.document.ProviderDigest
}

func (m Manifest) StandardLibraryDigest() string {
	return m.document.StandardLibraryDigest
}

func (m Manifest) ProviderIntegerRepresentation() string {
	return m.document.ProviderIntegerRepresentation
}

func (m Manifest) BuildProfile() (environmentcontract.BuildProfile, bool) {
	profile, err := environmentcontract.NewBuildProfileForToolchain(
		m.document.GoVersion,
		m.document.GOOS,
		m.document.GOARCH,
		m.document.CGOEnabled,
		m.document.BuildTags,
	)
	return profile, err == nil
}

func (m Manifest) Bindings() []Binding {
	return slices.Clone(m.bindings)
}

func (b Binding) SourceIdentity() string {
	return b.document.SourceIdentity
}

func (b Binding) SourceSignature() string {
	return b.document.SourceSignature
}

func (b Binding) SourceModulePath() string {
	return b.document.SourceModulePath
}

func (b Binding) SourceModuleVersion() string {
	return b.document.SourceModuleVersion
}

func (b Binding) SourceLocation() string {
	return b.document.SourceLocation
}

func (b Binding) TargetKind() TargetKind {
	return b.document.TargetKind
}

func (b Binding) ModuleTarget() (string, string, string, string, bool) {
	return b.document.ModuleSpecifier,
		b.document.Export,
		b.document.ImplementationOwner,
		b.document.TargetFingerprint,
		b.document.TargetKind == TargetModule
}

func (b Binding) SourceTarget() (string, string, string, bool) {
	return b.document.TargetIdentity,
		b.document.TargetSignature,
		b.document.TargetLocation,
		b.document.TargetKind == TargetSource
}

func documentDigest(document Document) (string, error) {
	document.ManifestDigest = ""
	payload, err := json.Marshal(document)
	if err != nil {
		return "", manifestError("", err.Error())
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}

func encodeDocument(document Document) ([]byte, error) {
	payload, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, manifestError("", err.Error())
	}
	return append(payload, '\n'), nil
}
