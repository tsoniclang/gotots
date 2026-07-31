package certify

import (
	"sort"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	runtimecontract "github.com/tsoniclang/gotots/internal/contracts/runtime"
)

type Certificate struct {
	manifest     gostdlib.Manifest
	toolchainKey string
	providerRoot string
	runtime      runtimecontract.Requirements
}

func Verify(config Config) (*Certificate, error) {
	resolved, err := resolveConfig(config)
	if err != nil {
		return nil, err
	}
	checkedBytes, checked, err := readManifest(resolved.manifestPath)
	if err != nil {
		return nil, err
	}
	generated, err := Generate(config)
	if err != nil {
		return nil, err
	}
	if err := compareCanonical(checkedBytes, generated); err != nil {
		return nil, err
	}
	runtimeRequirements, runtimeDigest, err := readRuntimeContract(
		resolved.runtimeContractPath,
	)
	if err != nil {
		return nil, err
	}
	if runtimeDigest != checked.RuntimeDigest() {
		return nil, certifyError(
			"verify runtime contract",
			resolved.runtimeContractPath,
			"content digest does not match the checked manifest",
		)
	}
	selectedToolchain, err := inspectToolchain(resolved)
	if err != nil {
		return nil, err
	}
	return &Certificate{
		manifest:     checked,
		toolchainKey: selectedToolchain.key,
		providerRoot: resolved.providerRoot,
		runtime:      runtimeRequirements,
	}, nil
}

func (c *Certificate) Valid() bool {
	return c != nil && c.manifest.Digest() != "" &&
		c.toolchainKey != "" && c.providerRoot != "" && c.runtime.Valid()
}

func (c *Certificate) ManifestDigest() string {
	if c == nil {
		return ""
	}
	return c.manifest.Digest()
}

func (c *Certificate) ToolchainKey() string {
	if c == nil {
		return ""
	}
	return c.toolchainKey
}

func (c *Certificate) ProviderModules() []string {
	if !c.Valid() {
		return nil
	}
	seen := make(map[string]struct{})
	for _, module := range c.manifest.Modules() {
		seen[module.Specifier()] = struct{}{}
	}
	for _, module := range c.manifest.FacetModules() {
		seen[module.Specifier()] = struct{}{}
	}
	modules := make([]string, 0, len(seen))
	for module := range seen {
		modules = append(modules, module)
	}
	sort.Strings(modules)
	return modules
}

func (c *Certificate) RuntimeRequirements() (
	runtimecontract.Requirements,
	bool,
) {
	if !c.Valid() {
		return runtimecontract.Requirements{}, false
	}
	return c.runtime, true
}

func (c *Certificate) Binding(identity string) (gostdlib.Binding, bool) {
	if !c.Valid() {
		return gostdlib.Binding{}, false
	}
	return c.manifest.Binding(identity)
}

func (c *Certificate) Modules() []gostdlib.Module {
	if !c.Valid() {
		return nil
	}
	return c.manifest.Modules()
}

func (c *Certificate) Facet(
	sourceIdentity string,
	kind gostdlib.FacetKind,
	capability gostdlib.FacetCapability,
) (gostdlib.Facet, bool) {
	if !c.Valid() {
		return gostdlib.Facet{}, false
	}
	return c.manifest.Facet(sourceIdentity, kind, capability)
}

func (c *Certificate) GenericCallableFacet(
	sourceIdentity string,
	profileKey string,
) (gostdlib.Facet, bool) {
	if !c.Valid() {
		return gostdlib.Facet{}, false
	}
	return c.manifest.GenericCallableFacet(sourceIdentity, profileKey)
}

func (c *Certificate) ProviderRepresentation(
	module string,
	export string,
) (gostdlib.ProviderRepresentation, bool) {
	if !c.Valid() {
		return gostdlib.ProviderRepresentation{}, false
	}
	return c.manifest.ProviderRepresentation(module, export)
}
