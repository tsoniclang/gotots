package structure

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/source"
)

type originKey struct {
	file   string
	line   int
	column int
	kind   identity.DefinitionKind
}

type checkedCandidate struct {
	root ast.Node
	join ast.Node
	kind identity.DefinitionKind
}

func attachCgo(
	universe *source.Universe,
	loaded *source.LoadedPackage,
	graph *PackageGraph,
	index *TransientIndex,
	work *Work,
	includeSynthetic bool,
) error {
	checked := loaded.CheckedDeclarations()
	if len(checked) == 0 {
		return nil
	}
	filesByPath := map[string]*source.LoadedFile{}
	for _, file := range loaded.Files() {
		if file.CgoOriginal() {
			if _, local := index.files[file.ID()]; !local {
				continue
			}
			filesByPath[file.Path()] = file
		}
	}
	if len(filesByPath) == 0 {
		return fmt.Errorf("package %s has checked declarations but no cgo originals", loaded.ID())
	}
	origins := map[originKey][]identity.DefinitionID{}
	for _, definition := range graph.Definitions() {
		file, exists := index.files[definition.id.File()]
		if !exists || !file.CgoOriginal() {
			continue
		}
		node := index.definitions[definition.id]
		join := definitionJoinNode(node)
		if join == nil {
			return fmt.Errorf("cgo definition %s has no origin join node", definition.id)
		}
		position := file.PhysicalFileSet().PositionFor(join.Pos(), false)
		key := originKey{
			file: file.Path(), line: position.Line, column: position.Column,
			kind: definition.id.Kind(),
		}
		origins[key] = append(origins[key], definition.id)
	}

	matched := map[identity.DefinitionID]bool{}
	synthetic := map[string]bool{}
	for _, declaration := range checked {
		display := universe.Fset().Position(declaration.Pos())
		if _, original := filesByPath[display.Filename]; !original {
			if !includeSynthetic {
				continue
			}
			if err := addSyntheticDeclaration(
				loaded.ID(),
				loaded.CheckerView(),
				universe.Fset(),
				declaration,
				graph,
				synthetic,
				work,
			); err != nil {
				return err
			}
			continue
		}
		for _, candidate := range checkedDefinitionCandidates(declaration) {
			position := universe.Fset().Position(candidate.join.Pos())
			origin := originKey{
				file: position.Filename, line: position.Line, column: position.Column,
				kind: candidate.kind,
			}
			definition, match, err := resolveOrigin(origins, origin)
			if err != nil {
				return err
			}
			if matched[definition] {
				return fmt.Errorf("cgo definition %s has multiple checked counterparts", definition)
			}
			matched[definition] = true
			index.checked[definition] = candidate.root
			checkedDigest, err := checkedNodeDigest(
				universe.Fset(), candidate.root,
			)
			if err != nil {
				return err
			}
			mapping := CheckedDefinitionMapping{
				definition:    definition,
				originLine:    origin.line,
				originColumn:  origin.column,
				originMatch:   match,
				checkedDigest: checkedDigest,
			}
			for fileIndex := range graph.files {
				if graph.files[fileIndex].owner.id.file == definition.File() {
					graph.files[fileIndex].mappings = append(
						graph.files[fileIndex].mappings, mapping,
					)
					break
				}
			}
			work.RecordAppends++
		}
	}
	for _, definition := range graph.Definitions() {
		file, exists := index.files[definition.id.File()]
		if exists && file.CgoOriginal() && !matched[definition.id] {
			return fmt.Errorf("cgo definition %s has no checked counterpart", definition.id)
		}
	}
	for fileIndex := range graph.files {
		sort.Slice(graph.files[fileIndex].mappings, func(i, j int) bool {
			work.SortComparisons++
			return graph.files[fileIndex].mappings[i].definition.String() <
				graph.files[fileIndex].mappings[j].definition.String()
		})
	}
	return nil
}

func definitionJoinNode(node ast.Node) ast.Node {
	switch node := node.(type) {
	case *ast.FuncDecl:
		if node.Body != nil {
			return node.Body
		}
		return node
	case *ast.FuncLit:
		return node
	case *ast.ValueSpec:
		return node
	default:
		return nil
	}
}

func checkedDefinitionCandidates(declaration ast.Node) []checkedCandidate {
	var out []checkedCandidate
	switch node := declaration.(type) {
	case *ast.FuncDecl:
		kind := identity.DefinitionFuncDecl
		join := ast.Node(node.Body)
		if node.Body == nil {
			kind = identity.DefinitionBodylessDecl
			join = node
		}
		out = append(out, checkedCandidate{root: node, join: join, kind: kind})
		if node.Body != nil {
			out = append(out, checkedLiterals(node.Body)...)
		}
	case *ast.GenDecl:
		if node.Tok != token.VAR {
			break
		}
		for _, spec := range node.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok || len(value.Values) == 0 {
				continue
			}
			out = append(out, checkedCandidate{
				root: value, join: value, kind: identity.DefinitionPackageInitializer,
			})
			out = append(out, checkedLiterals(value)...)
		}
	}
	return out
}

