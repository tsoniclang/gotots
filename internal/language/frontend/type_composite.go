package frontend

import (
	"fmt"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/language/typesemantics"
)

func (builder *typeBuilder) signature(
	signature *types.Signature,
) (semantic.Signature, error) {
	return builder.signatureDescriptor(signature, true)
}

func (builder *typeBuilder) signatureDescriptor(
	signature *types.Signature,
	includeReceiver bool,
) (semantic.Signature, error) {
	out := semantic.Signature{Variadic: signature.Variadic()}
	var err error
	if includeReceiver && signature.Recv() != nil {
		out.Receiver, err = builder.reference(signature.Recv().Type())
		if err != nil {
			return semantic.Signature{}, err
		}
	}
	if includeReceiver {
		out.ReceiverTypeParameters, err = builder.typeParameters(
			signature.RecvTypeParams(),
		)
		if err != nil {
			return semantic.Signature{}, err
		}
	}
	out.TypeParameters, err = builder.typeParameters(
		signature.TypeParams(),
	)
	if err != nil {
		return semantic.Signature{}, err
	}
	out.Parameters, err = builder.tuple(signature.Params())
	if err != nil {
		return semantic.Signature{}, err
	}
	out.Results, err = builder.tuple(signature.Results())
	return out, err
}

func (builder *typeBuilder) structFields(
	structure *types.Struct,
) ([]semantic.TypeField, error) {
	out := make([]semantic.TypeField, 0, structure.NumFields())
	for ordinal := 0; ordinal < structure.NumFields(); ordinal++ {
		field := structure.Field(ordinal)
		typeID, err := builder.reference(field.Type())
		if err != nil {
			return nil, err
		}
		pkg, err := builder.memberPackage(field)
		if err != nil {
			return nil, err
		}
		out = append(out, semantic.TypeField{
			Name: field.Name(), Package: pkg, Type: typeID,
			Embedded: field.Embedded(), Tag: structure.Tag(ordinal),
			Ordinal: ordinal,
		})
	}
	return out, nil
}

func (builder *typeBuilder) interfaceSpec(
	iface *types.Interface,
) (semantic.TypeSpec, error) {
	iface.Complete()
	methods := make(
		[]semantic.TypeMethod, 0, iface.NumExplicitMethods(),
	)
	for index := 0; index < iface.NumExplicitMethods(); index++ {
		method, err := builder.methodDescriptor(
			iface.ExplicitMethod(index),
		)
		if err != nil {
			return semantic.TypeSpec{}, err
		}
		methods = append(methods, method)
	}
	sortMethods(methods)
	embeddeds := make(
		[]identity.SemanticTypeID, 0, iface.NumEmbeddeds(),
	)
	for index := 0; index < iface.NumEmbeddeds(); index++ {
		embedded, err := builder.reference(
			iface.EmbeddedType(index),
		)
		if err != nil {
			return semantic.TypeSpec{}, err
		}
		embeddeds = append(embeddeds, embedded)
	}
	sort.Slice(embeddeds, func(left, right int) bool {
		return embeddeds[left].String() <
			embeddeds[right].String()
	})
	setKind, normalized, ok := typesemantics.NormalizedTerms(iface)
	if !ok {
		return semantic.TypeSpec{}, fmt.Errorf(
			"interface type set cannot be normalized",
		)
	}
	terms := make([]semantic.TypeTerm, 0, len(normalized))
	for _, term := range normalized {
		typeID, err := builder.reference(term.Type)
		if err != nil {
			return semantic.TypeSpec{}, err
		}
		terms = append(terms, semantic.TypeTerm{
			Tilde: term.Tilde, Type: typeID,
		})
	}
	sortTypeTerms(terms)
	return semantic.TypeSpec{
		Kind:       semantic.TypeInterface,
		Methods:    methods,
		Embeddeds:  embeddeds,
		Terms:      terms,
		TypeSet:    semanticTypeSetKind(setKind),
		Comparable: iface.IsComparable(),
	}, nil
}

func (builder *typeBuilder) namedMethods(
	named *types.Named,
) ([]semantic.TypeMethod, error) {
	out := make([]semantic.TypeMethod, 0, named.NumMethods())
	for index := 0; index < named.NumMethods(); index++ {
		method, err := builder.methodDescriptor(
			named.Method(index),
		)
		if err != nil {
			return nil, err
		}
		out = append(out, method)
	}
	sortMethods(out)
	return out, nil
}

