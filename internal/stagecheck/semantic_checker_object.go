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
)

func (verifier *checkerSemanticVerifier) verifyDeclarationReferenceIdentity(
	id identity.SemanticDeclarationID,
	object types.Object,
) error {
	if function, ok := object.(*types.Func); ok {
		object = function.Origin()
	}
	if id.Form() == identity.SemanticDeclarationMember {
		function, method := object.(*types.Func)
		if !method {
			return fmt.Errorf(
				"member declaration %s requires contextual owner evidence",
				id,
			)
		}
		signature, _ := function.Type().(*types.Signature)
		if signature == nil || signature.Recv() == nil {
			return fmt.Errorf(
				"member declaration %s requires contextual owner evidence",
				id,
			)
		}
		owner := independentOriginMemberOwner(
			signature.Recv().Type(),
		)
		if err := verifier.types.verify(
			id.OwnerType(), owner,
		); err != nil {
			return fmt.Errorf("method declaration owner: %w", err)
		}
		if err := verifier.verifyMemberDeclarationIdentity(
			id, object, owner, 0,
		); err != nil {
			return err
		}
		target, present := verifier.actual.ResolveDeclarationTarget(id)
		if !present {
			return fmt.Errorf(
				"method declaration target %s is absent", id,
			)
		}
		return verifier.verifyMemberTargetPayload(
			target, object, 0,
		)
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
	if declaration, present := verifier.declaration(
		selection.Object(),
	); present {
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
	verifier.types.indexCheckerDeclarationTypeParameters(object)
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
		verifier.expected.occurrences.get(occurrence.Parent())
	if !present ||
		keyValueOccurrence.Kind() != catalog.KindKeyValueExpr {
		return false, nil
	}
	compositeOccurrence, present :=
		verifier.expected.occurrences.get(keyValueOccurrence.Parent())
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
		return independentOriginMemberOwner(
			signature.Recv().Type(),
		), nil
	}
	current := independentOriginMemberOwner(checker.Recv())
	index := checker.Index()
	for _, part := range index[:len(index)-1] {
		underlying, ok := types.Unalias(current).Underlying().(*types.Struct)
		if !ok || part >= underlying.NumFields() {
			return nil, fmt.Errorf(
				"selection path crosses non-struct %T", current,
			)
		}
		current = independentOriginMemberOwner(
			underlying.Field(part).Type(),
		)
	}
	return independentOriginMemberOwner(current), nil
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
