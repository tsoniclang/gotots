package structure

import (
	"fmt"
	"strings"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

func encodeFileGraph(file FileGraph, byteDigest string) artifactFile {
	record := artifactFile{
		File:        file.owner.id.file.String(),
		ByteDigest:  byteDigest,
		Occurrences: encodeOccurrences(file.occurrences),
		Owner: artifactOwner{
			Members:    occurrenceStrings(file.owner.members),
			Directives: encodeDirectives(file.owner.directives),
		},
		Anchors: occurrenceStrings(file.containment.anchors),
	}
	for _, definition := range file.definitions {
		record.Definitions = append(
			record.Definitions,
			artifactDefinition{
				ID:       definition.id.String(),
				Owner:    definition.owner.String(),
				Header:   definition.header.String(),
				Boundary: definition.boundary.String(),
				Name:     definition.name,
			},
		)
	}
	for _, site := range file.sites {
		record.Sites = append(record.Sites, artifactSite{
			Kind:             uint8(site.kind),
			Definition:       site.definition.String(),
			Owner:            site.owner.String(),
			ParentDefinition: site.parentDefinition.String(),
			Terminal:         site.terminal.String(),
		})
	}
	for _, header := range file.headers {
		record.Headers = append(record.Headers, artifactHeader{
			ID:      header.id.String(),
			Digest:  header.digest,
			Members: occurrenceStrings(header.members),
		})
	}
	for _, boundary := range file.boundaries {
		encoded := artifactBoundary{
			ID:             boundary.id.String(),
			Kind:           uint8(boundary.kind),
			CombinedDigest: boundary.combinedDigest,
			Implicit:       uint8(boundary.implicit),
			Synthetic:      uint8(boundary.synthetic),
		}
		for _, entry := range boundary.entries {
			encoded.Entries = append(encoded.Entries, artifactEntry{
				ID: entry.id.String(), Hash: entry.hash,
			})
		}
		record.Boundaries = append(record.Boundaries, encoded)
	}
	for _, mapping := range file.mappings {
		record.Mappings = append(record.Mappings, artifactCheckedMap{
			Definition:    mapping.definition.String(),
			OriginLine:    mapping.originLine,
			OriginColumn:  mapping.originColumn,
			OriginMatch:   uint8(mapping.originMatch),
			CheckedDigest: mapping.checkedDigest,
		})
	}
	return record
}

func occurrenceStrings(ids []identity.OccurrenceID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}

func encodeOccurrences(
	occurrences *OccurrenceStore,
) []artifactOccurrence {
	out := make([]artifactOccurrence, 0, occurrences.Count())
	if occurrences == nil {
		return out
	}
	if err := occurrences.Visit(func(reference OccurrenceRef) error {
		occurrence := reference.Occurrence()
		out = append(out, artifactOccurrence{
			ID:      occurrence.id.String(),
			Kind:    uint16(occurrence.kind),
			Parent:  occurrence.parent.String(),
			Edge:    uint16(occurrence.edge),
			Ordinal: occurrence.ordinal,
			Span:    occurrence.span,
			Display: occurrence.display,
			Token:   uint16(occurrence.token),
		})
		return nil
	}); err != nil {
		panic(err)
	}
	return out
}

func encodeDirectives(
	directives []Directive,
) []artifactDirective {
	out := make([]artifactDirective, 0, len(directives))
	for _, directive := range directives {
		out = append(out, artifactDirective{
			Kind:    uint16(directive.kind),
			Tool:    directive.tool,
			Name:    directive.name,
			Args:    directive.args,
			Span:    directive.span,
			Display: directive.display,
		})
	}
	return out
}

func decodeFileGraph(record artifactFile) (FileGraph, error) {
	file, err := identity.ParseFileID(record.File)
	if err != nil {
		return FileGraph{}, err
	}
	ownerID, err := SourceFileOwner(file)
	if err != nil {
		return FileGraph{}, err
	}
	occurrences, err := decodeOccurrences(record.Occurrences, file)
	if err != nil {
		return FileGraph{}, err
	}
	members, err := parseOccurrenceStrings(record.Owner.Members, file)
	if err != nil {
		return FileGraph{}, err
	}
	anchors, err := parseOccurrenceStrings(record.Anchors, file)
	if err != nil {
		return FileGraph{}, err
	}
	directives, err := decodeDirectives(record.Owner.Directives)
	if err != nil {
		return FileGraph{}, err
	}
	graph := FileGraph{
		owner: OwnerRegion{
			id: ownerID, members: members, directives: directives,
		},
		occurrences: occurrences,
		containment: ContainmentGraph{
			owner: ownerID, anchors: anchors,
		},
	}
	if err := decodeDefinitions(record, &graph); err != nil {
		return FileGraph{}, err
	}
	if err := validateDecodedFile(graph); err != nil {
		return FileGraph{}, err
	}
	return graph, nil
}

func decodeDefinitions(
	record artifactFile,
	graph *FileGraph,
) error {
	for _, encoded := range record.Definitions {
		definition, err := identity.ParseDefinitionID(encoded.ID)
		if err != nil {
			return err
		}
		owner, err := parseOwnerRegion(encoded.Owner)
		if err != nil {
			return err
		}
		header, err := identity.NewHeaderRegionID(definition)
		if err != nil {
			return err
		}
		boundary, err := identity.NewExecutionBoundaryID(definition)
		if err != nil {
			return err
		}
		if header.String() != encoded.Header ||
			boundary.String() != encoded.Boundary {
			return fmt.Errorf("definition region identities are noncanonical")
		}
		graph.definitions = append(
			graph.definitions,
			ImplementationDefinition{
				id:       definition,
				owner:    owner,
				header:   header,
				boundary: boundary,
				name:     encoded.Name,
			},
		)
	}
	for _, encoded := range record.Sites {
		site, err := decodeSite(encoded)
		if err != nil {
			return err
		}
		graph.sites = append(graph.sites, site)
	}
	for _, encoded := range record.Headers {
		header, err := decodeHeader(encoded)
		if err != nil {
			return err
		}
		graph.headers = append(graph.headers, header)
	}
	for _, encoded := range record.Boundaries {
		boundary, err := decodeBoundary(encoded)
		if err != nil {
			return err
		}
		graph.boundaries = append(graph.boundaries, boundary)
	}
	for _, encoded := range record.Mappings {
		definition, err := identity.ParseDefinitionID(encoded.Definition)
		if err != nil {
			return err
		}
		if definition.File() != graph.owner.id.file {
			return fmt.Errorf("checked mapping belongs to another file")
		}
		graph.mappings = append(
			graph.mappings,
			CheckedDefinitionMapping{
				definition:    definition,
				originLine:    encoded.OriginLine,
				originColumn:  encoded.OriginColumn,
				originMatch:   CheckedOriginMatch(encoded.OriginMatch),
				checkedDigest: encoded.CheckedDigest,
			},
		)
	}
	return nil
}

func decodeSite(encoded artifactSite) (DefinitionSite, error) {
	definition, err := identity.ParseDefinitionID(encoded.Definition)
	if err != nil {
		return DefinitionSite{}, err
	}
	owner, err := parseOwnerRegion(encoded.Owner)
	if err != nil {
		return DefinitionSite{}, err
	}
	var parent identity.DefinitionID
	if encoded.ParentDefinition != "" {
		parent, err = identity.ParseDefinitionID(encoded.ParentDefinition)
		if err != nil {
			return DefinitionSite{}, err
		}
	}
	var terminal identity.OccurrenceID
	if encoded.Terminal != "" {
		terminal, err = identity.ParseOccurrenceID(encoded.Terminal)
		if err != nil {
			return DefinitionSite{}, err
		}
	}
	kind := DefinitionSiteKind(encoded.Kind)
	if kind != DefinitionSiteSource &&
		kind != DefinitionSiteSynthetic {
		return DefinitionSite{}, fmt.Errorf("definition site kind is invalid")
	}
	return DefinitionSite{
		kind:             kind,
		definition:       definition,
		owner:            owner,
		parentDefinition: parent,
		terminal:         terminal,
	}, nil
}

func decodeHeader(encoded artifactHeader) (HeaderRegion, error) {
	definitionText := strings.TrimSuffix(encoded.ID, "#header")
	definition, err := identity.ParseDefinitionID(definitionText)
	if err != nil {
		return HeaderRegion{}, err
	}
	id, err := identity.NewHeaderRegionID(definition)
	if err != nil {
		return HeaderRegion{}, err
	}
	if id.String() != encoded.ID || encoded.Digest == "" {
		return HeaderRegion{}, fmt.Errorf("header identity or digest is invalid")
	}
	members, err := parseOccurrenceStrings(
		encoded.Members, definition.File(),
	)
	if err != nil {
		return HeaderRegion{}, err
	}
	return HeaderRegion{
		id: id, digest: encoded.Digest, members: members,
	}, nil
}

func decodeBoundary(
	encoded artifactBoundary,
) (ExecutionBoundary, error) {
	definitionText := strings.TrimSuffix(encoded.ID, "#execution")
	definition, err := identity.ParseDefinitionID(definitionText)
	if err != nil {
		return ExecutionBoundary{}, err
	}
	id, err := identity.NewExecutionBoundaryID(definition)
	if err != nil {
		return ExecutionBoundary{}, err
	}
	kind := ExecutionBoundaryKind(encoded.Kind)
	if id.String() != encoded.ID ||
		!kind.Valid() ||
		encoded.CombinedDigest == "" {
		return ExecutionBoundary{}, fmt.Errorf(
			"execution boundary identity, kind, or digest is invalid",
		)
	}
	boundary := ExecutionBoundary{
		id:             id,
		kind:           kind,
		combinedDigest: encoded.CombinedDigest,
		implicit:       identity.ImplicitDefinitionOp(encoded.Implicit),
		synthetic:      identity.SyntheticDefinitionRole(encoded.Synthetic),
	}
	for _, entry := range encoded.Entries {
		occurrence, err := identity.ParseOccurrenceID(entry.ID)
		if err != nil {
			return ExecutionBoundary{}, err
		}
		if entry.Hash == "" {
			return ExecutionBoundary{}, fmt.Errorf(
				"execution entry has no content hash",
			)
		}
		boundary.entries = append(
			boundary.entries,
			ExecutionEntry{id: occurrence, hash: entry.Hash},
		)
	}
	return boundary, nil
}

func decodeOccurrences(
	encoded []artifactOccurrence,
	file identity.FileID,
) (*OccurrenceStore, error) {
	builder, err := NewOccurrenceStoreBuilder(file, len(encoded))
	if err != nil {
		return nil, err
	}
	for _, record := range encoded {
		id, err := identity.ParseOccurrenceID(record.ID)
		if err != nil {
			return nil, err
		}
		if id.Span().File() != file {
			return nil, fmt.Errorf("occurrence belongs to another file")
		}
		var parent identity.OccurrenceID
		if record.Parent != "" {
			parent, err = identity.ParseOccurrenceID(record.Parent)
			if err != nil {
				return nil, err
			}
		}
		kind := catalog.Kind(record.Kind)
		edge := catalog.Edge(record.Edge)
		token := catalog.TokenKind(record.Token)
		if !kind.Valid() ||
			(record.Edge != 0 && !edge.Valid()) ||
			(record.Token != 0 && !token.Valid()) {
			return nil, fmt.Errorf(
				"occurrence has invalid kind, edge, or token",
			)
		}
		occurrence, err := NewOccurrence(
			id,
			kind,
			parent,
			edge,
			record.Ordinal,
			record.Span,
			record.Display,
			token,
		)
		if err != nil {
			return nil, err
		}
		if _, err := builder.Append(occurrence); err != nil {
			return nil, err
		}
	}
	return builder.Seal()
}

func parseOccurrenceStrings(
	encoded []string,
	file identity.FileID,
) ([]identity.OccurrenceID, error) {
	out := make([]identity.OccurrenceID, 0, len(encoded))
	for _, text := range encoded {
		id, err := identity.ParseOccurrenceID(text)
		if err != nil {
			return nil, err
		}
		if !file.IsZero() && id.Span().File() != file {
			return nil, fmt.Errorf(
				"occurrence membership belongs to another file",
			)
		}
		out = append(out, id)
	}
	return out, nil
}

func decodeDirectives(
	encoded []artifactDirective,
) ([]Directive, error) {
	out := make([]Directive, 0, len(encoded))
	for _, record := range encoded {
		kind := catalog.DirectiveKind(record.Kind)
		directive, err := NewDirective(
			kind,
			record.Tool,
			record.Name,
			record.Args,
			record.Span,
			record.Display,
		)
		if err != nil {
			return nil, err
		}
		out = append(out, directive)
	}
	return out, nil
}
