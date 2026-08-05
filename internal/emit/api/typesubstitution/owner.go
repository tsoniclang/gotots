package typesubstitution

import "go/types"

type Error struct {
	Reason string
}

func (e *Error) Error() string {
	return "substitute Go type: " + e.Reason
}

func SubstituteType(
	source types.Type,
	replacements map[*types.TypeParam]types.Type,
) (types.Type, error) {
	if source == nil || len(replacements) == 0 {
		return source, nil
	}
	switch source := source.(type) {
	case *types.Basic:
		return source, nil
	case *types.TypeParam:
		if target := replacements[source]; target != nil {
			return target, nil
		}
		return source, nil
	case *types.Pointer:
		element, err := SubstituteType(source.Elem(), replacements)
		return types.NewPointer(element), err
	case *types.Slice:
		element, err := SubstituteType(source.Elem(), replacements)
		return types.NewSlice(element), err
	case *types.Array:
		element, err := SubstituteType(source.Elem(), replacements)
		return types.NewArray(element, source.Len()), err
	case *types.Map:
		key, err := SubstituteType(source.Key(), replacements)
		if err != nil {
			return nil, err
		}
		element, err := SubstituteType(source.Elem(), replacements)
		if err != nil {
			return nil, err
		}
		return types.NewMap(key, element), nil
	case *types.Chan:
		element, err := SubstituteType(source.Elem(), replacements)
		return types.NewChan(source.Dir(), element), err
	case *types.Named:
		return substituteNamedType(source, replacements)
	case *types.Alias:
		return substituteAliasType(source, replacements)
	case *types.Tuple:
		return substituteTupleType(source, replacements)
	case *types.Signature:
		return substituteSignatureType(source, replacements)
	case *types.Struct:
		return substituteStructType(source, replacements)
	case *types.Interface:
		return substituteInterfaceType(source, replacements)
	case *types.Union:
		return substituteUnionType(source, replacements)
	default:
		return nil, &Error{Reason: "type substitution encountered an unsupported type form"}
	}
}

func substituteNamedType(
	source *types.Named,
	replacements map[*types.TypeParam]types.Type,
) (types.Type, error) {
	if source.TypeArgs().Len() == 0 {
		return source, nil
	}
	arguments, changed, err := substituteTypeList(source.TypeArgs(), replacements)
	if err != nil || !changed {
		return source, err
	}
	return types.Instantiate(nil, source.Origin(), arguments, false)
}

func substituteAliasType(
	source *types.Alias,
	replacements map[*types.TypeParam]types.Type,
) (types.Type, error) {
	if source.TypeArgs().Len() == 0 {
		return source, nil
	}
	arguments, changed, err := substituteTypeList(source.TypeArgs(), replacements)
	if err != nil || !changed {
		return source, err
	}
	return types.Instantiate(nil, source.Origin(), arguments, false)
}

func substituteTupleType(
	source *types.Tuple,
	replacements map[*types.TypeParam]types.Type,
) (*types.Tuple, error) {
	if source == nil {
		return nil, nil
	}
	variables := make([]*types.Var, 0, source.Len())
	for index := range source.Len() {
		variable := source.At(index)
		target, err := SubstituteType(variable.Type(), replacements)
		if err != nil {
			return nil, err
		}
		variables = append(variables, types.NewVar(
			variable.Pos(),
			variable.Pkg(),
			variable.Name(),
			target,
		))
	}
	return types.NewTuple(variables...), nil
}

func substituteSignatureType(
	source *types.Signature,
	replacements map[*types.TypeParam]types.Type,
) (types.Type, error) {
	parameters, err := substituteTupleType(source.Params(), replacements)
	if err != nil {
		return nil, err
	}
	results, err := substituteTupleType(source.Results(), replacements)
	if err != nil {
		return nil, err
	}
	receiver, err := substituteVariable(source.Recv(), replacements)
	if err != nil {
		return nil, err
	}
	receiverParameters, err := remainingTypeParameters(
		source.RecvTypeParams(),
		replacements,
	)
	if err != nil {
		return nil, err
	}
	typeParameters, err := remainingTypeParameters(
		source.TypeParams(),
		replacements,
	)
	if err != nil {
		return nil, err
	}
	if receiver == nil || !ContainsGenericTypeParameter(receiver.Type()) {
		receiverParameters = nil
	}
	return types.NewSignatureType(
		receiver,
		receiverParameters,
		typeParameters,
		parameters,
		results,
		source.Variadic(),
	), nil
}

func substituteVariable(
	source *types.Var,
	replacements map[*types.TypeParam]types.Type,
) (*types.Var, error) {
	if source == nil {
		return nil, nil
	}
	target, err := SubstituteType(source.Type(), replacements)
	if err != nil {
		return nil, err
	}
	return types.NewVar(
		source.Pos(),
		source.Pkg(),
		source.Name(),
		target,
	), nil
}

func remainingTypeParameters(
	source *types.TypeParamList,
	replacements map[*types.TypeParam]types.Type,
) ([]*types.TypeParam, error) {
	result := make([]*types.TypeParam, 0, source.Len())
	selected := false
	for index := range source.Len() {
		parameter := source.At(index)
		if replacements[parameter] != nil {
			selected = true
			continue
		}
		result = append(result, parameter)
	}
	if selected && len(result) != 0 {
		return nil, &Error{Reason: "type substitution partially selected a parameter list"}
	}
	return result, nil
}

