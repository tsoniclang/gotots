package stagecheck

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/language/typesemantics"
	"github.com/tsoniclang/gotots/internal/source"
)

func independentSemanticVariant(
	expected semanticPackageExpectation,
	index *structure.TransientIndex,
	occurrence structure.OccurrenceRef,
	node ast.Node,
) (catalog.Variant, error) {
	view := expected.loaded.CheckerView()
	commaOK := independentCommaOK(
		expected, index, occurrence,
	)
	var variant catalog.Variant
	switch node := node.(type) {
	case *ast.CallExpr:
		variant = independentCallVariant(view, node)
	case *ast.IndexExpr:
		value, present := view.TypeOf(node)
		if present && value.IsType() {
			variant = catalog.VariantGenericInstantiation
			break
		}
		if identifier := independentGenericIdentifier(node.X); identifier != nil {
			if _, present := view.InstanceOf(identifier); present {
				variant = catalog.VariantGenericInstantiation
				break
			}
		}
		base, present := view.TypeOf(node.X)
		if !present {
			return 0, fmt.Errorf("index base has no checker type")
		}
		core, _ := typesemantics.Core(base.Type)
		if _, mapType := core.(*types.Map); mapType {
			if commaOK {
				variant = catalog.VariantMapLookupCommaOk
			} else {
				variant = catalog.VariantMapLookupValue
			}
		} else {
			variant = catalog.VariantIndexElement
		}
	case *ast.IndexListExpr:
		variant = catalog.VariantGenericInstantiation
	case *ast.TypeAssertExpr:
		switch {
		case node.Type == nil:
			variant = catalog.VariantTypeSwitchGuard
		case commaOK:
			variant = catalog.VariantAssertCommaOk
		default:
			variant = catalog.VariantAssertValue
		}
	case *ast.SelectorExpr:
		var err error
		variant, err = independentSelectorVariant(view, node)
		if err != nil {
			return 0, err
		}
	case *ast.AssignStmt:
		variant = independentAssignmentVariant(node)
	case *ast.CompositeLit:
		value, present := view.TypeOf(node)
		if !present {
			return 0, fmt.Errorf(
				"composite literal has no checker type",
			)
		}
		variant = independentCompositeVariant(value.Type)
	case *ast.KeyValueExpr:
		variant = independentKeyedVariant(
			expected, index, occurrence,
		)
	case *ast.UnaryExpr:
		switch {
		case node.Op != token.ARROW:
			variant = catalog.VariantNone
		case commaOK:
			variant = catalog.VariantReceiveCommaOk
		default:
			variant = catalog.VariantReceiveValue
		}
	case *ast.StarExpr:
		value, present := view.TypeOf(node)
		if present && value.IsType() {
			variant = catalog.VariantStarPointerType
		} else {
			variant = catalog.VariantStarDereference
		}
	case *ast.ReturnStmt:
		switch {
		case len(node.Results) != 0:
			variant = catalog.VariantReturnValues
		case independentDefinitionSignature(
			expected, index, occurrence,
		) != nil &&
			independentDefinitionSignature(
				expected, index, occurrence,
			).Results().Len() != 0:
			variant = catalog.VariantReturnBare
		default:
			variant = catalog.VariantReturnVoid
		}
	case *ast.RangeStmt:
		value, present := view.TypeOf(node.X)
		if !present {
			return 0, fmt.Errorf("range operand has no checker type")
		}
		variant = independentRangeVariant(value.Type)
	case *ast.SwitchStmt:
		if node.Tag == nil {
			variant = catalog.VariantSwitchTrue
		} else {
			variant = catalog.VariantSwitchExpression
		}
	case *ast.TypeSpec:
		if node.Assign.IsValid() {
			variant = catalog.VariantTypeAlias
		} else {
			variant = catalog.VariantTypeDefinition
		}
	case *ast.CommClause:
		switch node.Comm.(type) {
		case nil:
			variant = catalog.VariantCommDefault
		case *ast.SendStmt:
			variant = catalog.VariantCommSend
		default:
			variant = catalog.VariantCommReceive
		}
	default:
		variant = catalog.VariantNone
	}
	if !catalog.VariantAllowed(occurrence.Kind(), variant) {
		return 0, fmt.Errorf(
			"variant %s is illegal for %s",
			variant, occurrence.Kind(),
		)
	}
	return variant, nil
}

