package identity

import (
	"encoding/hex"
	"fmt"
	"unicode"
	"unicode/utf8"
)

// SemanticTypeID identifies one complete canonical Go type descriptor by its
// full sha256 digest. The semantic package owns descriptor construction and
// collision checks; identity owns only the validated opaque identity.
type SemanticTypeID struct {
	digest string
}

func NewSemanticTypeID(digest string) (SemanticTypeID, error) {
	raw, err := hex.DecodeString(digest)
	if err != nil || len(raw) != 32 || hex.EncodeToString(raw) != digest {
		return SemanticTypeID{}, &Error{
			Identity: "semantic-type",
			Value:    digest,
			Reason:   "digest must be exactly 64 lowercase hexadecimal characters",
		}
	}
	return SemanticTypeID{digest: digest}, nil
}

func (id SemanticTypeID) IsZero() bool { return id == SemanticTypeID{} }
func (id SemanticTypeID) Digest() string {
	return id.digest
}
func (id SemanticTypeID) String() string {
	if id.IsZero() {
		return ""
	}
	return "semantic-type/sha256:" + id.digest
}

// SemanticObjectClass is the closed identity class of a declared Go object.
// It is an identity component, not a semantic-resolution result.
type SemanticObjectClass uint8

const (
	SemanticObjectInvalid SemanticObjectClass = iota
	SemanticObjectPackage
	SemanticObjectConstant
	SemanticObjectType
	SemanticObjectAlias
	SemanticObjectVariable
	SemanticObjectFunction
	SemanticObjectMethod
	SemanticObjectField
	SemanticObjectBuiltin
	SemanticObjectNil

	semanticObjectClassCount = SemanticObjectNil
)

var semanticObjectClassNames = [semanticObjectClassCount + 1]string{
	SemanticObjectPackage:  "package",
	SemanticObjectConstant: "constant",
	SemanticObjectType:     "type",
	SemanticObjectAlias:    "alias",
	SemanticObjectVariable: "variable",
	SemanticObjectFunction: "function",
	SemanticObjectMethod:   "method",
	SemanticObjectField:    "field",
	SemanticObjectBuiltin:  "builtin",
	SemanticObjectNil:      "nil",
}

func (class SemanticObjectClass) Valid() bool {
	return class > SemanticObjectInvalid &&
		class <= semanticObjectClassCount
}

func (class SemanticObjectClass) String() string {
	if class.Valid() {
		return semanticObjectClassNames[class]
	}
	return fmt.Sprintf(
		"identity.SemanticObjectClass(%d)", uint8(class),
	)
}

// SemanticDeclarationForm is the closed anchor form of a declaration.
type SemanticDeclarationForm uint8

const (
	SemanticDeclarationInvalid SemanticDeclarationForm = iota
	SemanticDeclarationPackageObject
	SemanticDeclarationMember
	SemanticDeclarationPredeclared
	SemanticDeclarationOccurrence
)

// SemanticDeclarationID is the canonical identity of a package declaration,
// member declaration, or predeclared object.
type SemanticDeclarationID struct {
	form        SemanticDeclarationForm
	pkg         PackageID
	ownerType   SemanticTypeID
	memberPkg   PackageID
	class       SemanticObjectClass
	name        string
	ordinal     int
	predeclared uint16
	owner       OccurrenceID
	occurrence  OccurrenceID
}

func NewPackageDeclarationID(
	pkg PackageID,
	class SemanticObjectClass,
	name string,
) (SemanticDeclarationID, error) {
	if pkg.IsZero() || !class.Valid() || name == "" || hasReserved(name) {
		return SemanticDeclarationID{}, &Error{
			Identity: "semantic-declaration",
			Value:    name,
			Reason:   "package declaration requires package, closed class, and canonical name",
		}
	}
	if class == SemanticObjectPackage ||
		class == SemanticObjectField ||
		class == SemanticObjectMethod ||
		class == SemanticObjectNil {
		return SemanticDeclarationID{}, &Error{
			Identity: "semantic-declaration",
			Value:    name,
			Reason:   "object class is not a package declaration class",
		}
	}
	return SemanticDeclarationID{
		form:  SemanticDeclarationPackageObject,
		pkg:   pkg,
		class: class,
		name:  name,
	}, nil
}

