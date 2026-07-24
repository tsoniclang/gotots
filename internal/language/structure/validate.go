package structure

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

func validateCompleteGraph(graph *Graph) error {
	if graph == nil || graph.version != ArtifactVersion {
		return fmt.Errorf("invalid structural artifact version")
	}
	if graph.byOccurrence == nil ||
		graph.byDefinition == nil ||
		graph.byBoundary == nil {
		return fmt.Errorf("structural artifact is not sealed")
	}
	packageIDs := map[identity.PackageID]bool{}
	ownerIDs := map[OwnerRegionID]int{}
	definitions := map[identity.DefinitionID]int{}
	definitionRecords := map[identity.DefinitionID]ImplementationDefinition{}
	sites := map[identity.DefinitionID]DefinitionSite{}
	headers := map[identity.DefinitionID]HeaderRegion{}
	boundaries := map[identity.DefinitionID]ExecutionBoundary{}
	declaredAnchors := map[identity.OccurrenceID]bool{}
	for _, pkg := range graph.packages {
		if pkg.id.IsZero() || packageIDs[pkg.id] {
			return fmt.Errorf("structural graph has zero or duplicate package %s", pkg.id)
		}
		packageIDs[pkg.id] = true
		if err := validatePackageStorage(pkg); err != nil {
			return err
		}
		for _, file := range pkg.files {
			if err := validateFileGraph(file, graph.byOccurrence); err != nil {
				return fmt.Errorf("%s: %w", pkg.id, err)
			}
			ownerIDs[file.owner.id]++
			for _, anchor := range file.containment.anchors {
				if declaredAnchors[anchor] {
					return fmt.Errorf(
						"containment anchor %s is stored in multiple file graphs",
						anchor,
					)
				}
				declaredAnchors[anchor] = true
			}
		}
		for _, owner := range pkg.synthetic {
			if owner.id.kind != OwnerRegionSynthetic ||
				owner.id.pkg != pkg.id ||
				!owner.id.synthetic.Valid() ||
				len(owner.members) != 0 {
				return fmt.Errorf(
					"package %s has invalid synthetic owner %s", pkg.id, owner.id,
				)
			}
			ownerIDs[owner.id]++
		}
		for _, definition := range pkg.Definitions() {
			if definition.id.IsZero() || definition.owner.IsZero() {
				return fmt.Errorf("package %s has an invalid definition", pkg.id)
			}
			if definition.name == "" {
				return fmt.Errorf(
					"definition %s has no diagnostic name", definition.id,
				)
			}
			definitions[definition.id]++
			definitionRecords[definition.id] = definition
			if definition.header.Definition() != definition.id ||
				definition.boundary.Definition() != definition.id {
				return fmt.Errorf(
					"definition %s has noncanonical region identities",
					definition.id,
				)
			}
		}
		for _, site := range pkg.Sites() {
			if _, duplicate := sites[site.definition]; duplicate {
				return fmt.Errorf("definition %s has multiple sites", site.definition)
			}
			sites[site.definition] = site
		}
		for _, header := range pkg.Headers() {
			definition := header.id.Definition()
			if _, duplicate := headers[definition]; duplicate {
				return fmt.Errorf("definition %s has multiple headers", definition)
			}
			headers[definition] = header
		}
		for _, boundary := range pkg.Boundaries() {
			definition := boundary.id.Definition()
			if _, duplicate := boundaries[definition]; duplicate {
				return fmt.Errorf(
					"definition %s has multiple execution boundaries", definition,
				)
			}
			boundaries[definition] = boundary
		}
	}
	if err := validateSealedIndexes(graph); err != nil {
		return err
	}
	for owner, count := range ownerIDs {
		if count != 1 {
			return fmt.Errorf("owner region %s occurs %d times", owner, count)
		}
	}
	requiredAnchors := map[identity.OccurrenceID]bool{}
	for definition, count := range definitions {
		if count != 1 {
			return fmt.Errorf(
				"definition %s occurs %d times", definition, count,
			)
		}
		site, hasSite := sites[definition]
		header, hasHeader := headers[definition]
		boundary, hasBoundary := boundaries[definition]
		if !hasSite || !hasHeader || !hasBoundary {
			return fmt.Errorf(
				"definition %s site/header/boundary=%t/%t/%t",
				definition, hasSite, hasHeader, hasBoundary,
			)
		}
		if err := validateDefinition(
			graph,
			definition,
			site,
			header,
			boundary,
			ownerIDs,
			definitionRecords,
			requiredAnchors,
		); err != nil {
			return err
		}
	}
	if len(sites) != len(definitions) ||
		len(headers) != len(definitions) ||
		len(boundaries) != len(definitions) {
		return fmt.Errorf(
			"orphan structural records definitions/sites/headers/boundaries=%d/%d/%d/%d",
			len(definitions), len(sites), len(headers), len(boundaries),
		)
	}
	if err := validateDefinitionForest(sites, definitionRecords); err != nil {
		return err
	}
	for anchor := range declaredAnchors {
		if _, present := graph.byOccurrence[anchor]; !present {
			return fmt.Errorf(
				"containment anchor %s has no canonical occurrence", anchor,
			)
		}
		if !requiredAnchors[anchor] {
			return fmt.Errorf("containment anchor %s is unused", anchor)
		}
	}
	for anchor := range requiredAnchors {
		if !declaredAnchors[anchor] {
			return fmt.Errorf("containment path omits anchor %s", anchor)
		}
	}
	return nil
}

