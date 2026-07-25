package structure

import (
	"fmt"
	"go/ast"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

func (b *fileBuilder) buildDefinitionParts(
	definition identity.DefinitionID,
	headerID identity.HeaderRegionID,
	boundaryID identity.ExecutionBoundaryID,
	node ast.Node,
	root Occurrence,
	context catalog.DefinitionContext,
) (HeaderRegion, ExecutionBoundary, error) {
	members := []identity.OccurrenceID{root.id}
	if err := b.walkHeader(node, root, &members); err != nil {
		return HeaderRegion{}, ExecutionBoundary{}, err
	}
	headerDigest, err := b.headerDigest(node, members)
	if err != nil {
		return HeaderRegion{}, ExecutionBoundary{}, err
	}
	boundary, err := b.executionBoundary(
		boundaryID, node, root, context,
	)
	if err != nil {
		return HeaderRegion{}, ExecutionBoundary{}, err
	}
	return HeaderRegion{
		id: headerID, digest: headerDigest, members: members,
	}, boundary, nil
}

func (b *fileBuilder) walkHeader(
	node ast.Node,
	parent Occurrence,
	out *[]identity.OccurrenceID,
) error {
	nested, err := children(node, parent.kind)
	if err != nil {
		return b.defect(node, catalog.EdgeInvalid, err.Error())
	}
	for _, child := range nested {
		b.work.CatalogEdges++
		if child.edge.DefinitionEntry() {
			continue
		}
		occurrence, err := b.makeOccurrence(
			child.node, parent.id, child.edge, child.ordinal,
		)
		if err != nil {
			return err
		}
		if err := b.recordOccurrence(
			occurrence, child.node,
		); err != nil {
			return err
		}
		*out = append(*out, occurrence.id)
		b.work.RecordAppends++
		if err := b.walkHeader(child.node, occurrence, out); err != nil {
			return err
		}
	}
	return nil
}

func definitionEntryChildren(
	node ast.Node,
	kind catalog.Kind,
) ([]Child, error) {
	all, err := children(node, kind)
	if err != nil {
		return nil, err
	}
	var entries []Child
	for _, child := range all {
		if child.edge.DefinitionEntry() {
			entries = append(entries, child)
		}
	}
	return entries, nil
}

func (b *fileBuilder) executionBoundary(
	id identity.ExecutionBoundaryID,
	definition ast.Node,
	root Occurrence,
	context catalog.DefinitionContext,
) (ExecutionBoundary, error) {
	boundary := ExecutionBoundary{id: id}
	entries, err := definitionEntryChildren(definition, root.kind)
	if err != nil {
		return ExecutionBoundary{}, err
	}
	definitionKind, isDefinition, err := catalog.DefinitionKind(
		root.kind, context, len(entries) > 0,
	)
	if err != nil || !isDefinition {
		return ExecutionBoundary{}, b.defect(
			definition,
			catalog.EdgeInvalid,
			"unsupported definition shape",
		)
	}
	switch definitionKind {
	case identity.DefinitionFuncDecl, identity.DefinitionFuncLit:
		boundary.kind = BoundaryBlock
	case identity.DefinitionPackageInitializer:
		boundary.kind = BoundaryInitializers
	case identity.DefinitionBodylessDecl:
		boundary.kind = BoundaryBodyless
		boundary.combinedDigest = digestStrings(
			id.Definition().String(), "bodyless-obligation",
		)
		return boundary, nil
	default:
		return ExecutionBoundary{}, b.defect(
			definition,
			catalog.EdgeInvalid,
			"unsupported definition kind",
		)
	}
	for _, entry := range entries {
		occurrence, err := b.makeOccurrence(
			entry.node, root.id, entry.edge, entry.ordinal,
		)
		if err != nil {
			return ExecutionBoundary{}, err
		}
		if err := b.recordOccurrence(
			occurrence, entry.node,
		); err != nil {
			return ExecutionBoundary{}, err
		}
		hash, err := digestSpan(b.raw, occurrence.span)
		if err != nil {
			return ExecutionBoundary{}, err
		}
		boundary.entries = append(boundary.entries, ExecutionEntry{
			id: occurrence.id, hash: hash,
		})
		b.work.RecordAppends++
	}
	parts := make([]string, 0, len(boundary.entries))
	for _, entry := range boundary.entries {
		parts = append(parts, entry.hash)
	}
	boundary.combinedDigest = digestStrings(parts...)
	if boundary.combinedDigest == "" {
		return ExecutionBoundary{}, fmt.Errorf(
			"definition %s has an empty execution digest", id.Definition(),
		)
	}
	return boundary, nil
}