func NewMemberDeclarationID(
	owner SemanticTypeID,
	pkg PackageID,
	class SemanticObjectClass,
	name string,
	ordinal int,
) (SemanticDeclarationID, error) {
	exported := semanticNameExported(name)
	if owner.IsZero() ||
		(class != SemanticObjectField && class != SemanticObjectMethod) ||
		name == "" ||
		hasReserved(name) ||
		ordinal < 0 ||
		(!exported && pkg.IsZero()) {
		return SemanticDeclarationID{}, &Error{
			Identity: "semantic-declaration",
			Value:    name,
			Reason:   "member declaration requires owner type, unexported-member package, field/method class, canonical name, and ordinal",
		}
	}
	if class == SemanticObjectMethod && ordinal != 0 {
		return SemanticDeclarationID{}, &Error{
			Identity: "semantic-declaration",
			Value:    name,
			Reason:   "method identity is receiver type plus name and has no positional ordinal",
		}
	}
	if exported {
		pkg = PackageID{}
	}
	return SemanticDeclarationID{
		form:      SemanticDeclarationMember,
		ownerType: owner,
		memberPkg: pkg,
		class:     class,
		name:      name,
		ordinal:   ordinal,
	}, nil
}

func semanticNameExported(name string) bool {
	first, _ := utf8.DecodeRuneInString(name)
	return first != utf8.RuneError && unicode.IsUpper(first)
}

func NewPredeclaredDeclarationID(
	predeclared uint16,
	class SemanticObjectClass,
) (SemanticDeclarationID, error) {
	if predeclared == 0 ||
		(class != SemanticObjectType &&
			class != SemanticObjectConstant &&
			class != SemanticObjectBuiltin &&
			class != SemanticObjectNil) {
		return SemanticDeclarationID{}, &Error{
			Identity: "semantic-declaration",
			Value:    fmt.Sprint(predeclared),
			Reason:   "predeclared declaration requires pinned identity and compatible class",
		}
	}
	return SemanticDeclarationID{
		form:        SemanticDeclarationPredeclared,
		class:       class,
		predeclared: predeclared,
	}, nil
}

func NewOccurrenceDeclarationID(
	owner OccurrenceID,
	occurrence OccurrenceID,
	class SemanticObjectClass,
	name string,
	ordinal int,
) (SemanticDeclarationID, error) {
	if owner.IsZero() ||
		occurrence.IsZero() ||
		owner.Span().File() != occurrence.Span().File() ||
		!class.Valid() ||
		name == "" ||
		hasReserved(name) ||
		ordinal < 0 {
		return SemanticDeclarationID{}, &Error{
			Identity: "semantic-declaration",
			Value:    name,
			Reason:   "occurrence declaration requires same-file owner/declaration, closed class, canonical name, and ordinal",
		}
	}
	if class != SemanticObjectConstant &&
		class != SemanticObjectType &&
		class != SemanticObjectAlias {
		return SemanticDeclarationID{}, &Error{
			Identity: "semantic-declaration",
			Value:    name,
			Reason:   "object class is not an occurrence declaration class",
		}
	}
	return SemanticDeclarationID{
		form:       SemanticDeclarationOccurrence,
		class:      class,
		name:       name,
		ordinal:    ordinal,
		owner:      owner,
		occurrence: occurrence,
	}, nil
}

