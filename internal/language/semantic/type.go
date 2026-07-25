package semantic

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/tsoniclang/gotots/internal/identity"
)

type TypeField struct {
	Name     string
	Package  identity.PackageID
	Type     identity.SemanticTypeID
	Embedded bool
	Tag      string
	Ordinal  int
}

type TypeMethod struct {
	Name      string
	Package   identity.PackageID
	Signature identity.SemanticTypeID
	Ordinal   int
}

type TypeTerm struct {
	Tilde bool
	Type  identity.SemanticTypeID
}

type Signature struct {
	Receiver               identity.SemanticTypeID
	ReceiverTypeParameters []identity.SemanticTypeID
	TypeParameters         []identity.SemanticTypeID
	Parameters             []identity.SemanticTypeID
	Results                []identity.SemanticTypeID
	Variadic               bool
}

type TypeSpec struct {
	Kind TypeKind

	Basic       BasicKind
	Declaration identity.SemanticDeclarationID
	Parameter   TypeParameterOwner
	Arguments   []identity.SemanticTypeID
	Underlying  identity.SemanticTypeID
	Target      identity.SemanticTypeID
	Constraint  identity.SemanticTypeID
	Element     identity.SemanticTypeID
	Key         identity.SemanticTypeID
	Length      int64
	Direction   ChannelDirection
	Signature   Signature
	Fields      []TypeField
	Methods     []TypeMethod
	Embeddeds   []identity.SemanticTypeID
	Terms       []TypeTerm
	TypeSet     TypeSetKind
	Comparable  bool
	Elements    []identity.SemanticTypeID
}

type Type struct {
	id   identity.SemanticTypeID
	spec TypeSpec
}

func NewType(spec TypeSpec) (Type, error) {
	spec = cloneTypeSpec(spec)
	if err := validateTypeSpec(spec); err != nil {
		return Type{}, err
	}
	id, err := semanticTypeID(spec)
	if err != nil {
		return Type{}, err
	}
	return Type{
		id:   id,
		spec: spec,
	}, nil
}

// NominalTypeID returns the identity available before a named or alias
// descriptor's recursively referenced content is built.
func NominalTypeID(
	kind TypeKind,
	declaration identity.SemanticDeclarationID,
	arguments []identity.SemanticTypeID,
) (identity.SemanticTypeID, error) {
	spec := TypeSpec{
		Kind:        kind,
		Declaration: declaration,
		Arguments:   append([]identity.SemanticTypeID(nil), arguments...),
	}
	switch kind {
	case TypeNamed, TypeAlias:
		if declaration.IsZero() {
			return identity.SemanticTypeID{}, fmt.Errorf(
				"%s nominal identity requires declaration only", kind,
			)
		}
	default:
		return identity.SemanticTypeID{}, fmt.Errorf(
			"%s is not nominal", kind,
		)
	}
	return semanticTypeID(spec)
}

func TypeParameterTypeID(
	owner TypeParameterOwner,
) (identity.SemanticTypeID, error) {
	if owner.IsZero() {
		return identity.SemanticTypeID{}, fmt.Errorf(
			"type-parameter identity requires canonical owner",
		)
	}
	spec := TypeSpec{Kind: TypeParameter, Parameter: owner}
	return semanticTypeID(spec)
}

func (record Type) ID() identity.SemanticTypeID { return record.id }
func (record Type) Kind() TypeKind              { return record.spec.Kind }
func (record Type) Spec() TypeSpec              { return cloneTypeSpec(record.spec) }

func cloneTypeSpec(spec TypeSpec) TypeSpec {
	spec.Arguments = append(
		[]identity.SemanticTypeID(nil), spec.Arguments...,
	)
	spec.Signature.TypeParameters = append(
		[]identity.SemanticTypeID(nil),
		spec.Signature.TypeParameters...,
	)
	spec.Signature.ReceiverTypeParameters = append(
		[]identity.SemanticTypeID(nil),
		spec.Signature.ReceiverTypeParameters...,
	)
	spec.Signature.Parameters = append(
		[]identity.SemanticTypeID(nil),
		spec.Signature.Parameters...,
	)
	spec.Signature.Results = append(
		[]identity.SemanticTypeID(nil),
		spec.Signature.Results...,
	)
	spec.Fields = append([]TypeField(nil), spec.Fields...)
	spec.Methods = append([]TypeMethod(nil), spec.Methods...)
	spec.Embeddeds = append(
		[]identity.SemanticTypeID(nil), spec.Embeddeds...,
	)
	spec.Terms = append([]TypeTerm(nil), spec.Terms...)
	spec.Elements = append(
		[]identity.SemanticTypeID(nil), spec.Elements...,
	)
	return spec
}