func independentCommaOK(
	expected semanticPackageExpectation,
	index *structure.TransientIndex,
	occurrence structure.OccurrenceRef,
) bool {
	parent := expected.occurrence(occurrence.Parent())
	parentNode, present := index.OccurrenceNode(parent.ID())
	if !present {
		return false
	}
	switch node := parentNode.(type) {
	case *ast.AssignStmt:
		return occurrence.Role() == catalog.RoleAssignedValue &&
			len(node.Lhs) == 2 &&
			len(node.Rhs) == 1 &&
			independentHasOK(
				expected.loaded.CheckerView(), node.Rhs[0],
			)
	case *ast.ValueSpec:
		return occurrence.Role() == catalog.RoleInitializerValue &&
			len(node.Names) == 2 &&
			len(node.Values) == 1 &&
			independentHasOK(
				expected.loaded.CheckerView(), node.Values[0],
			)
	default:
		return false
	}
}

func independentHasOK(
	view *source.TypeInfoView,
	expression ast.Expr,
) bool {
	value, present := view.TypeOf(expression)
	return present && value.HasOk()
}

func independentCallVariant(
	view *source.TypeInfoView,
	node *ast.CallExpr,
) catalog.Variant {
	if value, present := view.TypeOf(node.Fun); present &&
		value.IsType() {
		return catalog.VariantConversion
	}
	if identifier := independentGenericIdentifier(node.Fun); identifier != nil {
		if object, present := view.UseOf(identifier); present {
			if _, builtin := object.(*types.Builtin); builtin {
				return catalog.VariantCallBuiltin
			}
		}
	}
	if selector, ok := ast.Unparen(node.Fun).(*ast.SelectorExpr); ok {
		if selection, present := view.SelectionOf(selector); present &&
			selection.Kind() == types.MethodVal {
			return catalog.VariantCallMethod
		}
	}
	return catalog.VariantCallFunction
}

func independentSelectorVariant(
	view *source.TypeInfoView,
	node *ast.SelectorExpr,
) (catalog.Variant, error) {
	if selection, present := view.SelectionOf(node); present {
		switch selection.Kind() {
		case types.MethodVal:
			return catalog.VariantSelectMethodValue, nil
		case types.MethodExpr:
			return catalog.VariantSelectMethodExpression, nil
		case types.FieldVal:
			if len(selection.Index()) > 1 {
				return catalog.VariantSelectPromotedField, nil
			}
			return catalog.VariantSelectField, nil
		}
	}
	if base, ok := ast.Unparen(node.X).(*ast.Ident); ok {
		if object, present := view.UseOf(base); present {
			if _, packageName := object.(*types.PkgName); packageName {
				return catalog.VariantSelectPackageMember, nil
			}
		}
	}
	if _, present := view.UseOf(node.Sel); present {
		return catalog.VariantSelectPackageMember, nil
	}
	if _, present := view.DefOf(node.Sel); present {
		return catalog.VariantSelectPackageMember, nil
	}
	return 0, fmt.Errorf("selector has no checker selection")
}

func independentAssignmentVariant(
	node *ast.AssignStmt,
) catalog.Variant {
	define := node.Tok == token.DEFINE
	if !define && node.Tok != token.ASSIGN {
		return catalog.VariantAssignCompound
	}
	if len(node.Lhs) != len(node.Rhs) && len(node.Rhs) == 1 {
		if _, call := ast.Unparen(node.Rhs[0]).(*ast.CallExpr); call {
			if define {
				return catalog.VariantDefineFromCall
			}
			return catalog.VariantAssignFromCall
		}
		if define {
			return catalog.VariantDefineCommaOk
		}
		return catalog.VariantAssignCommaOk
	}
	if define {
		return catalog.VariantDefineBalanced
	}
	return catalog.VariantAssignBalanced
}

