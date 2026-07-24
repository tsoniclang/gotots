package structure

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/source"
)

type pathStep struct {
	node       ast.Node
	occurrence Occurrence
}

type fileBuilder struct {
	file        *source.LoadedFile
	fset        *token.FileSet
	raw         []byte
	displayFile string
	ownerID     OwnerRegionID
	owner       OwnerRegion
	occurrences map[identity.OccurrenceID]Occurrence
	order       []identity.OccurrenceID
	anchors     map[identity.OccurrenceID]bool
	anchorOrder []identity.OccurrenceID
	definitions []ImplementationDefinition
	sites       []DefinitionSite
	headers     []HeaderRegion
	boundaries  []ExecutionBoundary
	path        []pathStep
	work        *Work
	index       *TransientIndex
}

func buildFile(
	file *source.LoadedFile,
	syntax *ast.File,
	work *Work,
	index *TransientIndex,
) (FileGraph, error) {
	ownerID, err := SourceFileOwner(file.ID())
	if err != nil {
		return FileGraph{}, err
	}
	builder := &fileBuilder{
		file:        file,
		fset:        file.PhysicalFileSet(),
		raw:         file.SelectedBytes(),
		displayFile: file.ID().String(),
		ownerID:     ownerID,
		occurrences: map[identity.OccurrenceID]Occurrence{},
		anchors:     map[identity.OccurrenceID]bool{},
		work:        work,
		index:       index,
	}
	index.files[file.ID()] = file
	if err := index.bindCheckedFile(file); err != nil {
		return FileGraph{}, err
	}
	root, err := builder.makeOccurrence(
		syntax, identity.OccurrenceID{}, catalog.EdgeInvalid, 0,
	)
	if err != nil {
		return FileGraph{}, err
	}
	if err := builder.recordOccurrence(root, syntax); err != nil {
		return FileGraph{}, err
	}
	builder.owner.id = ownerID
	builder.owner.members = append(builder.owner.members, root.id)
	builder.work.RecordAppends++
	context, err := catalog.NewDefinitionContext(
		catalog.DefinitionScopePackage, catalog.TokenInvalid,
	)
	if err != nil {
		return FileGraph{}, err
	}
	if err := builder.walkOwner(syntax, root, context); err != nil {
		return FileGraph{}, err
	}
	directives, err := builder.scanDirectives(syntax)
	if err != nil {
		return FileGraph{}, err
	}
	builder.owner.directives = directives
	occurrences := make([]Occurrence, 0, len(builder.order))
	for _, id := range builder.order {
		occurrences = append(occurrences, builder.occurrences[id])
	}
	return FileGraph{
		owner:       builder.owner,
		occurrences: occurrences,
		containment: ContainmentGraph{
			owner: ownerID,
			anchors: append(
				[]identity.OccurrenceID(nil), builder.anchorOrder...,
			),
		},
		definitions: builder.definitions,
		sites:       builder.sites,
		headers:     builder.headers,
		boundaries:  builder.boundaries,
	}, nil
}

func (b *fileBuilder) walkOwner(
	node ast.Node,
	current Occurrence,
	context catalog.DefinitionContext,
) error {
	nested, err := children(node, current.kind)
	if err != nil {
		return b.defect(node, catalog.EdgeInvalid, err.Error())
	}
	childContext, err := contextForChildren(node, current.kind, context)
	if err != nil {
		return b.defect(node, catalog.EdgeInvalid, err.Error())
	}
	for _, child := range nested {
		b.work.CatalogEdges++
		occurrence, err := b.makeOccurrence(
			child.node, current.id, child.edge, child.ordinal,
		)
		if err != nil {
			return err
		}
		definitionKind, isDefinition, err := definitionKind(
			child.node, childContext,
		)
		if err != nil {
			return err
		}
		b.path = append(
			b.path, pathStep{node: child.node, occurrence: occurrence},
		)
		if isDefinition {
			err = b.addDefinition(
				child.node,
				occurrence,
				definitionKind,
				identity.DefinitionID{},
				0,
				childContext,
			)
		} else {
			err = b.recordOccurrence(occurrence, child.node)
			if err == nil {
				b.owner.members = append(
					b.owner.members, occurrence.id,
				)
				b.work.RecordAppends++
				err = b.walkOwner(
					child.node,
					occurrence,
					childContext,
				)
			}
		}
		b.path = b.path[:len(b.path)-1]
		if err != nil {
			return err
		}
	}
	return nil
}

func contextForChildren(
	node ast.Node,
	kind catalog.Kind,
	context catalog.DefinitionContext,
) (catalog.DefinitionContext, error) {
	if kind != catalog.KindGenDecl {
		return context, nil
	}
	declaration, err := tokenEvidence(node)
	if err != nil {
		return catalog.DefinitionContext{}, err
	}
	return context.WithDeclaration(declaration)
}

