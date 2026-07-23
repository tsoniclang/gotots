package semantic

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

type ConstantKind uint8

const (
	ConstantInvalid ConstantKind = iota
	ConstantBool
	ConstantString
	ConstantInteger
	ConstantFloat
	ConstantComplex
)

func (kind ConstantKind) Valid() bool {
	return kind >= ConstantBool && kind <= ConstantComplex
}

type Constant struct {
	kind  ConstantKind
	exact string
}

func NewConstant(
	kind ConstantKind,
	exact string,
) (Constant, error) {
	if !kind.Valid() || exact == "" {
		return Constant{}, fmt.Errorf(
			"constant requires closed kind and exact value",
		)
	}
	return Constant{kind: kind, exact: exact}, nil
}

func (constant Constant) IsZero() bool {
	return constant == Constant{}
}
func (constant Constant) Kind() ConstantKind { return constant.kind }
func (constant Constant) Exact() string      { return constant.exact }

type ObjectReference struct {
	kind        ObjectReferenceKind
	declaration identity.SemanticDeclarationID
	binding     identity.SemanticBindingID
}

func NoObjectReference() ObjectReference {
	return ObjectReference{kind: ObjectReferenceNone}
}

func DeclarationReference(
	declaration identity.SemanticDeclarationID,
) (ObjectReference, error) {
	if declaration.IsZero() {
		return ObjectReference{}, fmt.Errorf(
			"declaration reference requires identity",
		)
	}
	return ObjectReference{
		kind:        ObjectReferenceDeclaration,
		declaration: declaration,
	}, nil
}

func BindingReference(
	binding identity.SemanticBindingID,
) (ObjectReference, error) {
	if binding.IsZero() {
		return ObjectReference{}, fmt.Errorf(
			"binding reference requires identity",
		)
	}
	return ObjectReference{
		kind:    ObjectReferenceBinding,
		binding: binding,
	}, nil
}

func (reference ObjectReference) Kind() ObjectReferenceKind {
	return reference.kind
}
func (reference ObjectReference) Declaration() identity.SemanticDeclarationID {
	return reference.declaration
}
func (reference ObjectReference) Binding() identity.SemanticBindingID {
	return reference.binding
}
func (reference ObjectReference) Valid() bool {
	switch reference.kind {
	case ObjectReferenceNone:
		return reference.declaration.IsZero() &&
			reference.binding.IsZero()
	case ObjectReferenceDeclaration:
		return !reference.declaration.IsZero() &&
			reference.binding.IsZero()
	case ObjectReferenceBinding:
		return reference.declaration.IsZero() &&
			!reference.binding.IsZero()
	default:
		return false
	}
}

type Declaration struct {
	id        identity.SemanticDeclarationID
	pkg       identity.PackageID
	class     identity.SemanticObjectClass
	name      string
	typeID    identity.SemanticTypeID
	source    identity.OccurrenceID
	exported  bool
	constant  Constant
	authority Authority
}

func NewDeclaration(
	id identity.SemanticDeclarationID,
	pkg identity.PackageID,
	class identity.SemanticObjectClass,
	name string,
	typeID identity.SemanticTypeID,
	source identity.OccurrenceID,
	exported bool,
	constant Constant,
	authority Authority,
) (Declaration, error) {
	if id.IsZero() ||
		!class.Valid() ||
		name == "" ||
		typeID.IsZero() ||
		!authority.Valid() {
		return Declaration{}, fmt.Errorf(
			"declaration requires identity, class, name, type, and authority",
		)
	}
	if class == identity.SemanticObjectConstant {
		if constant.IsZero() {
			return Declaration{}, fmt.Errorf(
				"constant declaration requires exact value",
			)
		}
	} else if !constant.IsZero() {
		return Declaration{}, fmt.Errorf(
			"non-constant declaration carries a constant value",
		)
	}
	if id.Form() != identity.SemanticDeclarationPredeclared &&
		pkg.IsZero() {
		return Declaration{}, fmt.Errorf(
			"non-predeclared declaration requires package",
		)
	}
	return Declaration{
		id: id, pkg: pkg, class: class, name: name,
		typeID: typeID, source: source, exported: exported,
		constant: constant, authority: authority,
	}, nil
}

