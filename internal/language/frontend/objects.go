package frontend

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

type bindingCandidate struct {
	object     types.Object
	source     identity.OccurrenceID
	scope      identity.OccurrenceID
	role       identity.SemanticBindingRole
	ordinal    int
	definition identity.DefinitionID
	typ        types.Type
	name       string
}

type objectIndex struct {
	input                   *packageInput
	contexts                *contextIndex
	packageByPath           map[string]identity.PackageID
	predeclared             map[types.Object]catalog.PredeclaredKind
	sourceByObject          map[types.Object]identity.OccurrenceID
	objectBySource          map[identity.OccurrenceID]types.Object
	definitionByObject      map[types.Object]identity.DefinitionID
	scopeOwners             map[*types.Scope]identity.OccurrenceID
	memberOwnerRelations    map[types.Object][]types.Type
	bindingByObject         map[types.Object]*bindingCandidate
	bindingBySource         map[identity.OccurrenceID]*bindingCandidate
	declarationIDs          map[types.Object]identity.SemanticDeclarationID
	declarationByID         map[identity.SemanticDeclarationID]types.Object
	declarationOwnerPackage map[identity.SemanticDeclarationID]identity.PackageID
	localOrdinals           map[types.Object]int
	bindingIDs              map[*bindingCandidate]identity.SemanticBindingID
	typeParameterOwners     map[*types.TypeParam]semantic.TypeParameterOwner
	typeParameterByLocation map[typeParameterLocation]semantic.TypeParameterOwner
	typeBuilder             *typeBuilder
	memberVisits            map[memberVisit]bool
	scopeByOccurrence       map[identity.OccurrenceID]identity.OccurrenceID
	scopeOccurrenceResolved map[identity.OccurrenceID]bool
	checkerScopeOwner       map[*types.Scope]identity.OccurrenceID
	checkerScopeResolved    map[*types.Scope]bool
	work                    *Work
}

type memberVisit struct {
	typ     types.Type
	nominal types.Type
}

