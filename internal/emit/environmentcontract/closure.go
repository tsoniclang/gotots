package environmentcontract

import (
	"slices"
	"sort"
	"strings"

	environmentidentity "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	gostdlibcertify "github.com/tsoniclang/gotots/internal/contracts/gostdlib/certify"
)

// ClosureRoot is one provider-routed settled environment declaration
// entering the used-provider closure: its canonical Go identity, joined
// closed demands, and typed provider selections.
type ClosureRoot struct {
	Identity   string
	Demands    []environmentidentity.UseDemand
	Selections []gostdlib.UseSelection
}

// ClosureError is the typed diagnostic of one failed used-provider closure
// join. Every list is one-sided, sorted, and identity-keyed.
type ClosureError struct {
	MissingBindings        []string
	MissingImplementations []string
	UsedProfileBoundaries  []string
	UsedPlaceholders       []string
}

func (e *ClosureError) Error() string {
	var sections []string
	if len(e.MissingBindings) != 0 {
		sections = append(
			sections,
			"used roots without certified bindings: "+
				strings.Join(e.MissingBindings, ", "),
		)
	}
	if len(e.MissingImplementations) != 0 {
		sections = append(
			sections,
			"certified edges without implementation documents: "+
				strings.Join(e.MissingImplementations, ", "),
		)
	}
	if len(e.UsedProfileBoundaries) != 0 {
		sections = append(
			sections,
			"used provider profile boundaries: "+
				strings.Join(e.UsedProfileBoundaries, ", "),
		)
	}
	if len(e.UsedPlaceholders) != 0 {
		sections = append(
			sections,
			"used provider placeholders: "+
				strings.Join(e.UsedPlaceholders, ", "),
		)
	}
	return "used-provider closure: " + strings.Join(sections, "; ")
}

// executableClosureDemand reports whether one closed use demand makes
// provider behavior executable. A type contract alone demands no body.
func executableClosureDemand(demand environmentidentity.UseDemand) bool {
	return demand != environmentidentity.UseDemandTypeContract &&
		demand.Valid()
}

// VerifyProviderClosure joins the provider-routed settled environment roots
// to the certified provider implementation evidence and fails closed on any
// used placeholder, missing node, or unresolved edge. Construction is
// linear in nodes plus edges; unused catalog entries stay outside the
// closure.
func VerifyProviderClosure(
	roots []ClosureRoot,
	certificate *gostdlibcertify.Certificate,
) error {
	if certificate == nil || !certificate.Valid() {
		return &ClosureError{
			MissingBindings: []string{"<provider certificate>"},
		}
	}
	bindings := make(map[string]gostdlib.Binding)
	for _, module := range certificate.Modules() {
		for _, binding := range module.Bindings() {
			bindings[binding.Identity()] = binding
		}
	}
	implementations := make(map[string]gostdlib.ImplementationDocument)
	for _, implementation := range certificate.Implementations() {
		implementations[implementation.Identity] = implementation
	}
	// Public binding bodies certify their behavior inline; index their
	// sites so private edges into public bodies join the same closure.
	for _, module := range certificate.Modules() {
		for _, binding := range module.Bindings() {
			for _, site := range binding.ImplementationSites() {
				if _, private := implementations[site]; private {
					continue
				}
				implementations[site] = gostdlib.ImplementationDocument{
					Identity:     site,
					Disposition:  binding.Disposition(),
					Dependencies: binding.ImplementationDependencies(),
				}
			}
		}
	}
	representationMethods := make(
		map[string]gostdlib.ProviderRepresentationMethodDocument,
	)
	representationTypes := make(map[string]struct{})
	for _, module := range certificate.FacetModules() {
		for _, representation := range module.Representations() {
			for _, identity := range representation.SourceTypes() {
				representationTypes[identity] = struct{}{}
			}
			for _, identity := range representation.SourceInterfaces() {
				representationTypes[identity] = struct{}{}
			}
			for _, method := range representation.Methods() {
				representationMethods[method.SourceIdentity] = method
			}
		}
	}
	failure := &ClosureError{}
	visited := make(map[string]struct{})
	for _, root := range roots {
		executable := false
		for _, demand := range root.Demands {
			if executableClosureDemand(demand) {
				executable = true
			}
		}
		binding, ok := bindings[root.Identity]
		if !ok {
			if method, covered := representationMethods[root.Identity]; covered {
				if executable {
					recordUsedDisposition(
						root.Identity,
						method.Disposition,
						failure,
					)
					expandClosureEdges(
						root.Identity,
						method.Dependencies,
						implementations,
						visited,
						failure,
					)
				}
				continue
			}
			if _, covered := representationTypes[root.Identity]; covered {
				// A representation-covered type joins its behavior through
				// its certified method documents.
				continue
			}
			failure.MissingBindings = append(
				failure.MissingBindings,
				root.Identity,
			)
			continue
		}
		if executable && binding.Disposition().Valid() {
			recordUsedDisposition(
				root.Identity,
				binding.Disposition(),
				failure,
			)
			expandClosureEdges(
				root.Identity,
				binding.ImplementationDependencies(),
				implementations,
				visited,
				failure,
			)
		}
		for _, selection := range root.Selections {
			sites, ok := selectionImplementationSites(
				certificate,
				root.Identity,
				selection,
			)
			if !ok {
				failure.MissingImplementations = append(
					failure.MissingImplementations,
					root.Identity+"|"+selection.EvidenceKey(),
				)
				continue
			}
			for _, site := range sites {
				expandClosureSite(
					root.Identity+"|"+selection.EvidenceKey(),
					site,
					implementations,
					visited,
					failure,
				)
			}
		}
	}
	sort.Strings(failure.MissingBindings)
	sort.Strings(failure.MissingImplementations)
	sort.Strings(failure.UsedProfileBoundaries)
	sort.Strings(failure.UsedPlaceholders)
	failure.MissingBindings = slices.Compact(failure.MissingBindings)
	failure.MissingImplementations = slices.Compact(
		failure.MissingImplementations,
	)
	failure.UsedProfileBoundaries = slices.Compact(
		failure.UsedProfileBoundaries,
	)
	failure.UsedPlaceholders = slices.Compact(failure.UsedPlaceholders)
	if len(failure.MissingBindings) != 0 ||
		len(failure.MissingImplementations) != 0 ||
		len(failure.UsedProfileBoundaries) != 0 ||
		len(failure.UsedPlaceholders) != 0 {
		return failure
	}
	return nil
}