func (record Declaration) ID() identity.SemanticDeclarationID {
	return record.id
}
func (record Declaration) Package() identity.PackageID { return record.pkg }
func (record Declaration) Class() identity.SemanticObjectClass {
	return record.class
}
func (record Declaration) Name() string                  { return record.name }
func (record Declaration) Type() identity.SemanticTypeID { return record.typeID }
func (record Declaration) Source() identity.OccurrenceID { return record.source }
func (record Declaration) Exported() bool                { return record.exported }
func (record Declaration) Constant() Constant            { return record.constant }
func (record Declaration) Authority() Authority          { return record.authority }

type Binding struct {
	id         identity.SemanticBindingID
	pkg        identity.PackageID
	definition identity.DefinitionID
	role       identity.SemanticBindingRole
	name       string
	typeID     identity.SemanticTypeID
	source     identity.OccurrenceID
	captures   []identity.DefinitionID
	authority  Authority
}

func NewBinding(
	id identity.SemanticBindingID,
	pkg identity.PackageID,
	definition identity.DefinitionID,
	role identity.SemanticBindingRole,
	name string,
	typeID identity.SemanticTypeID,
	source identity.OccurrenceID,
	captures []identity.DefinitionID,
	authority Authority,
) (Binding, error) {
	if id.IsZero() ||
		pkg.IsZero() ||
		!role.Valid() ||
		typeID.IsZero() ||
		!authority.Valid() {
		return Binding{}, fmt.Errorf(
			"binding requires identity, package, role, type, and authority",
		)
	}
	if id.Role() != role {
		return Binding{}, fmt.Errorf(
			"binding role disagrees with identity",
		)
	}
	if source.IsZero() != id.Declaration().IsZero() ||
		(!source.IsZero() && source != id.Declaration()) {
		return Binding{}, fmt.Errorf(
			"binding source disagrees with identity",
		)
	}
	seen := map[identity.DefinitionID]bool{}
	if definition.IsZero() && len(captures) != 0 {
		return Binding{}, fmt.Errorf(
			"non-executable binding cannot carry capture owners",
		)
	}
	for _, capture := range captures {
		if capture.IsZero() || seen[capture] {
			return Binding{}, fmt.Errorf(
				"binding capture set is invalid",
			)
		}
		seen[capture] = true
	}
	return Binding{
		id: id, pkg: pkg, definition: definition, role: role, name: name,
		typeID: typeID, source: source,
		captures:  append([]identity.DefinitionID(nil), captures...),
		authority: authority,
	}, nil
}

func (record Binding) ID() identity.SemanticBindingID {
	return record.id
}
func (record Binding) Package() identity.PackageID {
	return record.pkg
}
func (record Binding) Definition() identity.DefinitionID {
	return record.definition
}
func (record Binding) Role() identity.SemanticBindingRole {
	return record.role
}
func (record Binding) Name() string                  { return record.name }
func (record Binding) Type() identity.SemanticTypeID { return record.typeID }
func (record Binding) Source() identity.OccurrenceID { return record.source }
func (record Binding) CapturedBy() []identity.DefinitionID {
	return append([]identity.DefinitionID(nil), record.captures...)
}
func (record Binding) Authority() Authority { return record.authority }

// TypeWitness binds one canonical type descriptor to the exact semantic
// authority that materialized it for one package. Type descriptors are pure
// values and may be shared; authority remains an explicit package relation.
type TypeWitness struct {
	pkg       identity.PackageID
	typeID    identity.SemanticTypeID
	authority Authority
}

func NewTypeWitness(
	pkg identity.PackageID,
	typeID identity.SemanticTypeID,
	authority Authority,
) (TypeWitness, error) {
	if pkg.IsZero() || typeID.IsZero() || !authority.Valid() {
		return TypeWitness{}, fmt.Errorf(
			"type witness requires package, type, and authority",
		)
	}
	return TypeWitness{
		pkg: pkg, typeID: typeID, authority: authority,
	}, nil
}

