package stagecheck

import (
	"fmt"
	"go/ast"
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/source"
)

func (verifier *checkerSemanticVerifier) verifyDeclarationReferenceIdentity(
	id identity.SemanticDeclarationID,
	object types.Object,
) error {
	if function, ok := object.(*types.Func); ok {
		object = function.Origin()
	}
	if object.Pkg() == nil {
		predeclared := independentPredeclaredKind(object)
		class := independentPredeclaredClass(predeclared.Class())
		expected, err := identity.NewPredeclaredDeclarationID(
			uint16(predeclared), class,
		)
		if !predeclared.Valid() ||
			!class.Valid() ||
			err != nil ||
			id != expected {
			return fmt.Errorf(
				"predeclared declaration identity differs for %s",
				object.Name(),
			)
		}
		return nil
	}
	if expected := verifier.types.localDeclarations[object]; !expected.IsZero() {
		if id != expected {
			return fmt.Errorf(
				"local declaration identity differs for %s: semantic=%s checker=%s",
				object.Name(), id, expected,
			)
		}
		return nil
	}
	pkg := verifier.types.packageByPath[object.Pkg().Path()]
	class := independentObjectClass(object)
	if pkg.IsZero() ||
		id.Form() != identity.SemanticDeclarationPackageObject {
		return fmt.Errorf(
			"declaration %s is absent from this shard and is not a package object",
			id,
		)
	}
	expected, err := identity.NewPackageDeclarationID(
		pkg, class, object.Name(),
	)
	if err != nil || id != expected {
		return fmt.Errorf(
			"package declaration identity differs for %s.%s",
			object.Pkg().Path(), object.Name(),
		)
	}
	return nil
}

func (verifier *checkerSemanticVerifier) verifySelection(
	node ast.Node,
	record semantic.Selection,
) error {
	selector, selectorNode := node.(*ast.SelectorExpr)
	if !selectorNode {
		if !record.IsZero() {
			return fmt.Errorf("non-selector carries selection")
		}
		return nil
	}
	checker, present := verifier.view.SelectionOf(selector)
	if !present {
		if !record.IsZero() {
			return fmt.Errorf("package selector carries field selection")
		}
		return nil
	}
	var kind semantic.SelectionKind
	switch checker.Kind() {
	case types.FieldVal:
		kind = semantic.SelectionField
	case types.MethodVal:
		kind = semantic.SelectionMethodValue
	case types.MethodExpr:
		kind = semantic.SelectionMethodExpression
	}
	if record.IsZero() ||
		record.Kind() != kind ||
		record.Indirect() != checker.Indirect() ||
		!slices.Equal(record.Index(), checker.Index()) {
		return fmt.Errorf("selection kind/index/indirection differs")
	}
	if err := verifier.types.verify(
		record.Receiver(), checker.Recv(),
	); err != nil {
		return err
	}
	return verifier.verifySelectionDeclaration(record, checker)
}

func (verifier *checkerSemanticVerifier) verifySelectionDeclaration(
	selection semantic.Selection,
	checker *types.Selection,
) error {
	if err := verifier.verifyCheckerSelectionDeclaration(
		selection.Object(), checker,
	); err != nil {
		return err
	}
	object := checker.Obj()
	class := independentObjectClass(object)
	if declaration, present := verifier.declarations[selection.Object()]; present {
		if declaration.Name() != object.Name() ||
			declaration.Class() != class ||
			declaration.Exported() != object.Exported() {
			return fmt.Errorf(
				"selection declaration %s metadata differs",
				declaration.ID(),
			)
		}
	}
	return nil
}

func (verifier *checkerSemanticVerifier) verifyCheckerSelectionDeclaration(
	id identity.SemanticDeclarationID,
	checker *types.Selection,
) error {
	object := checker.Obj()
	if function, ok := object.(*types.Func); ok {
		object = function.Origin()
	}
	owner, err := independentSelectionOwner(checker)
	if err != nil {
		return err
	}
	if err := verifier.types.verify(id.OwnerType(), owner); err != nil {
		return fmt.Errorf("selection owner: %w", err)
	}
	return verifier.verifyMemberDeclarationIdentity(
		id, object, owner, selectionMemberOrdinal(checker),
	)
}

func selectionMemberOrdinal(checker *types.Selection) int {
	if _, field := checker.Obj().(*types.Var); !field {
		return 0
	}
	index := checker.Index()
	return index[len(index)-1]
}

func (
	verifier *checkerSemanticVerifier,
) verifyMemberDeclarationIdentity(
	id identity.SemanticDeclarationID,
	object types.Object,
	owner types.Type,
	ordinal int,
) error {
	class := independentObjectClass(object)
	var pkg identity.PackageID
	if !object.Exported() {
		if object.Pkg() == nil {
			return fmt.Errorf(
				"unexported selection %s has no package",
				object.Name(),
			)
		}
		pkg = verifier.types.packageByPath[object.Pkg().Path()]
	}
	expected, err := identity.NewMemberDeclarationID(
		id.OwnerType(), pkg, class, object.Name(), ordinal,
	)
	if err != nil || expected != id {
		return fmt.Errorf(
			"selection member identity differs: semantic=%s checker=%s",
			id, expected,
		)
	}
	return nil
}