func substituteStructType(
	source *types.Struct,
	replacements map[*types.TypeParam]types.Type,
) (types.Type, error) {
	fields := make([]*types.Var, 0, source.NumFields())
	tags := make([]string, 0, source.NumFields())
	for index := range source.NumFields() {
		field := source.Field(index)
		fieldType, err := SubstituteType(field.Type(), replacements)
		if err != nil {
			return nil, err
		}
		fields = append(fields, types.NewField(
			field.Pos(),
			field.Pkg(),
			field.Name(),
			fieldType,
			field.Embedded(),
		))
		tags = append(tags, source.Tag(index))
	}
	return types.NewStruct(fields, tags), nil
}

func substituteInterfaceType(
	source *types.Interface,
	replacements map[*types.TypeParam]types.Type,
) (types.Type, error) {
	methods := make([]*types.Func, 0, source.NumExplicitMethods())
	for index := range source.NumExplicitMethods() {
		method := source.ExplicitMethod(index)
		methodType, err := SubstituteType(method.Type(), replacements)
		if err != nil {
			return nil, err
		}
		signature, ok := methodType.(*types.Signature)
		if !ok {
			return nil, &Error{Reason: "substituted interface method is not callable"}
		}
		methods = append(methods, types.NewFunc(
			method.Pos(),
			method.Pkg(),
			method.Name(),
			signature,
		))
	}
	embeddeds := make([]types.Type, 0, source.NumEmbeddeds())
	for index := range source.NumEmbeddeds() {
		embedded, err := SubstituteType(
			source.EmbeddedType(index),
			replacements,
		)
		if err != nil {
			return nil, err
		}
		embeddeds = append(embeddeds, embedded)
	}
	target := types.NewInterfaceType(methods, embeddeds)
	target.Complete()
	return target, nil
}

func substituteUnionType(
	source *types.Union,
	replacements map[*types.TypeParam]types.Type,
) (types.Type, error) {
	terms := make([]*types.Term, 0, source.Len())
	for index := range source.Len() {
		sourceTerm := source.Term(index)
		termType, err := SubstituteType(sourceTerm.Type(), replacements)
		if err != nil {
			return nil, err
		}
		terms = append(terms, types.NewTerm(sourceTerm.Tilde(), termType))
	}
	return types.NewUnion(terms), nil
}

func substituteTypeList(
	source *types.TypeList,
	replacements map[*types.TypeParam]types.Type,
) ([]types.Type, bool, error) {
	result := make([]types.Type, 0, source.Len())
	changed := false
	for index := range source.Len() {
		original := source.At(index)
		target, err := SubstituteType(original, replacements)
		if err != nil {
			return nil, false, err
		}
		result = append(result, target)
		changed = changed || !types.Identical(original, target)
	}
	return result, changed, nil
}

func ContainsGenericTypeParameter(sourceType types.Type) bool {
	return len(GenericTypeParametersIn(sourceType)) != 0
}

func GenericTypeParametersIn(
	sourceType types.Type,
) map[*types.TypeParam]struct{} {
	found := make(map[*types.TypeParam]struct{})
	collectGenericTypeParameters(
		sourceType,
		make(map[types.Type]bool),
		found,
	)
	return found
}

func collectGenericTypeParameters(
	sourceType types.Type,
	visiting map[types.Type]bool,
	found map[*types.TypeParam]struct{},
) {
	if sourceType == nil || visiting[sourceType] {
		return
	}
	sourceType = types.Unalias(sourceType)
	if parameter, ok := sourceType.(*types.TypeParam); ok {
		found[parameter] = struct{}{}
		return
	}
	visiting[sourceType] = true
	defer delete(visiting, sourceType)
	switch source := sourceType.(type) {
	case *types.Named:
		for index := range source.TypeArgs().Len() {
			collectGenericTypeParameters(
				source.TypeArgs().At(index),
				visiting,
				found,
			)
		}
	case *types.Pointer:
		collectGenericTypeParameters(source.Elem(), visiting, found)
	case *types.Slice:
		collectGenericTypeParameters(source.Elem(), visiting, found)
	case *types.Array:
		collectGenericTypeParameters(source.Elem(), visiting, found)
	case *types.Map:
		collectGenericTypeParameters(source.Key(), visiting, found)
		collectGenericTypeParameters(source.Elem(), visiting, found)
	case *types.Chan:
		collectGenericTypeParameters(source.Elem(), visiting, found)
	case *types.Struct:
		for index := range source.NumFields() {
			collectGenericTypeParameters(
				source.Field(index).Type(),
				visiting,
				found,
			)
		}
	case *types.Tuple:
		for index := range source.Len() {
			collectGenericTypeParameters(
				source.At(index).Type(),
				visiting,
				found,
			)
		}
	case *types.Signature:
		collectGenericTypeParameters(source.Params(), visiting, found)
		collectGenericTypeParameters(source.Results(), visiting, found)
	case *types.Interface:
		for index := range source.NumMethods() {
			collectGenericTypeParameters(
				source.Method(index).Type(),
				visiting,
				found,
			)
		}
		for index := range source.NumEmbeddeds() {
			collectGenericTypeParameters(
				source.EmbeddedType(index),
				visiting,
				found,
			)
		}
	case *types.Union:
		for index := range source.Len() {
			collectGenericTypeParameters(
				source.Term(index).Type(),
				visiting,
				found,
			)
		}
	}
}