func buildObjectIndex(
	stage *stageInput,
	input *packageInput,
	contexts *contextIndex,
	work *Work,
) (*objectIndex, error) {
	out := &objectIndex{
		input: input, contexts: contexts,
		packageByPath:           map[string]identity.PackageID{},
		predeclared:             predeclaredObjects(),
		sourceByObject:          map[types.Object]identity.OccurrenceID{},
		objectBySource:          map[identity.OccurrenceID]types.Object{},
		definitionByObject:      map[types.Object]identity.DefinitionID{},
		scopeOwners:             map[*types.Scope]identity.OccurrenceID{},
		memberOwnerRelations:    map[types.Object][]types.Type{},
		bindingByObject:         map[types.Object]*bindingCandidate{},
		bindingBySource:         map[identity.OccurrenceID]*bindingCandidate{},
		declarationIDs:          map[types.Object]identity.SemanticDeclarationID{},
		declarationByID:         map[identity.SemanticDeclarationID]types.Object{},
		declarationOwnerPackage: map[identity.SemanticDeclarationID]identity.PackageID{},
		localOrdinals:           map[types.Object]int{},
		bindingIDs:              map[*bindingCandidate]identity.SemanticBindingID{},
		typeParameterOwners:     map[*types.TypeParam]semantic.TypeParameterOwner{},
		typeParameterByLocation: map[typeParameterLocation]semantic.TypeParameterOwner{},
		memberVisits:            map[memberVisit]bool{},
		scopeByOccurrence:       map[identity.OccurrenceID]identity.OccurrenceID{},
		scopeOccurrenceResolved: map[identity.OccurrenceID]bool{},
		checkerScopeOwner:       map[*types.Scope]identity.OccurrenceID{},
		checkerScopeResolved:    map[*types.Scope]bool{},
		work:                    work,
	}
	for _, loaded := range stage.universe.Packages() {
		path := loaded.ID().ImportPath()
		if existing := out.packageByPath[path]; !existing.IsZero() &&
			existing != loaded.ID() {
			return nil, fmt.Errorf(
				"checker import path %s has package identities %s and %s",
				path, existing, loaded.ID(),
			)
		}
		out.packageByPath[path] = loaded.ID()
	}
	view := input.loaded.CheckerView()
	if view == nil &&
		input.provenance == semantic.ProvenanceLanguagePseudo &&
		input.id.ImportPath() == "builtin" {
		newTypeBuilder(out)
		return out, nil
	}
	if input.loaded.Types() == nil {
		return nil, fmt.Errorf(
			"local semantic package %s has no transient checker",
			input.id,
		)
	}
	if view == nil {
		return nil, fmt.Errorf(
			"local semantic package %s has no transient checker",
			input.id,
		)
	}
	scope := input.loaded.Types().Scope()
	for _, name := range scope.Names() {
		object := scope.Lookup(name)
		if object != nil {
			out.indexMembers(object.Type(), nil)
		}
	}
	for _, occurrenceID := range input.order {
		work.ObjectOccurrenceVisits++
		record := input.occurrences[occurrenceID]
		if selector, ok := record.node.(*ast.SelectorExpr); ok {
			if selection, present := view.SelectionOf(selector); present {
				out.indexMembers(
					selection.Recv(), nil,
				)
			}
		}
		if scope, present := view.ScopeOf(record.node); present {
			if existing := out.scopeOwners[scope]; !existing.IsZero() &&
				existing != occurrenceID {
				return nil, fmt.Errorf(
					"checker scope is owned by occurrences %s and %s",
					existing, occurrenceID,
				)
			}
			out.scopeOwners[scope] = occurrenceID
		}
		if identifier, ok := record.node.(*ast.Ident); ok {
			if object, present := view.DefOf(identifier); present &&
				object != nil {
				variable, field := object.(*types.Var)
				if identifier.Name != "_" ||
					(field && variable.IsField()) {
					if err := out.bindObjectSource(
						object, occurrenceID,
					); err != nil {
						return nil, err
					}
				}
			} else if identifier.Name != "_" {
				if object, present := view.UseOf(identifier); present &&
					object != nil &&
					!packageObject(object) &&
					!packageNameObject(object) &&
					object.Pkg() == input.loaded.Types() &&
					object.Pos().IsValid() {
					object = out.canonicalDeclarationObject(object)
					if out.sourceByObject[object].IsZero() {
						source, err := input.index.IdentifierOccurrence(
							object.Pos(), object.Name(),
						)
						if err != nil {
							return nil, err
						}
						if err := out.bindObjectSource(
							object, source,
						); err != nil {
							return nil, err
						}
					}
				}
			}
		}
		if object, present := view.ImplicitOf(record.node); present &&
			object != nil {
			if variable, field := object.(*types.Var); field && variable.IsField() {
				if err := out.bindObjectSource(
					object, occurrenceID,
				); err != nil {
					return nil, err
				}
			}
		}
		if expression, ok := record.node.(ast.Expr); ok {
			if value, present := view.TypeOf(expression); present {
				out.indexMembers(value.Type, nil)
			}
		}
	}
	if err := out.bindCheckedDefinitionSources(); err != nil {
		return nil, err
	}
	if err := out.bindIntrinsicDefinitionSources(); err != nil {
		return nil, err
	}
	if err := out.bindSyntheticDefinitions(); err != nil {
		return nil, err
	}
	for object, source := range out.sourceByObject {
		record := input.occurrences[source]
		if record != nil && !record.owner.IsZero() {
			if err := out.bindObjectDefinition(
				object, record.owner,
			); err != nil {
				return nil, err
			}
		}
	}
	if err := out.createBindingCandidates(); err != nil {
		return nil, err
	}
	if err := out.assignBindingIdentities(); err != nil {
		return nil, err
	}
	if err := out.assignLocalDeclarationOrdinals(); err != nil {
		return nil, err
	}
	newTypeBuilder(out)
	if err := out.indexPackageTypeParameterOwners(
		input.loaded.Types(),
	); err != nil {
		return nil, err
	}
	return out, nil
}

