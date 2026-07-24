package stagecheck

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/source"
)

func independentExpectedType(
	expected semanticPackageExpectation,
	index *structure.TransientIndex,
	occurrence structure.Occurrence,
) types.Type {
	parent := expected.occurrence(occurrence.Parent())
	parentNode, present := index.OccurrenceNode(parent.ID())
	if !present {
		return nil
	}
	view := expected.loaded.CheckerView()
	role := occurrence.Role()
	ordinal := occurrence.Ordinal()
	switch node := parentNode.(type) {
	case *ast.AssignStmt:
		if role == catalog.RoleAssignedValue &&
			!(len(node.Rhs) == 1 && len(node.Lhs) > 1) &&
			ordinal < len(node.Lhs) {
			return independentExpressionType(view, node.Lhs[ordinal])
		}
	case *ast.ValueSpec:
		if role != catalog.RoleInitializerValue {
			return nil
		}
		if len(node.Values) == 1 && len(node.Names) > 1 {
			return nil
		}
		if node.Type != nil {
			return independentExpressionType(view, node.Type)
		}
		if ordinal < len(node.Names) {
			object, _ := view.DefOf(node.Names[ordinal])
			if object != nil {
				return object.Type()
			}
		}
	case *ast.CallExpr:
		return independentCallExpectedType(view, node, role, ordinal)
	case *ast.ReturnStmt:
		if role != catalog.RoleReturnValue {
			return nil
		}
		signature := independentDefinitionSignature(
			expected, index, parent.Occurrence,
		)
		if signature == nil ||
			(len(node.Results) == 1 && signature.Results().Len() > 1) {
			return nil
		}
		if ordinal < signature.Results().Len() {
			return signature.Results().At(ordinal).Type()
		}
	case *ast.CompositeLit:
		if role != catalog.RoleCompositeElement {
			return nil
		}
		return independentCompositeElementType(
			view,
			independentAggregateType(
				independentExpressionType(view, node),
			),
			node,
			ordinal,
		)
	case *ast.KeyValueExpr:
		return independentKeyedExpectedType(
			expected, index, view, parent.Occurrence, node, role,
		)
	case *ast.SendStmt:
		if role == catalog.RoleSentValue {
			channel := independentAggregateType(
				independentExpressionType(view, node.Chan),
			)
			if typed, ok := channel.(*types.Chan); ok {
				return typed.Elem()
			}
		}
	case *ast.IfStmt:
		if role == catalog.RoleCondition {
			return types.Typ[types.Bool]
		}
	case *ast.ForStmt:
		if role == catalog.RoleCondition {
			return types.Typ[types.Bool]
		}
	case *ast.RangeStmt:
		return independentRangeExpectedType(view, node, role)
	}
	return nil
}

func independentCallExpectedType(
	view *source.TypeInfoView,
	node *ast.CallExpr,
	role catalog.Role,
	ordinal int,
) types.Type {
	if role != catalog.RoleCallArgument {
		return nil
	}
	if value, present := view.TypeOf(node.Fun); present && value.IsType() {
		return value.Type
	}
	signature := independentSignature(
		independentExpressionType(view, node.Fun),
	)
	if signature == nil {
		return nil
	}
	parameters := signature.Params()
	if signature.Variadic() && ordinal >= parameters.Len()-1 {
		if parameters.Len() == 0 {
			return nil
		}
		last := parameters.At(parameters.Len() - 1).Type()
		if node.Ellipsis.IsValid() {
			return last
		}
		if slice, ok := types.Unalias(last).Underlying().(*types.Slice); ok {
			return slice.Elem()
		}
		return nil
	}
	if ordinal < parameters.Len() {
		return parameters.At(ordinal).Type()
	}
	return nil
}

