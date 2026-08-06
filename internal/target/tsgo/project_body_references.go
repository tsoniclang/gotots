package tsgo

import (
	"fmt"
	"sort"
)

// BodyReferenceDeclaration is one checked declaration site of a resolved
// implementation-body reference.
type BodyReferenceDeclaration struct {
	Path  string
	Index uint32
	Kind  SyntaxKind
}

// BodyValueReference is one checked identifier reference inside an
// implementation body: the checker symbol name, whether the identifier sits
// in call position, whether the checker resolved it, whether every
// declaration is local to the walked body, and the exact declaration sites.
type BodyValueReference struct {
	Name         string
	CallPosition bool
	TypePosition bool
	Resolved     bool
	Local        bool
	Declarations []BodyReferenceDeclaration
}

// ParseNodeHandle decomposes one canonical declaration node handle into its
// node index, syntax kind, and source path.
func ParseNodeHandle(
	handle string,
) (uint32, SyntaxKind, string, error) {
	index, kind, path, err := parseProjectNodeHandle(handle)
	return index, SyntaxKind(kind), path, err
}

type getSymbolsAtLocationsParams struct {
	Snapshot  uint64   `json:"snapshot"`
	Project   string   `json:"project"`
	Locations []string `json:"locations"`
}

// ImplementationBodyReferences enumerates every identifier inside one
// checked declaration subtree and resolves each through the pinned
// checker. Import aliases are resolved through the checker's module symbol
// and export set to their true declaring sites; message strings and source
// text never participate.
func (p *ProjectInspection) ImplementationBodyReferences(
	declarationHandle string,
) ([]BodyValueReference, error) {
	return p.declarationBodyReferences(declarationHandle, false)
}

// ConstructionBodyReferences enumerates only the construction behavior of
// one checked class declaration: its constructor and property-initializer
// subtrees. Method bodies are selected per member by their own evidence.
func (p *ProjectInspection) ConstructionBodyReferences(
	declarationHandle string,
) ([]BodyValueReference, error) {
	return p.declarationBodyReferences(declarationHandle, true)
}

func (p *ProjectInspection) declarationBodyReferences(
	declarationHandle string,
	constructionOnly bool,
) ([]BodyValueReference, error) {
	rootIndex, _, sourcePath, err := parseProjectNodeHandle(declarationHandle)
	if err != nil {
		return nil, err
	}
	source, err := p.projectSourceEvidence(sourcePath)
	if err != nil {
		return nil, err
	}
	if rootIndex == 0 || rootIndex >= uint32(len(source.nodes)) {
		return nil, projectNodeEvidenceError(
			declarationHandle,
			"declaration is outside the source",
		)
	}
	identifiers := make([]uint32, 0, 64)
	for index := uint32(1); index < uint32(len(source.nodes)); index++ {
		if source.nodes[index].kind != uint32(SyntaxKindIdentifier) {
			continue
		}
		if !source.underAncestor(index, rootIndex) {
			continue
		}
		if constructionOnly &&
			!source.constructionPosition(index, rootIndex) {
			continue
		}
		identifiers = append(identifiers, index)
	}
	if len(identifiers) == 0 {
		return nil, nil
	}
	symbols, err := p.symbolsAtProjectNodes(sourcePath, source, identifiers)
	if err != nil {
		return nil, err
	}
	references := make([]BodyValueReference, 0, len(identifiers))
	for position, index := range identifiers {
		reference := BodyValueReference{
			CallPosition: source.callPosition(index),
			TypePosition: source.typePosition(index, rootIndex),
		}
		symbol := symbols[position]
		if symbol == nil {
			references = append(references, reference)
			continue
		}
		reference.Name = symbol.Name
		reference.Resolved = true
		declarations, local, err := p.resolveReferenceDeclarations(
			sourcePath,
			source,
			rootIndex,
			*symbol,
		)
		if err != nil {
			return nil, err
		}
		reference.Local = local
		reference.Declarations = declarations
		references = append(references, reference)
	}
	return references, nil
}