func validateTypeSpec(spec TypeSpec) error {
	if !spec.Kind.Valid() {
		return fmt.Errorf("semantic type requires a closed kind")
	}
	if err := validateTypeMembers(spec); err != nil {
		return err
	}
	switch spec.Kind {
	case TypeBasic:
		if !spec.Basic.Valid() {
			return fmt.Errorf("basic type requires a closed basic kind")
		}
	case TypeNamed:
		if spec.Declaration.IsZero() ||
			spec.Underlying.IsZero() {
			return fmt.Errorf(
				"named type requires declaration and underlying type",
			)
		}
		if err := validateMethods(spec.Methods); err != nil {
			return err
		}
	case TypeAlias:
		if spec.Declaration.IsZero() || spec.Target.IsZero() {
			return fmt.Errorf(
				"alias type requires declaration and target",
			)
		}
	case TypeParameter:
		if spec.Parameter.IsZero() || spec.Constraint.IsZero() {
			return fmt.Errorf(
				"type parameter requires canonical owner and constraint",
			)
		}
	case TypePointer, TypeSlice:
		if spec.Element.IsZero() {
			return fmt.Errorf("%s type requires element", spec.Kind)
		}
	case TypeArray:
		if spec.Element.IsZero() || spec.Length < 0 {
			return fmt.Errorf(
				"array type requires element and non-negative length",
			)
		}
	case TypeMap:
		if spec.Key.IsZero() || spec.Element.IsZero() {
			return fmt.Errorf("map type requires key and element")
		}
	case TypeChannel:
		if spec.Element.IsZero() || !spec.Direction.Valid() {
			return fmt.Errorf(
				"channel type requires element and direction",
			)
		}
	case TypeSignature:
		if err := validateSignature(spec.Signature); err != nil {
			return err
		}
	case TypeStruct:
		for index, field := range spec.Fields {
			if field.Name == "" ||
				field.Type.IsZero() ||
				(!semanticNameExported(field.Name) &&
					field.Package.IsZero()) ||
				(semanticNameExported(field.Name) &&
					!field.Package.IsZero()) ||
				field.Ordinal != index {
				return fmt.Errorf(
					"struct field %d is not canonical", index,
				)
			}
		}
	case TypeInterface:
		if err := validateMethods(spec.Methods); err != nil {
			return err
		}
		if !spec.TypeSet.Valid() ||
			(spec.TypeSet == TypeSetFinite) !=
				(len(spec.Terms) != 0) {
			return fmt.Errorf(
				"interface has invalid normalized type set",
			)
		}
		for _, embedded := range spec.Embeddeds {
			if embedded.IsZero() {
				return fmt.Errorf(
					"interface has zero embedded type",
				)
			}
		}
		for _, term := range spec.Terms {
			if term.Type.IsZero() {
				return fmt.Errorf("interface has zero type term")
			}
		}
	case TypeTuple:
		for _, element := range spec.Elements {
			if element.IsZero() {
				return fmt.Errorf("tuple has zero element type")
			}
		}
	case TypeUnion:
		if len(spec.Terms) == 0 {
			return fmt.Errorf("union requires at least one term")
		}
		for _, term := range spec.Terms {
			if term.Type.IsZero() {
				return fmt.Errorf("union has zero term type")
			}
		}
	}
	return nil
}

