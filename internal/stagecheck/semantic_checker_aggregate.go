package stagecheck

import (
	"fmt"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/language/typesemantics"
)

func (verifier *checkerTypeVerifier) verifySignature(
	spec semantic.Signature,
	signature *types.Signature,
) error {
	if signature == nil || spec.Variadic != signature.Variadic() {
		return fmt.Errorf("signature variadic state differs")
	}
	if signature.Recv() == nil {
		if !spec.Receiver.IsZero() {
			return fmt.Errorf("signature has unexpected receiver")
		}
	} else if err := verifier.verify(
		spec.Receiver, signature.Recv().Type(),
	); err != nil {
		return err
	}
	if err := verifier.verifyTypeParameterList(
		spec.ReceiverTypeParameters,
		signature.RecvTypeParams(),
	); err != nil {
		return err
	}
	if err := verifier.verifyTypeParameterList(
		spec.TypeParameters, signature.TypeParams(),
	); err != nil {
		return err
	}
	if err := verifier.verifyTuple(
		spec.Parameters, signature.Params(),
	); err != nil {
		return err
	}
	return verifier.verifyTuple(spec.Results, signature.Results())
}

func (verifier *checkerTypeVerifier) verifyTypeParameterList(
	ids []identity.SemanticTypeID,
	list *types.TypeParamList,
) error {
	length := 0
	if list != nil {
		length = list.Len()
	}
	if len(ids) != length {
		return fmt.Errorf(
			"type parameter count %d differs from %d",
			len(ids), length,
		)
	}
	for index, id := range ids {
		if err := verifier.verify(id, list.At(index)); err != nil {
			return err
		}
	}
	return nil
}

func (verifier *checkerTypeVerifier) verifyStruct(
	fields []semantic.TypeField,
	structure *types.Struct,
) error {
	if structure == nil || len(fields) != structure.NumFields() {
		return fmt.Errorf("struct field count differs")
	}
	for ordinal, field := range fields {
		checker := structure.Field(ordinal)
		var pkg identity.PackageID
		if !checker.Exported() {
			if checker.Pkg() == nil {
				return fmt.Errorf(
					"unexported field %s has no package", checker.Name(),
				)
			}
			pkg = verifier.packageByPath[checker.Pkg().Path()]
		}
		if field.Name != checker.Name() ||
			field.Package != pkg ||
			field.Embedded != checker.Embedded() ||
			field.Tag != structure.Tag(ordinal) ||
			field.Ordinal != ordinal {
			return fmt.Errorf(
				"struct field %d metadata differs", ordinal,
			)
		}
		if err := verifier.verify(
			field.Type, checker.Type(),
		); err != nil {
			return err
		}
	}
	return nil
}

func (verifier *checkerTypeVerifier) verifyNamedMethods(
	methods []semantic.TypeMethod,
	named *types.Named,
) error {
	if len(methods) != named.NumMethods() {
		return fmt.Errorf("named method count differs")
	}
	checker := make([]*types.Func, 0, named.NumMethods())
	for index := 0; index < named.NumMethods(); index++ {
		checker = append(checker, named.Method(index))
	}
	return verifier.verifyMethods(methods, checker)
}

func (verifier *checkerTypeVerifier) verifyMethods(
	methods []semantic.TypeMethod,
	checker []*types.Func,
) error {
	type methodEntry struct {
		name string
		pkg  identity.PackageID
		fn   *types.Func
	}
	entries := make([]methodEntry, 0, len(checker))
	for _, method := range checker {
		var pkg identity.PackageID
		if !method.Exported() {
			if method.Pkg() == nil {
				return fmt.Errorf(
					"unexported method %s has no package",
					method.Name(),
				)
			}
			pkg = verifier.packageByPath[method.Pkg().Path()]
		}
		entries = append(entries, methodEntry{
			name: method.Name(), pkg: pkg, fn: method,
		})
	}
	sort.Slice(entries, func(left, right int) bool {
		if entries[left].pkg != entries[right].pkg {
			return entries[left].pkg.String() <
				entries[right].pkg.String()
		}
		return entries[left].name < entries[right].name
	})
	if len(methods) != len(entries) {
		return fmt.Errorf("method count differs")
	}
	for ordinal, method := range methods {
		entry := entries[ordinal]
		if method.Name != entry.name ||
			method.Package != entry.pkg ||
			method.Ordinal != ordinal {
			return fmt.Errorf(
				"method %d metadata differs: semantic=%s/%s/%d checker=%s/%s/%d",
				ordinal,
				method.Package,
				method.Name,
				method.Ordinal,
				entry.pkg,
				entry.name,
				ordinal,
			)
		}
		signature, ok := entry.fn.Type().(*types.Signature)
		if !ok {
			return fmt.Errorf(
				"method %s has non-signature type", entry.name,
			)
		}
		if err := verifier.verifyMethodSignature(
			method.Signature, signature,
		); err != nil {
			return err
		}
	}
	return nil
}

