package semantic

type wireTypeRecord struct {
	ID      wireTypeReference `json:"id"`
	Kind    uint8             `json:"kind"`
	Payload wireTypePayload   `json:"payload"`
}

type wireTypePayload struct {
	Tag       uint8              `json:"tag"`
	Basic     *wireBasicType     `json:"basic,omitempty"`
	Nominal   *wireNominalType   `json:"nominal,omitempty"`
	Parameter *wireTypeParameter `json:"parameter,omitempty"`
	Element   *wireElementType   `json:"element,omitempty"`
	Array     *wireArrayType     `json:"array,omitempty"`
	Map       *wireMapType       `json:"map,omitempty"`
	Channel   *wireChannelType   `json:"channel,omitempty"`
	Signature *wireSignatureType `json:"signature,omitempty"`
	Struct    *wireStructType    `json:"struct,omitempty"`
	Interface *wireInterfaceType `json:"interface,omitempty"`
	Tuple     *wireTupleType     `json:"tuple,omitempty"`
	Union     *wireUnionType     `json:"union,omitempty"`
}

type wireBasicType struct {
	Kind uint8 `json:"kind"`
}

type wireNominalType struct {
	Declaration wireDeclarationReference              `json:"declaration"`
	Arguments   wireReferenceRange[wireTypeReference] `json:"arguments"`
	Target      wireTypeReference                     `json:"target"`
	Methods     wireTypeMethodRange                   `json:"methods"`
}

type wireTypeParameter struct {
	Declaration wireDeclarationReference `json:"declaration,omitempty"`
	Definition  wireDefinitionReference  `json:"definition,omitempty"`
	Role        uint8                    `json:"role"`
	Ordinal     int                      `json:"ordinal"`
	Constraint  wireTypeReference        `json:"constraint"`
}

type wireElementType struct {
	Element wireTypeReference `json:"element"`
}

type wireArrayType struct {
	Element wireTypeReference `json:"element"`
	Length  int64             `json:"length"`
}

type wireMapType struct {
	Key     wireTypeReference `json:"key"`
	Element wireTypeReference `json:"element"`
}

type wireChannelType struct {
	Element   wireTypeReference `json:"element"`
	Direction uint8             `json:"direction"`
}

type wireSignatureType struct {
	Receiver               wireTypeReference                     `json:"receiver,omitempty"`
	ReceiverTypeParameters wireReferenceRange[wireTypeReference] `json:"receiverTypeParameters"`
	TypeParameters         wireReferenceRange[wireTypeReference] `json:"typeParameters"`
	Parameters             wireReferenceRange[wireTypeReference] `json:"parameters"`
	Results                wireReferenceRange[wireTypeReference] `json:"results"`
	Variadic               bool                                  `json:"variadic,omitempty"`
}

type wireStructType struct {
	Fields wireTypeFieldRange `json:"fields"`
}

type wireInterfaceType struct {
	Methods    wireTypeMethodRange                   `json:"methods"`
	Embeddeds  wireReferenceRange[wireTypeReference] `json:"embeddeds"`
	Terms      wireTypeTermRange                     `json:"terms"`
	TypeSet    uint8                                 `json:"typeSet"`
	Comparable bool                                  `json:"comparable,omitempty"`
}

type wireTupleType struct {
	Elements wireReferenceRange[wireTypeReference] `json:"elements"`
}

type wireUnionType struct {
	Terms wireTypeTermRange `json:"terms"`
}

type wireTypeField struct {
	Name     string               `json:"name"`
	Package  wirePackageReference `json:"package,omitempty"`
	Type     wireTypeReference    `json:"type"`
	Embedded bool                 `json:"embedded,omitempty"`
	Tag      string               `json:"tag,omitempty"`
	Ordinal  int                  `json:"ordinal"`
}

type wireTypeFieldRange struct {
	Start  uint64          `json:"start"`
	Count  uint64          `json:"count"`
	Values []wireTypeField `json:"values,omitempty"`
}

type wireTypeMethod struct {
	Name      string               `json:"name"`
	Package   wirePackageReference `json:"package,omitempty"`
	Signature wireTypeReference    `json:"signature"`
	Ordinal   int                  `json:"ordinal"`
}

type wireTypeMethodRange struct {
	Start  uint64           `json:"start"`
	Count  uint64           `json:"count"`
	Values []wireTypeMethod `json:"values,omitempty"`
}

type wireTypeTerm struct {
	Tilde bool              `json:"tilde,omitempty"`
	Type  wireTypeReference `json:"type"`
}

type wireTypeTermRange struct {
	Start  uint64         `json:"start"`
	Count  uint64         `json:"count"`
	Values []wireTypeTerm `json:"values,omitempty"`
}