// resolveReferenceDeclarations parses one resolved symbol's declaration
// sites, reporting locality against the walked body and resolving import
// aliases through the checker's module exports.
func (p *ProjectInspection) resolveReferenceDeclarations(
	sourcePath string,
	source projectSourceEvidence,
	rootIndex uint32,
	symbol symbolResponse,
) ([]BodyReferenceDeclaration, bool, error) {
	declarations := make([]BodyReferenceDeclaration, 0, len(symbol.Declarations))
	local := len(symbol.Declarations) != 0
	aliasIndex := uint32(0)
	for _, handle := range symbol.Declarations {
		index, kind, path, err := parseProjectNodeHandle(handle)
		if err != nil {
			return nil, false, err
		}
		if !sameProjectPath(path, sourcePath) ||
			!source.underAncestor(index, rootIndex) {
			local = false
		}
		switch SyntaxKind(kind) {
		case SyntaxKindImportSpecifier,
			SyntaxKindImportClause,
			SyntaxKindNamespaceImport:
			if sameProjectPath(path, sourcePath) {
				aliasIndex = index
			}
		}
		declarations = append(declarations, BodyReferenceDeclaration{
			Path:  path,
			Index: index,
			Kind:  SyntaxKind(kind),
		})
	}
	if aliasIndex == 0 {
		return declarations, local, nil
	}
	resolved, err := p.resolveImportAlias(
		sourcePath,
		source,
		aliasIndex,
		symbol.Name,
	)
	if err != nil {
		return nil, false, err
	}
	if len(resolved) != 0 {
		return resolved, false, nil
	}
	return declarations, false, nil
}

// resolveImportAlias resolves one import-specifier declaration to the true
// declaring sites of the imported symbol: the enclosing import declaration's
// module-specifier literal resolves to the checker's module symbol, whose
// export set names the target symbol.
func (p *ProjectInspection) resolveImportAlias(
	sourcePath string,
	source projectSourceEvidence,
	specifierIndex uint32,
	name string,
) ([]BodyReferenceDeclaration, error) {
	importIndex := specifierIndex
	for importIndex != 0 &&
		source.nodes[importIndex].kind != uint32(SyntaxKindImportDeclaration) &&
		source.nodes[importIndex].kind != uint32(SyntaxKindExportDeclaration) {
		importIndex = source.nodes[importIndex].parent
	}
	if importIndex == 0 {
		return nil, projectNodeEvidenceError(
			sourcePath,
			"alias specifier has no import or export declaration",
		)
	}
	moduleLiteral := uint32(0)
	for _, child := range source.directChildren(importIndex) {
		if source.nodes[child].kind == uint32(SyntaxKindStringLiteral) {
			moduleLiteral = child
		}
	}
	if moduleLiteral == 0 {
		// A local export declaration has no module specifier; the alias
		// resolves through the file's own bindings instead.
		return nil, nil
	}
	module, err := p.symbolAtProjectNode(
		sourcePath,
		moduleLiteral,
		"import module specifier",
	)
	if err != nil {
		return nil, err
	}
	var exports []symbolResponse
	if err := requestProjectJSON(
		p.client,
		"getExportsOfSymbol",
		getExportsOfSymbolParams{
			Snapshot: p.snapshot,
			Symbol:   module.ID,
		},
		&exports,
	); err != nil {
		return nil, err
	}
	for _, exported := range exports {
		if exported.Name != name {
			continue
		}
		declarations := make(
			[]BodyReferenceDeclaration,
			0,
			len(exported.Declarations),
		)
		for _, handle := range exported.Declarations {
			index, kind, path, parseErr := parseProjectNodeHandle(handle)
			if parseErr != nil {
				return nil, parseErr
			}
			declarations = append(declarations, BodyReferenceDeclaration{
				Path:  path,
				Index: index,
				Kind:  SyntaxKind(kind),
			})
		}
		sort.Slice(declarations, func(left, right int) bool {
			if declarations[left].Path != declarations[right].Path {
				return declarations[left].Path < declarations[right].Path
			}
			return declarations[left].Index < declarations[right].Index
		})
		return declarations, nil
	}
	return nil, nil
}

