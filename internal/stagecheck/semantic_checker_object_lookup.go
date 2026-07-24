package stagecheck

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/source"
)

func independentCheckerObject(
	view interface {
		DefOf(*ast.Ident) (types.Object, bool)
		UseOf(*ast.Ident) (types.Object, bool)
		SelectionOf(*ast.SelectorExpr) (*types.Selection, bool)
	},
	node ast.Node,
) types.Object {
	switch node := node.(type) {
	case *ast.Ident:
		if object, present := view.DefOf(node); present {
			return object
		}
		object, _ := view.UseOf(node)
		return object
	case *ast.TypeSpec:
		object, _ := view.DefOf(node.Name)
		return object
	case *ast.SelectorExpr:
		if selection, present := view.SelectionOf(node); present {
			return selection.Obj()
		}
		if object, present := view.UseOf(node.Sel); present {
			return object
		}
		object, _ := view.DefOf(node.Sel)
		return object
	default:
		return nil
	}
}

func independentOperationObject(
	view *source.TypeInfoView,
	node ast.Node,
) types.Object {
	switch node := node.(type) {
	case *ast.CallExpr:
		return independentExpressionObject(view, node.Fun)
	case *ast.IndexExpr:
		return independentExpressionObject(view, node.X)
	case *ast.IndexListExpr:
		return independentExpressionObject(view, node.X)
	default:
		return independentCheckerObject(view, node)
	}
}

func (verifier *checkerSemanticVerifier) verifyOperationObject(
	occurrence structure.OccurrenceRef,
	node ast.Node,
	reference semantic.ObjectReference,
) error {
	if identifier, blank := node.(*ast.Ident); blank &&
		identifier.Name == "_" {
		if reference.Kind() != semantic.ObjectReferenceNone {
			return fmt.Errorf(
				"blank identifier %s carries semantic object",
				occurrence.ID(),
			)
		}
		return nil
	}
	object := independentOperationObject(verifier.view, node)
	if object == nil {
		if reference.Kind() != semantic.ObjectReferenceNone {
			return fmt.Errorf("semantic object exists without checker object")
		}
		return nil
	}
	if reference.Kind() ==
		semantic.ObjectReferenceDeclaration &&
		reference.Declaration().Form() ==
			identity.SemanticDeclarationMember {
		if selection := independentOperationSelection(
			verifier.view, node,
		); selection != nil {
			return verifier.verifyCheckerSelectionDeclaration(
				reference.Declaration(), selection,
			)
		}
		handled, err := verifier.verifyCompositeFieldReference(
			occurrence, reference.Declaration(), object,
		)
		if handled {
			return err
		}
	}
	return verifier.verifyObjectReference(reference, object)
}

func independentOperationSelection(
	view *source.TypeInfoView,
	node ast.Node,
) *types.Selection {
	var expression ast.Expr
	switch typed := node.(type) {
	case *ast.CallExpr:
		expression = typed.Fun
	case ast.Expr:
		expression = typed
	}
	for expression != nil {
		switch typed := ast.Unparen(expression).(type) {
		case *ast.SelectorExpr:
			selection, _ := view.SelectionOf(typed)
			return selection
		case *ast.IndexExpr:
			expression = typed.X
		case *ast.IndexListExpr:
			expression = typed.X
		default:
			return nil
		}
	}
	return nil
}

func independentExpressionObject(
	view *source.TypeInfoView,
	expression ast.Expr,
) types.Object {
	switch expression := ast.Unparen(expression).(type) {
	case *ast.Ident:
		object, _ := view.UseOf(expression)
		return object
	case *ast.SelectorExpr:
		return independentCheckerObject(view, expression)
	case *ast.IndexExpr:
		return independentExpressionObject(view, expression.X)
	case *ast.IndexListExpr:
		return independentExpressionObject(view, expression.X)
	default:
		return nil
	}
}

func independentNodeType(
	view *source.TypeInfoView,
	node ast.Node,
) types.Type {
	if expression, ok := node.(ast.Expr); ok {
		if value, present := view.TypeOf(expression); present {
			return value.Type
		}
	}
	if object := independentCheckerObject(view, node); object != nil {
		return object.Type()
	}
	return nil
}

func independentGenericNodeIdentifier(node ast.Node) *ast.Ident {
	switch node := node.(type) {
	case ast.Expr:
		return independentGenericIdentifier(node)
	case *ast.CallExpr:
		return independentGenericIdentifier(node.Fun)
	default:
		return nil
	}
}

func independentObjectClass(
	object types.Object,
) identity.SemanticObjectClass {
	switch object := object.(type) {
	case *types.PkgName:
		return identity.SemanticObjectPackage
	case *types.Const:
		return identity.SemanticObjectConstant
	case *types.TypeName:
		if object.IsAlias() {
			return identity.SemanticObjectAlias
		}
		return identity.SemanticObjectType
	case *types.Var:
		if object.IsField() {
			return identity.SemanticObjectField
		}
		return identity.SemanticObjectVariable
	case *types.Func:
		signature, _ := object.Type().(*types.Signature)
		if signature != nil && signature.Recv() != nil {
			return identity.SemanticObjectMethod
		}
		return identity.SemanticObjectFunction
	case *types.Builtin:
		return identity.SemanticObjectBuiltin
	case *types.Nil:
		return identity.SemanticObjectNil
	case *types.Label:
		return identity.SemanticObjectInvalid
	default:
		return identity.SemanticObjectInvalid
	}
}

func independentPredeclaredKind(
	object types.Object,
) catalog.PredeclaredKind {
	for _, member := range catalog.AllPredeclared() {
		if types.Universe.Lookup(member.Name()) == object {
			return member
		}
	}
	return catalog.PredeclaredInvalid
}
