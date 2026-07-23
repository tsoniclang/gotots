package frontend

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

type pendingOperation struct {
	record  *occurrenceInput
	context occurrenceContext
	variant catalog.Variant
	kind    semantic.OperationKind
}

func (builder *packageBuilder) resolveOccurrences() error {
	pending := make([]pendingOperation, 0)
	for _, occurrenceID := range builder.input.order {
		record := builder.input.occurrences[occurrenceID]
		context := builder.contexts.byOccurrence[occurrenceID]
		variant := catalog.VariantNone
		var err error
		if !record.checkedUnmapped &&
			!builder.intrinsicHeader(record) &&
			(!structuralCompileTime(record, context) ||
				hasDirectCheckerEvidence(
					builder.input.loaded.CheckerView(), record.node,
				)) {
			variant, err = resolveVariant(
				builder.input, record, context,
			)
			if err != nil {
				return err
			}
		}
		builder.variantByOccurrence[occurrenceID] = variant
		resolution, operation, err := builder.classifyOccurrence(
			record, context, variant,
		)
		if err != nil {
			return err
		}
		if operation != semantic.OperationInvalid {
			operationID, err := identity.NewOperationID(
				record.owner, occurrenceID,
			)
			if err != nil {
				return err
			}
			builder.operationByOccurrence[occurrenceID] = operationID
			pending = append(pending, pendingOperation{
				record: record, context: context,
				variant: variant, kind: operation,
			})
			continue
		}
		if err := builder.admitResolution(resolution); err != nil {
			return err
		}
	}
	for _, item := range pending {
		operation, err := builder.buildOperation(item)
		if err != nil {
			return err
		}
		builder.operations = append(builder.operations, operation)
		resolution, err := semantic.NewOccurrenceResolution(
			builder.resolutionSpec(
				item.record, item.variant,
				semantic.ResolutionOperation,
			).withOperation(operation.ID()),
		)
		if err != nil {
			return err
		}
		if err := builder.admitResolution(resolution); err != nil {
			return err
		}
	}
	if len(builder.resolutions) != len(builder.input.occurrences) {
		return fmt.Errorf(
			"semantic package %s resolved %d of %d occurrences",
			builder.input.id,
			len(builder.resolutions),
			len(builder.input.occurrences),
		)
	}
	return builder.buildImplicitOperations()
}

func hasDirectCheckerEvidence(
	view checkerExpressionView,
	node ast.Node,
) bool {
	if view == nil {
		return false
	}
	if expression, ok := node.(ast.Expr); ok {
		if _, present := view.TypeOf(expression); present {
			return true
		}
	}
	switch node := node.(type) {
	case *ast.Ident:
		_, defined := view.DefOf(node)
		_, used := view.UseOf(node)
		return defined || used
	case *ast.SelectorExpr:
		if _, present := view.SelectionOf(node); present {
			return true
		}
		return false
	case *ast.TypeSpec:
		_, present := view.DefOf(node.Name)
		return present
	}
	return false
}

func (builder *packageBuilder) classifyOccurrence(
	record *occurrenceInput,
	context occurrenceContext,
	variant catalog.Variant,
) (semantic.OccurrenceResolution, semantic.OperationKind, error) {
	if record.checkedUnmapped {
		return builder.checkedViewUnsupported(record, variant)
	}
	if definition := builder.definitionByRoot[record.occurrence.ID()]; !definition.IsZero() {
		return builder.definitionComponent(
			record, variant, definition,
			semantic.DefinitionComponentRoot,
		)
	}
	if record.domain == catalog.ResolutionDomainBoundary {
		return builder.definitionComponent(
			record, variant, record.owner,
			boundaryComponent(record.owner),
		)
	}
	if record.occurrence.Kind().Disposition() !=
		catalog.DispositionActive {
		return builder.unsupportedResolution(record, variant)
	}
	if declarationNameIsDefinitionComponent(record) {
		return builder.definitionComponent(
			record,
			variant,
			record.owner,
			semantic.DefinitionComponentName,
		)
	}
	if builder.intrinsicHeader(record) {
		return builder.intrinsicHeaderResolution(record)
	}
	if resolution, present, err := builder.objectResolution(
		record, context, variant,
	); present || err != nil {
		return resolution, semantic.OperationInvalid, err
	}
	if typeID, present, err := builder.typeResolution(
		record, context,
	); present || err != nil {
		if err != nil {
			return semantic.OccurrenceResolution{},
				semantic.OperationInvalid, err
		}
		resolution, err := semantic.NewOccurrenceResolution(
			builder.resolutionSpec(
				record, variant, semantic.ResolutionType,
			).withType(typeID),
		)
		return resolution, semantic.OperationInvalid, err
	}
	if record.occurrence.Kind() == catalog.KindFuncLit {
		return builder.definitionComponent(
			record, variant, record.owner,
			semantic.DefinitionComponentRoot,
		)
	}
	if record.domain == catalog.ResolutionDomainExecutable {
		kind, err := operationKind(
			builder.input.loaded.CheckerView(),
			record,
			variant,
		)
		if err != nil {
			return semantic.OccurrenceResolution{},
				semantic.OperationInvalid, err
		}
		if kind.Valid() {
			return semantic.OccurrenceResolution{}, kind, nil
		}
	}
	structural, err := builder.structuralResolution(
		record, context, variant,
	)
	return structural, semantic.OperationInvalid, err
}