// ResolveDeclarationHandles maps export/import alias declarations to the
// true implementation declaration handles through the pinned checker, so
// re-export facades never masquerade as implementation bodies.
func (p *ProjectInspection) ResolveDeclarationHandles(
	handles []string,
) ([]string, error) {
	resolved := make([]string, 0, len(handles))
	seen := make(map[string]struct{}, len(handles))
	pending := append([]string(nil), handles...)
	for steps := 0; len(pending) != 0; steps++ {
		if steps > 4096 {
			return nil, projectNodeEvidenceError(
				"declaration resolution",
				"alias resolution did not terminate",
			)
		}
		handle := pending[0]
		pending = pending[1:]
		if _, duplicate := seen[handle]; duplicate {
			continue
		}
		seen[handle] = struct{}{}
		index, kind, sourcePath, err := parseProjectNodeHandle(handle)
		if err != nil {
			return nil, err
		}
		switch SyntaxKind(kind) {
		case SyntaxKindExportSpecifier,
			SyntaxKindImportSpecifier,
			SyntaxKindImportClause,
			SyntaxKindNamespaceImport:
		default:
			resolved = append(resolved, handle)
			continue
		}
		source, err := p.projectSourceEvidence(sourcePath)
		if err != nil {
			return nil, err
		}
		if index == 0 || index >= uint32(len(source.nodes)) {
			return nil, projectNodeEvidenceError(
				handle,
				"alias declaration is outside the source",
			)
		}
		nameIndex := source.firstChildOfKind(
			index,
			uint32(SyntaxKindIdentifier),
		)
		if nameIndex == 0 {
			return nil, projectNodeEvidenceError(
				handle,
				"alias declaration has no name",
			)
		}
		symbol, err := p.symbolAtProjectNode(
			sourcePath,
			nameIndex,
			"alias declaration name",
		)
		if err != nil {
			return nil, err
		}
		if len(symbol.Declarations) == 0 {
			return nil, projectNodeEvidenceError(
				handle,
				"alias declaration resolved no target",
			)
		}
		progressed := false
		for _, target := range symbol.Declarations {
			if _, duplicate := seen[target]; !duplicate {
				progressed = true
			}
			pending = append(pending, target)
		}
		if !progressed {
			// The checker returned only already-seen alias declarations;
			// resolve through the module's export set instead.
			targets, moduleErr := p.moduleExportDeclarations(
				sourcePath,
				source,
				index,
				symbol.Name,
			)
			if moduleErr != nil {
				return nil, moduleErr
			}
			if len(targets) == 0 {
				targets, moduleErr = p.resolveLocalAlias(
					sourcePath,
					source,
					symbol.Name,
					symbol.ID,
				)
				if moduleErr != nil {
					return nil, moduleErr
				}
			}
			if len(targets) == 0 {
				return nil, projectNodeEvidenceError(
					handle,
					"alias declaration has no module or local target",
				)
			}
			pending = append(pending, targets...)
		}
	}
	sort.Strings(resolved)
	return resolved, nil
}

// moduleExportDeclarations resolves one alias specifier's target through
// its declaration's module specifier and the module's checked export set.
func (p *ProjectInspection) moduleExportDeclarations(
	sourcePath string,
	source projectSourceEvidence,
	specifierIndex uint32,
	name string,
) ([]string, error) {
	references, err := p.resolveImportAlias(
		sourcePath,
		source,
		specifierIndex,
		name,
	)
	if err != nil {
		return nil, err
	}
	handles := make([]string, 0, len(references))
	for _, reference := range references {
		handles = append(handles, projectNodeHandle(
			reference.Path,
			reference.Index,
			uint32(reference.Kind),
		))
	}
	return handles, nil
}

