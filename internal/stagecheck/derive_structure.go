package stagecheck

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/source"
)

type derivedOccurrence struct {
	id      identity.OccurrenceID
	kind    catalog.Kind
	parent  identity.OccurrenceID
	edge    catalog.Edge
	ordinal int
	span    structure.Span
	display structure.DisplaySpan
	token   catalog.TokenKind
}

type derivedChild struct {
	node    ast.Node
	edge    catalog.Edge
	ordinal int
}

type derivedPathStep struct {
	node       ast.Node
	occurrence derivedOccurrence
}

type derivedFile struct {
	ledger      *structuralLedger
	file        *source.LoadedFile
	fset        *token.FileSet
	raw         []byte
	owner       structure.OwnerRegionID
	occurrences map[identity.OccurrenceID]derivedOccurrence
	anchors     map[identity.OccurrenceID]bool
	definitions map[identity.DefinitionID]ast.Node
}

func deriveFile(file *source.LoadedFile) (*derivedFile, error) {
	raw := file.SelectedBytes()
	fset := token.NewFileSet()
	syntax, err := parser.ParseFile(
		fset,
		file.Path(),
		raw,
		parser.ParseComments|parser.SkipObjectResolution,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"%s: independent parse failed: %w", file.ID(), err,
		)
	}
	owner, err := structure.SourceFileOwner(file.ID())
	if err != nil {
		return nil, err
	}
	builder := &derivedFile{
		ledger:      newStructuralLedger(),
		file:        file,
		fset:        fset,
		raw:         raw,
		owner:       owner,
		occurrences: map[identity.OccurrenceID]derivedOccurrence{},
		anchors:     map[identity.OccurrenceID]bool{},
		definitions: map[identity.DefinitionID]ast.Node{},
	}
	addRecord(&builder.ledger.owners, owner)
	addRecord(&builder.ledger.containmentGraphs, owner)
	root, err := builder.occurrence(
		syntax, identity.OccurrenceID{}, catalog.EdgeInvalid, 0,
	)
	if err != nil {
		return nil, err
	}
	if err := builder.recordOccurrence(root); err != nil {
		return nil, err
	}
	addRecord(&builder.ledger.ownerMembers, ownerMemberLedgerRecord{
		owner: owner, member: root.id,
	})
	context, err := catalog.NewDefinitionContext(
		catalog.DefinitionScopePackage, catalog.TokenInvalid,
	)
	if err != nil {
		return nil, err
	}
	if err := builder.walkOwner(syntax, root, nil, context); err != nil {
		return nil, err
	}
	if err := builder.deriveDirectives(syntax); err != nil {
		return nil, err
	}
	return builder, nil
}

func (b *derivedFile) walkOwner(
	node ast.Node,
	current derivedOccurrence,
	path []derivedPathStep,
	context catalog.DefinitionContext,
) error {
	children, err := independentChildren(node, current.kind)
	if err != nil {
		return err
	}
	childContext, err := independentChildContext(
		node, current.kind, context,
	)
	if err != nil {
		return err
	}
	for _, child := range children {
		occurrence, err := b.occurrence(
			child.node, current.id, child.edge, child.ordinal,
		)
		if err != nil {
			return err
		}
		next := append(path, derivedPathStep{
			node: child.node, occurrence: occurrence,
		})
		definitionKind, isDefinition, err := independentDefinitionKind(
			child.node, childContext,
		)
		if err != nil {
			return err
		}
		if isDefinition {
			if err := b.addDefinition(
				child.node,
				occurrence,
				definitionKind,
				identity.DefinitionID{},
				next,
				childContext,
			); err != nil {
				return err
			}
			continue
		}
		if err := b.recordOccurrence(occurrence); err != nil {
			return err
		}
		addRecord(&b.ledger.ownerMembers, ownerMemberLedgerRecord{
			owner: b.owner, member: occurrence.id,
		})
		if err := b.walkOwner(
			child.node, occurrence, next, childContext,
		); err != nil {
			return err
		}
	}
	return nil
}

