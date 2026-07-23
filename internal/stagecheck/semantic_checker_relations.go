package stagecheck

import (
	"fmt"
	"go/ast"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/source"
)

func (verifier *checkerSemanticVerifier) operationOperands(
	occurrence structure.Occurrence,
) ([]identity.OccurrenceID, error) {
	children := append(
		[]identity.OccurrenceID(nil),
		verifier.children[occurrence.ID()]...,
	)
	edgeRank := map[catalog.Edge]int{}
	for index, edge := range catalog.EdgesOf(occurrence.Kind()) {
		edgeRank[edge] = index
	}
	sort.Slice(children, func(left, right int) bool {
		leftRecord := verifier.expected.occurrences[children[left]]
		rightRecord := verifier.expected.occurrences[children[right]]
		if edgeRank[leftRecord.Edge()] !=
			edgeRank[rightRecord.Edge()] {
			return edgeRank[leftRecord.Edge()] <
				edgeRank[rightRecord.Edge()]
		}
		return leftRecord.Ordinal() < rightRecord.Ordinal()
	})
	var ranks map[catalog.Role]int
	switch occurrence.Kind() {
	case catalog.KindForStmt:
		ranks = map[catalog.Role]int{
			catalog.RoleInitStatement: 0,
			catalog.RoleCondition:     1,
			catalog.RoleBody:          2,
			catalog.RolePostStatement: 3,
		}
	case catalog.KindRangeStmt:
		ranks = map[catalog.Role]int{
			catalog.RoleRangeOperand: 0,
			catalog.RoleRangeKey:     1,
			catalog.RoleRangeValue:   2,
			catalog.RoleBody:         3,
		}
	}
	if ranks != nil {
		sort.SliceStable(children, func(left, right int) bool {
			leftRole := verifier.expected.occurrences[children[left]].Role()
			rightRole := verifier.expected.occurrences[children[right]].Role()
			return ranks[leftRole] < ranks[rightRole]
		})
	}
	out := make([]identity.OccurrenceID, 0, len(children))
	for _, childID := range children {
		child, present := verifier.expected.occurrences[childID]
		if !present {
			continue
		}
		if verifier.runtimeOperand(child) {
			out = append(out, childID)
		}
	}
	return out, nil
}

func (verifier *checkerSemanticVerifier) runtimeOperand(
	occurrence structure.Occurrence,
) bool {
	switch occurrence.Role() {
	case catalog.RoleDocumentation,
		catalog.RoleTrailingDocumentation,
		catalog.RoleCommentText,
		catalog.RolePackageName,
		catalog.RoleDeclaration,
		catalog.RoleDeclarationName,
		catalog.RoleTypeExpression,
		catalog.RoleFieldTag,
		catalog.RoleFieldGroup,
		catalog.RoleConstructedType,
		catalog.RoleSelectedName,
		catalog.RoleAssertedType,
		catalog.RoleArrayLength,
		catalog.RoleElementType,
		catalog.RoleStructFields,
		catalog.RoleTypeParameters,
		catalog.RoleParameters,
		catalog.RoleResults,
		catalog.RoleInterfaceMethods,
		catalog.RoleKeyType,
		catalog.RoleValueType,
		catalog.RoleLabelDeclaration,
		catalog.RoleLabelReference,
		catalog.RoleImportAlias,
		catalog.RoleImportPath,
		catalog.RoleReceiver,
		catalog.RoleFunctionSignature,
		catalog.RoleFunctionBody,
		catalog.RoleSpecification:
		return false
	case catalog.RoleElementKey:
		parent := verifier.resolutions[occurrence.Parent()]
		return parent.Variant() != catalog.VariantKeyFieldName
	default:
		return true
	}
}

func (verifier *checkerSemanticVerifier) operationDefinitions(
	operation semantic.Operation,
) []identity.DefinitionID {
	region, present := verifier.expected.regions[operation.Definition()]
	if !present {
		return nil
	}
	type ordered struct {
		ordinal int
		id      identity.DefinitionID
	}
	var records []ordered
	for _, reference := range region.References() {
		if reference.Parent() == operation.Occurrence() {
			records = append(records, ordered{
				ordinal: reference.Ordinal(),
				id:      reference.Child(),
			})
		}
	}
	sort.Slice(records, func(left, right int) bool {
		return records[left].ordinal < records[right].ordinal
	})
	out := make([]identity.DefinitionID, 0, len(records))
	for _, record := range records {
		out = append(out, record.id)
	}
	return out
}

