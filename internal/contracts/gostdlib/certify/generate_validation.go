package certify

import (
	"bytes"
	"fmt"
	"go/types"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func validateSeeds(source []moduleSeed) ([]moduleSeed, error) {
	if len(source) == 0 {
		return nil, certifyError("configure modules", "", "module set is empty")
	}
	result := append([]moduleSeed(nil), source...)
	specifiers := make(map[string]struct{}, len(result))
	sources := make(map[string]struct{}, len(result))
	for index, seed := range result {
		if seed.GoImportPath == "" || seed.GoImportPath == "." ||
			path.Clean(seed.GoImportPath) != seed.GoImportPath ||
			strings.HasPrefix(seed.GoImportPath, "../") ||
			strings.HasPrefix(seed.GoImportPath, "/") {
			return nil, certifyError(
				"configure modules",
				seed.GoImportPath,
				"Go import path is not canonical",
			)
		}
		if _, ok := providerSubpath(seed.Specifier); !ok ||
			path.Clean(seed.SourcePath) != seed.SourcePath ||
			!strings.HasPrefix(seed.SourcePath, "src/") ||
			!strings.HasSuffix(seed.SourcePath, ".ts") {
			return nil, certifyError("configure modules", seed.GoImportPath, "identity is incomplete")
		}
		if index != 0 && result[index-1].GoImportPath >= seed.GoImportPath {
			return nil, certifyError(
				"configure modules",
				seed.GoImportPath,
				"modules are not strictly ordered",
			)
		}
		if _, duplicate := specifiers[seed.Specifier]; duplicate {
			return nil, certifyError("configure modules", seed.Specifier, "specifier is duplicated")
		}
		if _, duplicate := sources[seed.SourcePath]; duplicate {
			return nil, certifyError("configure modules", seed.SourcePath, "source is duplicated")
		}
		specifiers[seed.Specifier] = struct{}{}
		sources[seed.SourcePath] = struct{}{}
	}
	return result, nil
}

func verifyPublicName(name string, targetType string) error {
	if name == "" || targetType == "" {
		return fmt.Errorf("public symbol identity is incomplete")
	}
	for _, forbidden := range []string{
		"$argument",
		"__from_",
		"$cooperative_",
		"$contract",
		"$state",
	} {
		if strings.Contains(name, forbidden) || strings.Contains(targetType, forbidden) {
			return fmt.Errorf("public symbol contains encoded ABI spelling %q", forbidden)
		}
	}
	return nil
}

func verifyProviderBoundaryCoverage(
	source goSurface,
	modules []gostdlib.ModuleDocument,
	facets []gostdlib.FacetModuleDocument,
) error {
	if err := verifyCallableParameterBindings(source, modules); err != nil {
		return err
	}
	if err := verifyDefinedCallableEffects(source, modules, facets); err != nil {
		return err
	}
	return verifyProviderProfileInterfaceClosure(source, modules, facets)
}

func compareCanonical(left []byte, right []byte) error {
	if bytes.Equal(left, right) {
		return nil
	}
	return certifyError(
		"verify manifest",
		"canonical bytes",
		"checked manifest differs from independently regenerated evidence",
	)
}

func readManifest(path string) ([]byte, gostdlib.Manifest, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, gostdlib.Manifest{}, certifyError("read manifest", path, err.Error())
	}
	manifest, err := gostdlib.Parse(payload)
	if err != nil {
		return nil, gostdlib.Manifest{}, err
	}
	canonical, err := gostdlib.Encode(manifest)
	if err != nil {
		return nil, gostdlib.Manifest{}, err
	}
	return canonical, manifest, nil
}

// implementationBehavior is the mechanically derived behavior evidence of
// one checked implementation body: its closed disposition and its
// conservative private value-level dependency edges.
type implementationBehavior struct {
	disposition  gostdlib.ImplementationDisposition
	dependencies []string
	// discovered maps each private in-provider dependency identity to one
	// declaration node handle for closure continuation.
	discovered map[string]string
}