func (id SemanticDeclarationID) IsZero() bool {
	return id == SemanticDeclarationID{}
}
func (id SemanticDeclarationID) Form() SemanticDeclarationForm {
	return id.form
}
func (id SemanticDeclarationID) Package() PackageID {
	return id.pkg
}
func (id SemanticDeclarationID) OwnerType() SemanticTypeID {
	return id.ownerType
}
func (id SemanticDeclarationID) MemberPackage() PackageID {
	return id.memberPkg
}
func (id SemanticDeclarationID) Class() SemanticObjectClass {
	return id.class
}
func (id SemanticDeclarationID) Name() string {
	return id.name
}
func (id SemanticDeclarationID) Ordinal() int {
	return id.ordinal
}
func (id SemanticDeclarationID) Predeclared() uint16 {
	return id.predeclared
}
func (id SemanticDeclarationID) OwnerOccurrence() OccurrenceID {
	return id.owner
}
func (id SemanticDeclarationID) Occurrence() OccurrenceID {
	return id.occurrence
}
func (id SemanticDeclarationID) String() string {
	switch id.form {
	case SemanticDeclarationPackageObject:
		return fmt.Sprintf(
			"%s#declaration/%s/%s",
			id.pkg, id.class, id.name,
		)
	case SemanticDeclarationMember:
		namespace := "exported"
		if !id.memberPkg.IsZero() {
			namespace = id.memberPkg.String()
		}
		if id.class == SemanticObjectMethod {
			return fmt.Sprintf(
				"%s#member/%s/%s/%s",
				id.ownerType, namespace, id.class, id.name,
			)
		}
		return fmt.Sprintf(
			"%s#member/%s/%s/%d/%s",
			id.ownerType, namespace, id.class, id.ordinal, id.name,
		)
	case SemanticDeclarationPredeclared:
		return fmt.Sprintf(
			"lang#predeclared/%d/%s",
			id.predeclared, id.class,
		)
	case SemanticDeclarationOccurrence:
		return fmt.Sprintf(
			"%s#local-declaration/%s/%d/%s/%s",
			id.owner, id.class, id.ordinal, id.name, id.occurrence,
		)
	default:
		return ""
	}
}

// SemanticBindingRole is the closed identity role of a lexical binding.
type SemanticBindingRole uint8

const (
	SemanticBindingInvalid SemanticBindingRole = iota
	SemanticBindingImport
	SemanticBindingLocal
	SemanticBindingReceiver
	SemanticBindingParameter
	SemanticBindingResult
	SemanticBindingTypeParameter
	SemanticBindingRange
	SemanticBindingTypeSwitch
	SemanticBindingLabel
	SemanticBindingImplicit

	semanticBindingRoleCount = SemanticBindingImplicit
)

var semanticBindingRoleNames = [semanticBindingRoleCount + 1]string{
	SemanticBindingImport:        "import",
	SemanticBindingLocal:         "local",
	SemanticBindingReceiver:      "receiver",
	SemanticBindingParameter:     "parameter",
	SemanticBindingResult:        "result",
	SemanticBindingTypeParameter: "type-parameter",
	SemanticBindingRange:         "range",
	SemanticBindingTypeSwitch:    "type-switch",
	SemanticBindingLabel:         "label",
	SemanticBindingImplicit:      "implicit",
}

func (role SemanticBindingRole) Valid() bool {
	return role > SemanticBindingInvalid &&
		role <= semanticBindingRoleCount
}

func (role SemanticBindingRole) String() string {
	if role.Valid() {
		return semanticBindingRoleNames[role]
	}
	return fmt.Sprintf(
		"identity.SemanticBindingRole(%d)", uint8(role),
	)
}

// SemanticBindingID identifies one lexical or implicit binding. Owner is the
// nearest canonical scope/definition occurrence. Declaration may be zero only
// for an unnamed or implicit binding, in which case role and ordinal remain
// complete identity evidence.
type SemanticBindingID struct {
	owner       OccurrenceID
	declaration OccurrenceID
	role        SemanticBindingRole
	ordinal     int
}

func NewSemanticBindingID(
	owner OccurrenceID,
	declaration OccurrenceID,
	role SemanticBindingRole,
	ordinal int,
) (SemanticBindingID, error) {
	if owner.IsZero() || !role.Valid() || ordinal < 0 {
		return SemanticBindingID{}, &Error{
			Identity: "semantic-binding",
			Value:    owner.String(),
			Reason:   "binding requires owner, closed role, and non-negative ordinal",
		}
	}
	if !declaration.IsZero() &&
		declaration.Span().File() != owner.Span().File() {
		return SemanticBindingID{}, &Error{
			Identity: "semantic-binding",
			Value:    declaration.String(),
			Reason:   "binding owner and declaration must share a source file",
		}
	}
	return SemanticBindingID{
		owner:       owner,
		declaration: declaration,
		role:        role,
		ordinal:     ordinal,
	}, nil
}