func validateMethods(methods []TypeMethod) error {
	for index, method := range methods {
		if method.Name == "" ||
			(!semanticNameExported(method.Name) &&
				method.Package.IsZero()) ||
			(semanticNameExported(method.Name) &&
				!method.Package.IsZero()) ||
			method.Signature.IsZero() ||
			method.Ordinal != index {
			return fmt.Errorf(
				"type method %d is not canonical", index,
			)
		}
		if index != 0 {
			previous := methods[index-1]
			packageOrder := previous.Package.Compare(method.Package)
			if packageOrder > 0 ||
				(packageOrder == 0 && previous.Name >= method.Name) {
				return fmt.Errorf(
					"type methods are not canonical at %d",
					index,
				)
			}
		}
	}
	return nil
}

func semanticNameExported(name string) bool {
	first, _ := utf8.DecodeRuneInString(name)
	return first != utf8.RuneError && unicode.IsUpper(first)
}

func signaturePresent(signature Signature) bool {
	return !signature.Receiver.IsZero() ||
		len(signature.ReceiverTypeParameters) != 0 ||
		len(signature.TypeParameters) != 0 ||
		len(signature.Parameters) != 0 ||
		len(signature.Results) != 0 ||
		signature.Variadic
}

func validateSignature(signature Signature) error {
	for _, typeID := range signature.ReceiverTypeParameters {
		if typeID.IsZero() {
			return fmt.Errorf(
				"signature has zero receiver type parameter",
			)
		}
	}
	for _, typeID := range signature.TypeParameters {
		if typeID.IsZero() {
			return fmt.Errorf(
				"signature has zero type parameter",
			)
		}
	}
	for _, typeID := range signature.Parameters {
		if typeID.IsZero() {
			return fmt.Errorf("signature has zero parameter type")
		}
	}
	for _, typeID := range signature.Results {
		if typeID.IsZero() {
			return fmt.Errorf("signature has zero result type")
		}
	}
	if signature.Variadic && len(signature.Parameters) == 0 {
		return fmt.Errorf(
			"variadic signature requires a final parameter",
		)
	}
	return nil
}

func semanticTypeID(
	spec TypeSpec,
) (identity.SemanticTypeID, error) {
	encoded := encodeTypeIdentityBytes(spec)
	digest := sha256.Sum256(encoded)
	var rendered [sha256.Size * 2]byte
	hex.Encode(rendered[:], digest[:])
	return identity.NewSemanticTypeID(string(rendered[:]))
}

func encodeTypeIdentityBytes(spec TypeSpec) []byte {
	encoded := make([]byte, 0, 256)
	encoded = appendTypePart(
		encoded, "semantic-type-identity/v1",
	)
	encoded = appendTypeInt(encoded, int64(spec.Kind))
	switch spec.Kind {
	case TypeNamed, TypeAlias:
		encoded = appendTypePart(
			encoded, spec.Declaration.String(),
		)
		encoded = appendTypeIDs(encoded, spec.Arguments)
	case TypeParameter:
		encoded = appendTypePart(
			encoded, spec.Parameter.String(),
		)
	default:
		encoded = appendFramedTypeSpec(encoded, spec)
	}
	return encoded
}

func appendFramedTypeSpec(
	encoded []byte,
	spec TypeSpec,
) []byte {
	const prefixReserve = 24
	prefixStart := len(encoded)
	encoded = append(encoded, make([]byte, prefixReserve)...)
	contentStart := len(encoded)
	encoded = appendTypeSpec(encoded, spec)
	contentLength := len(encoded) - contentStart
	var prefix [prefixReserve]byte
	renderedPrefix := strconv.AppendInt(
		prefix[:0], int64(contentLength), 10,
	)
	renderedPrefix = append(renderedPrefix, ':')
	copy(
		encoded[prefixStart+len(renderedPrefix):],
		encoded[contentStart:],
	)
	copy(encoded[prefixStart:], renderedPrefix)
	framedEnd := prefixStart + len(renderedPrefix) + contentLength
	encoded = encoded[:framedEnd]
	return append(encoded, '|')
}

