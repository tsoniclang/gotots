// Package selectionfacts materializes the finite, request-declared semantic
// facts needed by conditional provider rules. It is the sole producer of
// those facts; scope and the typed frontend consume the same immutable
// records.
package selectionfacts

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/scope/sourceplan"
	"github.com/tsoniclang/gotots/internal/source"
)

// SchemaVersion identifies the sole current fact-production contract.
const SchemaVersion = 1

// ID is the canonical identity of one definition-selection fact.
type ID struct {
	definition identity.DefinitionID
	kind       contract.SelectionFactKind
}

func NewID(
	definition identity.DefinitionID,
	kind contract.SelectionFactKind,
) (ID, error) {
	if definition.IsZero() || !kind.Valid() {
		return ID{}, fmt.Errorf("selection-fact identity requires definition and kind")
	}
	return ID{definition: definition, kind: kind}, nil
}

func (id ID) Definition() identity.DefinitionID { return id.definition }
func (id ID) Kind() contract.SelectionFactKind  { return id.kind }
func (id ID) String() string {
	if id.definition.IsZero() || !id.kind.Valid() {
		return ""
	}
	return id.definition.String() + "#selection-fact/" + id.kind.String()
}

// Fact is one immutable boolean selection fact and its producer/evidence
// digest.
type Fact struct {
	id             ID
	value          bool
	producerDigest string
	evidenceDigest string
}

func (f Fact) ID() ID                 { return f.id }
func (f Fact) Value() bool            { return f.value }
func (f Fact) ProducerDigest() string { return f.producerDigest }
func (f Fact) EvidenceDigest() string { return f.evidenceDigest }

// Artifact is the exact finite fact set requested by the contract.
type Artifact struct {
	facts       []Fact
	byID        map[ID]*Fact
	fingerprint string
}

func (a *Artifact) Facts() []Fact       { return append([]Fact(nil), a.facts...) }
func (a *Artifact) Fingerprint() string { return a.fingerprint }
func (a *Artifact) Value(
	definition identity.DefinitionID,
	kind contract.SelectionFactKind,
) (bool, bool) {
	id, err := NewID(definition, kind)
	if err != nil {
		return false, false
	}
	fact, ok := a.byID[id]
	if !ok {
		return false, false
	}
	return fact.value, true
}

// Materialize evaluates exactly the contract-requested facts over the one
// transient checker graph.
func Materialize(
	universe *source.Universe,
	graph *structure.Graph,
	index *structure.TransientIndex,
	plan *sourceplan.Plan,
	selected contract.Contract,
	certified *structure.ProviderArtifact,
) (*Artifact, error) {
	if plan == nil {
		return nil, fmt.Errorf(
			"selection-fact materialization requires a structural-source plan",
		)
	}
	return materialize(
		universe, graph, index, plan, selected, certified, false,
	)
}

// MaterializeForAudit derives every requested fact from the one local
// transient graph before producing a certified provider artifact.
func MaterializeForAudit(
	universe *source.Universe,
	graph *structure.Graph,
	index *structure.TransientIndex,
	selected contract.Contract,
) (*Artifact, error) {
	return materialize(
		universe, graph, index, nil, selected, nil, true,
	)
}

func materialize(
	universe *source.Universe,
	graph *structure.Graph,
	index *structure.TransientIndex,
	plan *sourceplan.Plan,
	selected contract.Contract,
	certified *structure.ProviderArtifact,
	audit bool,
) (*Artifact, error) {
	if universe == nil || !universe.Hydrated() {
		return nil, fmt.Errorf(
			"selection facts require the selectively hydrated universe",
		)
	}
	out := &Artifact{byID: map[ID]*Fact{}}
	evidence, err := buildEvidenceIndex(universe, graph)
	if err != nil {
		return nil, err
	}
	loadedPackages := map[identity.PackageID]*source.LoadedPackage{}
	for _, pkg := range universe.Packages() {
		loadedPackages[pkg.ID()] = pkg
	}
	for _, packageGraph := range graph.Packages() {
		loaded := loadedPackages[packageGraph.ID()]
		if loaded == nil {
			return nil, fmt.Errorf("definition package %s is absent from source universe", packageGraph.ID())
		}
		syntheticObjects := cgoSyntheticObjects(universe, loaded)
		for _, definition := range packageGraph.Definitions() {
			for _, kind := range selected.RequestedFacts(definition.ID(), packageGraph.ID()) {
				id, _ := NewID(definition.ID(), kind)
				value, evidenceDigest, producer := false, "", ""
				certifiedAuthority, err := factUsesCertifiedAuthority(
					plan, definition.ID(), packageGraph.ID(), audit,
				)
				if err != nil {
					return nil, err
				}
				if certifiedAuthority {
					stored, present, err := certifiedFact(
						certified, definition.ID(), kind,
					)
					if err != nil {
						return nil, err
					}
					if !present {
						return nil, fmt.Errorf(
							"certified definition %s lacks requested fact %s",
							definition.ID(), kind,
						)
					}
					value = stored.Value()
					evidenceDigest = stored.EvidenceDigest()
					producer = stored.ProducerDigest()
				} else {
					value, err = materializeOne(
						kind, definition, loaded, index, syntheticObjects,
					)
					if err != nil {
						return nil, err
					}
					producer = producerDigest(
						universe,
						universe.Toolchain().BinaryDigest(),
						kind.String(),
					)
					evidenceDigest, err = evidence.digest(
						definition.ID(), kind, value,
					)
					if err != nil {
						return nil, err
					}
				}
				fact := Fact{
					id: id, value: value,
					producerDigest: producer,
					evidenceDigest: evidenceDigest,
				}
				out.facts = append(out.facts, fact)
			}
		}
	}
	sort.Slice(out.facts, func(i, j int) bool {
		return out.facts[i].id.String() < out.facts[j].id.String()
	})
	for index := range out.facts {
		fact := &out.facts[index]
		if _, duplicate := out.byID[fact.id]; duplicate {
			return nil, fmt.Errorf(
				"duplicate selection fact %s", fact.id.String(),
			)
		}
		out.byID[fact.id] = fact
	}
	hash := sha256.New()
	fmt.Fprintf(hash, "selection-facts-schema:%d\n", SchemaVersion)
	for _, fact := range out.facts {
		fmt.Fprintf(hash, "%s|%t|%s|%s\n",
			fact.id.String(), fact.value, fact.producerDigest, fact.evidenceDigest)
	}
	out.fingerprint = fmt.Sprintf("%x", hash.Sum(nil))
	return out, nil
}

