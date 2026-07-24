package stagecheck

import (
	"fmt"
	"go/ast"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

type checkerLocalDeclarationGroup struct {
	scope identity.OccurrenceID
	class identity.SemanticObjectClass
}

type checkerLocalDeclaration struct {
	object types.Object
	source identity.OccurrenceID
}

func (verifier *checkerSemanticVerifier) independentLocalDeclarationIDs() (
	map[types.Object]identity.SemanticDeclarationID,
	error,
) {
	byObject := map[types.Object]identity.OccurrenceID{}
	for object, occurrenceID := range verifier.checkerSourceByObject {
		if _, local := independentLocalDeclarationClass(object); !local {
			continue
		}
		if existing := byObject[object]; !existing.IsZero() &&
			existing != occurrenceID {
			return nil, fmt.Errorf(
				"local declaration %s has sources %s and %s",
				object.Name(), existing, occurrenceID,
			)
		}
		byObject[object] = occurrenceID
	}
	groups := map[checkerLocalDeclarationGroup][]checkerLocalDeclaration{}
	for object, source := range byObject {
		class, _ := independentLocalDeclarationClass(object)
		scope, err := verifier.independentBindingScope(
			object, source, verifier.scopeOwners,
		)
		if err != nil {
			return nil, err
		}
		group := checkerLocalDeclarationGroup{
			scope: scope,
			class: class,
		}
		groups[group] = append(
			groups[group],
			checkerLocalDeclaration{object: object, source: source},
		)
	}
	out := map[types.Object]identity.SemanticDeclarationID{}
	for group, records := range groups {
		sort.Slice(records, func(left, right int) bool {
			leftSource := records[left].source
			rightSource := records[right].source
			if leftSource.Span().Start() != rightSource.Span().Start() {
				return leftSource.Span().Start() <
					rightSource.Span().Start()
			}
			if leftSource.Span().End() != rightSource.Span().End() {
				return leftSource.Span().End() <
					rightSource.Span().End()
			}
			return records[left].object.Name() <
				records[right].object.Name()
		})
		for ordinal, record := range records {
			id, err := identity.NewOccurrenceDeclarationID(
				group.scope,
				record.source,
				group.class,
				record.object.Name(),
				ordinal,
			)
			if err != nil {
				return nil, err
			}
			out[record.object] = id
		}
	}
	return out, nil
}

func (verifier *checkerSemanticVerifier) deriveIndependentDefinitionOwnership() error {
	for definition := range verifier.expected.definitions {
		node, present := verifier.definitionNode(definition)
		if !present {
			if definition.ImplicitOp().Valid() {
				continue
			}
			return fmt.Errorf(
				"definition %s has no checker ownership node",
				definition,
			)
		}
		objects := verifier.independentDeclarationObjects(node)
		if len(objects) == 0 {
			continue
		}
		var sources []identity.OccurrenceID
		if !definition.Root().IsZero() {
			rootReference := verifier.expected.occurrences.reference(
				definition.Root(),
			)
			for _, childReference := range verifier.childReferences(
				rootReference,
			) {
				child := verifier.expected.occurrenceRecord(
					childReference,
				)
				if child.Role() ==
					catalog.RoleDeclarationName {
					sources = append(sources, child.ID())
				}
			}
		}
		if !definition.SyntheticRole().Valid() &&
			len(objects) != len(sources) {
			return fmt.Errorf(
				"definition %s has %d checker objects and %d canonical names",
				definition, len(objects), len(sources),
			)
		}
		for index, object := range objects {
			if object == nil || object.Name() == "_" {
				continue
			}
			if existing := verifier.definitionByObject[object]; !existing.IsZero() &&
				existing != definition {
				return fmt.Errorf(
					"checker object %s has definitions %s and %s",
					object.Name(), existing, definition,
				)
			}
			verifier.definitionByObject[object] = definition
			if definition.SyntheticRole().Valid() {
				continue
			}
			source := sources[index]
			if existing := verifier.sourceByObject[object]; !existing.IsZero() &&
				existing != source {
				return fmt.Errorf(
					"checker object %s has canonical sources %s and %s",
					object.Name(), existing, source,
				)
			}
			verifier.sourceByObject[object] = source
		}
	}
	return nil
}

func (
	verifier *checkerSemanticVerifier,
) deriveIndependentPackageDeclarationSources() error {
	for _, occurrenceReference := range verifier.expected.order {
		occurrence := verifier.expected.occurrenceRecord(
			occurrenceReference,
		)
		occurrenceID := occurrence.ID()
		if occurrence.Role() != catalog.RoleDeclarationName {
			continue
		}
		node, present := verifier.index.OccurrenceNode(occurrenceID)
		identifier, identifierNode := node.(*ast.Ident)
		if !present || !identifierNode {
			continue
		}
		object, defined := verifier.view.DefOf(identifier)
		if !defined || !independentPackageDeclarationObject(object) {
			continue
		}
		if existing := verifier.sourceByObject[object]; !existing.IsZero() &&
			existing != occurrenceID {
			return fmt.Errorf(
				"package declaration %s has canonical sources %s and %s",
				object.Name(), existing, occurrenceID,
			)
		}
		verifier.sourceByObject[object] = occurrenceID
	}
	return nil
}

func (verifier *checkerSemanticVerifier) independentDeclarationObjects(
	node ast.Node,
) []types.Object {
	switch typed := node.(type) {
	case *ast.FuncDecl:
		object, _ := verifier.view.DefOf(typed.Name)
		return []types.Object{object}
	case *ast.TypeSpec:
		object, _ := verifier.view.DefOf(typed.Name)
		return []types.Object{object}
	case *ast.ValueSpec:
		out := make([]types.Object, 0, len(typed.Names))
		for _, name := range typed.Names {
			object, _ := verifier.view.DefOf(name)
			out = append(out, object)
		}
		return out
	default:
		return nil
	}
}

func independentLocalDeclarationClass(
	object types.Object,
) (identity.SemanticObjectClass, bool) {
	if independentPackageDeclarationObject(object) {
		return identity.SemanticObjectInvalid, false
	}
	switch typed := object.(type) {
	case *types.Const:
		return identity.SemanticObjectConstant, true
	case *types.TypeName:
		if _, parameter := typed.Type().(*types.TypeParam); parameter {
			return identity.SemanticObjectInvalid, false
		}
		if typed.IsAlias() {
			return identity.SemanticObjectAlias, true
		}
		return identity.SemanticObjectType, true
	default:
		return identity.SemanticObjectInvalid, false
	}
}

func independentPackageDeclarationObject(object types.Object) bool {
	return object != nil &&
		object.Pkg() != nil &&
		object.Parent() == object.Pkg().Scope() &&
		object.Pkg().Scope().Lookup(object.Name()) == object
}