func (b *derivedFile) addDefinition(
	node ast.Node,
	root derivedOccurrence,
	kind identity.DefinitionKind,
	parent identity.DefinitionID,
	path []derivedPathStep,
	context catalog.DefinitionContext,
) error {
	definition, err := identity.NewSourceDefinitionID(root.id, kind)
	if err != nil {
		return err
	}
	if _, duplicate := b.definitions[definition]; duplicate {
		return fmt.Errorf(
			"independent derivation duplicates %s", definition,
		)
	}
	if err := b.recordOccurrence(root); err != nil {
		return err
	}
	b.definitions[definition] = node
	header, _ := identity.NewHeaderRegionID(definition)
	boundary, _ := identity.NewExecutionBoundaryID(definition)
	if len(path) == 0 || path[len(path)-1].occurrence.id != root.id {
		return fmt.Errorf(
			"independent definition %s has no exact path", definition,
		)
	}
	if err := b.ensurePath(path[:len(path)-1]); err != nil {
		return err
	}
	addRecord(&b.ledger.definitions, definitionLedgerRecord{
		id:       definition,
		owner:    b.owner,
		header:   header,
		boundary: boundary,
		name:     independentDefinitionName(node),
	})
	addRecord(&b.ledger.definitionSites, definitionSiteLedgerRecord{
		kind:       structure.DefinitionSiteSource,
		definition: definition,
		owner:      b.owner,
		parent:     parent,
		terminal:   root.id,
	})
	entries, err := independentDefinitionEntries(node)
	if err != nil {
		return err
	}
	if err := b.addHeader(header, node, root); err != nil {
		return err
	}
	if err := b.addBoundary(
		definition, boundary, kind, root, entries,
	); err != nil {
		return err
	}
	executableContext, _ := catalog.NewDefinitionContext(
		catalog.DefinitionScopeExecutable, catalog.TokenInvalid,
	)
	for _, entry := range entries {
		entryOccurrence, err := b.occurrence(
			entry.node, root.id, entry.edge, entry.ordinal,
		)
		if err != nil {
			return err
		}
		stored, present := b.occurrences[entryOccurrence.id]
		if !present {
			return fmt.Errorf(
				"independent boundary entry is absent from occurrence store",
			)
		}
		if stored != entryOccurrence {
			return fmt.Errorf(
				"independent boundary entry has conflicting occurrence payload",
			)
		}
		entryOccurrence = stored
		nestedKind, nested, err := independentDefinitionKind(
			entry.node, executableContext,
		)
		if err != nil {
			return err
		}
		if nested {
			if err := b.addDefinition(
				entry.node,
				entryOccurrence,
				nestedKind,
				definition,
				[]derivedPathStep{{
					node: entry.node, occurrence: entryOccurrence,
				}},
				executableContext,
			); err != nil {
				return err
			}
			continue
		}
		if err := b.scanNested(
			entry.node,
			definition,
			[]derivedPathStep{{
				node: entry.node, occurrence: entryOccurrence,
			}},
			executableContext,
		); err != nil {
			return err
		}
	}
	return nil
}

func (b *derivedFile) scanNested(
	node ast.Node,
	parent identity.DefinitionID,
	path []derivedPathStep,
	context catalog.DefinitionContext,
) error {
	kind, err := independentKind(node)
	if err != nil {
		return err
	}
	children, err := independentChildren(node, kind)
	if err != nil {
		return err
	}
	childContext, err := independentChildContext(node, kind, context)
	if err != nil {
		return err
	}
	parentOccurrence := path[len(path)-1].occurrence
	for _, child := range children {
		occurrence, err := b.occurrence(
			child.node,
			parentOccurrence.id,
			child.edge,
			child.ordinal,
		)
		if err != nil {
			return err
		}
		next := append(path, derivedPathStep{
			node: child.node, occurrence: occurrence,
		})
		definitionKind, nested, err := independentDefinitionKind(
			child.node, childContext,
		)
		if err != nil {
			return err
		}
		if nested {
			if err := b.addDefinition(
				child.node,
				occurrence,
				definitionKind,
				parent,
				next,
				childContext,
			); err != nil {
				return err
			}
			continue
		}
		if err := b.scanNested(
			child.node, parent, next, childContext,
		); err != nil {
			return err
		}
	}
	return nil
}

func (b *derivedFile) ensurePath(path []derivedPathStep) error {
	for _, step := range path {
		if err := b.recordOccurrence(step.occurrence); err != nil {
			return err
		}
		if b.anchors[step.occurrence.id] {
			continue
		}
		b.anchors[step.occurrence.id] = true
		addRecord(
			&b.ledger.containmentAnchors,
			containmentAnchorLedgerRecord{
				owner: b.owner, anchor: step.occurrence.id,
			},
		)
	}
	return nil
}

func (b *derivedFile) addHeader(
	header identity.HeaderRegionID,
	node ast.Node,
	root derivedOccurrence,
) error {
	members := []derivedOccurrence{root}
	if err := b.walkHeader(node, root, &members); err != nil {
		return err
	}
	digest, err := independentHeaderDigest(
		b.raw, b.span, node, members,
	)
	if err != nil {
		return err
	}
	addRecord(&b.ledger.headers, headerLedgerRecord{
		id: header, digest: digest,
	})
	for index, occurrence := range members {
		addRecord(&b.ledger.headerMembers, headerMemberLedgerRecord{
			header: header, ordinal: index, occurrence: occurrence.id,
		})
	}
	return nil
}

func (b *derivedFile) walkHeader(
	node ast.Node,
	parent derivedOccurrence,
	out *[]derivedOccurrence,
) error {
	children, err := independentChildren(node, parent.kind)
	if err != nil {
		return err
	}
	for _, child := range children {
		if child.edge.DefinitionEntry() {
			continue
		}
		occurrence, err := b.occurrence(
			child.node, parent.id, child.edge, child.ordinal,
		)
		if err != nil {
			return err
		}
		if err := b.recordOccurrence(occurrence); err != nil {
			return err
		}
		*out = append(*out, occurrence)
		if err := b.walkHeader(child.node, occurrence, out); err != nil {
			return err
		}
	}
	return nil
}