func (builder *typeBuilder) methodDescriptor(
	method *types.Func,
) (semantic.TypeMethod, error) {
	signature, ok := method.Type().(*types.Signature)
	if !ok {
		return semantic.TypeMethod{}, fmt.Errorf(
			"method %s has non-signature type %T",
			method.Name(),
			method.Type(),
		)
	}
	if signature.RecvTypeParams() != nil &&
		signature.RecvTypeParams().Len() != 0 {
		if _, err := builder.objects.declarationID(method); err != nil {
			return semantic.TypeMethod{}, err
		}
	}
	signatureID, err := builder.methodSignature(signature)
	if err != nil {
		return semantic.TypeMethod{}, err
	}
	pkg, err := builder.memberPackage(method)
	if err != nil {
		return semantic.TypeMethod{}, err
	}
	return semantic.TypeMethod{
		Name: method.Name(), Package: pkg, Signature: signatureID,
	}, nil
}

func (builder *typeBuilder) methodSignature(
	signature *types.Signature,
) (identity.SemanticTypeID, error) {
	descriptor, err := builder.signatureDescriptor(
		signature, false,
	)
	if err != nil {
		return identity.SemanticTypeID{}, err
	}
	record, err := semantic.NewType(semantic.TypeSpec{
		Kind:      semantic.TypeSignature,
		Signature: descriptor,
	})
	if err != nil {
		return identity.SemanticTypeID{}, err
	}
	if existing, present := builder.records[record.ID()]; present &&
		existing.Canonical() != record.Canonical() {
		return identity.SemanticTypeID{}, fmt.Errorf(
			"semantic method-signature collision %s",
			record.ID(),
		)
	}
	builder.records[record.ID()] = record
	builder.work++
	return record.ID(), nil
}

func sortMethods(methods []semantic.TypeMethod) {
	sort.Slice(methods, func(left, right int) bool {
		leftKey := methods[left].Package.String() + "|" +
			methods[left].Name + "|" +
			methods[left].Signature.String()
		rightKey := methods[right].Package.String() + "|" +
			methods[right].Name + "|" +
			methods[right].Signature.String()
		return leftKey < rightKey
	})
	for index := range methods {
		methods[index].Ordinal = index
	}
}

func (builder *typeBuilder) unionTerms(
	union *types.Union,
) ([]semantic.TypeTerm, error) {
	out := make([]semantic.TypeTerm, 0, union.Len())
	for index := 0; index < union.Len(); index++ {
		term := union.Term(index)
		typeID, err := builder.reference(term.Type())
		if err != nil {
			return nil, err
		}
		out = append(out, semantic.TypeTerm{
			Tilde: term.Tilde(), Type: typeID,
		})
	}
	sortTypeTerms(out)
	return out, nil
}

func sortTypeTerms(terms []semantic.TypeTerm) {
	sort.Slice(terms, func(left, right int) bool {
		leftKey := fmt.Sprintf(
			"%t|%s", terms[left].Tilde, terms[left].Type,
		)
		rightKey := fmt.Sprintf(
			"%t|%s", terms[right].Tilde, terms[right].Type,
		)
		return leftKey < rightKey
	})
}

func (builder *typeBuilder) tuple(
	tuple *types.Tuple,
) ([]identity.SemanticTypeID, error) {
	if tuple == nil {
		return nil, nil
	}
	out := make([]identity.SemanticTypeID, 0, tuple.Len())
	for index := 0; index < tuple.Len(); index++ {
		typeID, err := builder.reference(tuple.At(index).Type())
		if err != nil {
			return nil, err
		}
		out = append(out, typeID)
	}
	return out, nil
}

func (builder *typeBuilder) typeParameters(
	list *types.TypeParamList,
) ([]identity.SemanticTypeID, error) {
	if list == nil {
		return nil, nil
	}
	out := make([]identity.SemanticTypeID, 0, list.Len())
	for index := 0; index < list.Len(); index++ {
		typeID, err := builder.reference(list.At(index))
		if err != nil {
			return nil, err
		}
		out = append(out, typeID)
	}
	return out, nil
}

func (builder *typeBuilder) typeList(
	list *types.TypeList,
) ([]identity.SemanticTypeID, error) {
	if list == nil {
		return nil, nil
	}
	out := make([]identity.SemanticTypeID, 0, list.Len())
	for index := 0; index < list.Len(); index++ {
		typeID, err := builder.reference(list.At(index))
		if err != nil {
			return nil, err
		}
		out = append(out, typeID)
	}
	return out, nil
}
