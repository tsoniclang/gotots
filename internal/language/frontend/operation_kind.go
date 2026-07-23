package frontend

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

func operationKind(
	view checkerExpressionView,
	record *occurrenceInput,
	variant catalog.Variant,
) (semantic.OperationKind, error) {
	switch record.occurrence.Kind() {
	case catalog.KindIdent:
		return identifierOperationKind(view, record), nil
	case catalog.KindBasicLit:
		return semantic.OperationLiteral, nil
	case catalog.KindCompositeLit:
		return semantic.OperationComposite, nil
	case catalog.KindParenExpr:
		return semantic.OperationParenthesized, nil
	case catalog.KindSelectorExpr:
		switch variant {
		case catalog.VariantSelectField,
			catalog.VariantSelectPromotedField:
			return semantic.OperationFieldSelect, nil
		case catalog.VariantSelectMethodValue:
			return semantic.OperationMethodValue, nil
		case catalog.VariantSelectMethodExpression:
			return semantic.OperationMethodExpression, nil
		case catalog.VariantSelectPackageMember:
			return semantic.OperationPackageSelect, nil
		}
	case catalog.KindIndexExpr:
		switch variant {
		case catalog.VariantMapLookupValue,
			catalog.VariantMapLookupCommaOk:
			return semantic.OperationMapLookup, nil
		case catalog.VariantGenericInstantiation:
			return semantic.OperationGenericInstantiate, nil
		default:
			return semantic.OperationIndex, nil
		}
	case catalog.KindIndexListExpr:
		return semantic.OperationGenericInstantiate, nil
	case catalog.KindSliceExpr:
		return semantic.OperationSlice, nil
	case catalog.KindTypeAssertExpr:
		return semantic.OperationTypeAssert, nil
	case catalog.KindCallExpr:
		switch variant {
		case catalog.VariantCallBuiltin:
			return semantic.OperationBuiltinCall, nil
		case catalog.VariantConversion:
			return semantic.OperationConvert, nil
		default:
			return semantic.OperationCall, nil
		}
	case catalog.KindStarExpr:
		return semantic.OperationDereference, nil
	case catalog.KindUnaryExpr:
		node, _ := record.node.(*ast.UnaryExpr)
		if node != nil && node.Op == token.AND {
			return semantic.OperationAddress, nil
		}
		if variant == catalog.VariantReceiveValue ||
			variant == catalog.VariantReceiveCommaOk {
			return semantic.OperationReceive, nil
		}
		return semantic.OperationUnary, nil
	case catalog.KindBinaryExpr:
		return semantic.OperationBinary, nil
	case catalog.KindKeyValueExpr:
		return semantic.OperationKeyedElement, nil
	case catalog.KindDeclStmt:
		return semantic.OperationDeclarationStatement, nil
	case catalog.KindEmptyStmt:
		return semantic.OperationEmpty, nil
	case catalog.KindLabeledStmt:
		return semantic.OperationLabel, nil
	case catalog.KindExprStmt:
		return semantic.OperationExpressionStatement, nil
	case catalog.KindSendStmt:
		return semantic.OperationSend, nil
	case catalog.KindIncDecStmt:
		return semantic.OperationIncrement, nil
	case catalog.KindAssignStmt:
		return semantic.OperationAssign, nil
	case catalog.KindGoStmt:
		return semantic.OperationSpawn, nil
	case catalog.KindDeferStmt:
		return semantic.OperationDefer, nil
	case catalog.KindReturnStmt:
		return semantic.OperationReturn, nil
	case catalog.KindBranchStmt:
		return semantic.OperationBranch, nil
	case catalog.KindBlockStmt:
		return semantic.OperationBlock, nil
	case catalog.KindIfStmt:
		return semantic.OperationIf, nil
	case catalog.KindCaseClause:
		return semantic.OperationCase, nil
	case catalog.KindSwitchStmt:
		return semantic.OperationSwitch, nil
	case catalog.KindTypeSwitchStmt:
		return semantic.OperationTypeSwitch, nil
	case catalog.KindCommClause:
		return semantic.OperationCommClause, nil
	case catalog.KindSelectStmt:
		return semantic.OperationSelect, nil
	case catalog.KindForStmt:
		return semantic.OperationFor, nil
	case catalog.KindRangeStmt:
		return semantic.OperationRange, nil
	}
	if catalog.AllowsResolution(
		record.occurrence.Kind(),
		record.occurrence.Role(),
		variant,
		record.domain,
		catalog.ResolutionClassStructural,
	) {
		return semantic.OperationInvalid, nil
	}
	return semantic.OperationInvalid, &Error{
		Definition: record.owner,
		Occurrence: record.occurrence.ID(),
		Kind:       record.occurrence.Kind(),
		Reason: fmt.Sprintf(
			"executable construct has no semantic operation for variant %s",
			variant,
		),
	}
}

func identifierOperationKind(
	view checkerExpressionView,
	record *occurrenceInput,
) semantic.OperationKind {
	identifier, _ := record.node.(*ast.Ident)
	if identifier != nil && view != nil {
		if object, present := view.DefOf(identifier); present &&
			object != nil {
			return semantic.OperationDeclare
		}
		if object, present := view.UseOf(identifier); present {
			switch object.(type) {
			case *types.PkgName:
				return semantic.OperationPackageSelect
			case *types.Func, *types.Builtin:
				return semantic.OperationFunctionValue
			}
		}
	}
	switch record.occurrence.Role() {
	case catalog.RoleAssignmentTarget,
		catalog.RoleAssignablePlace,
		catalog.RoleRangeKey,
		catalog.RoleRangeValue:
		return semantic.OperationStore
	default:
		return semantic.OperationLoad
	}
}
