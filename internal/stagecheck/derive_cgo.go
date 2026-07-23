package stagecheck

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/source"
)

type independentOrigin struct {
	file   string
	line   int
	column int
	kind   identity.DefinitionKind
}

type independentCheckedDefinition struct {
	root ast.Node
	join ast.Node
	kind identity.DefinitionKind
}

func deriveCgoPackage(
	universe *source.Universe,
	pkg *source.LoadedPackage,
	local map[identity.FileID]*derivedFile,
	includeSynthetic bool,
	ledger *structuralLedger,
) error {
	if len(pkg.CheckedDeclarations()) == 0 {
		return nil
	}
	allOriginalPaths := map[string]bool{}
	localOriginalPaths := map[string]bool{}
	for _, file := range pkg.Files() {
		if !file.CgoOriginal() {
			continue
		}
		allOriginalPaths[file.Path()] = true
		if local[file.ID()] != nil {
			localOriginalPaths[file.Path()] = true
		}
	}
	origins := map[independentOrigin][]identity.DefinitionID{}
	for _, file := range local {
		if !file.file.CgoOriginal() {
			continue
		}
		for definition, node := range file.definitions {
			join, err := independentOriginNode(definition.Kind(), node)
			if err != nil {
				return err
			}
			position := file.fset.PositionFor(join.Pos(), false)
			key := independentOrigin{
				file: file.file.Path(),
				line: position.Line, column: position.Column,
				kind: definition.Kind(),
			}
			origins[key] = append(origins[key], definition)
		}
	}
	matched := map[identity.DefinitionID]bool{}
	syntheticSeen := map[identity.DefinitionID]bool{}
	for _, declaration := range pkg.CheckedDeclarations() {
		display := universe.Fset().Position(declaration.Pos())
		if allOriginalPaths[display.Filename] &&
			!localOriginalPaths[display.Filename] {
			continue
		}
		if !allOriginalPaths[display.Filename] {
			if includeSynthetic {
				if err := deriveSynthetic(
					pkg,
					universe.Fset(),
					declaration,
					syntheticSeen,
					ledger,
				); err != nil {
					return err
				}
			}
			continue
		}
		packageContext, err := catalog.NewDefinitionContext(
			catalog.DefinitionScopePackage, catalog.TokenInvalid,
		)
		if err != nil {
			return err
		}
		candidates, err := independentCheckedDefinitions(
			declaration, packageContext,
		)
		if err != nil {
			return err
		}
		for _, candidate := range candidates {
			position := universe.Fset().Position(candidate.join.Pos())
			origin := independentOrigin{
				file:   position.Filename,
				line:   position.Line,
				column: position.Column,
				kind:   candidate.kind,
			}
			definition, match, err := independentResolveOrigin(
				origins,
				origin,
			)
			if err != nil {
				return err
			}
			if matched[definition] {
				return fmt.Errorf(
					"independent cgo origin duplicates %s", definition,
				)
			}
			matched[definition] = true
			checkedDigest, err := independentCheckedNodeDigest(
				universe.Fset(), candidate.root,
			)
			if err != nil {
				return err
			}
			ledger.add(
				"checked-mapping",
				fmt.Sprintf(
					"%s|%d|%d|%d|%s",
					definition,
					origin.line,
					origin.column,
					uint8(match),
					checkedDigest,
				),
			)
		}
	}
	for _, definitions := range origins {
		for _, definition := range definitions {
			if !matched[definition] {
				return fmt.Errorf(
					"independent cgo definition %s has no checked counterpart",
					definition,
				)
			}
		}
	}
	return nil
}

func independentOriginNode(
	kind identity.DefinitionKind,
	node ast.Node,
) (ast.Node, error) {
	switch kind {
	case identity.DefinitionFuncDecl:
		entries, err := independentDefinitionEntries(node)
		if err != nil || len(entries) != 1 {
			return nil, fmt.Errorf(
				"cgo function definition has invalid entries",
			)
		}
		return entries[0].node, nil
	case identity.DefinitionFuncLit,
		identity.DefinitionPackageInitializer,
		identity.DefinitionBodylessDecl:
		return node, nil
	default:
		return nil, fmt.Errorf("cgo definition has unsupported kind %s", kind)
	}
}

func independentCheckedDefinitions(
	node ast.Node,
	context catalog.DefinitionContext,
) ([]independentCheckedDefinition, error) {
	kind, err := independentKind(node)
	if err != nil {
		return nil, err
	}
	entries, err := independentDefinitionEntries(node)
	if err != nil {
		return nil, err
	}
	definitionKind, definition, err := catalog.DefinitionKind(
		kind, context, len(entries) > 0,
	)
	if err != nil {
		return nil, err
	}
	if definition {
		join, err := independentOriginNode(definitionKind, node)
		if err != nil {
			return nil, err
		}
		out := []independentCheckedDefinition{{
			root: node, join: join, kind: definitionKind,
		}}
		for _, entry := range entries {
			executableContext, contextErr := catalog.NewDefinitionContext(
				catalog.DefinitionScopeExecutable, catalog.TokenInvalid,
			)
			if contextErr != nil {
				return nil, contextErr
			}
			nested, err := independentCheckedDefinitions(
				entry.node, executableContext,
			)
			if err != nil {
				return nil, err
			}
			out = append(out, nested...)
		}
		return out, nil
	}
	children, err := independentChildren(node, kind)
	if err != nil {
		return nil, err
	}
	childContext, err := independentChildContext(node, kind, context)
	if err != nil {
		return nil, err
	}
	var out []independentCheckedDefinition
	for _, child := range children {
		nested, err := independentCheckedDefinitions(
			child.node, childContext,
		)
		if err != nil {
			return nil, err
		}
		out = append(out, nested...)
	}
	return out, nil
}