func (verifier *checkerSemanticVerifier) verifyResolutionTarget(
	occurrence structure.Occurrence,
	resolution semantic.OccurrenceResolution,
	node ast.Node,
) error {
	switch resolution.Kind() {
	case semantic.ResolutionDeclaration:
		if occurrence.Role() == catalog.RoleSelectedName {
			if handled, err := verifier.verifySelectedNameResolution(
				occurrence, resolution,
			); handled {
				return err
			}
		}
		return verifier.verifyObjectReference(
			mustDeclarationReference(resolution.Declaration()),
			independentResolutionObject(
				verifier.view, occurrence, node,
			),
		)
	case semantic.ResolutionBinding:
		return verifier.verifyObjectReference(
			mustBindingReference(resolution.Binding()),
			independentResolutionObject(
				verifier.view, occurrence, node,
			),
		)
	case semantic.ResolutionType:
		typ := independentNodeType(verifier.view, node)
		if typ == nil &&
			occurrence.Kind() == catalog.KindFuncType &&
			occurrence.Role() == catalog.RoleFunctionSignature {
			typ = independentDefinitionSignature(
				verifier.expected, verifier.index, occurrence,
			)
		}
		return verifier.types.verify(resolution.Type(), typ)
	case semantic.ResolutionStructuralOnly:
		evidence := resolution.Structural()
		if evidence.Disposition() ==
			semantic.StructuralCompileTimeExpression {
			object, typ := verifier.independentStructuralCoverage(
				occurrence,
			)
			if !evidence.Declaration().IsZero() {
				return verifier.verifyObjectReference(
					mustDeclarationReference(
						evidence.Declaration(),
					),
					object,
				)
			}
			return verifier.types.verify(evidence.Type(), typ)
		}
	}
	return nil
}

func independentResolutionObject(
	view *source.TypeInfoView,
	occurrence structure.Occurrence,
	node ast.Node,
) types.Object {
	identifier, ok := node.(*ast.Ident)
	if !ok {
		return independentCheckerObject(view, node)
	}
	defined, hasDefinition := view.DefOf(identifier)
	used, hasUse := view.UseOf(identifier)
	if hasDefinition && hasUse {
		field, fieldDefinition := defined.(*types.Var)
		_, typeUse := used.(*types.TypeName)
		if fieldDefinition && field.IsField() && typeUse {
			return used
		}
		definedType, definedTypeName := defined.(*types.TypeName)
		usedType, usedTypeName := used.(*types.TypeName)
		if definedTypeName && usedTypeName &&
			occurrence.Role() == catalog.RoleIndex {
			_, definedParameter := definedType.Type().(*types.TypeParam)
			_, usedParameter := usedType.Type().(*types.TypeParam)
			if definedParameter && usedParameter {
				return defined
			}
		}
		return nil
	}
	if hasDefinition {
		return defined
	}
	if hasUse {
		return used
	}
	return nil
}