// selectionImplementationSites resolves one typed provider selection to the
// implementation sites of its certified facet or profile.
func selectionImplementationSites(
	certificate *gostdlibcertify.Certificate,
	identity string,
	selection gostdlib.UseSelection,
) ([]string, bool) {
	switch selection.Kind() {
	case gostdlib.UseSelectionFacet:
		kind, capability, _ := selection.Facet()
		facet, ok := certificate.Facet(identity, kind, capability)
		if !ok {
			return nil, false
		}
		return facet.ImplementationSites(), true
	case gostdlib.UseSelectionCallableProfile:
		key, _ := selection.ProfileKey()
		profile, ok := certificate.ProviderCallableProfile(identity, key)
		if !ok {
			return nil, false
		}
		return profile.ImplementationSites(), true
	case gostdlib.UseSelectionStatefulProfile:
		key, _ := selection.ProfileKey()
		profile, ok := certificate.ProviderStatefulProfile(identity, key)
		if !ok {
			return nil, false
		}
		return profile.ImplementationSites(), true
	default:
		return nil, false
	}
}

// expandClosureSite verifies one implementation site and its transitive
// certified dependencies.
func expandClosureSite(
	root string,
	site string,
	implementations map[string]gostdlib.ImplementationDocument,
	visited map[string]struct{},
	failure *ClosureError,
) {
	implementation, ok := implementations[site]
	if !ok {
		failure.MissingImplementations = append(
			failure.MissingImplementations,
			root+" -> "+site,
		)
		return
	}
	recordUsedDisposition(
		root+" -> "+site,
		implementation.Disposition,
		failure,
	)
	expandClosureEdges(
		root,
		implementation.Dependencies,
		implementations,
		visited,
		failure,
	)
}

// expandClosureEdges walks certified dependency edges once each, reporting
// any reached placeholder or missing implementation node against the
// closure root that reached it.
func expandClosureEdges(
	root string,
	edges []string,
	implementations map[string]gostdlib.ImplementationDocument,
	visited map[string]struct{},
	failure *ClosureError,
) {
	pending := append([]string(nil), edges...)
	for len(pending) != 0 {
		edge := pending[0]
		pending = pending[1:]
		if _, done := visited[edge]; done {
			continue
		}
		visited[edge] = struct{}{}
		implementation, ok := implementations[edge]
		if !ok {
			failure.MissingImplementations = append(
				failure.MissingImplementations,
				root+" -> "+edge,
			)
			continue
		}
		recordUsedDisposition(
			root+" -> "+edge,
			implementation.Disposition,
			failure,
		)
		pending = append(pending, implementation.Dependencies...)
	}
}

// recordUsedDisposition records behavior that cannot enter an executable
// selected-provider closure. A profile boundary is valid provider evidence,
// but reaching it proves that the selected product did not exclude it.
func recordUsedDisposition(
	identity string,
	disposition gostdlib.ImplementationDisposition,
	failure *ClosureError,
) {
	switch disposition {
	case gostdlib.DispositionProfileBoundary:
		failure.UsedProfileBoundaries = append(
			failure.UsedProfileBoundaries,
			identity,
		)
	case gostdlib.DispositionPlaceholder:
		failure.UsedPlaceholders = append(
			failure.UsedPlaceholders,
			identity,
		)
	}
}