func independentCompositeVariant(
	typ types.Type,
) catalog.Variant {
	underlying := types.Unalias(typ).Underlying()
	if pointer, ok := underlying.(*types.Pointer); ok {
		underlying = types.Unalias(pointer.Elem()).Underlying()
	}
	switch underlying.(type) {
	case *types.Struct:
		return catalog.VariantLitStruct
	case *types.Array:
		return catalog.VariantLitArray
	case *types.Slice:
		return catalog.VariantLitSlice
	case *types.Map:
		return catalog.VariantLitMap
	default:
		return catalog.VariantInvalid
	}
}

func independentKeyedVariant(
	expected semanticPackageExpectation,
	index *structure.TransientIndex,
	occurrence structure.OccurrenceRef,
) catalog.Variant {
	parentNode, present := index.OccurrenceNode(
		occurrence.Parent(),
	)
	if !present {
		return catalog.VariantInvalid
	}
	literal, ok := parentNode.(*ast.CompositeLit)
	if !ok {
		return catalog.VariantInvalid
	}
	value, present := expected.loaded.CheckerView().TypeOf(literal)
	if !present {
		return catalog.VariantInvalid
	}
	switch types.Unalias(value.Type).Underlying().(type) {
	case *types.Struct:
		return catalog.VariantKeyFieldName
	case *types.Map:
		return catalog.VariantKeyMapKey
	case *types.Array, *types.Slice:
		return catalog.VariantKeyArrayIndex
	default:
		return catalog.VariantInvalid
	}
}

func independentRangeVariant(typ types.Type) catalog.Variant {
	core, ok := typesemantics.Core(types.Default(typ))
	if !ok {
		return catalog.VariantInvalid
	}
	switch typed := core.(type) {
	case *types.Array:
		return catalog.VariantRangeArray
	case *types.Pointer:
		if element, ok := typesemantics.Core(typed.Elem()); ok {
			if _, array := element.(*types.Array); array {
				return catalog.VariantRangePointerToArray
			}
		}
	case *types.Slice:
		return catalog.VariantRangeSlice
	case *types.Map:
		return catalog.VariantRangeMap
	case *types.Chan:
		return catalog.VariantRangeChannel
	case *types.Signature:
		return catalog.VariantRangeFunc
	case *types.Basic:
		if typed.Info()&types.IsString != 0 {
			return catalog.VariantRangeString
		}
		if typed.Info()&types.IsInteger != 0 {
			return catalog.VariantRangeInteger
		}
	}
	return catalog.VariantInvalid
}

func independentGenericIdentifier(expression ast.Expr) *ast.Ident {
	switch expression := ast.Unparen(expression).(type) {
	case *ast.Ident:
		return expression
	case *ast.SelectorExpr:
		return expression.Sel
	case *ast.IndexExpr:
		return independentGenericIdentifier(expression.X)
	case *ast.IndexListExpr:
		return independentGenericIdentifier(expression.X)
	case *ast.CallExpr:
		return independentGenericIdentifier(expression.Fun)
	default:
		return nil
	}
}

func independentDefinitionSignature(
	expected semanticPackageExpectation,
	index *structure.TransientIndex,
	occurrence structure.OccurrenceRef,
) *types.Signature {
	definition := expected.occurrenceOwner(occurrence.ID())
	node, present := index.CheckedDefinitionNode(definition)
	if !present {
		node, present = index.DefinitionNode(definition)
	}
	if !present {
		return nil
	}
	view := expected.loaded.CheckerView()
	var typ types.Type
	switch node := node.(type) {
	case *ast.FuncDecl:
		object, _ := view.DefOf(node.Name)
		if object != nil {
			typ = object.Type()
		}
	case *ast.FuncLit:
		if value, present := view.TypeOf(node); present {
			typ = value.Type
		}
	}
	if typ == nil {
		return nil
	}
	signature, _ := types.Unalias(typ).Underlying().(*types.Signature)
	return signature
}