// typeOnlyDeclarationKind reports declaration kinds that certify a type
// contract rather than executable behavior.
func typeOnlyDeclarationKind(kind tsgo.SyntaxKind) bool {
	switch kind {
	case tsgo.SyntaxKindInterfaceDeclaration,
		tsgo.SyntaxKindTypeAliasDeclaration,
		tsgo.SyntaxKindTypeParameter,
		tsgo.SyntaxKindPropertySignature,
		tsgo.SyntaxKindMethodSignature:
		return true
	default:
		return false
	}
}

// deriveImplementationBehavior walks one binding's checked implementation
// declarations, resolves every value-level reference to exact symbols, and
// derives the closed disposition plus conservative dependency edges. An
// unresolved call fails certification; message strings never participate.
func deriveImplementationBehavior(
	config resolvedConfig,
	project *tsgo.ProjectInspection,
	subject string,
	handles []string,
) (implementationBehavior, error) {
	return deriveBehavior(config, project, subject, handles, false)
}

// deriveConstructionBehavior derives only the construction behavior of one
// public class declaration; member behavior joins per method binding.
func deriveConstructionBehavior(
	config resolvedConfig,
	project *tsgo.ProjectInspection,
	subject string,
	handles []string,
) (implementationBehavior, error) {
	return deriveBehavior(config, project, subject, handles, true)
}

func deriveBehavior(
	config resolvedConfig,
	project *tsgo.ProjectInspection,
	subject string,
	handles []string,
	constructionOnly bool,
) (implementationBehavior, error) {
	behavior := implementationBehavior{
		disposition: gostdlib.DispositionImplemented,
		discovered:  make(map[string]string),
	}
	edges := make(map[string]struct{})
	for _, handle := range handles {
		var references []tsgo.BodyValueReference
		var err error
		if constructionOnly {
			references, err = project.ConstructionBodyReferences(handle)
		} else {
			references, err = project.ImplementationBodyReferences(handle)
		}
		if err != nil {
			return implementationBehavior{}, err
		}
		for _, reference := range references {
			if !reference.Resolved {
				if reference.CallPosition {
					return implementationBehavior{}, certifyError(
						"certify implementation",
						subject,
						"implementation body contains an unresolved call",
					)
				}
				continue
			}
			if reference.Local || reference.TypePosition {
				continue
			}
			for _, declaration := range reference.Declarations {
				relative := providerSourceRelativePath(
					config,
					declaration.Path,
				)
				if relative == "" {
					// Outside the provider's own sources: the shared
					// certified runtime, ambient host declarations, or the
					// certified node backend platform.
					continue
				}
				if typeOnlyDeclarationKind(declaration.Kind) {
					continue
				}
				identity := gostdlib.ImplementationSiteIdentity(
					relative,
					reference.Name,
					declaration.Index,
				)
				if _, duplicate := edges[identity]; !duplicate {
					edges[identity] = struct{}{}
					behavior.dependencies = append(
						behavior.dependencies,
						identity,
					)
					behavior.discovered[identity] = tsgoNodeHandle(
						declaration,
					)
				}
				if gostdlib.CanonicalPlaceholderDependency(identity) {
					behavior.disposition = gostdlib.DispositionPlaceholder
				}
			}
		}
	}
	sort.Strings(behavior.dependencies)
	return behavior, nil
}

// providerSourceRelativePath resolves one absolute declaration path to its
// provider-relative form when it belongs to the provider's own sources
// under src/; every other location is a certified terminal.
func providerSourceRelativePath(
	config resolvedConfig,
	sourcePath string,
) string {
	relative := relativeProviderPath(config, sourcePath)
	if relative == "" || filepath.IsAbs(relative) {
		return ""
	}
	slashed := filepath.ToSlash(relative)
	if !strings.HasPrefix(slashed, "src/") {
		return ""
	}
	return slashed
}

// tsgoNodeHandle rebuilds the canonical node handle of one declaration
// site for closure continuation.
func tsgoNodeHandle(declaration tsgo.BodyReferenceDeclaration) string {
	return fmt.Sprintf(
		"%d.%d.%s",
		declaration.Index,
		uint32(declaration.Kind),
		declaration.Path,
	)
}