// resolveLocalAlias resolves one locally exported name through the file's
// own checked binding declarations: the module-scope declaration whose
// checker symbol carries the same name is the alias target.
func (p *ProjectInspection) resolveLocalAlias(
	sourcePath string,
	source projectSourceEvidence,
	name string,
	aliasSymbol uint64,
) ([]string, error) {
	candidates := make([]uint32, 0, 32)
	for index := uint32(1); index < uint32(len(source.nodes)); index++ {
		switch SyntaxKind(source.nodes[index].kind) {
		case SyntaxKindImportSpecifier,
			SyntaxKindImportClause,
			SyntaxKindNamespaceImport,
			SyntaxKindFunctionDeclaration,
			SyntaxKindClassDeclaration,
			SyntaxKindVariableDeclaration,
			SyntaxKindEnumDeclaration:
		default:
			continue
		}
		nameIndex := source.firstChildOfKind(
			index,
			uint32(SyntaxKindIdentifier),
		)
		if nameIndex != 0 {
			candidates = append(candidates, nameIndex)
		}
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	symbols, err := p.symbolsAtProjectNodes(sourcePath, source, candidates)
	if err != nil {
		return nil, err
	}
	for _, symbol := range symbols {
		if symbol == nil ||
			symbol.Name != name ||
			symbol.ID == aliasSymbol {
			continue
		}
		return cloneStrings(symbol.Declarations), nil
	}
	return nil, nil
}

// constructionPosition reports whether one node inside a class subtree
// belongs to construction behavior: a constructor body, a property
// initializer, or a heritage clause.
func (s projectSourceEvidence) constructionPosition(
	index uint32,
	root uint32,
) bool {
	for current := s.nodes[index].parent; current != 0; current = s.nodes[current].parent {
		switch SyntaxKind(s.nodes[current].kind) {
		case SyntaxKindConstructor,
			SyntaxKindPropertyDeclaration,
			SyntaxKindHeritageClause:
			return true
		case SyntaxKindMethodDeclaration,
			SyntaxKindGetAccessor,
			SyntaxKindSetAccessor:
			return false
		}
		if current == root {
			return false
		}
	}
	return false
}

func (s projectSourceEvidence) firstChildOfKind(
	parent uint32,
	kind uint32,
) uint32 {
	for index := uint32(1); index < uint32(len(s.nodes)); index++ {
		if s.nodes[index].parent == parent && s.nodes[index].kind == kind {
			return index
		}
	}
	return 0
}

// symbolsAtProjectNodes batch-resolves checker symbols for many nodes of
// one source file through the pinned batch endpoint.
func (p *ProjectInspection) symbolsAtProjectNodes(
	sourcePath string,
	source projectSourceEvidence,
	indices []uint32,
) ([]*symbolResponse, error) {
	locations := make([]string, len(indices))
	for position, index := range indices {
		locations[position] = projectNodeHandle(
			sourcePath,
			index,
			source.nodes[index].kind,
		)
	}
	var symbols []*symbolResponse
	if err := requestProjectJSON(
		p.client,
		"getSymbolsAtLocations",
		getSymbolsAtLocationsParams{
			Snapshot:  p.snapshot,
			Project:   p.project,
			Locations: locations,
		},
		&symbols,
	); err != nil {
		return nil, err
	}
	if len(symbols) != len(indices) {
		return nil, fmt.Errorf(
			"batch symbol resolution returned %d results for %d locations",
			len(symbols),
			len(indices),
		)
	}
	return symbols, nil
}

// underAncestor reports whether the ancestor chain of one node passes
// through the given root node.
func (s projectSourceEvidence) underAncestor(
	index uint32,
	root uint32,
) bool {
	for current := index; current != 0; current = s.nodes[current].parent {
		if current == root {
			return true
		}
	}
	return false
}

// callPosition reports whether one identifier is the invoked callee of a
// call expression, directly or through one property access.
func (s projectSourceEvidence) callPosition(index uint32) bool {
	parent := s.nodes[index].parent
	if parent == 0 {
		return false
	}
	if s.nodes[parent].kind == uint32(SyntaxKindCallExpression) {
		return s.firstChild(parent) == index
	}
	if s.nodes[parent].kind != uint32(SyntaxKindPropertyAccessExpression) {
		return false
	}
	grandparent := s.nodes[parent].parent
	return grandparent != 0 &&
		s.nodes[grandparent].kind == uint32(SyntaxKindCallExpression) &&
		s.firstChild(grandparent) == parent
}

// typePosition reports whether one identifier sits inside a type
// reference or type query, where it names a contract rather than
// executable behavior.
func (s projectSourceEvidence) typePosition(
	index uint32,
	root uint32,
) bool {
	for current := s.nodes[index].parent; current != 0 && current != root; current = s.nodes[current].parent {
		switch SyntaxKind(s.nodes[current].kind) {
		case SyntaxKindTypeReference, SyntaxKindTypeQuery:
			return true
		}
	}
	return false
}

func (s projectSourceEvidence) firstChild(parent uint32) uint32 {
	for index := uint32(1); index < uint32(len(s.nodes)); index++ {
		if s.nodes[index].parent == parent {
			return index
		}
	}
	return 0
}