func appendTypeSpec(encoded []byte, spec TypeSpec) []byte {
	encoded = appendTypePart(encoded, "semantic-type/v1")
	encoded = appendTypeInt(encoded, int64(spec.Kind))
	encoded = appendTypeInt(encoded, int64(spec.Basic))
	encoded = appendTypePart(encoded, spec.Declaration.String())
	encoded = appendTypePart(encoded, spec.Parameter.String())
	encoded = appendTypeIDs(encoded, spec.Arguments)
	encoded = appendSemanticTypeIDPart(encoded, spec.Underlying)
	encoded = appendSemanticTypeIDPart(encoded, spec.Target)
	encoded = appendSemanticTypeIDPart(encoded, spec.Constraint)
	encoded = appendSemanticTypeIDPart(encoded, spec.Element)
	encoded = appendSemanticTypeIDPart(encoded, spec.Key)
	encoded = appendTypeInt(encoded, spec.Length)
	encoded = appendTypeInt(encoded, int64(spec.Direction))
	encoded = appendSemanticTypeIDPart(
		encoded, spec.Signature.Receiver,
	)
	encoded = appendTypeIDs(
		encoded, spec.Signature.ReceiverTypeParameters,
	)
	encoded = appendTypeIDs(
		encoded, spec.Signature.TypeParameters,
	)
	encoded = appendTypeIDs(
		encoded, spec.Signature.Parameters,
	)
	encoded = appendTypeIDs(encoded, spec.Signature.Results)
	encoded = appendTypeBool(encoded, spec.Signature.Variadic)
	encoded = appendTypeInt(encoded, int64(len(spec.Fields)))
	for _, field := range spec.Fields {
		encoded = appendTypePart(encoded, field.Name)
		encoded = appendTypePart(
			encoded, field.Package.String(),
		)
		encoded = appendSemanticTypeIDPart(
			encoded, field.Type,
		)
		encoded = appendTypeBool(encoded, field.Embedded)
		encoded = appendTypePart(encoded, field.Tag)
		encoded = appendTypeInt(
			encoded, int64(field.Ordinal),
		)
	}
	encoded = appendTypeInt(encoded, int64(len(spec.Methods)))
	for _, method := range spec.Methods {
		encoded = appendTypePart(encoded, method.Name)
		encoded = appendTypePart(
			encoded, method.Package.String(),
		)
		encoded = appendSemanticTypeIDPart(
			encoded, method.Signature,
		)
		encoded = appendTypeInt(
			encoded, int64(method.Ordinal),
		)
	}
	encoded = appendTypeIDs(encoded, spec.Embeddeds)
	encoded = appendTypeInt(encoded, int64(len(spec.Terms)))
	for _, term := range spec.Terms {
		encoded = appendTypeBool(encoded, term.Tilde)
		encoded = appendSemanticTypeIDPart(
			encoded, term.Type,
		)
	}
	encoded = appendTypeBool(encoded, spec.Comparable)
	encoded = appendTypeInt(encoded, int64(spec.TypeSet))
	return appendTypeIDs(encoded, spec.Elements)
}

func appendTypeIDs(
	encoded []byte,
	ids []identity.SemanticTypeID,
) []byte {
	encoded = appendTypeInt(encoded, int64(len(ids)))
	for _, id := range ids {
		encoded = appendSemanticTypeIDPart(encoded, id)
	}
	return encoded
}

func appendSemanticTypeIDPart(
	encoded []byte,
	id identity.SemanticTypeID,
) []byte {
	if id.IsZero() {
		return appendTypePart(encoded, "")
	}
	const prefix = "semantic-type/sha256:"
	length := len(prefix) + len(id.Digest())
	encoded = strconv.AppendInt(encoded, int64(length), 10)
	encoded = append(encoded, ':')
	encoded = append(encoded, prefix...)
	encoded = append(encoded, id.Digest()...)
	return append(encoded, '|')
}

func appendTypePart(encoded []byte, value string) []byte {
	encoded = strconv.AppendInt(encoded, int64(len(value)), 10)
	encoded = append(encoded, ':')
	encoded = append(encoded, value...)
	return append(encoded, '|')
}

func appendTypeInt(encoded []byte, value int64) []byte {
	var number [32]byte
	rendered := strconv.AppendInt(number[:0], value, 10)
	encoded = strconv.AppendInt(
		encoded, int64(len(rendered)), 10,
	)
	encoded = append(encoded, ':')
	encoded = append(encoded, rendered...)
	return append(encoded, '|')
}

func appendTypeBool(encoded []byte, value bool) []byte {
	if value {
		return appendTypePart(encoded, "1")
	}
	return appendTypePart(encoded, "0")
}