// certifyPrivateImplementations expands the certified private dependency
// closure: every discovered in-provider dependency of a public binding is
// itself certified with one disposition and its own conservative edges,
// so a placeholder moved behind a private helper remains detected.
func certifyPrivateImplementations(
	config resolvedConfig,
	project *tsgo.ProjectInspection,
	public map[string]struct{},
	worklist map[string]string,
) ([]gostdlib.ImplementationDocument, error) {
	certified := make(map[string]gostdlib.ImplementationDocument)
	pending := make([]string, 0, len(worklist))
	for identity := range worklist {
		pending = append(pending, identity)
	}
	sort.Strings(pending)
	for len(pending) != 0 {
		identity := pending[0]
		pending = pending[1:]
		if _, done := certified[identity]; done {
			continue
		}
		if _, isPublic := public[identity]; isPublic {
			continue
		}
		if gostdlib.CanonicalPlaceholderDependency(identity) {
			certified[identity] = gostdlib.ImplementationDocument{
				Identity:    identity,
				Disposition: gostdlib.DispositionPlaceholder,
			}
			continue
		}
		handle, ok := worklist[identity]
		if !ok || handle == "" {
			return nil, certifyError(
				"certify implementation",
				identity,
				"private dependency has no declaration evidence",
			)
		}
		behavior, err := deriveImplementationBehavior(
			config,
			project,
			identity,
			[]string{handle},
		)
		if err != nil {
			return nil, err
		}
		certified[identity] = gostdlib.ImplementationDocument{
			Identity:     identity,
			Disposition:  behavior.disposition,
			Dependencies: behavior.dependencies,
		}
		for dependency, dependencyHandle := range behavior.discovered {
			if _, done := certified[dependency]; done {
				continue
			}
			if _, queued := worklist[dependency]; !queued {
				worklist[dependency] = dependencyHandle
				pending = append(pending, dependency)
			}
		}
	}
	result := make([]gostdlib.ImplementationDocument, 0, len(certified))
	for _, document := range certified {
		result = append(result, document)
	}
	sort.Slice(result, func(left, right int) bool {
		return result[left].Identity < result[right].Identity
	})
	return result, nil
}

// implementationEvidence accumulates the public implementation sites and
// the private dependency worklist across all certified modules.
type implementationEvidence struct {
	public   map[string]struct{}
	worklist map[string]string
}

func newImplementationEvidence() *implementationEvidence {
	return &implementationEvidence{
		public:   make(map[string]struct{}),
		worklist: make(map[string]string),
	}
}

// certifyBindingBehavior derives one binding's implementation sites,
// disposition, and conservative dependencies, and accumulates the private
// dependency worklist for closure expansion.
func certifyBindingBehavior(
	config resolvedConfig,
	project *tsgo.ProjectInspection,
	subject string,
	name string,
	handles []string,
	evidence *implementationEvidence,
) ([]string, implementationBehavior, error) {
	return certifyBehavior(
		config, project, subject, name, handles, evidence, false,
	)
}

// certifyConstructionBehavior certifies one public class export's
// construction behavior; member behavior joins per method binding.
func certifyConstructionBehavior(
	config resolvedConfig,
	project *tsgo.ProjectInspection,
	subject string,
	name string,
	handles []string,
	evidence *implementationEvidence,
) ([]string, implementationBehavior, error) {
	return certifyBehavior(
		config, project, subject, name, handles, evidence, true,
	)
}