func validateFileGraph(
	file FileGraph,
	all map[identity.OccurrenceID]*Occurrence,
) error {
	if file.owner.id.kind != OwnerRegionSourceFile ||
		file.owner.id.file.IsZero() ||
		file.containment.owner != file.owner.id {
		return fmt.Errorf("file graph has invalid owner/containment identity")
	}
	seen := map[identity.OccurrenceID]bool{}
	for _, occurrence := range file.occurrences {
		if seen[occurrence.id] {
			return fmt.Errorf(
				"file stores occurrence %s more than once", occurrence.id,
			)
		}
		seen[occurrence.id] = true
		if occurrence.id.Span().File() != file.owner.id.file {
			return fmt.Errorf(
				"occurrence %s belongs to another file", occurrence.id,
			)
		}
		if err := validateOccurrence(occurrence, all); err != nil {
			return err
		}
	}
	memberSet := map[identity.OccurrenceID]bool{}
	type primaryOccurrenceOwner struct {
		source bool
		header identity.HeaderRegionID
	}
	primaryOwner := map[identity.OccurrenceID]primaryOccurrenceOwner{}
	referenced := map[identity.OccurrenceID]bool{}
	rootCount := 0
	for _, id := range file.owner.members {
		if memberSet[id] {
			return fmt.Errorf("owner repeats member %s", id)
		}
		memberSet[id] = true
		primaryOwner[id] = primaryOccurrenceOwner{source: true}
		referenced[id] = true
		occurrence, present := all[id]
		if !present {
			return fmt.Errorf("owner member %s has no canonical occurrence", id)
		}
		if occurrence.parent.IsZero() {
			if occurrence.kind != catalog.KindFile {
				return fmt.Errorf(
					"source owner root %s is %s, not File", id, occurrence.kind,
				)
			}
			rootCount++
		} else if !memberSet[occurrence.parent] {
			return fmt.Errorf(
				"owner member %s precedes or omits parent %s",
				id, occurrence.parent,
			)
		}
	}
	if rootCount != 1 {
		return fmt.Errorf("source owner has %d File roots", rootCount)
	}
	anchorSet := map[identity.OccurrenceID]bool{}
	for _, anchor := range file.containment.anchors {
		if anchorSet[anchor] {
			return fmt.Errorf("containment graph repeats anchor %s", anchor)
		}
		anchorSet[anchor] = true
		referenced[anchor] = true
		occurrence, present := all[anchor]
		if !present || occurrence.id.Span().File() != file.owner.id.file {
			return fmt.Errorf(
				"containment anchor %s lacks same-file canonical payload", anchor,
			)
		}
	}
	for _, header := range file.headers {
		for index, member := range header.members {
			if owner := primaryOwner[member]; owner.source ||
				!owner.header.IsZero() {
				ownerName := "source owner"
				if !owner.header.IsZero() {
					ownerName = owner.header.String()
				}
				return fmt.Errorf(
					"occurrence %s is owned by both %s and header %s",
					member, ownerName, header.id,
				)
			}
			primaryOwner[member] = primaryOccurrenceOwner{
				header: header.id,
			}
			referenced[member] = true
			if index > 0 {
				occurrence := all[member]
				if primaryOwner[occurrence.parent].header != header.id {
					return fmt.Errorf(
						"header %s omits or reorders parent %s of %s",
						header.id, occurrence.parent, member,
					)
				}
			}
		}
	}
	for _, boundary := range file.boundaries {
		for _, entry := range boundary.entries {
			referenced[entry.id] = true
		}
	}
	for id := range seen {
		if !referenced[id] {
			return fmt.Errorf(
				"canonical occurrence %s has no owning structural relation", id,
			)
		}
	}
	if err := validateCheckedMappings(file, all); err != nil {
		return err
	}
	return nil
}