func (index *objectIndex) bindCheckedDefinitionSources() error {
	view := index.input.loaded.CheckerView()
	for definition := range index.input.definitions {
		checked, present := index.input.index.
			CheckedDefinitionNode(definition)
		if !present || definition.Root().IsZero() {
			continue
		}
		objects := checkedDeclarationObjects(view, checked)
		sources := definitionNameSources(index.input, definition)
		if len(objects) != len(sources) {
			return fmt.Errorf(
				"checked definition %s has %d objects and %d source names",
				definition, len(objects), len(sources),
			)
		}
		for ordinal, object := range objects {
			if object == nil {
				return fmt.Errorf(
					"checked definition %s name %d has no checker object",
					definition, ordinal,
				)
			}
			if object.Name() == "_" {
				continue
			}
			if err := index.bindObjectSource(
				object, sources[ordinal],
			); err != nil {
				return err
			}
			if err := index.bindObjectDefinition(
				object, definition,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func (index *objectIndex) bindSyntheticDefinitions() error {
	view := index.input.loaded.CheckerView()
	for definition := range index.input.definitions {
		node, present := index.input.index.
			SyntheticDefinitionNode(definition)
		if !present {
			continue
		}
		object := declarationObject(
			view, node, definition.SyntheticName(),
		)
		if object == nil {
			return fmt.Errorf(
				"synthetic definition %s has no checker object",
				definition,
			)
		}
		if err := index.bindObjectDefinition(
			object, definition,
		); err != nil {
			return err
		}
	}
	return nil
}

func (index *objectIndex) bindObjectDefinition(
	object types.Object,
	definition identity.DefinitionID,
) error {
	if object == nil || definition.IsZero() {
		return fmt.Errorf(
			"object-definition relation requires both identities",
		)
	}
	if existing := index.definitionByObject[object]; !existing.IsZero() &&
		existing != definition {
		return fmt.Errorf(
			"checker object %s belongs to definitions %s and %s",
			object.Name(), existing, definition,
		)
	}
	index.definitionByObject[object] = definition
	return nil
}

func packageNameObject(object types.Object) bool {
	_, packageName := object.(*types.PkgName)
	return packageName
}

func checkedDeclarationObjects(
	view checkerExpressionView,
	node ast.Node,
) []types.Object {
	switch node := node.(type) {
	case *ast.FuncDecl:
		object, _ := view.DefOf(node.Name)
		return []types.Object{object}
	case *ast.TypeSpec:
		object, _ := view.DefOf(node.Name)
		return []types.Object{object}
	case *ast.ValueSpec:
		out := make([]types.Object, 0, len(node.Names))
		for _, name := range node.Names {
			object, _ := view.DefOf(name)
			out = append(out, object)
		}
		return out
	default:
		return nil
	}
}

func definitionNameSources(
	input *packageInput,
	definition identity.DefinitionID,
) []identity.OccurrenceID {
	root := input.occurrences[definition.Root()]
	if root == nil {
		return nil
	}
	var out []identity.OccurrenceID
	for _, childID := range root.children {
		child := input.occurrences[childID]
		if child != nil &&
			child.occurrence.Role() ==
				catalog.RoleDeclarationName {
			out = append(out, childID)
		}
	}
	return out
}

func (index *objectIndex) bindObjectSource(
	object types.Object,
	source identity.OccurrenceID,
) error {
	if existing, present := index.sourceByObject[object]; present &&
		existing != source {
		return fmt.Errorf(
			"checker object %s has defining occurrences %s and %s",
			object.Name(), existing, source,
		)
	}
	if existing := index.objectBySource[source]; existing != nil &&
		existing != object {
		return fmt.Errorf(
			"occurrence %s defines checker objects %s and %s",
			source, existing.Name(), object.Name(),
		)
	}
	index.sourceByObject[object] = source
	index.objectBySource[source] = object
	return nil
}