func (
	verifier *checkerSemanticVerifier,
) verifyCompositeFieldReference(
	occurrence structure.Occurrence,
	id identity.SemanticDeclarationID,
	object types.Object,
) (bool, error) {
	if occurrence.Role() != catalog.RoleElementKey {
		return false, nil
	}
	field, fieldObject := object.(*types.Var)
	if !fieldObject || !field.IsField() {
		return false, nil
	}
	keyValueOccurrence, present :=
		verifier.expected.occurrences[occurrence.Parent()]
	if !present ||
		keyValueOccurrence.Kind() != catalog.KindKeyValueExpr {
		return false, nil
	}
	compositeOccurrence, present :=
		verifier.expected.occurrences[keyValueOccurrence.Parent()]
	if !present ||
		compositeOccurrence.Kind() != catalog.KindCompositeLit {
		return false, nil
	}
	node, present := verifier.index.OccurrenceNode(
		compositeOccurrence.ID(),
	)
	composite, compositeNode := node.(*ast.CompositeLit)
	if !present || !compositeNode {
		return true, fmt.Errorf(
			"composite field %s has no owning literal",
			id,
		)
	}
	value, present := verifier.view.TypeOf(composite)
	if !present || value.Type == nil {
		return true, fmt.Errorf(
			"composite field %s has no checker type",
			id,
		)
	}
	owner := stripCheckerPointer(value.Type)
	underlying, structType := types.Unalias(owner).
		Underlying().(*types.Struct)
	if !structType {
		return true, fmt.Errorf(
			"composite field %s owner is not a struct",
			id,
		)
	}
	ordinal := -1
	for index := 0; index < underlying.NumFields(); index++ {
		if underlying.Field(index) == field {
			ordinal = index
			break
		}
	}
	if ordinal < 0 {
		return true, fmt.Errorf(
			"composite field %s is absent from checker owner",
			id,
		)
	}
	if named, namedOwner := owner.(*types.Named); namedOwner {
		owner = named.Origin()
	}
	if err := verifier.types.verify(id.OwnerType(), owner); err != nil {
		return true, fmt.Errorf(
			"composite field %s owner: %w",
			id, err,
		)
	}
	return true, verifier.verifyMemberDeclarationIdentity(
		id, field, owner, ordinal,
	)
}

func independentSelectionOwner(
	checker *types.Selection,
) (types.Type, error) {
	if function, ok := checker.Obj().(*types.Func); ok {
		origin := function.Origin()
		signature, _ := origin.Type().(*types.Signature)
		if signature == nil || signature.Recv() == nil {
			return nil, fmt.Errorf(
				"selected method %s has no origin receiver",
				function.Name(),
			)
		}
		owner := stripCheckerPointer(signature.Recv().Type())
		if named, ok := owner.(*types.Named); ok {
			return named.Origin(), nil
		}
		return owner, nil
	}
	current := stripCheckerPointer(checker.Recv())
	index := checker.Index()
	for _, part := range index[:len(index)-1] {
		underlying, ok := types.Unalias(current).Underlying().(*types.Struct)
		if !ok || part >= underlying.NumFields() {
			return nil, fmt.Errorf(
				"selection path crosses non-struct %T", current,
			)
		}
		current = stripCheckerPointer(
			underlying.Field(part).Type(),
		)
	}
	if named, ok := current.(*types.Named); ok {
		return named.Origin(), nil
	}
	return current, nil
}

func stripCheckerPointer(typ types.Type) types.Type {
	for {
		pointer, ok := types.Unalias(typ).(*types.Pointer)
		if !ok {
			return typ
		}
		typ = pointer.Elem()
	}
}

func (verifier *checkerSemanticVerifier) verifyInstance(
	node ast.Node,
	kind semantic.OperationKind,
	record semantic.Instance,
) error {
	if kind != semantic.OperationGenericInstantiate &&
		kind != semantic.OperationCall {
		if !record.IsZero() {
			return fmt.Errorf(
				"non-generic operation carries an instance",
			)
		}
		return nil
	}
	identifier := independentGenericNodeIdentifier(node)
	if identifier == nil {
		if !record.IsZero() {
			return fmt.Errorf("non-generic node carries instance")
		}
		return nil
	}
	checker, present := verifier.view.InstanceOf(identifier)
	if !present {
		if !record.IsZero() {
			return fmt.Errorf("non-instantiated node carries instance")
		}
		return nil
	}
	if record.IsZero() ||
		len(record.Types()) != checker.TypeArgs.Len() {
		return fmt.Errorf("generic argument count differs")
	}
	object, present := verifier.view.UseOf(identifier)
	if !present {
		return fmt.Errorf("generic identifier has no checker object")
	}
	if err := verifier.verifyObjectReference(
		record.Target(), object,
	); err != nil {
		return err
	}
	for index, typeID := range record.Types() {
		if err := verifier.types.verify(
			typeID, checker.TypeArgs.At(index),
		); err != nil {
			return err
		}
	}
	return verifier.types.verify(record.Signature(), checker.Type)
}

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
	occurrence structure.Occurrence,
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