func independentOperationKind(
	view *source.TypeInfoView,
	occurrence structure.OccurrenceRef,
	node ast.Node,
	variant catalog.Variant,
) semantic.OperationKind {
	switch node := node.(type) {
	case *ast.Ident:
		if object, present := view.DefOf(node); present && object != nil {
			return semantic.OperationDeclare
		}
		if object, present := view.UseOf(node); present {
			switch object.(type) {
			case *types.PkgName:
				return semantic.OperationPackageSelect
			case *types.Func, *types.Builtin:
				return semantic.OperationFunctionValue
			}
		}
		switch occurrence.Role() {
		case catalog.RoleAssignmentTarget,
			catalog.RoleAssignablePlace,
			catalog.RoleRangeKey,
			catalog.RoleRangeValue:
			return semantic.OperationStore
		default:
			return semantic.OperationLoad
		}
	case *ast.BasicLit:
		return semantic.OperationLiteral
	case *ast.CompositeLit:
		return semantic.OperationComposite
	case *ast.ParenExpr:
		return semantic.OperationParenthesized
	case *ast.SelectorExpr:
		switch variant {
		case catalog.VariantSelectField,
			catalog.VariantSelectPromotedField:
			return semantic.OperationFieldSelect
		case catalog.VariantSelectMethodValue:
			return semantic.OperationMethodValue
		case catalog.VariantSelectMethodExpression:
			return semantic.OperationMethodExpression
		case catalog.VariantSelectPackageMember:
			return semantic.OperationPackageSelect
		}
	case *ast.IndexExpr:
		switch variant {
		case catalog.VariantMapLookupValue,
			catalog.VariantMapLookupCommaOk:
			return semantic.OperationMapLookup
		case catalog.VariantGenericInstantiation:
			return semantic.OperationGenericInstantiate
		default:
			return semantic.OperationIndex
		}
	case *ast.IndexListExpr:
		return semantic.OperationGenericInstantiate
	case *ast.SliceExpr:
		return semantic.OperationSlice
	case *ast.TypeAssertExpr:
		return semantic.OperationTypeAssert
	case *ast.CallExpr:
		switch variant {
		case catalog.VariantCallBuiltin:
			return semantic.OperationBuiltinCall
		case catalog.VariantConversion:
			return semantic.OperationConvert
		default:
			return semantic.OperationCall
		}
	case *ast.StarExpr:
		return semantic.OperationDereference
	case *ast.UnaryExpr:
		if node.Op == token.AND {
			return semantic.OperationAddress
		}
		if node.Op == token.ARROW {
			return semantic.OperationReceive
		}
		return semantic.OperationUnary
	case *ast.BinaryExpr:
		return semantic.OperationBinary
	case *ast.KeyValueExpr:
		return semantic.OperationKeyedElement
	case *ast.DeclStmt:
		return semantic.OperationDeclarationStatement
	case *ast.EmptyStmt:
		return semantic.OperationEmpty
	case *ast.LabeledStmt:
		return semantic.OperationLabel
	case *ast.ExprStmt:
		return semantic.OperationExpressionStatement
	case *ast.SendStmt:
		return semantic.OperationSend
	case *ast.IncDecStmt:
		return semantic.OperationIncrement
	case *ast.AssignStmt:
		return semantic.OperationAssign
	case *ast.GoStmt:
		return semantic.OperationSpawn
	case *ast.DeferStmt:
		return semantic.OperationDefer
	case *ast.ReturnStmt:
		return semantic.OperationReturn
	case *ast.BranchStmt:
		return semantic.OperationBranch
	case *ast.BlockStmt:
		return semantic.OperationBlock
	case *ast.IfStmt:
		return semantic.OperationIf
	case *ast.CaseClause:
		return semantic.OperationCase
	case *ast.SwitchStmt:
		return semantic.OperationSwitch
	case *ast.TypeSwitchStmt:
		return semantic.OperationTypeSwitch
	case *ast.CommClause:
		return semantic.OperationCommClause
	case *ast.SelectStmt:
		return semantic.OperationSelect
	case *ast.ForStmt:
		return semantic.OperationFor
	case *ast.RangeStmt:
		return semantic.OperationRange
	}
	return semantic.OperationInvalid
}
