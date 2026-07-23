package frontend

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/typesemantics"
)

func resolveVariant(
	input *packageInput,
	record *occurrenceInput,
	context occurrenceContext,
) (catalog.Variant, error) {
	view := input.loaded.CheckerView()
	var (
		variant catalog.Variant
		reason  string
	)
	switch node := record.node.(type) {
	case *ast.CallExpr:
		variant, reason = callVariant(view, node)
	case *ast.IndexExpr:
		variant, reason = indexVariant(
			view, node.X, node, context,
		)
	case *ast.IndexListExpr:
		variant = catalog.VariantGenericInstantiation
	case *ast.TypeAssertExpr:
		switch {
		case node.Type == nil:
			variant = catalog.VariantTypeSwitchGuard
		case context.commaOK:
			variant = catalog.VariantAssertCommaOk
		default:
			variant = catalog.VariantAssertValue
		}
	case *ast.SelectorExpr:
		variant, reason = selectorVariant(view, node)
	case *ast.AssignStmt:
		variant = assignmentVariant(node)
	case *ast.CompositeLit:
		variant, reason = compositeVariant(view, node)
	case *ast.KeyValueExpr:
		variant, reason = keyedVariant(context.composite)
	case *ast.UnaryExpr:
		switch {
		case node.Op != token.ARROW:
			variant = catalog.VariantNone
		case context.commaOK:
			variant = catalog.VariantReceiveCommaOk
		default:
			variant = catalog.VariantReceiveValue
		}
	case *ast.StarExpr:
		if value, present := view.TypeOf(node); present &&
			value.IsType() {
			variant = catalog.VariantStarPointerType
		} else {
			variant = catalog.VariantStarDereference
		}
	case *ast.ReturnStmt:
		switch {
		case len(node.Results) != 0:
			variant = catalog.VariantReturnValues
		case context.signature != nil &&
			context.signature.Results().Len() != 0:
			variant = catalog.VariantReturnBare
		default:
			variant = catalog.VariantReturnVoid
		}
	case *ast.RangeStmt:
		variant, reason = rangeVariant(view, node)
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
	if reason != "" || !catalog.VariantAllowed(
		record.occurrence.Kind(), variant,
	) {
		if reason == "" {
			reason = "resolved semantic variant is illegal for construct"
		}
		return catalog.VariantInvalid, &Error{
			Package: input.id, Definition: record.owner,
			Occurrence: record.occurrence.ID(),
			Kind:       record.occurrence.Kind(),
			Reason:     reason,
		}
	}
	return variant, nil
}

func callVariant(
	view checkerExpressionView,
	node *ast.CallExpr,
) (catalog.Variant, string) {
	if value, present := view.TypeOf(node.Fun); present &&
		value.IsType() {
		return catalog.VariantConversion, ""
	}
	if callee := calleeIdentifier(node.Fun); callee != nil {
		if object, present := view.UseOf(callee); present {
			if _, builtin := object.(*types.Builtin); builtin {
				return catalog.VariantCallBuiltin, ""
			}
		}
	}
	if selector, ok := ast.Unparen(node.Fun).(*ast.SelectorExpr); ok {
		if selection, present := view.SelectionOf(selector); present &&
			selection.Kind() == types.MethodVal {
			return catalog.VariantCallMethod, ""
		}
	}
	return catalog.VariantCallFunction, ""
}

func indexVariant(
	view checkerExpressionView,
	base ast.Expr,
	node *ast.IndexExpr,
	context occurrenceContext,
) (catalog.Variant, string) {
	if value, present := view.TypeOf(node); present &&
		value.IsType() {
		return catalog.VariantGenericInstantiation, ""
	}
	if callee := calleeIdentifier(base); callee != nil {
		if _, present := view.InstanceOf(callee); present {
			return catalog.VariantGenericInstantiation, ""
		}
	}
	value, present := view.TypeOf(base)
	if !present {
		return catalog.VariantInvalid,
			"index operand has no checker type"
	}
	core, ok := typesemantics.Core(value.Type)
	if ok {
		if _, isMap := core.(*types.Map); isMap {
			if context.commaOK {
				return catalog.VariantMapLookupCommaOk, ""
			}
			return catalog.VariantMapLookupValue, ""
		}
	}
	return catalog.VariantIndexElement, ""
}

func selectorVariant(
	view checkerExpressionView,
	node *ast.SelectorExpr,
) (catalog.Variant, string) {
	if selection, present := view.SelectionOf(node); present {
		switch selection.Kind() {
		case types.MethodVal:
			return catalog.VariantSelectMethodValue, ""
		case types.MethodExpr:
			return catalog.VariantSelectMethodExpression, ""
		case types.FieldVal:
			if len(selection.Index()) > 1 {
				return catalog.VariantSelectPromotedField, ""
			}
			return catalog.VariantSelectField, ""
		}
	}
	if base, ok := ast.Unparen(node.X).(*ast.Ident); ok {
		if object, present := view.UseOf(base); present {
			if _, isPackage := object.(*types.PkgName); isPackage {
				return catalog.VariantSelectPackageMember, ""
			}
		}
	}
	if _, present := view.UseOf(node.Sel); present {
		return catalog.VariantSelectPackageMember, ""
	}
	if _, present := view.DefOf(node.Sel); present {
		return catalog.VariantSelectPackageMember, ""
	}
	return catalog.VariantInvalid,
		"selector has no checker selection or package object"
}

func assignmentVariant(node *ast.AssignStmt) catalog.Variant {
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

func compositeVariant(
	view checkerExpressionView,
	node *ast.CompositeLit,
) (catalog.Variant, string) {
	value, present := view.TypeOf(node)
	if !present {
		return catalog.VariantInvalid,
			"composite literal has no checker type"
	}
	switch aggregateType(value.Type).(type) {
	case *types.Struct:
		return catalog.VariantLitStruct, ""
	case *types.Array:
		return catalog.VariantLitArray, ""
	case *types.Slice:
		return catalog.VariantLitSlice, ""
	case *types.Map:
		return catalog.VariantLitMap, ""
	default:
		return catalog.VariantInvalid,
			"composite literal has unsupported core type"
	}
}

func keyedVariant(
	composite types.Type,
) (catalog.Variant, string) {
	switch composite.(type) {
	case *types.Struct:
		return catalog.VariantKeyFieldName, ""
	case *types.Map:
		return catalog.VariantKeyMapKey, ""
	case *types.Array, *types.Slice:
		return catalog.VariantKeyArrayIndex, ""
	default:
		return catalog.VariantInvalid,
			"keyed element has no supported composite owner"
	}
}

func rangeVariant(
	view checkerExpressionView,
	node *ast.RangeStmt,
) (catalog.Variant, string) {
	value, present := view.TypeOf(node.X)
	if !present {
		return catalog.VariantInvalid,
			"range operand has no checker type"
	}
	core, ok := typesemantics.Core(types.Default(value.Type))
	if !ok {
		return catalog.VariantInvalid,
			"range operand has no core type"
	}
	switch typed := core.(type) {
	case *types.Array:
		return catalog.VariantRangeArray, ""
	case *types.Pointer:
		if element, ok := typesemantics.Core(typed.Elem()); ok {
			if _, array := element.(*types.Array); array {
				return catalog.VariantRangePointerToArray, ""
			}
		}
	case *types.Slice:
		return catalog.VariantRangeSlice, ""
	case *types.Map:
		return catalog.VariantRangeMap, ""
	case *types.Chan:
		return catalog.VariantRangeChannel, ""
	case *types.Signature:
		return catalog.VariantRangeFunc, ""
	case *types.Basic:
		if typed.Info()&types.IsString != 0 {
			return catalog.VariantRangeString, ""
		}
		if typed.Info()&types.IsInteger != 0 {
			return catalog.VariantRangeInteger, ""
		}
	}
	return catalog.VariantInvalid,
		"range operand has unsupported core type"
}

func calleeIdentifier(expression ast.Expr) *ast.Ident {
	switch expression := ast.Unparen(expression).(type) {
	case *ast.Ident:
		return expression
	case *ast.SelectorExpr:
		return expression.Sel
	default:
		return nil
	}
}

type checkerExpressionView interface {
	TypeOf(ast.Expr) (types.TypeAndValue, bool)
	UseOf(*ast.Ident) (types.Object, bool)
	DefOf(*ast.Ident) (types.Object, bool)
	SelectionOf(*ast.SelectorExpr) (*types.Selection, bool)
	InstanceOf(*ast.Ident) (types.Instance, bool)
}
