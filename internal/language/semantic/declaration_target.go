package semantic

import (
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

type DeclarationTargetKind uint8

const (
	DeclarationTargetInvalid DeclarationTargetKind = iota
	DeclarationTargetStandalone
	DeclarationTargetField
	DeclarationTargetMethod
)

func (kind DeclarationTargetKind) Valid() bool {
	return kind >= DeclarationTargetStandalone &&
		kind <= DeclarationTargetMethod
}

type DeclarationTarget struct {
	kind        DeclarationTargetKind
	id          identity.SemanticDeclarationID
	owner       identity.SemanticTypeID
	declaration Declaration
	field       TypeField
	method      TypeMethod
}

func (target DeclarationTarget) Kind() DeclarationTargetKind {
	return target.kind
}

func (target DeclarationTarget) ID() identity.SemanticDeclarationID {
	return target.id
}

func (target DeclarationTarget) OwnerType() identity.SemanticTypeID {
	return target.owner
}

func (target DeclarationTarget) Standalone() (Declaration, bool) {
	return target.declaration,
		target.kind == DeclarationTargetStandalone
}

func (target DeclarationTarget) Field() (TypeField, bool) {
	return target.field, target.kind == DeclarationTargetField
}

func (target DeclarationTarget) Method() (TypeMethod, bool) {
	return target.method, target.kind == DeclarationTargetMethod
}

func (pkg Package) ResolveDeclarationTarget(
	id identity.SemanticDeclarationID,
) (DeclarationTarget, bool) {
	if id.IsZero() {
		return DeclarationTarget{}, false
	}
	if id.Form() != identity.SemanticDeclarationMember {
		declaration, present := pkg.Declaration(id)
		if !present {
			return DeclarationTarget{}, false
		}
		return DeclarationTarget{
			kind:        DeclarationTargetStandalone,
			id:          id,
			declaration: declaration,
		}, true
	}
	switch id.Class() {
	case identity.SemanticObjectField:
		return pkg.resolveFieldTarget(id)
	case identity.SemanticObjectMethod:
		return pkg.resolveMethodTarget(id)
	default:
		return DeclarationTarget{}, false
	}
}

func (pkg Package) resolveFieldTarget(
	id identity.SemanticDeclarationID,
) (DeclarationTarget, bool) {
	owner := id.OwnerType()
	current := pkg.identities.typeReference(owner)
	if current == 0 {
		return DeclarationTarget{}, false
	}
	for depth := 0; depth <= pkg.TypeCount(); depth++ {
		record, present := pkg.types.storedType(current)
		if !present {
			return DeclarationTarget{}, false
		}
		switch record.kind {
		case TypeNamed:
			nominal, err := payloadAt(
				pkg.types.nominal, record.payload,
			)
			if err != nil {
				return DeclarationTarget{}, false
			}
			current = nominal.target
		case TypeStruct:
			ordinal := id.Ordinal()
			relation, err := payloadAt(
				pkg.types.structs, record.payload,
			)
			if err != nil {
				return DeclarationTarget{}, false
			}
			field, present := pkg.types.fieldAt(relation, ordinal)
			if !present {
				return DeclarationTarget{}, false
			}
			candidate, err := identity.NewMemberDeclarationID(
				owner,
				pkg.identities.packageID(field.pkg),
				identity.SemanticObjectField,
				field.name,
				field.ordinal,
			)
			if err != nil || candidate != id {
				return DeclarationTarget{}, false
			}
			return DeclarationTarget{
				kind:  DeclarationTargetField,
				id:    id,
				owner: owner,
				field: pkg.types.projectField(
					pkg.identities, field,
				),
			}, true
		default:
			return DeclarationTarget{}, false
		}
	}
	return DeclarationTarget{}, false
}

func (pkg Package) resolveMethodTarget(
	id identity.SemanticDeclarationID,
) (DeclarationTarget, bool) {
	owner := id.OwnerType()
	current := pkg.identities.typeReference(owner)
	if current == 0 {
		return DeclarationTarget{}, false
	}
	for depth := 0; depth <= pkg.TypeCount(); depth++ {
		record, present := pkg.types.storedType(current)
		if !present {
			return DeclarationTarget{}, false
		}
		switch record.kind {
		case TypeNamed:
			nominal, err := payloadAt(
				pkg.types.nominal, record.payload,
			)
			if err != nil {
				return DeclarationTarget{}, false
			}
			if method, present := pkg.types.methodAt(
				nominal.methods,
				pkg.identities.packageReference(
					id.MemberPackage(),
				),
				id.Name(),
			); present {
				return methodDeclarationTarget(
					id,
					owner,
					pkg.types.projectMethod(
						pkg.identities, method,
					),
				)
			}
			current = nominal.target
		case TypeInterface:
			iface, err := payloadAt(
				pkg.types.interfaces, record.payload,
			)
			if err != nil {
				return DeclarationTarget{}, false
			}
			method, present := pkg.types.methodAt(
				iface.methods,
				pkg.identities.packageReference(
					id.MemberPackage(),
				),
				id.Name(),
			)
			if !present {
				return DeclarationTarget{}, false
			}
			return methodDeclarationTarget(
				id,
				owner,
				pkg.types.projectMethod(pkg.identities, method),
			)
		default:
			return DeclarationTarget{}, false
		}
	}
	return DeclarationTarget{}, false
}

func methodDeclarationTarget(
	id identity.SemanticDeclarationID,
	owner identity.SemanticTypeID,
	method TypeMethod,
) (DeclarationTarget, bool) {
	candidate, err := identity.NewMemberDeclarationID(
		owner,
		method.Package,
		identity.SemanticObjectMethod,
		method.Name,
		0,
	)
	if err != nil || candidate != id {
		return DeclarationTarget{}, false
	}
	return DeclarationTarget{
		kind: DeclarationTargetMethod,
		id:   id, owner: owner, method: method,
	}, true
}

func (store packageTypeStore) storedType(
	id typeRef,
) (storedType, bool) {
	index := sort.Search(len(store.records), func(index int) bool {
		return store.records[index].id >= id
	})
	if index == len(store.records) ||
		store.records[index].id != id {
		return storedType{}, false
	}
	return store.records[index], true
}

func (store packageTypeStore) fieldAt(
	relation typeFieldRange,
	ordinal int,
) (storedTypeField, bool) {
	if ordinal < 0 ||
		uint64(ordinal) >= relation.count ||
		relation.start > uint64(len(store.fields)) ||
		relation.count >
			uint64(len(store.fields))-relation.start {
		return storedTypeField{}, false
	}
	field := store.fields[relation.start+uint64(ordinal)]
	return field, field.ordinal == ordinal
}

func (store packageTypeStore) methodAt(
	relation typeMethodRange,
	pkg packageRef,
	name string,
) (storedTypeMethod, bool) {
	if relation.start > uint64(len(store.methods)) ||
		relation.count >
			uint64(len(store.methods))-relation.start {
		return storedTypeMethod{}, false
	}
	methods := store.methods[relation.start : relation.start+relation.count]
	index := sort.Search(len(methods), func(index int) bool {
		if methods[index].pkg != pkg {
			return methods[index].pkg >= pkg
		}
		return methods[index].name >= name
	})
	if index == len(methods) ||
		methods[index].pkg != pkg ||
		methods[index].name != name {
		return storedTypeMethod{}, false
	}
	return methods[index], true
}

func (store packageTypeStore) projectField(
	identities packageIdentityTable,
	field storedTypeField,
) TypeField {
	return TypeField{
		Name:     field.name,
		Package:  identities.packageID(field.pkg),
		Type:     identities.typeID(field.typeID),
		Embedded: field.embedded,
		Tag:      field.tag,
		Ordinal:  field.ordinal,
	}
}

func (store packageTypeStore) projectMethod(
	identities packageIdentityTable,
	method storedTypeMethod,
) TypeMethod {
	return TypeMethod{
		Name:      method.name,
		Package:   identities.packageID(method.pkg),
		Signature: identities.typeID(method.signature),
		Ordinal:   method.ordinal,
	}
}