func (verifier *checkerSemanticVerifier) independentStructuralCoverage(
	occurrence structure.Occurrence,
) (types.Object, types.Type) {
	if verifier.expected.domains[occurrence.ID()] ==
		catalog.ResolutionDomainHeader {
		definition := verifier.expected.owners[occurrence.ID()]
		node, present := verifier.index.CheckedDefinitionNode(definition)
		if !present {
			node, present = verifier.index.DefinitionNode(definition)
		}
		if present {
			switch node := node.(type) {
			case *ast.FuncDecl:
				object, _ := verifier.view.DefOf(node.Name)
				if object != nil {
					return object, nil
				}
			case *ast.TypeSpec:
				object, _ := verifier.view.DefOf(node.Name)
				if object != nil {
					return object, nil
				}
			case *ast.FuncLit:
				return nil, independentExpressionType(
					verifier.view, node,
				)
			}
		}
	}
	current := occurrence
	var typeCoverage types.Type
	for !current.Parent().IsZero() {
		parent := verifier.expected.occurrences[current.Parent()]
		node, present := verifier.index.OccurrenceNode(parent.ID())
		if !present {
			return nil, nil
		}
		if expression, ok := node.(ast.Expr); ok {
			if value, present := verifier.view.TypeOf(expression); present &&
				value.IsType() &&
				typeCoverage == nil {
				typeCoverage = value.Type
			}
		}
		switch node := node.(type) {
		case *ast.FuncDecl:
			if current.Role() == catalog.RoleFunctionSignature {
				object, _ := verifier.view.DefOf(node.Name)
				if object != nil {
					return object, nil
				}
			}
		case *ast.FuncLit:
			if current.Role() == catalog.RoleFunctionSignature {
				if typeCoverage == nil {
					typeCoverage = independentExpressionType(
						verifier.view, node,
					)
				}
			}
		case *ast.ValueSpec:
			switch current.Role() {
			case catalog.RoleTypeExpression:
				if typeCoverage == nil {
					typeCoverage = independentExpressionType(
						verifier.view, node.Type,
					)
				}
			case catalog.RoleInitializerValue:
				if current.Ordinal() < len(node.Names) {
					object, _ := verifier.view.DefOf(
						node.Names[current.Ordinal()],
					)
					if object != nil {
						return object, nil
					}
				}
			}
		case *ast.TypeSpec:
			if current.Role() == catalog.RoleTypeExpression ||
				current.Role() == catalog.RoleTypeParameters {
				object, _ := verifier.view.DefOf(node.Name)
				if object != nil {
					return object, nil
				}
				if typeCoverage == nil {
					typeCoverage = independentExpressionType(
						verifier.view, node.Type,
					)
				}
			}
		}
		current = parent
	}
	return nil, typeCoverage
}

func (verifier *checkerSemanticVerifier) verifySelectedNameResolution(
	occurrence structure.Occurrence,
	resolution semantic.OccurrenceResolution,
) (bool, error) {
	parentNode, present := verifier.index.OccurrenceNode(
		occurrence.Parent(),
	)
	selector, selectorNode := parentNode.(*ast.SelectorExpr)
	if !present || !selectorNode {
		return false, nil
	}
	checker, present := verifier.view.SelectionOf(selector)
	if !present {
		return false, nil
	}
	parentResolution := verifier.resolutions[occurrence.Parent()]
	operation := verifier.operations[parentResolution.Operation()]
	selection := operation.Spec().Selection
	if !selection.IsZero() {
		if selection.Object() != resolution.Declaration() {
			return true, fmt.Errorf(
				"selected name %s differs from parent selection",
				occurrence.ID(),
			)
		}
		return true, verifier.verifySelectionDeclaration(
			selection, checker,
		)
	}
	return true, verifier.verifyCheckerSelectionDeclaration(
		resolution.Declaration(), checker,
	)
}

func mustDeclarationReference(
	id identity.SemanticDeclarationID,
) semantic.ObjectReference {
	reference, _ := semantic.DeclarationReference(id)
	return reference
}

func mustBindingReference(
	id identity.SemanticBindingID,
) semantic.ObjectReference {
	reference, _ := semantic.BindingReference(id)
	return reference
}