func checkedLiterals(root ast.Node) []checkedCandidate {
	var out []checkedCandidate
	ast.Inspect(root, func(node ast.Node) bool {
		if literal, ok := node.(*ast.FuncLit); ok {
			out = append(out, checkedCandidate{
				root: literal, join: literal, kind: identity.DefinitionFuncLit,
			})
		}
		return true
	})
	return out
}

func resolveOrigin(
	origins map[originKey][]identity.DefinitionID,
	key originKey,
) (identity.DefinitionID, CheckedOriginMatch, error) {
	candidates := origins[key]
	if len(candidates) == 1 {
		return candidates[0], CheckedOriginExact, nil
	}
	if len(candidates) > 1 {
		return identity.DefinitionID{}, CheckedOriginInvalid, fmt.Errorf(
			"cgo origin %s:%d:%d kind %s resolves %d exact definitions",
			key.file, key.line, key.column, key.kind, len(candidates),
		)
	}
	if len(candidates) == 0 {
		for candidateKey, definitions := range origins {
			if candidateKey.file == key.file && candidateKey.line == key.line &&
				candidateKey.kind == key.kind {
				candidates = append(candidates, definitions...)
			}
		}
	}
	if len(candidates) != 1 {
		return identity.DefinitionID{}, CheckedOriginInvalid, fmt.Errorf(
			"cgo origin %s:%d:%d kind %s resolves %d definitions",
			key.file, key.line, key.column, key.kind, len(candidates),
		)
	}
	return candidates[0], CheckedOriginUniqueLine, nil
}

func addSyntheticDeclaration(
	pkg identity.PackageID,
	view *source.TypeInfoView,
	fset *token.FileSet,
	declaration ast.Node,
	graph *PackageGraph,
	seen map[string]bool,
	work *Work,
) error {
	for _, descriptor := range syntheticDescriptors(declaration, view) {
		definitionID, err := identity.NewSyntheticDefinitionID(pkg, descriptor.role, descriptor.name)
		if err != nil {
			return err
		}
		if seen[definitionID.String()] {
			return fmt.Errorf("duplicate cgo synthetic definition %s", definitionID)
		}
		seen[definitionID.String()] = true
		ownerID, err := SyntheticOwner(pkg, SyntheticOwnerCgoGenerated)
		if err != nil {
			return err
		}
		if len(graph.synthetic) == 0 || graph.synthetic[len(graph.synthetic)-1].id != ownerID {
			graph.synthetic = append(graph.synthetic, OwnerRegion{id: ownerID})
		}
		headerID, _ := identity.NewHeaderRegionID(definitionID)
		boundaryID, _ := identity.NewExecutionBoundaryID(definitionID)
		headerContent, boundaryContent, err := syntheticContentDigests(
			fset, descriptor.node, descriptor.nameIndex,
		)
		if err != nil {
			return err
		}
		graph.ownedDefinitions = append(
			graph.ownedDefinitions,
			ImplementationDefinition{
				id: definitionID, owner: ownerID, header: headerID, boundary: boundaryID,
				name: descriptor.name,
			},
		)
		graph.ownedSites = append(graph.ownedSites, DefinitionSite{
			kind:       DefinitionSiteSynthetic,
			definition: definitionID,
			owner:      ownerID,
		})
		graph.ownedHeaders = append(graph.ownedHeaders, HeaderRegion{
			id: headerID,
			digest: digestStrings(
				definitionID.String(),
				"synthetic-header",
				headerContent,
			),
		})
		graph.ownedBoundaries = append(graph.ownedBoundaries, ExecutionBoundary{
			id: boundaryID, kind: BoundaryImplicit, synthetic: descriptor.role,
			combinedDigest: digestStrings(
				definitionID.String(),
				"synthetic-boundary",
				boundaryContent,
			),
		})
		work.RecordAppends += 4
	}
	return nil
}

type syntheticDescriptor struct {
	name      string
	role      identity.SyntheticDefinitionRole
	node      ast.Node
	nameIndex int
}

func syntheticDescriptors(
	node ast.Node,
	view *source.TypeInfoView,
) []syntheticDescriptor {
	var out []syntheticDescriptor
	switch node := node.(type) {
	case *ast.FuncDecl:
		if node.Name != nil && node.Name.Name != "" {
			name := node.Name.Name
			if view != nil {
				if object, present := view.DefOf(node.Name); present {
					if function, ok := object.(*types.Func); ok {
						name = function.FullName()
					}
				}
			}
			out = append(out, syntheticDescriptor{
				name: name,
				role: identity.SyntheticDefinitionAdapter,
				node: node,
			})
		}
	case *ast.GenDecl:
		switch node.Tok {
		case token.TYPE:
			for _, spec := range node.Specs {
				if named, ok := spec.(*ast.TypeSpec); ok && named.Name != nil {
					out = append(out, syntheticDescriptor{
						name: named.Name.Name,
						role: identity.SyntheticDefinitionType,
						node: named,
					})
				}
			}
		case token.VAR, token.CONST:
			for _, spec := range node.Specs {
				if values, ok := spec.(*ast.ValueSpec); ok {
					for nameIndex, name := range values.Names {
						if name.Name != "" && name.Name != "_" {
							out = append(out, syntheticDescriptor{
								name:      name.Name,
								role:      identity.SyntheticDefinitionData,
								node:      values,
								nameIndex: nameIndex,
							})
						}
					}
				}
			}
		}
	}
	return out
}