func independentResolveOrigin(
	origins map[independentOrigin][]identity.DefinitionID,
	key independentOrigin,
) (identity.DefinitionID, structure.CheckedOriginMatch, error) {
	candidates := append([]identity.DefinitionID(nil), origins[key]...)
	if len(candidates) == 1 {
		return candidates[0], structure.CheckedOriginExact, nil
	}
	if len(candidates) > 1 {
		return identity.DefinitionID{},
			structure.CheckedOriginInvalid,
			fmt.Errorf(
				"independent cgo origin %s:%d:%d kind %s resolves %v",
				key.file, key.line, key.column, key.kind, candidates,
			)
	}
	if len(candidates) == 0 {
		for candidate, definitions := range origins {
			if candidate.file == key.file &&
				candidate.line == key.line &&
				candidate.kind == key.kind {
				candidates = append(candidates, definitions...)
			}
		}
	}
	if len(candidates) != 1 {
		sort.Slice(candidates, func(left, right int) bool {
			return candidates[left].String() < candidates[right].String()
		})
		return identity.DefinitionID{},
			structure.CheckedOriginInvalid,
			fmt.Errorf(
				"independent cgo origin %s:%d:%d kind %s resolves %v",
				key.file, key.line, key.column, key.kind, candidates,
			)
	}
	return candidates[0], structure.CheckedOriginUniqueLine, nil
}

func deriveSynthetic(
	pkg *source.LoadedPackage,
	fset *token.FileSet,
	declaration ast.Node,
	seen map[identity.DefinitionID]bool,
	ledger *structuralLedger,
) error {
	for _, descriptor := range independentSyntheticDescriptors(
		declaration, pkg.CheckerView(),
	) {
		definition, err := identity.NewSyntheticDefinitionID(
			pkg.ID(), descriptor.role, descriptor.name,
		)
		if err != nil {
			return err
		}
		if seen[definition] {
			return fmt.Errorf(
				"independent cgo synthetic duplicates %s", definition,
			)
		}
		firstSynthetic := len(seen) == 0
		seen[definition] = true
		ownerID, err := structure.SyntheticOwner(
			pkg.ID(), structure.SyntheticOwnerCgoGenerated,
		)
		if err != nil {
			return err
		}
		owner := ownerID.String()
		header, _ := identity.NewHeaderRegionID(definition)
		boundary, _ := identity.NewExecutionBoundaryID(definition)
		headerContent, boundaryContent, err :=
			independentSyntheticContentDigests(
				fset,
				descriptor.node,
				descriptor.nameIndex,
			)
		if err != nil {
			return err
		}
		if firstSynthetic {
			ledger.add("owner", owner)
		}
		ledger.add(
			"definition",
			fmt.Sprintf(
				"%s|%s|%s|%s|%s",
				definition,
				owner,
				header,
				boundary,
				descriptor.name,
			),
		)
		ledger.add(
			"definition-site",
			fmt.Sprintf(
				"%d|%s|%s||",
				uint8(structure.DefinitionSiteSynthetic),
				definition,
				owner,
			),
		)
		ledger.add(
			"header",
			fmt.Sprintf(
				"%s|%s",
				header,
				independentDigest(
					definition.String(),
					"synthetic-header",
					headerContent,
				),
			),
		)
		ledger.add(
			"execution-boundary",
			fmt.Sprintf(
				"%s|%d|%s|0|%d",
				boundary,
				uint8(structure.BoundaryImplicit),
				independentDigest(
					definition.String(),
					"synthetic-boundary",
					boundaryContent,
				),
				uint8(descriptor.role),
			),
		)
	}
	return nil
}

type independentSyntheticDescriptor struct {
	name      string
	role      identity.SyntheticDefinitionRole
	node      ast.Node
	nameIndex int
}

func independentSyntheticDescriptors(
	node ast.Node,
	view *source.TypeInfoView,
) []independentSyntheticDescriptor {
	var out []independentSyntheticDescriptor
	switch typed := node.(type) {
	case *ast.FuncDecl:
		if typed.Name != nil && typed.Name.Name != "" {
			name := typed.Name.Name
			if view != nil {
				if object, present := view.DefOf(typed.Name); present {
					if function, ok := object.(*types.Func); ok {
						name = function.FullName()
					}
				}
			}
			out = append(out, independentSyntheticDescriptor{
				name: name,
				role: identity.SyntheticDefinitionAdapter,
				node: typed,
			})
		}
	case *ast.GenDecl:
		for _, spec := range typed.Specs {
			switch declaration := spec.(type) {
			case *ast.TypeSpec:
				if typed.Tok == token.TYPE && declaration.Name != nil {
					out = append(out, independentSyntheticDescriptor{
						name: declaration.Name.Name,
						role: identity.SyntheticDefinitionType,
						node: declaration,
					})
				}
			case *ast.ValueSpec:
				if typed.Tok != token.VAR && typed.Tok != token.CONST {
					continue
				}
				for nameIndex, name := range declaration.Names {
					if name.Name != "" && name.Name != "_" {
						out = append(out, independentSyntheticDescriptor{
							name:      name.Name,
							role:      identity.SyntheticDefinitionData,
							node:      declaration,
							nameIndex: nameIndex,
						})
					}
				}
			}
		}
	}
	return out
}