func validateOccurrence(
	occurrence Occurrence,
	all map[identity.OccurrenceID]*Occurrence,
) error {
	canonical, err := NewOccurrence(
		occurrence.id,
		occurrence.kind,
		occurrence.parent,
		occurrence.edge,
		occurrence.ordinal,
		occurrence.span,
		occurrence.display,
		occurrence.token,
	)
	if err != nil || canonical != occurrence {
		return fmt.Errorf(
			"occurrence %s has noncanonical identity facts", occurrence.id,
		)
	}
	if occurrence.parent.IsZero() {
		if occurrence.edge != catalog.EdgeInvalid || occurrence.ordinal != 0 {
			return fmt.Errorf(
				"root occurrence %s has child-edge facts", occurrence.id,
			)
		}
		return nil
	}
	parent, present := all[occurrence.parent]
	if !present {
		return fmt.Errorf(
			"occurrence %s has absent parent %s", occurrence.id, occurrence.parent,
		)
	}
	if !occurrence.edge.Valid() ||
		occurrence.edge.Parent() != parent.kind ||
		(!occurrence.edge.IsList() && occurrence.ordinal != 0) {
		return fmt.Errorf(
			"occurrence %s has invalid parent edge %s", occurrence.id, occurrence.edge,
		)
	}
	return nil
}

func validateDefinition(
	graph *Graph,
	definition identity.DefinitionID,
	site DefinitionSite,
	header HeaderRegion,
	boundary ExecutionBoundary,
	owners map[OwnerRegionID]int,
	definitions map[identity.DefinitionID]ImplementationDefinition,
	requiredAnchors map[identity.OccurrenceID]bool,
) error {
	record := graph.byDefinition[definition]
	if record.owner != site.owner ||
		record.header != header.id ||
		record.boundary != boundary.id ||
		owners[record.owner] != 1 ||
		!validSHA256(header.digest) ||
		!validSHA256(boundary.combinedDigest) {
		return fmt.Errorf("definition %s has incoherent owned records", definition)
	}
	if definition.File().IsZero() {
		if site.kind != DefinitionSiteSynthetic ||
			!site.terminal.IsZero() ||
			!site.parentDefinition.IsZero() ||
			record.owner.kind != OwnerRegionSynthetic {
			return fmt.Errorf("implicit definition %s has a source-shaped site", definition)
		}
	} else {
		if site.kind != DefinitionSiteSource ||
			site.terminal != definition.Root() ||
			record.owner.kind != OwnerRegionSourceFile ||
			record.owner.file != definition.File() {
			return fmt.Errorf("source definition %s has an invalid site", definition)
		}
		if err := validateContainmentPath(
			graph, site, definitions, requiredAnchors,
		); err != nil {
			return err
		}
	}
	if err := validateHeader(graph, definition, header); err != nil {
		return err
	}
	return validateBoundary(graph, definition, boundary)
}

func validateContainmentPath(
	graph *Graph,
	site DefinitionSite,
	definitions map[identity.DefinitionID]ImplementationDefinition,
	requiredAnchors map[identity.OccurrenceID]bool,
) error {
	current, present := graph.byOccurrence[site.terminal]
	if !present {
		return fmt.Errorf(
			"definition %s terminal %s has no occurrence",
			site.definition, site.terminal,
		)
	}
	visited := map[identity.OccurrenceID]bool{current.id: true}
	for !current.parent.IsZero() {
		if !site.parentDefinition.IsZero() &&
			current.parent == site.parentDefinition.Root() {
			parent, present := definitions[site.parentDefinition]
			if !present ||
				parent.owner != site.owner ||
				parent.id.File() != site.definition.File() {
				return fmt.Errorf(
					"definition %s has invalid enclosing definition %s",
					site.definition, site.parentDefinition,
				)
			}
			return nil
		}
		parent, exists := graph.byOccurrence[current.parent]
		if !exists {
			return fmt.Errorf(
				"definition %s path loses parent %s",
				site.definition, current.parent,
			)
		}
		if visited[parent.id] {
			return fmt.Errorf(
				"definition %s containment path cycles at %s",
				site.definition, parent.id,
			)
		}
		visited[parent.id] = true
		if parent.parent.IsZero() {
			if !site.parentDefinition.IsZero() ||
				parent.kind != catalog.KindFile ||
				parent.id.Span().File() != site.owner.file {
				return fmt.Errorf(
					"definition %s path reaches wrong owner root", site.definition,
				)
			}
			return nil
		}
		requiredAnchors[parent.id] = true
		current = parent
	}
	return fmt.Errorf(
		"definition %s containment path is not rooted", site.definition,
	)
}