func definitionKind(
	node ast.Node,
	context catalog.DefinitionContext,
) (identity.DefinitionKind, bool, error) {
	kind, err := Classify(node)
	if err != nil {
		return identity.DefinitionInvalid, false, err
	}
	entries, err := definitionEntryChildren(node, kind)
	if err != nil {
		return identity.DefinitionInvalid, false, err
	}
	return catalog.DefinitionKind(kind, context, len(entries) > 0)
}

func (b *fileBuilder) addDefinition(
	node ast.Node,
	root Occurrence,
	kind identity.DefinitionKind,
	parent identity.DefinitionID,
	pathStart int,
	context catalog.DefinitionContext,
) error {
	definitionID, err := identity.NewSourceDefinitionID(root.id, kind)
	if err != nil {
		return err
	}
	if _, duplicate := b.index.definitions[definitionID]; duplicate {
		return b.defect(node, catalog.EdgeInvalid, "duplicate definition identity")
	}
	if err := b.recordOccurrence(root, node); err != nil {
		return err
	}
	if err := b.index.bindStructuralOwner(
		root.id, definitionID,
	); err != nil {
		return err
	}
	b.index.definitions[definitionID] = node
	entries, err := definitionEntryChildren(node, root.kind)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		b.index.entries[definitionID] = append(
			b.index.entries[definitionID], entry.node,
		)
	}
	headerID, err := identity.NewHeaderRegionID(definitionID)
	if err != nil {
		return err
	}
	boundaryID, err := identity.NewExecutionBoundaryID(definitionID)
	if err != nil {
		return err
	}
	if pathStart < 0 ||
		pathStart >= len(b.path) ||
		b.path[len(b.path)-1].occurrence.id != root.id {
		return b.defect(node, catalog.EdgeInvalid, "definition has no exact containment path")
	}
	if err := b.ensurePath(
		b.path[pathStart : len(b.path)-1],
	); err != nil {
		return err
	}
	header, boundary, err := b.buildDefinitionParts(
		definitionID, headerID, boundaryID, node, root, context,
	)
	if err != nil {
		return err
	}
	b.definitions = append(b.definitions, ImplementationDefinition{
		id:       definitionID,
		owner:    b.ownerID,
		header:   headerID,
		boundary: boundaryID,
		name:     definitionName(node),
	})
	b.sites = append(b.sites, DefinitionSite{
		kind:             DefinitionSiteSource,
		definition:       definitionID,
		owner:            b.ownerID,
		parentDefinition: parent,
		terminal:         root.id,
	})
	b.headers = append(b.headers, header)
	b.boundaries = append(b.boundaries, boundary)
	b.work.RecordAppends += 4

	executableContext, err := catalog.NewDefinitionContext(
		catalog.DefinitionScopeExecutable, catalog.TokenInvalid,
	)
	if err != nil {
		return err
	}
	for entryIndex, entry := range boundary.entries {
		if entryIndex >= len(entries) {
			return b.defect(
				node,
				catalog.EdgeInvalid,
				"execution entry cannot be recovered from its definition",
			)
		}
		entryNode := entries[entryIndex].node
		entryOccurrence, present := b.occurrences[entry.id]
		if !present {
			return b.defect(
				entryNode,
				entries[entryIndex].edge,
				"execution entry has no canonical occurrence",
			)
		}
		nestedKind, nestedDefinition, err := definitionKind(
			entryNode, executableContext,
		)
		if err != nil {
			return err
		}
		b.path = append(b.path, pathStep{
			node: entryNode, occurrence: entryOccurrence,
		})
		entryPathStart := len(b.path) - 1
		if nestedDefinition {
			err = b.addDefinition(
				entryNode,
				entryOccurrence,
				nestedKind,
				definitionID,
				entryPathStart,
				executableContext,
			)
		} else {
			err = b.scanNestedDefinitions(
				entryNode,
				definitionID,
				entryPathStart,
				executableContext,
			)
		}
		b.path = b.path[:len(b.path)-1]
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *fileBuilder) ensurePath(path []pathStep) error {
	firstNew := 0
	for index := len(path) - 1; index >= 0; index-- {
		b.work.IdentityProbes++
		if b.anchors[path[index].occurrence.id] {
			firstNew = index + 1
			break
		}
	}
	for _, step := range path[firstNew:] {
		if err := b.recordOccurrence(
			step.occurrence, step.node,
		); err != nil {
			return err
		}
		b.anchors[step.occurrence.id] = true
		b.anchorOrder = append(
			b.anchorOrder, step.occurrence.id,
		)
		b.work.RecordAppends++
	}
	return nil
}

