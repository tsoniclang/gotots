package structure

import (
	"fmt"
	"strings"

	"github.com/tsoniclang/gotots/internal/identity"
)

func parseOwnerRegion(text string) (OwnerRegionID, error) {
	if strings.HasPrefix(text, "source:") {
		file, err := identity.ParseFileID(
			strings.TrimPrefix(text, "source:"),
		)
		if err != nil {
			return OwnerRegionID{}, err
		}
		return SourceFileOwner(file)
	}
	if strings.HasPrefix(text, "synthetic:") {
		payload := strings.TrimPrefix(text, "synthetic:")
		slash := strings.LastIndex(payload, "/")
		if slash < 0 {
			return OwnerRegionID{}, fmt.Errorf(
				"synthetic owner lacks a closed kind",
			)
		}
		pkg, err := identity.ParsePackageID(payload[:slash])
		if err != nil {
			return OwnerRegionID{}, err
		}
		var kind SyntheticOwnerKind
		for candidate := SyntheticOwnerKind(1); candidate.Valid(); candidate++ {
			if candidate.String() == payload[slash+1:] {
				kind = candidate
				break
			}
		}
		return SyntheticOwner(pkg, kind)
	}
	return OwnerRegionID{}, fmt.Errorf("invalid owner region %q", text)
}

func validateDecodedFile(graph FileGraph) error {
	if graph.owner.id.IsZero() ||
		graph.containment.owner != graph.owner.id {
		return fmt.Errorf("decoded file graph has invalid owner")
	}
	occurrences := map[identity.OccurrenceID]OccurrenceRef{}
	if err := graph.VisitOccurrenceRefs(func(
		occurrence OccurrenceRef,
	) error {
		id := occurrence.ID()
		if _, duplicate := occurrences[id]; duplicate {
			return fmt.Errorf(
				"decoded file duplicates occurrence %s", id,
			)
		}
		occurrences[id] = occurrence
		return nil
	}); err != nil {
		return err
	}
	if err := graph.VisitOccurrenceRefs(func(
		occurrence OccurrenceRef,
	) error {
		return validateOccurrence(occurrence, occurrences)
	}); err != nil {
		return err
	}
	definitions := map[identity.DefinitionID]int{}
	sites := map[identity.DefinitionID]int{}
	headers := map[identity.DefinitionID]int{}
	boundaries := map[identity.DefinitionID]int{}
	for _, definition := range graph.definitions {
		definitions[definition.id]++
	}
	for _, site := range graph.sites {
		sites[site.definition]++
	}
	for _, header := range graph.headers {
		headers[header.id.Definition()]++
	}
	for _, boundary := range graph.boundaries {
		boundaries[boundary.id.Definition()]++
	}
	for definition, count := range definitions {
		if count != 1 ||
			sites[definition] != 1 ||
			headers[definition] != 1 ||
			boundaries[definition] != 1 {
			return fmt.Errorf(
				"decoded definition %s has invalid cardinalities", definition,
			)
		}
	}
	if len(definitions) != len(sites) ||
		len(definitions) != len(headers) ||
		len(definitions) != len(boundaries) {
		return fmt.Errorf("decoded file has orphan definition records")
	}
	return nil
}

func decodeSyntheticPackage(
	pkg identity.PackageID,
	record artifactPackage,
) (PackageGraph, error) {
	graph := PackageGraph{id: pkg}
	owners := map[OwnerRegionID]bool{}
	for _, encoded := range record.Owners {
		owner, err := parseOwnerRegion(encoded)
		if err != nil {
			return PackageGraph{}, err
		}
		if owner.Kind() != OwnerRegionSynthetic ||
			owner.Package() != pkg ||
			owners[owner] {
			return PackageGraph{}, fmt.Errorf(
				"invalid or duplicate synthetic owner %s", encoded,
			)
		}
		owners[owner] = true
		graph.synthetic = append(
			graph.synthetic, OwnerRegion{id: owner},
		)
	}
	for _, encoded := range record.Definitions {
		definition, err := identity.ParseDefinitionID(encoded.ID)
		if err != nil {
			return PackageGraph{}, err
		}
		if definition.Package() != pkg ||
			!definition.SyntheticRole().Valid() {
			return PackageGraph{}, fmt.Errorf(
				"provider synthetic definition has invalid ownership",
			)
		}
		owner, err := parseOwnerRegion(encoded.Owner)
		if err != nil || !owners[owner] {
			return PackageGraph{}, fmt.Errorf(
				"provider synthetic definition has unknown owner",
			)
		}
		header, _ := identity.NewHeaderRegionID(definition)
		boundary, _ := identity.NewExecutionBoundaryID(definition)
		if header.String() != encoded.Header ||
			boundary.String() != encoded.Boundary {
			return PackageGraph{}, fmt.Errorf(
				"provider synthetic definition has noncanonical regions",
			)
		}
		graph.ownedDefinitions = append(
			graph.ownedDefinitions,
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
		if err != nil ||
			site.kind != DefinitionSiteSynthetic ||
			!site.terminal.IsZero() ||
			!site.parentDefinition.IsZero() ||
			!owners[site.owner] {
			return PackageGraph{}, fmt.Errorf(
				"invalid synthetic definition site",
			)
		}
		graph.ownedSites = append(graph.ownedSites, site)
	}
	for _, encoded := range record.Headers {
		header, err := decodeHeader(encoded)
		if err != nil || len(header.members) != 0 {
			return PackageGraph{}, fmt.Errorf("invalid synthetic header")
		}
		graph.ownedHeaders = append(graph.ownedHeaders, header)
	}
	for _, encoded := range record.Boundaries {
		boundary, err := decodeBoundary(encoded)
		if err != nil ||
			boundary.kind != BoundaryImplicit ||
			!boundary.synthetic.Valid() ||
			len(boundary.entries) != 0 {
			return PackageGraph{}, fmt.Errorf(
				"invalid synthetic execution boundary",
			)
		}
		graph.ownedBoundaries = append(graph.ownedBoundaries, boundary)
	}
	if err := validatePackageGraph(graph); err != nil {
		return PackageGraph{}, err
	}
	return graph, nil
}