func (builder *packageBuilder) intrinsicHeader(
	record *occurrenceInput,
) bool {
	if record == nil ||
		record.domain != catalog.ResolutionDomainHeader ||
		builder.input.id.ImportPath() != "unsafe" {
		return false
	}
	definition, present := builder.input.definitions[record.owner]
	if !present {
		return false
	}
	member := catalog.UnsafeMemberByName(definition.Name())
	return member.Valid() &&
		member.Class() == catalog.UnsafeMemberClassBuiltin
}

func (builder *packageBuilder) intrinsicHeaderResolution(
	record *occurrenceInput,
) (
	semantic.OccurrenceResolution,
	semantic.OperationKind,
	error,
) {
	evidence, err := semantic.NewStructuralEvidence(
		semantic.StructuralIntrinsicContract,
		identity.SemanticDeclarationID{},
		identity.SemanticTypeID{},
	)
	if err != nil {
		return semantic.OccurrenceResolution{},
			semantic.OperationInvalid, err
	}
	resolution, err := semantic.NewOccurrenceResolution(
		builder.resolutionSpec(
			record,
			catalog.VariantNone,
			semantic.ResolutionStructuralOnly,
		).withStructural(evidence),
	)
	return resolution, semantic.OperationInvalid, err
}

func declarationNameIsDefinitionComponent(
	record *occurrenceInput,
) bool {
	identifier, ok := record.node.(*ast.Ident)
	return ok &&
		identifier.Name == "init" &&
		record.occurrence.Role() == catalog.RoleDeclarationName &&
		record.owner.Kind() == identity.DefinitionFuncDecl
}

func (builder *packageBuilder) checkedViewUnsupported(
	record *occurrenceInput,
	variant catalog.Variant,
) (semantic.OccurrenceResolution, semantic.OperationKind, error) {
	id, err := identity.NewUnsupportedID(
		record.owner, record.occurrence.ID(),
	)
	if err != nil {
		return semantic.OccurrenceResolution{},
			semantic.OperationInvalid, err
	}
	unsupported, err := semantic.NewUnsupported(
		id,
		semantic.UnsupportedCheckedViewTransform,
		"checked-view transformation has no same-kind edge/ordinal counterpart",
		builder.input.authority,
	)
	if err != nil {
		return semantic.OccurrenceResolution{},
			semantic.OperationInvalid, err
	}
	builder.unsupported = append(builder.unsupported, unsupported)
	resolution, err := semantic.NewOccurrenceResolution(
		builder.resolutionSpec(
			record, variant, semantic.ResolutionUnsupported,
		).withUnsupported(id),
	)
	return resolution, semantic.OperationInvalid, err
}

func (builder *packageBuilder) objectResolution(
	record *occurrenceInput,
	context occurrenceContext,
	variant catalog.Variant,
) (semantic.OccurrenceResolution, bool, error) {
	var object types.Object
	switch node := record.node.(type) {
	case *ast.Ident:
		if node.Name == "_" && catalog.AllowsResolution(
			record.occurrence.Kind(),
			record.occurrence.Role(),
			variant,
			record.domain,
			catalog.ResolutionClassStructural,
		) {
			structural, err := semantic.NewStructuralEvidence(
				semantic.StructuralBlankIdentifier,
				identity.SemanticDeclarationID{},
				identity.SemanticTypeID{},
			)
			if err != nil {
				return semantic.OccurrenceResolution{}, true, err
			}
			resolution, err := semantic.NewOccurrenceResolution(
				builder.resolutionSpec(
					record, variant,
					semantic.ResolutionStructuralOnly,
				).withStructural(structural),
			)
			return resolution, true, err
		}
		var err error
		object, err = resolutionObject(
			builder.input.loaded.CheckerView(),
			record,
			context.selectedObject,
		)
		if err != nil {
			return semantic.OccurrenceResolution{}, true, err
		}
	case *ast.TypeSpec:
		object, _ = builder.input.loaded.CheckerView().DefOf(node.Name)
	}
	if object == nil {
		return semantic.OccurrenceResolution{}, false, nil
	}
	if record.domain == catalog.ResolutionDomainExecutable &&
		identifierExecutes(
			builder.input.loaded.CheckerView(), record, object,
		) {
		return semantic.OccurrenceResolution{}, false, nil
	}
	if _, isType := object.(*types.TypeName); isType &&
		identifierIsType(
			builder.input.loaded.CheckerView(), record,
		) {
		typeID, err := builder.types.build(object.Type())
		if err != nil {
			return semantic.OccurrenceResolution{}, true, err
		}
		resolution, err := semantic.NewOccurrenceResolution(
			builder.resolutionSpec(
				record, variant, semantic.ResolutionType,
			).withType(typeID),
		)
		return resolution, true, err
	}
	if binding, present := builder.objects.bindingID(object); present {
		resolution, err := semantic.NewOccurrenceResolution(
			builder.resolutionSpec(
				record, variant, semantic.ResolutionBinding,
			).withBinding(binding),
		)
		return resolution, true, err
	}
	declaration, err := builder.objects.declarationID(object)
	if err != nil {
		return semantic.OccurrenceResolution{}, true, err
	}
	resolution, err := semantic.NewOccurrenceResolution(
		builder.resolutionSpec(
			record, variant, semantic.ResolutionDeclaration,
		).withDeclaration(declaration),
	)
	return resolution, true, err
}

