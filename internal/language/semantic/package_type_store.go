package semantic

import "github.com/tsoniclang/gotots/internal/identity"

type typeFieldRange struct {
	start uint64
	count uint64
}

type typeMethodRange struct {
	start uint64
	count uint64
}

type typeTermRange struct {
	start uint64
	count uint64
}

type storedType struct {
	id      typeRef
	kind    TypeKind
	payload uint64
}

type storedNominalType struct {
	declaration declarationRef
	arguments   typeRefRange
	target      typeRef
	methods     typeMethodRange
}

type storedTypeParameter struct {
	declaration declarationRef
	definition  definitionRef
	role        TypeParameterRole
	ordinal     int
	constraint  typeRef
}

type storedArrayType struct {
	element typeRef
	length  int64
}

type storedMapType struct {
	key     typeRef
	element typeRef
}

type storedChannelType struct {
	element   typeRef
	direction ChannelDirection
}

type storedSignature struct {
	receiver               typeRef
	receiverTypeParameters typeRefRange
	typeParameters         typeRefRange
	parameters             typeRefRange
	results                typeRefRange
	variadic               bool
}

type storedInterfaceType struct {
	methods    typeMethodRange
	embeddeds  typeRefRange
	terms      typeTermRange
	typeSet    TypeSetKind
	comparable bool
}

type storedTypeField struct {
	name     string
	pkg      packageRef
	typeID   typeRef
	embedded bool
	tag      string
	ordinal  int
}

type storedTypeMethod struct {
	name      string
	pkg       packageRef
	signature typeRef
	ordinal   int
}

type storedTypeTerm struct {
	tilde  bool
	typeID typeRef
}

type packageTypeBuilder struct {
	records       []storedType
	basic         []BasicKind
	nominal       []storedNominalType
	parameters    []storedTypeParameter
	elements      []typeRef
	arrays        []storedArrayType
	maps          []storedMapType
	channels      []storedChannelType
	signatures    []storedSignature
	structs       []typeFieldRange
	interfaces    []storedInterfaceType
	tuples        []typeRefRange
	unions        []typeTermRange
	typeRelations []typeRef
	fields        []storedTypeField
	methods       []storedTypeMethod
	terms         []storedTypeTerm
}

func (builder *packageTypeBuilder) add(
	identities *packageIdentityBuilder,
	record Type,
) {
	spec := record.spec
	stored := storedType{
		id:   identities.typeID(record.id),
		kind: spec.Kind,
	}
	switch spec.Kind {
	case TypeBasic:
		builder.basic = append(builder.basic, spec.Basic)
		stored.payload = uint64(len(builder.basic))
	case TypeNamed, TypeAlias:
		target := spec.Underlying
		if spec.Kind == TypeAlias {
			target = spec.Target
		}
		builder.nominal = append(
			builder.nominal,
			storedNominalType{
				declaration: identities.declaration(
					spec.Declaration,
				),
				arguments: builder.addTypes(
					identities, spec.Arguments,
				),
				target: identities.typeID(target),
				methods: builder.addMethods(
					identities, spec.Methods,
				),
			},
		)
		stored.payload = uint64(len(builder.nominal))
	case TypeParameter:
		builder.parameters = append(
			builder.parameters,
			storedTypeParameter{
				declaration: identities.declaration(
					spec.Parameter.Declaration(),
				),
				definition: identities.definition(
					spec.Parameter.Definition(),
				),
				role:       spec.Parameter.Role(),
				ordinal:    spec.Parameter.Ordinal(),
				constraint: identities.typeID(spec.Constraint),
			},
		)
		stored.payload = uint64(len(builder.parameters))
	case TypePointer, TypeSlice:
		builder.elements = append(
			builder.elements,
			identities.typeID(spec.Element),
		)
		stored.payload = uint64(len(builder.elements))
	case TypeArray:
		builder.arrays = append(
			builder.arrays,
			storedArrayType{
				element: identities.typeID(spec.Element),
				length:  spec.Length,
			},
		)
		stored.payload = uint64(len(builder.arrays))
	case TypeMap:
		builder.maps = append(
			builder.maps,
			storedMapType{
				key:     identities.typeID(spec.Key),
				element: identities.typeID(spec.Element),
			},
		)
		stored.payload = uint64(len(builder.maps))
	case TypeChannel:
		builder.channels = append(
			builder.channels,
			storedChannelType{
				element:   identities.typeID(spec.Element),
				direction: spec.Direction,
			},
		)
		stored.payload = uint64(len(builder.channels))
	case TypeSignature:
		builder.signatures = append(
			builder.signatures,
			builder.addSignature(identities, spec.Signature),
		)
		stored.payload = uint64(len(builder.signatures))
	case TypeStruct:
		builder.structs = append(
			builder.structs,
			builder.addFields(identities, spec.Fields),
		)
		stored.payload = uint64(len(builder.structs))
	case TypeInterface:
		builder.interfaces = append(
			builder.interfaces,
			storedInterfaceType{
				methods: builder.addMethods(
					identities, spec.Methods,
				),
				embeddeds: builder.addTypes(
					identities, spec.Embeddeds,
				),
				terms: builder.addTerms(
					identities, spec.Terms,
				),
				typeSet:    spec.TypeSet,
				comparable: spec.Comparable,
			},
		)
		stored.payload = uint64(len(builder.interfaces))
	case TypeTuple:
		builder.tuples = append(
			builder.tuples,
			builder.addTypes(identities, spec.Elements),
		)
		stored.payload = uint64(len(builder.tuples))
	case TypeUnion:
		builder.unions = append(
			builder.unions,
			builder.addTerms(identities, spec.Terms),
		)
		stored.payload = uint64(len(builder.unions))
	}
	builder.records = append(builder.records, stored)
}