func (verifier *checkerTypeVerifier) verifyMethodSignature(
	id identity.SemanticTypeID,
	signature *types.Signature,
) error {
	record, present := verifier.types[id]
	if !present ||
		record.Kind() != semantic.TypeSignature {
		return fmt.Errorf(
			"method signature %s is absent", id,
		)
	}
	spec := record.Spec().Signature
	if !spec.Receiver.IsZero() ||
		len(spec.ReceiverTypeParameters) != 0 {
		return fmt.Errorf(
			"method descriptor signature retains receiver",
		)
	}
	if spec.Variadic != signature.Variadic() {
		return fmt.Errorf("method signature variadic state differs")
	}
	if err := verifier.verifyTypeParameterList(
		spec.TypeParameters, signature.TypeParams(),
	); err != nil {
		return err
	}
	if err := verifier.verifyTuple(
		spec.Parameters, signature.Params(),
	); err != nil {
		return err
	}
	return verifier.verifyTuple(spec.Results, signature.Results())
}

func (verifier *checkerTypeVerifier) verifyInterface(
	spec semantic.TypeSpec,
	iface *types.Interface,
) error {
	iface.Complete()
	methods := make(
		[]*types.Func, 0, iface.NumExplicitMethods(),
	)
	for index := 0; index < iface.NumExplicitMethods(); index++ {
		methods = append(methods, iface.ExplicitMethod(index))
	}
	if err := verifier.verifyMethods(spec.Methods, methods); err != nil {
		return err
	}
	if len(spec.Embeddeds) != iface.NumEmbeddeds() {
		return fmt.Errorf("interface embedded count differs")
	}
	if err := verifier.matchUnorderedTypes(
		spec.Embeddeds,
		func(index int) types.Type {
			return iface.EmbeddedType(index)
		},
		iface.NumEmbeddeds(),
	); err != nil {
		return err
	}
	setKind, terms, ok := typesemantics.NormalizedTerms(iface)
	if !ok ||
		spec.TypeSet != independentTypeSetKind(setKind) ||
		spec.Comparable != iface.IsComparable() ||
		len(spec.Terms) != len(terms) {
		return fmt.Errorf("interface type set differs")
	}
	return verifier.matchUnorderedTerms(spec.Terms, terms)
}

func (verifier *checkerTypeVerifier) verifyUnion(
	terms []semantic.TypeTerm,
	union *types.Union,
) error {
	if len(terms) != union.Len() {
		return fmt.Errorf("union term count differs")
	}
	checker := make([]typesemantics.Term, 0, union.Len())
	for index := 0; index < union.Len(); index++ {
		term := union.Term(index)
		checker = append(checker, typesemantics.Term{
			Type: term.Type(), Tilde: term.Tilde(),
		})
	}
	return verifier.matchUnorderedTerms(terms, checker)
}

func (verifier *checkerTypeVerifier) matchUnorderedTypes(
	ids []identity.SemanticTypeID,
	at func(int) types.Type,
	length int,
) error {
	used := make([]bool, length)
	for _, id := range ids {
		matched := false
		for index := 0; index < length; index++ {
			if used[index] {
				continue
			}
			if err := verifier.verify(id, at(index)); err == nil {
				used[index] = true
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf(
				"unordered semantic type %s has no checker match", id,
			)
		}
	}
	return nil
}

func (verifier *checkerTypeVerifier) matchUnorderedTerms(
	records []semantic.TypeTerm,
	terms []typesemantics.Term,
) error {
	used := make([]bool, len(terms))
	for _, record := range records {
		matched := false
		for index, term := range terms {
			if used[index] || record.Tilde != term.Tilde {
				continue
			}
			if err := verifier.verify(
				record.Type, term.Type,
			); err == nil {
				used[index] = true
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf(
				"semantic type-set term has no checker match",
			)
		}
	}
	return nil
}