func factUsesCertifiedAuthority(
	plan *sourceplan.Plan,
	definition identity.DefinitionID,
	pkg identity.PackageID,
	audit bool,
) (bool, error) {
	if audit || definition.ImplicitOp().Valid() {
		return false, nil
	}
	if definition.SyntheticRole().Valid() {
		decision, present := plan.SyntheticFor(pkg)
		if !present {
			return false, fmt.Errorf(
				"synthetic definition %s has no source-plan owner",
				definition,
			)
		}
		return decision.Kind() == sourceplan.KindCertifiedGraph, nil
	}
	decision, present := plan.For(definition.File())
	if !present {
		return false, fmt.Errorf(
			"definition %s has no source-plan file", definition,
		)
	}
	return decision.Kind() == sourceplan.KindCertifiedGraph, nil
}

func certifiedFact(
	artifact *structure.ProviderArtifact,
	definition identity.DefinitionID,
	kind contract.SelectionFactKind,
) (structure.CertifiedFact, bool, error) {
	if artifact == nil {
		return structure.CertifiedFact{}, false, nil
	}
	return artifact.SelectionFact(definition, kind)
}

// CertifiedFacts projects the immutable fact artifact into provider transport
// records without changing ownership.
func (a *Artifact) CertifiedFacts() []structure.CertifiedFact {
	out := make([]structure.CertifiedFact, 0, len(a.facts))
	for _, fact := range a.facts {
		certified, err := structure.NewCertifiedFact(
			fact.id.definition,
			fact.id.kind,
			fact.value,
			fact.producerDigest,
			fact.evidenceDigest,
		)
		if err != nil {
			panic(err)
		}
		out = append(out, certified)
	}
	return out
}

func materializeOne(
	kind contract.SelectionFactKind,
	definition structure.ImplementationDefinition,
	pkg *source.LoadedPackage,
	index *structure.TransientIndex,
	synthetic map[types.Object]bool,
) (bool, error) {
	switch kind {
	case contract.SelectionFactCDependent:
		node, present := index.CheckedDefinitionNode(definition.ID())
		if !present {
			node, present = index.DefinitionNode(definition.ID())
		}
		if !present {
			if definition.ID().SyntheticRole().Valid() {
				return true, nil
			}
			if definition.ID().ImplicitOp().Valid() {
				return false, nil
			}
			return false, fmt.Errorf(
				"definition %s lacks local or certified fact evidence",
				definition.ID(),
			)
		}
		value := usesSynthetic(node, pkg.CheckerView(), synthetic)
		return value, nil
	default:
		return false, fmt.Errorf("no producer for selection fact %s", kind)
	}
}

func producerDigest(
	universe *source.Universe,
	binaryDigest string,
	kind string,
) string {
	return digest(
		fmt.Sprintf("selectionfacts/v%d", SchemaVersion),
		kind,
		binaryDigest,
		universe.Toolchain().Version(),
		universe.Toolchain().BuildConfigurationDigest(),
		catalog.StructureDigest(),
	)
}

func cgoSyntheticObjects(
	universe *source.Universe,
	pkg *source.LoadedPackage,
) map[types.Object]bool {
	out := map[types.Object]bool{}
	original := map[string]bool{}
	for _, file := range pkg.Files() {
		original[file.Path()] = true
	}
	view := pkg.CheckerView()
	if view == nil {
		return out
	}
	for _, declaration := range pkg.CheckedDeclarations() {
		if original[universe.Fset().Position(declaration.Pos()).Filename] {
			continue
		}
		ast.Inspect(declaration, func(node ast.Node) bool {
			if identifier, ok := node.(*ast.Ident); ok {
				if object, found := view.DefOf(identifier); found && object != nil {
					out[object] = true
				}
			}
			return true
		})
	}
	return out
}

func usesSynthetic(
	root ast.Node,
	view *source.TypeInfoView,
	synthetic map[types.Object]bool,
) bool {
	if root == nil || view == nil || len(synthetic) == 0 {
		return false
	}
	found := false
	ast.Inspect(root, func(node ast.Node) bool {
		if found || node == nil {
			return false
		}
		if literal, ok := node.(*ast.FuncLit); ok && ast.Node(literal) != root {
			return false
		}
		if identifier, ok := node.(*ast.Ident); ok {
			if object, present := view.UseOf(identifier); present && synthetic[object] {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

func digest(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		fmt.Fprintf(hash, "%d:%s|", len(part), part)
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}
