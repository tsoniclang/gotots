package certify

import (
	"bytes"
	"os"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/externals"
)

type Certificate struct {
	manifest     externals.Manifest
	providerRoot string
}

func Verify(config Config) (*Certificate, error) {
	resolved, err := resolveConfig(config)
	if err != nil {
		return nil, err
	}
	checkedBytes, err := os.ReadFile(resolved.manifestPath)
	if err != nil {
		return nil, certifyError(
			"read manifest",
			resolved.manifestPath,
			err.Error(),
		)
	}
	checked, err := externals.Parse(checkedBytes)
	if err != nil {
		return nil, err
	}
	canonical, err := externals.Encode(checked)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(canonical, checkedBytes) {
		return nil, certifyError(
			"verify manifest",
			resolved.manifestPath,
			"checked bytes are not canonical",
		)
	}
	generated, err := Generate(config)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(checkedBytes, generated) {
		return nil, certifyError(
			"verify manifest",
			resolved.manifestPath,
			"checked manifest differs from regenerated evidence",
		)
	}
	return &Certificate{
		manifest:     checked,
		providerRoot: resolved.providerRoot,
	}, nil
}

func (c *Certificate) Valid() bool {
	if c == nil || c.manifest.Digest() == "" || c.providerRoot == "" {
		return false
	}
	_, ok := c.manifest.BuildProfile()
	return ok && len(c.manifest.Bindings()) != 0
}

func (c *Certificate) ManifestDigest() string {
	if c == nil {
		return ""
	}
	return c.manifest.Digest()
}

func (c *Certificate) StandardLibraryDigest() string {
	if c == nil {
		return ""
	}
	return c.manifest.StandardLibraryDigest()
}

func (c *Certificate) ProviderDigest() string {
	if c == nil {
		return ""
	}
	return c.manifest.ProviderDigest()
}

func (c *Certificate) Backend() string {
	if c == nil {
		return ""
	}
	return c.manifest.Backend()
}

func (c *Certificate) IntegerRepresentation() string {
	if c == nil {
		return ""
	}
	return c.manifest.IntegerRepresentation()
}

func (c *Certificate) ConcurrencySemantics() string {
	if c == nil {
		return ""
	}
	return c.manifest.ConcurrencySemantics()
}

func (c *Certificate) BuildProfile() (environmentcontract.BuildProfile, bool) {
	if c == nil {
		return environmentcontract.BuildProfile{}, false
	}
	return c.manifest.BuildProfile()
}

func (c *Certificate) Bindings() []externals.Binding {
	if !c.Valid() {
		return nil
	}
	return c.manifest.Bindings()
}