func (id SemanticBindingID) IsZero() bool {
	return id == SemanticBindingID{}
}
func (id SemanticBindingID) Owner() OccurrenceID {
	return id.owner
}
func (id SemanticBindingID) Declaration() OccurrenceID {
	return id.declaration
}
func (id SemanticBindingID) Role() SemanticBindingRole {
	return id.role
}
func (id SemanticBindingID) Ordinal() int {
	return id.ordinal
}
func (id SemanticBindingID) String() string {
	if id.IsZero() {
		return ""
	}
	declaration := "unnamed"
	if !id.declaration.IsZero() {
		declaration = id.declaration.String()
	}
	return fmt.Sprintf(
		"%s#binding/%s/%d/%s",
		id.owner, id.role, id.ordinal, declaration,
	)
}

// OperationID identifies either the one semantic operation owned by a source
// occurrence or one unspelled operation of an implicit definition.
type OperationID struct {
	definition DefinitionID
	occurrence OccurrenceID
	implicit   ImplicitDefinitionOp
	ordinal    int
}

func NewOperationID(
	definition DefinitionID,
	occurrence OccurrenceID,
) (OperationID, error) {
	if definition.IsZero() ||
		occurrence.IsZero() ||
		definition.Kind() == DefinitionImplicit ||
		definition.File() != occurrence.Span().File() {
		return OperationID{}, &Error{
			Identity: "operation",
			Reason:   "source operation requires same-file source definition and occurrence",
		}
	}
	return OperationID{
		definition: definition,
		occurrence: occurrence,
	}, nil
}

func NewImplicitOperationID(
	definition DefinitionID,
	operation ImplicitDefinitionOp,
	ordinal int,
) (OperationID, error) {
	if definition.IsZero() ||
		definition.Kind() != DefinitionImplicit ||
		definition.ImplicitOp() != operation ||
		!operation.Valid() ||
		ordinal < 0 {
		return OperationID{}, &Error{
			Identity: "operation",
			Value:    definition.String(),
			Reason:   "implicit operation requires matching implicit definition, operation, and ordinal",
		}
	}
	return OperationID{
		definition: definition,
		implicit:   operation,
		ordinal:    ordinal,
	}, nil
}

func (id OperationID) IsZero() bool { return id == OperationID{} }
func (id OperationID) Definition() DefinitionID {
	return id.definition
}
func (id OperationID) Occurrence() OccurrenceID {
	return id.occurrence
}
func (id OperationID) ImplicitOp() ImplicitDefinitionOp {
	return id.implicit
}
func (id OperationID) Ordinal() int { return id.ordinal }
func (id OperationID) Source() bool {
	return !id.occurrence.IsZero()
}
func (id OperationID) String() string {
	if id.IsZero() {
		return ""
	}
	if id.implicit.Valid() {
		return fmt.Sprintf(
			"%s#operation/implicit/%s/%d",
			id.definition, id.implicit, id.ordinal,
		)
	}
	return id.definition.String() + "#operation/" +
		id.occurrence.String()
}

// UnsupportedID identifies the explicit unsupported record for an occurrence.
type UnsupportedID struct {
	definition DefinitionID
	occurrence OccurrenceID
}

func NewUnsupportedID(
	definition DefinitionID,
	occurrence OccurrenceID,
) (UnsupportedID, error) {
	if definition.IsZero() || occurrence.IsZero() {
		return UnsupportedID{}, &Error{
			Identity: "unsupported",
			Reason:   "unsupported identity requires definition and occurrence",
		}
	}
	return UnsupportedID{
		definition: definition,
		occurrence: occurrence,
	}, nil
}

func (id UnsupportedID) IsZero() bool {
	return id == UnsupportedID{}
}
func (id UnsupportedID) Definition() DefinitionID {
	return id.definition
}
func (id UnsupportedID) Occurrence() OccurrenceID {
	return id.occurrence
}
func (id UnsupportedID) String() string {
	if id.IsZero() {
		return ""
	}
	return id.definition.String() + "#unsupported/" +
		id.occurrence.String()
}