func (b *derivedFile) addBoundary(
	definition identity.DefinitionID,
	boundary identity.ExecutionBoundaryID,
	kind identity.DefinitionKind,
	root derivedOccurrence,
	entries []derivedChild,
) error {
	boundaryKind := structure.BoundaryInvalid
	switch kind {
	case identity.DefinitionFuncDecl, identity.DefinitionFuncLit:
		boundaryKind = structure.BoundaryBlock
	case identity.DefinitionPackageInitializer:
		boundaryKind = structure.BoundaryInitializers
	case identity.DefinitionBodylessDecl:
		boundaryKind = structure.BoundaryBodyless
	}
	combined := ""
	if boundaryKind == structure.BoundaryBodyless {
		combined = independentDigest(
			definition.String(), "bodyless-obligation",
		)
	}
	var hashes []string
	for _, entry := range entries {
		occurrence, err := b.occurrence(
			entry.node, root.id, entry.edge, entry.ordinal,
		)
		if err != nil {
			return err
		}
		if err := b.recordOccurrence(occurrence); err != nil {
			return err
		}
		hash, err := independentDigestSpan(b.raw, occurrence.span)
		if err != nil {
			return err
		}
		hashes = append(hashes, hash)
		addRecord(&b.ledger.executionEntries, executionEntryLedgerRecord{
			boundary: boundary, occurrence: occurrence.id, hash: hash,
		})
	}
	if len(hashes) > 0 {
		combined = independentDigest(hashes...)
	}
	addRecord(
		&b.ledger.executionBoundaries,
		executionBoundaryLedgerRecord{
			id: boundary, kind: boundaryKind, digest: combined,
		},
	)
	return nil
}

func (b *derivedFile) recordOccurrence(
	occurrence derivedOccurrence,
) error {
	if existing, present := b.occurrences[occurrence.id]; present {
		if existing != occurrence {
			return fmt.Errorf(
				"independent occurrence %s has conflicting payloads",
				occurrence.id,
			)
		}
		return nil
	}
	b.occurrences[occurrence.id] = occurrence
	addRecord(
		&b.ledger.occurrences,
		occurrenceLedgerRecordFromDerived(occurrence),
	)
	return nil
}

func (b *derivedFile) occurrence(
	node ast.Node,
	parent identity.OccurrenceID,
	edge catalog.Edge,
	ordinal int,
) (derivedOccurrence, error) {
	kind, err := independentKind(node)
	if err != nil {
		return derivedOccurrence{}, err
	}
	if kind.Disposition() != catalog.DispositionActive {
		return derivedOccurrence{}, fmt.Errorf(
			"%s: independent derivation encountered %s disposition %s",
			b.file.ID(), kind, kind.Disposition(),
		)
	}
	span := b.span(node)
	spanID, err := identity.NewSpanID(
		b.file.ID(), span.Start.Offset, span.End.Offset,
	)
	if err != nil {
		return derivedOccurrence{}, err
	}
	id, err := identity.NewOccurrenceID(spanID, uint16(kind))
	if err != nil {
		return derivedOccurrence{}, err
	}
	tokenKind, err := independentToken(node)
	if err != nil {
		return derivedOccurrence{}, err
	}
	return derivedOccurrence{
		id:      id,
		kind:    kind,
		parent:  parent,
		edge:    edge,
		ordinal: ordinal,
		span:    span,
		display: b.display(node),
		token:   tokenKind,
	}, nil
}

func (b *derivedFile) span(node ast.Node) structure.Span {
	start := b.fset.PositionFor(node.Pos(), false)
	end := b.fset.PositionFor(node.End(), false)
	return structure.Span{
		Start: structure.Position{
			Line: start.Line, Column: start.Column, Offset: start.Offset,
		},
		End: structure.Position{
			Line: end.Line, Column: end.Column, Offset: end.Offset,
		},
	}
}

func (b *derivedFile) display(node ast.Node) structure.DisplaySpan {
	start := b.fset.Position(node.Pos())
	end := b.fset.Position(node.End())
	physicalStart := b.fset.PositionFor(node.Pos(), false)
	physicalEnd := b.fset.PositionFor(node.End(), false)
	return structure.DisplaySpan{
		Start: independentDisplayPosition(start, physicalStart, b.file.ID()),
		End:   independentDisplayPosition(end, physicalEnd, b.file.ID()),
	}
}

func independentDisplayPosition(
	adjusted token.Position,
	physical token.Position,
	file identity.FileID,
) structure.DisplayPosition {
	filename := adjusted.Filename
	if filename == physical.Filename {
		filename = file.String()
	}
	return structure.DisplayPosition{
		Filename: filename,
		Line:     adjusted.Line,
		Column:   adjusted.Column,
	}
}