func (verifier *checkerSemanticVerifier) verifyObjectReference(
	reference semantic.ObjectReference,
	object types.Object,
) error {
	if object == nil {
		return fmt.Errorf("semantic reference has no checker object")
	}
	switch reference.Kind() {
	case semantic.ObjectReferenceDeclaration:
		if function, ok := object.(*types.Func); ok {
			object = function.Origin()
		}
		record, present := verifier.declarations[reference.Declaration()]
		if !present {
			return verifier.verifyDeclarationReferenceIdentity(
				reference.Declaration(), object,
			)
		}
		class := independentObjectClass(object)
		if record.Name() != object.Name() ||
			record.Class() != class ||
			record.Exported() != object.Exported() {
			return fmt.Errorf(
				"declaration %s metadata differs from %T %s",
				record.ID(), object, object.Name(),
			)
		}
		if err := verifier.verifyDeclarationIdentity(
			record, object,
		); err != nil {
			return err
		}
		if _, builtin := object.(*types.Builtin); builtin {
			if !record.Type().IsZero() {
				return fmt.Errorf(
					"builtin declaration %s has ordinary type",
					record.ID(),
				)
			}
		} else if err := verifier.types.verify(
			record.Type(), object.Type(),
		); err != nil {
			return err
		}
		if value, constantObject := object.(*types.Const); constantObject {
			if record.Constant().Exact() !=
				value.Val().ExactString() ||
				record.Constant().Kind() !=
					semanticConstantKind(value.Val().Kind()) {
				return fmt.Errorf(
					"declaration %s constant differs", record.ID(),
				)
			}
		}
	case semantic.ObjectReferenceBinding:
		record, present := verifier.bindings[reference.Binding()]
		if !present {
			return fmt.Errorf(
				"binding %s is absent", reference.Binding(),
			)
		}
		expected := verifier.bindingByObject[object]
		if expected.IsZero() || expected != reference.Binding() {
			return fmt.Errorf(
				"binding reference %s differs from checker-derived %s",
				reference.Binding(), expected,
			)
		}
		if record.Name() != object.Name() {
			return fmt.Errorf(
				"binding %s name differs from %s",
				record.ID(), object.Name(),
			)
		}
		switch object.(type) {
		case *types.PkgName, *types.Label:
			if !record.Type().IsZero() {
				return fmt.Errorf(
					"typeless binding %s carries a type", record.ID(),
				)
			}
		default:
			if err := verifier.types.verify(
				record.Type(), object.Type(),
			); err != nil {
				return err
			}
		}
		if !record.Source().IsZero() {
			node, present := verifier.index.OccurrenceNode(
				record.Source(),
			)
			identifier, ok := node.(*ast.Ident)
			if !present || !ok {
				return fmt.Errorf(
					"binding %s source is not an identifier",
					record.ID(),
				)
			}
			defined, _ := verifier.view.DefOf(identifier)
			if defined != object {
				return fmt.Errorf(
					"binding %s source defines a different object",
					record.ID(),
				)
			}
		}
	default:
		return fmt.Errorf(
			"checker object %T has no semantic reference", object,
		)
	}
	return nil
}

func (verifier *checkerSemanticVerifier) verifyDeclarationIdentity(
	record semantic.Declaration,
	object types.Object,
) error {
	if object.Pkg() == nil {
		if predeclared := independentPredeclaredKind(object); predeclared.Valid() {
			class := independentPredeclaredClass(predeclared.Class())
			expected, err := identity.NewPredeclaredDeclarationID(
				uint16(predeclared),
				class,
			)
			if err != nil || expected != record.ID() {
				return fmt.Errorf(
					"predeclared identity differs for %s", object.Name(),
				)
			}
		}
		return nil
	}
	if expected := verifier.types.localDeclarations[object]; !expected.IsZero() {
		if record.ID() != expected {
			return fmt.Errorf(
				"local declaration identity differs for %s: semantic=%s checker=%s",
				object.Name(), record.ID(), expected,
			)
		}
		return nil
	}
	pkg := verifier.types.packageByPath[object.Pkg().Path()]
	if pkg.IsZero() || record.Package() != pkg {
		return fmt.Errorf(
			"declaration %s package differs from %s",
			record.ID(), object.Pkg().Path(),
		)
	}
	switch object.(type) {
	case *types.Const, *types.TypeName, *types.Func, *types.Var:
		expected, err := identity.NewPackageDeclarationID(
			pkg, independentObjectClass(object), object.Name(),
		)
		if record.ID().Form() ==
			identity.SemanticDeclarationPackageObject &&
			(err != nil || record.ID() != expected) {
			return fmt.Errorf(
				"package declaration identity differs for %s",
				object.Name(),
			)
		}
		if record.ID().Form() ==
			identity.SemanticDeclarationPackageObject {
			return nil
		}
		if record.ID().Name() != object.Name() ||
			record.ID().Class() != independentObjectClass(object) {
			return fmt.Errorf(
				"member/local declaration identity differs for %s",
				object.Name(),
			)
		}
	}
	return nil
}