func independentKeyedExpectedType(
	expected semanticPackageExpectation,
	index *structure.TransientIndex,
	view *source.TypeInfoView,
	parent structure.Occurrence,
	node *ast.KeyValueExpr,
	role catalog.Role,
) types.Type {
	literalOccurrence := expected.occurrence(parent.Parent())
	literalNode, present := index.OccurrenceNode(literalOccurrence.ID())
	if !present {
		return nil
	}
	literal, ok := literalNode.(*ast.CompositeLit)
	if !ok {
		return nil
	}
	aggregate := independentAggregateType(
		independentExpressionType(view, literal),
	)
	switch typed := aggregate.(type) {
	case *types.Map:
		if role == catalog.RoleElementKey {
			return typed.Key()
		}
		if role == catalog.RoleElementValue {
			return typed.Elem()
		}
	case *types.Array:
		if role == catalog.RoleElementKey {
			return types.Typ[types.Int]
		}
		if role == catalog.RoleElementValue {
			return typed.Elem()
		}
	case *types.Slice:
		if role == catalog.RoleElementKey {
			return types.Typ[types.Int]
		}
		if role == catalog.RoleElementValue {
			return typed.Elem()
		}
	case *types.Struct:
		if role == catalog.RoleElementValue {
			if identifier, ok := node.Key.(*ast.Ident); ok {
				object, _ := view.UseOf(identifier)
				if field, ok := object.(*types.Var); ok &&
					field.IsField() {
					return field.Type()
				}
			}
		}
	}
	return nil
}

func independentCompositeElementType(
	view *source.TypeInfoView,
	aggregate types.Type,
	literal *ast.CompositeLit,
	ordinal int,
) types.Type {
	switch typed := aggregate.(type) {
	case *types.Array:
		return typed.Elem()
	case *types.Slice:
		return typed.Elem()
	case *types.Map:
		return typed.Elem()
	case *types.Struct:
		if ordinal < len(literal.Elts) {
			if keyed, ok := literal.Elts[ordinal].(*ast.KeyValueExpr); ok {
				if identifier, ok := keyed.Key.(*ast.Ident); ok {
					object, _ := view.UseOf(identifier)
					if field, ok := object.(*types.Var); ok &&
						field.IsField() {
						return field.Type()
					}
				}
			}
		}
		if ordinal < typed.NumFields() {
			return typed.Field(ordinal).Type()
		}
	}
	return nil
}

func independentRangeExpectedType(
	view *source.TypeInfoView,
	node *ast.RangeStmt,
	role catalog.Role,
) types.Type {
	aggregate := independentAggregateType(
		independentExpressionType(view, node.X),
	)
	switch typed := aggregate.(type) {
	case *types.Array:
		if role == catalog.RoleRangeKey {
			return types.Typ[types.Int]
		}
		if role == catalog.RoleRangeValue {
			return typed.Elem()
		}
	case *types.Slice:
		if role == catalog.RoleRangeKey {
			return types.Typ[types.Int]
		}
		if role == catalog.RoleRangeValue {
			return typed.Elem()
		}
	case *types.Map:
		if role == catalog.RoleRangeKey {
			return typed.Key()
		}
		if role == catalog.RoleRangeValue {
			return typed.Elem()
		}
	case *types.Chan:
		if role == catalog.RoleRangeKey {
			return typed.Elem()
		}
	case *types.Basic:
		if typed.Info()&types.IsString != 0 {
			if role == catalog.RoleRangeKey {
				return types.Typ[types.Int]
			}
			if role == catalog.RoleRangeValue {
				return types.Typ[types.Rune]
			}
		}
	}
	return nil
}

func independentExpressionType(
	view *source.TypeInfoView,
	expression ast.Expr,
) types.Type {
	if expression == nil {
		return nil
	}
	value, present := view.TypeOf(expression)
	if !present {
		return nil
	}
	if basic, ok := value.Type.(*types.Basic); ok &&
		basic.Kind() == types.Invalid {
		return nil
	}
	return value.Type
}

func independentSignature(typ types.Type) *types.Signature {
	if typ == nil {
		return nil
	}
	signature, _ := types.Unalias(typ).Underlying().(*types.Signature)
	return signature
}

func independentAggregateType(typ types.Type) types.Type {
	if typ == nil {
		return nil
	}
	underlying := types.Unalias(typ).Underlying()
	if pointer, ok := underlying.(*types.Pointer); ok {
		underlying = types.Unalias(pointer.Elem()).Underlying()
	}
	return underlying
}