func certifyBehavior(
	config resolvedConfig,
	project *tsgo.ProjectInspection,
	subject string,
	name string,
	handles []string,
	evidence *implementationEvidence,
	constructionOnly bool,
) ([]string, implementationBehavior, error) {
	resolved, err := project.ResolveDeclarationHandles(handles)
	if err != nil {
		return nil, implementationBehavior{}, err
	}
	behavior, err := deriveBehavior(
		config,
		project,
		subject,
		resolved,
		constructionOnly,
	)
	if err != nil {
		return nil, implementationBehavior{}, err
	}
	sites := make([]string, 0, len(resolved))
	for _, handle := range resolved {
		index, _, sourcePath, parseErr := tsgo.ParseNodeHandle(handle)
		if parseErr != nil {
			return nil, implementationBehavior{}, parseErr
		}
		relative := providerSourceRelativePath(config, sourcePath)
		if relative == "" {
			return nil, implementationBehavior{}, certifyError(
				"certify implementation",
				subject,
				"implementation site is outside the provider sources",
			)
		}
		sites = append(sites, gostdlib.ImplementationSiteIdentity(
			relative,
			name,
			index,
		))
	}
	sort.Strings(sites)
	for _, site := range sites {
		evidence.public[site] = struct{}{}
	}
	for dependency, handle := range behavior.discovered {
		if _, queued := evidence.worklist[dependency]; !queued {
			evidence.worklist[dependency] = handle
		}
	}
	return sites, behavior, nil
}

func verifyGenericOperationBindings(
	source goSurface,
	modules []gostdlib.ModuleDocument,
	configured map[string][]gostdlib.GenericOperationDocument,
) error {
	bound := make(map[string]struct{}, len(configured))
	for _, module := range modules {
		for _, binding := range module.Bindings {
			if len(binding.GenericOperations) != 0 {
				bound[binding.Identity] = struct{}{}
			}
		}
	}
	for identity, operations := range configured {
		evidence, ok := source.objects[identity]
		if !ok {
			return certifyError(
				"configure generic operations",
				identity,
				"selected-GOROOT declaration is absent",
			)
		}
		function, ok := evidence.object.(*types.Func)
		if !ok {
			return certifyError(
				"configure generic operations",
				identity,
				"operation-set owner is not a function",
			)
		}
		signature, _ := function.Type().(*types.Signature)
		typeParameterCount := 0
		callableParameterCount := 0
		if signature != nil {
			typeParameterCount = signature.RecvTypeParams().Len() +
				signature.TypeParams().Len()
			callableParameterCount = signature.Params().Len()
		}
		for _, operation := range operations {
			for _, reference := range append(
				append(
					[]gostdlib.ContractTypeDocument(nil),
					operation.Parameters...,
				),
				operation.Results...,
			) {
				if err := verifyContractTypeParameters(
					reference,
					typeParameterCount,
					callableParameterCount,
				); err != nil {
					return certifyError(
						"configure generic operations",
						identity,
						err.Error(),
					)
				}
			}
		}
		if _, ok := bound[identity]; !ok {
			return certifyError(
				"configure generic operations",
				identity,
				"provider export is absent",
			)
		}
	}
	return nil
}

// facetImplementationSites records one facet-module export's checked
// implementation sites and seeds the private closure so its behavior is
// certified as implementation documents with one disposition owner.
func facetImplementationSites(
	config resolvedConfig,
	project *tsgo.ProjectInspection,
	subject string,
	name string,
	handles []string,
	evidence *implementationEvidence,
) ([]string, error) {
	resolved, err := project.ResolveDeclarationHandles(handles)
	if err != nil {
		return nil, err
	}
	sites := make([]string, 0, len(resolved))
	for _, handle := range resolved {
		index, _, sourcePath, parseErr := tsgo.ParseNodeHandle(handle)
		if parseErr != nil {
			return nil, parseErr
		}
		relative := providerSourceRelativePath(config, sourcePath)
		if relative == "" {
			return nil, certifyError(
				"certify implementation",
				subject,
				"implementation site is outside the provider sources",
			)
		}
		site := gostdlib.ImplementationSiteIdentity(relative, name, index)
		sites = append(sites, site)
		if _, isPublic := evidence.public[site]; isPublic {
			continue
		}
		if _, queued := evidence.worklist[site]; !queued {
			evidence.worklist[site] = handle
		}
	}
	sort.Strings(sites)
	return sites, nil
}
