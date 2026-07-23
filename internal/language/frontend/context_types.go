package frontend

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

func definitionSignature(
	input *packageInput,
	definition identity.DefinitionID,
) (*types.Signature, error) {
	if definition.Kind() == identity.DefinitionImplicit ||
		definition.Kind() == identity.DefinitionPackageInitializer {
		return nil, nil
	}
	node, present := input.index.CheckedDefinitionNode(definition)
	if !present {
		node, present = input.index.DefinitionNode(definition)
	}
	if !present {
		return nil, fmt.Errorf(
			"definition %s has no transient semantic root",
			definition,
		)
	}
	view := input.loaded.CheckerView()
	switch node := node.(type) {
	case *ast.FuncDecl:
		object, present := view.DefOf(node.Name)
		if !present || object == nil {
			object = intrinsicDefinitionObject(
				input, node.Name.Name,
			)
		}
		if object == nil {
			return nil, fmt.Errorf(
				"function definition %s has no checker object",
				definition,
			)
		}
		if _, builtin := object.(*types.Builtin); builtin {
			return nil, nil
		}
		return signatureOf(object.Type()), nil
	case *ast.FuncLit:
		return signatureOf(expressionType(view, node)), nil
	default:
		return nil, nil
	}
}

func definitionObject(
	input *packageInput,
	definition identity.DefinitionID,
) types.Object {
	node, present := input.index.CheckedDefinitionNode(definition)
	if !present {
		node, present = input.index.DefinitionNode(definition)
	}
	if !present {
		return nil
	}
	view := input.loaded.CheckerView()
	switch node := node.(type) {
	case *ast.FuncDecl:
		object, _ := view.DefOf(node.Name)
		if object == nil {
			object = intrinsicDefinitionObject(input, node.Name.Name)
		}
		return object
	case *ast.TypeSpec:
		object, _ := view.DefOf(node.Name)
		return object
	case *ast.ValueSpec:
		if len(node.Names) == 1 {
			object, _ := view.DefOf(node.Names[0])
			return object
		}
	}
	return nil
}

func expressionType(
	view interface {
		TypeOf(ast.Expr) (types.TypeAndValue, bool)
	},
	expression ast.Expr,
) types.Type {
	if expression == nil {
		return nil
	}
	value, present := view.TypeOf(expression)
	if !present {
		return nil
	}
	if basic, ok := value.Type.(*types.Basic); ok && basic.Kind() == types.Invalid {
		return nil
	}
	return value.Type
}

func signatureOf(value types.Type) *types.Signature {
	if value == nil {
		return nil
	}
	signature, _ := types.Unalias(value).Underlying().(*types.Signature)
	return signature
}

func aggregateType(value types.Type) types.Type {
	if value == nil {
		return nil
	}
	underlying := types.Unalias(value).Underlying()
	if pointer, ok := underlying.(*types.Pointer); ok {
		underlying = types.Unalias(pointer.Elem()).Underlying()
	}
	return underlying
}

func compositeElementType(
	view checkerExpressionView,
	aggregate types.Type,
	literal *ast.CompositeLit,
	ordinal int,
) types.Type {
	switch value := aggregate.(type) {
	case *types.Array:
		return value.Elem()
	case *types.Slice:
		return value.Elem()
	case *types.Map:
		return value.Elem()
	case *types.Struct:
		if ordinal < len(literal.Elts) {
			if keyed, ok := literal.Elts[ordinal].(*ast.KeyValueExpr); ok {
				if identifier, ok := keyed.Key.(*ast.Ident); ok {
					if object, present := view.UseOf(identifier); present {
						if field, ok := object.(*types.Var); ok &&
							field.IsField() {
							return field.Type()
						}
					}
				}
			}
		}
		if ordinal < value.NumFields() {
			return value.Field(ordinal).Type()
		}
	}
	return nil
}

func channelElement(
	view interface {
		TypeOf(ast.Expr) (types.TypeAndValue, bool)
	},
	channel ast.Expr,
) types.Type {
	value := aggregateType(expressionType(view, channel))
	if typed, ok := value.(*types.Chan); ok {
		return typed.Elem()
	}
	return nil
}

func nextCase(
	input *packageInput,
	current *occurrenceInput,
) identity.OccurrenceID {
	parent := input.occurrences[current.occurrence.Parent()]
	if parent == nil {
		return identity.OccurrenceID{}
	}
	for _, childID := range parent.children {
		child := input.occurrences[childID]
		if child == nil ||
			child.occurrence.Kind() != catalog.KindCaseClause ||
			child.occurrence.Ordinal() <= current.occurrence.Ordinal() {
			continue
		}
		return childID
	}
	return identity.OccurrenceID{}
}