func (b *fileBuilder) scanNestedDefinitions(
	node ast.Node,
	parent identity.DefinitionID,
	pathStart int,
	context catalog.DefinitionContext,
) error {
	current := b.path[len(b.path)-1].occurrence
	if err := b.index.bindStructuralSupport(
		current, node, parent,
	); err != nil {
		return err
	}
	kind, err := Classify(node)
	if err != nil {
		return err
	}
	nested, err := children(node, kind)
	if err != nil {
		return b.defect(node, catalog.EdgeInvalid, err.Error())
	}
	childContext, err := contextForChildren(node, kind, context)
	if err != nil {
		return err
	}
	for _, child := range nested {
		b.work.CatalogEdges++
		occurrence, err := b.makeOccurrence(
			child.node,
			current.id,
			child.edge,
			child.ordinal,
		)
		if err != nil {
			return err
		}
		b.path = append(
			b.path, pathStep{node: child.node, occurrence: occurrence},
		)
		nestedKind, isDefinition, err := definitionKind(
			child.node, childContext,
		)
		if err != nil {
			b.path = b.path[:len(b.path)-1]
			return err
		}
		if isDefinition {
			err = b.addDefinition(
				child.node,
				occurrence,
				nestedKind,
				parent,
				pathStart,
				childContext,
			)
		} else {
			err = b.scanNestedDefinitions(
				child.node, parent, pathStart, childContext,
			)
		}
		b.path = b.path[:len(b.path)-1]
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *fileBuilder) recordOccurrence(
	occurrence Occurrence,
	node ast.Node,
) error {
	b.work.IdentityProbes++
	if existing, present := b.occurrences[occurrence.id]; present {
		if existing != occurrence {
			return fmt.Errorf(
				"occurrence %s has conflicting canonical payloads",
				occurrence.id,
			)
		}
		return nil
	}
	if err := b.index.bindStructuralOccurrence(
		occurrence, node,
	); err != nil {
		return err
	}
	b.occurrences[occurrence.id] = occurrence
	b.order = append(b.order, occurrence.id)
	b.work.RecordAppends++
	return nil
}

func (b *fileBuilder) makeOccurrence(
	node ast.Node,
	parent identity.OccurrenceID,
	edge catalog.Edge,
	ordinal int,
) (Occurrence, error) {
	kind, err := Classify(node)
	if err != nil {
		return Occurrence{}, err
	}
	if kind.Disposition() != catalog.DispositionActive {
		return Occurrence{}, b.defect(
			node,
			edge,
			"construct disposition is "+kind.Disposition().String(),
		)
	}
	span := b.physicalSpan(node)
	spanID, err := identity.NewSpanID(
		b.file.ID(), span.Start.Offset, span.End.Offset,
	)
	if err != nil {
		return Occurrence{}, err
	}
	id, err := identity.NewOccurrenceID(spanID, uint16(kind))
	if err != nil {
		return Occurrence{}, err
	}
	lexical, err := tokenEvidence(node)
	if err != nil {
		return Occurrence{}, b.defect(node, edge, err.Error())
	}
	occurrence, err := NewOccurrence(
		id,
		kind,
		parent,
		edge,
		ordinal,
		span,
		b.displaySpan(node),
		lexical,
	)
	if err != nil {
		return Occurrence{}, err
	}
	return occurrence, nil
}

func (b *fileBuilder) physicalSpan(node ast.Node) Span {
	start := b.fset.PositionFor(node.Pos(), false)
	end := b.fset.PositionFor(node.End(), false)
	return Span{
		Start: Position{
			Line: start.Line, Column: start.Column, Offset: start.Offset,
		},
		End: Position{
			Line: end.Line, Column: end.Column, Offset: end.Offset,
		},
	}
}

func (b *fileBuilder) displaySpan(node ast.Node) DisplaySpan {
	start := b.fset.Position(node.Pos())
	end := b.fset.Position(node.End())
	physicalStart := b.fset.PositionFor(node.Pos(), false)
	physicalEnd := b.fset.PositionFor(node.End(), false)
	return DisplaySpan{
		Start: displayPosition(start, physicalStart, b.displayFile),
		End:   displayPosition(end, physicalEnd, b.displayFile),
	}
}

func displayPosition(
	adjusted token.Position,
	physical token.Position,
	physicalFile string,
) DisplayPosition {
	filename := adjusted.Filename
	if filename == physical.Filename {
		filename = physicalFile
	}
	return DisplayPosition{
		Filename: filename,
		Line:     adjusted.Line,
		Column:   adjusted.Column,
	}
}

func (b *fileBuilder) defect(
	node ast.Node,
	edge catalog.Edge,
	reason string,
) error {
	kind, _ := Classify(node)
	return &Error{
		Phase:  "DEFECT",
		File:   b.file.ID(),
		Kind:   kind,
		Edge:   edge,
		Span:   b.physicalSpan(node),
		Reason: reason,
	}
}

func definitionName(node ast.Node) string {
	switch node := node.(type) {
	case *ast.FuncDecl:
		return node.Name.Name
	case *ast.FuncLit:
		return "func literal"
	case *ast.ValueSpec:
		names := ""
		for index, name := range node.Names {
			if index > 0 {
				names += ","
			}
			names += name.Name
		}
		return names
	default:
		return ""
	}
}