func (builder *packageTypeBuilder) addTypes(
	identities *packageIdentityBuilder,
	values []identity.SemanticTypeID,
) typeRefRange {
	out := typeRefRange{
		start: uint64(len(builder.typeRelations)),
		count: uint64(len(values)),
	}
	for _, value := range values {
		builder.typeRelations = append(
			builder.typeRelations,
			identities.typeID(value),
		)
	}
	return out
}

func (builder *packageTypeBuilder) addSignature(
	identities *packageIdentityBuilder,
	signature Signature,
) storedSignature {
	return storedSignature{
		receiver: identities.typeID(signature.Receiver),
		receiverTypeParameters: builder.addTypes(
			identities, signature.ReceiverTypeParameters,
		),
		typeParameters: builder.addTypes(
			identities, signature.TypeParameters,
		),
		parameters: builder.addTypes(
			identities, signature.Parameters,
		),
		results: builder.addTypes(
			identities, signature.Results,
		),
		variadic: signature.Variadic,
	}
}

func (builder *packageTypeBuilder) addFields(
	identities *packageIdentityBuilder,
	values []TypeField,
) typeFieldRange {
	out := typeFieldRange{
		start: uint64(len(builder.fields)),
		count: uint64(len(values)),
	}
	for _, value := range values {
		builder.fields = append(builder.fields, storedTypeField{
			name:     value.Name,
			pkg:      identities.packageID(value.Package),
			typeID:   identities.typeID(value.Type),
			embedded: value.Embedded,
			tag:      value.Tag,
			ordinal:  value.Ordinal,
		})
	}
	return out
}

func (builder *packageTypeBuilder) addMethods(
	identities *packageIdentityBuilder,
	values []TypeMethod,
) typeMethodRange {
	out := typeMethodRange{
		start: uint64(len(builder.methods)),
		count: uint64(len(values)),
	}
	for _, value := range values {
		builder.methods = append(
			builder.methods,
			storedTypeMethod{
				name:      value.Name,
				pkg:       identities.packageID(value.Package),
				signature: identities.typeID(value.Signature),
				ordinal:   value.Ordinal,
			},
		)
	}
	return out
}

func (builder *packageTypeBuilder) addTerms(
	identities *packageIdentityBuilder,
	values []TypeTerm,
) typeTermRange {
	out := typeTermRange{
		start: uint64(len(builder.terms)),
		count: uint64(len(values)),
	}
	for _, value := range values {
		builder.terms = append(builder.terms, storedTypeTerm{
			tilde:  value.Tilde,
			typeID: identities.typeID(value.Type),
		})
	}
	return out
}

type packageTypeStore struct {
	records       []storedType
	basic         []BasicKind
	nominal       []storedNominalType
	parameters    []storedTypeParameter
	elements      []typeRef
	arrays        []storedArrayType
	maps          []storedMapType
	channels      []storedChannelType
	signatures    []storedSignature
	structs       []typeFieldRange
	interfaces    []storedInterfaceType
	tuples        []typeRefRange
	unions        []typeTermRange
	typeRelations []typeRef
	fields        []storedTypeField
	methods       []storedTypeMethod
	terms         []storedTypeTerm
}