func (record TypeWitness) Package() identity.PackageID {
	return record.pkg
}
func (record TypeWitness) Type() identity.SemanticTypeID {
	return record.typeID
}
func (record TypeWitness) Authority() Authority {
	return record.authority
}

type Selection struct {
	kind     SelectionKind
	receiver identity.SemanticTypeID
	object   identity.SemanticDeclarationID
	index    []int
	indirect bool
}

func NewSelection(
	kind SelectionKind,
	receiver identity.SemanticTypeID,
	object identity.SemanticDeclarationID,
	index []int,
	indirect bool,
) (Selection, error) {
	if !kind.Valid() ||
		receiver.IsZero() ||
		object.IsZero() ||
		len(index) == 0 {
		return Selection{}, fmt.Errorf(
			"selection requires kind, receiver, object, and index",
		)
	}
	for _, part := range index {
		if part < 0 {
			return Selection{}, fmt.Errorf(
				"selection index must be non-negative",
			)
		}
	}
	return Selection{
		kind: kind, receiver: receiver, object: object,
		index: append([]int(nil), index...), indirect: indirect,
	}, nil
}

func (selection Selection) IsZero() bool {
	return selection.kind == SelectionInvalid &&
		selection.receiver.IsZero() &&
		selection.object.IsZero() &&
		len(selection.index) == 0 &&
		!selection.indirect
}
func (selection Selection) Kind() SelectionKind { return selection.kind }
func (selection Selection) Receiver() identity.SemanticTypeID {
	return selection.receiver
}
func (selection Selection) Object() identity.SemanticDeclarationID {
	return selection.object
}
func (selection Selection) Index() []int {
	return append([]int(nil), selection.index...)
}
func (selection Selection) Indirect() bool { return selection.indirect }

type Instance struct {
	target    ObjectReference
	types     []identity.SemanticTypeID
	signature identity.SemanticTypeID
}

func NewInstance(
	target ObjectReference,
	types []identity.SemanticTypeID,
	signature identity.SemanticTypeID,
) (Instance, error) {
	if !target.Valid() ||
		target.Kind() == ObjectReferenceNone ||
		len(types) == 0 ||
		signature.IsZero() {
		return Instance{}, fmt.Errorf(
			"generic instance requires target, arguments, and signature",
		)
	}
	for _, typeID := range types {
		if typeID.IsZero() {
			return Instance{}, fmt.Errorf(
				"generic instance has zero type argument",
			)
		}
	}
	return Instance{
		target:    target,
		types:     append([]identity.SemanticTypeID(nil), types...),
		signature: signature,
	}, nil
}

func (instance Instance) IsZero() bool {
	return !instance.target.Valid()
}
func (instance Instance) Target() ObjectReference { return instance.target }
func (instance Instance) Types() []identity.SemanticTypeID {
	return append([]identity.SemanticTypeID(nil), instance.types...)
}
func (instance Instance) Signature() identity.SemanticTypeID {
	return instance.signature
}

type ImplicitOperation struct {
	kind   catalog.ImplicitOp
	source identity.SemanticTypeID
	target identity.SemanticTypeID
}

func NewImplicitOperation(
	kind catalog.ImplicitOp,
	source identity.SemanticTypeID,
	target identity.SemanticTypeID,
) (ImplicitOperation, error) {
	if !kind.Valid() {
		return ImplicitOperation{}, fmt.Errorf(
			"implicit operation requires catalog kind",
		)
	}
	return ImplicitOperation{
		kind: kind, source: source, target: target,
	}, nil
}

func (operation ImplicitOperation) Kind() catalog.ImplicitOp {
	return operation.kind
}
func (operation ImplicitOperation) Source() identity.SemanticTypeID {
	return operation.source
}
func (operation ImplicitOperation) Target() identity.SemanticTypeID {
	return operation.target
}