func resolutionObject(
	view checkerExpressionView,
	record *occurrenceInput,
	selected types.Object,
) (types.Object, error) {
	identifier, ok := record.node.(*ast.Ident)
	if !ok {
		return nil, nil
	}
	if selected != nil {
		if record.occurrence.Role() != catalog.RoleSelectedName {
			return nil, fmt.Errorf(
				"selected object leaked to %s role=%s at %s",
				record.occurrence.Kind(),
				record.occurrence.Role(),
				record.occurrence.ID(),
			)
		}
		return selected, nil
	}
	defined, hasDefinition := view.DefOf(identifier)
	used, hasUse := view.UseOf(identifier)
	if hasDefinition && hasUse {
		field, fieldDefinition := defined.(*types.Var)
		_, typeUse := used.(*types.TypeName)
		if fieldDefinition &&
			field.IsField() &&
			typeUse {
			return used, nil
		}
		definedType, definedTypeName := defined.(*types.TypeName)
		usedType, usedTypeName := used.(*types.TypeName)
		if definedTypeName && usedTypeName {
			_, definedParameter :=
				definedType.Type().(*types.TypeParam)
			_, usedParameter :=
				usedType.Type().(*types.TypeParam)
			if definedParameter &&
				usedParameter &&
				record.occurrence.Role() == catalog.RoleIndex {
				return defined, nil
			}
		}
		return nil, fmt.Errorf(
			"identifier %s has unsupported dual definition/use evidence at %s",
			identifier.Name,
			record.occurrence.ID(),
		)
	}
	if hasDefinition {
		return defined, nil
	}
	if hasUse {
		return used, nil
	}
	return nil, nil
}

func identifierExecutes(
	view checkerExpressionView,
	record *occurrenceInput,
	object types.Object,
) bool {
	if record.occurrence.Kind() != catalog.KindIdent {
		return false
	}
	if identifierIsType(view, record) {
		return false
	}
	switch record.occurrence.Role() {
	case catalog.RoleDeclarationName,
		catalog.RoleImportAlias,
		catalog.RoleLabelDeclaration,
		catalog.RoleLabelReference,
		catalog.RoleSelectedName,
		catalog.RolePackageName:
		return false
	}
	_, isLabel := object.(*types.Label)
	return !isLabel
}

func identifierIsType(
	view checkerExpressionView,
	record *occurrenceInput,
) bool {
	switch record.occurrence.Role() {
	case catalog.RoleTypeExpression,
		catalog.RoleConstructedType,
		catalog.RoleAssertedType,
		catalog.RoleElementType,
		catalog.RoleKeyType,
		catalog.RoleValueType,
		catalog.RoleTypeParameters:
		return true
	}
	if expression, ok := record.node.(ast.Expr); ok {
		value, present := view.TypeOf(expression)
		return present && value.IsType()
	}
	return false
}

func (builder *packageBuilder) typeResolution(
	record *occurrenceInput,
	context occurrenceContext,
) (identity.SemanticTypeID, bool, error) {
	expression, ok := record.node.(ast.Expr)
	if !ok {
		return identity.SemanticTypeID{}, false, nil
	}
	value, present := builder.input.loaded.CheckerView().TypeOf(expression)
	var resolved types.Type
	if present && value.IsType() {
		resolved = value.Type
	}
	if resolved == nil &&
		record.occurrence.Kind() == catalog.KindFuncType &&
		record.occurrence.Role() == catalog.RoleFunctionSignature {
		resolved = context.coverageType
		if resolved == nil {
			resolved = context.signature
		}
	}
	if resolved == nil {
		return identity.SemanticTypeID{}, false, nil
	}
	typeID, err := builder.types.build(resolved)
	return typeID, true, err
}