func validateHeader(
	graph *Graph,
	definition identity.DefinitionID,
	header HeaderRegion,
) error {
	if len(header.members) == 0 {
		if definition.Kind() == identity.DefinitionImplicit {
			return nil
		}
		return fmt.Errorf("source definition %s has an empty header", definition)
	}
	if header.members[0] != definition.Root() {
		return fmt.Errorf(
			"definition %s header is not rooted at its construct", definition,
		)
	}
	seen := map[identity.OccurrenceID]bool{}
	for index, member := range header.members {
		if seen[member] {
			return fmt.Errorf("definition %s repeats header member %s", definition, member)
		}
		seen[member] = true
		occurrence, present := graph.byOccurrence[member]
		if !present {
			return fmt.Errorf(
				"definition %s header member %s is absent", definition, member,
			)
		}
		if index > 0 && occurrence.edge.DefinitionEntry() {
			return fmt.Errorf(
				"definition %s header contains execution entry %s",
				definition, member,
			)
		}
	}
	return nil
}

func validateBoundary(
	graph *Graph,
	definition identity.DefinitionID,
	boundary ExecutionBoundary,
) error {
	if !boundary.kind.Valid() {
		return fmt.Errorf("definition %s has invalid boundary kind", definition)
	}
	switch boundary.kind {
	case BoundaryBlock:
		if len(boundary.entries) != 1 ||
			(definition.Kind() != identity.DefinitionFuncDecl &&
				definition.Kind() != identity.DefinitionFuncLit) {
			return fmt.Errorf("definition %s has invalid block boundary", definition)
		}
	case BoundaryInitializers:
		if len(boundary.entries) == 0 ||
			definition.Kind() != identity.DefinitionPackageInitializer {
			return fmt.Errorf(
				"definition %s has invalid initializer boundary", definition,
			)
		}
	case BoundaryBodyless:
		if len(boundary.entries) != 0 ||
			definition.Kind() != identity.DefinitionBodylessDecl {
			return fmt.Errorf("definition %s has invalid bodyless boundary", definition)
		}
	case BoundaryImplicit:
		if len(boundary.entries) != 0 ||
			definition.Kind() != identity.DefinitionImplicit ||
			(boundary.implicit.Valid() == boundary.synthetic.Valid()) {
			return fmt.Errorf("definition %s has invalid implicit boundary", definition)
		}
	}
	for _, entry := range boundary.entries {
		occurrence, present := graph.byOccurrence[entry.id]
		if !present ||
			occurrence.parent != definition.Root() ||
			!occurrence.edge.DefinitionEntry() ||
			!validSHA256(entry.hash) {
			return fmt.Errorf(
				"definition %s has invalid execution entry %s",
				definition, entry.id,
			)
		}
	}
	switch boundary.kind {
	case BoundaryBlock, BoundaryInitializers:
		hashes := make([]string, 0, len(boundary.entries))
		seen := map[identity.OccurrenceID]bool{}
		for _, entry := range boundary.entries {
			if seen[entry.id] {
				return fmt.Errorf(
					"definition %s repeats execution entry %s",
					definition, entry.id,
				)
			}
			seen[entry.id] = true
			hashes = append(hashes, entry.hash)
		}
		if boundary.combinedDigest != digestStrings(hashes...) {
			return fmt.Errorf(
				"definition %s has an invalid combined execution digest",
				definition,
			)
		}
	case BoundaryBodyless:
		if boundary.combinedDigest != digestStrings(
			definition.String(), "bodyless-obligation",
		) {
			return fmt.Errorf(
				"definition %s has an invalid bodyless digest", definition,
			)
		}
	case BoundaryImplicit:
		if boundary.implicit.Valid() {
			if boundary.combinedDigest != digestStrings(
				definition.String(), "execution",
			) {
				return fmt.Errorf(
					"definition %s has an invalid implicit digest",
					definition,
				)
			}
		} else if !validSHA256(boundary.combinedDigest) {
			return fmt.Errorf(
				"definition %s has an invalid synthetic digest",
				definition,
			)
		}
	}
	return nil
}

func validatePackageGraph(pkg PackageGraph) error {
	seen := map[identity.DefinitionID]bool{}
	for _, definition := range pkg.Definitions() {
		if definition.id.IsZero() || seen[definition.id] {
			return fmt.Errorf(
				"package %s has zero or duplicate definition %s",
				pkg.id, definition.id,
			)
		}
		seen[definition.id] = true
	}
	return nil
}
