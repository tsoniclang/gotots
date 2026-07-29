package instance

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func substituteTuple(
	source *types.Tuple,
	replacements map[*types.TypeParam]*types.TypeParam,
) (*types.Tuple, error) {
	if source == nil {
		return nil, nil
	}
	variables := make([]*types.Var, 0, source.Len())
	for index := range source.Len() {
		variable := source.At(index)
		target, err := substituteType(variable.Type(), replacements)
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

func substituteType(
	source types.Type,
	replacements map[*types.TypeParam]*types.TypeParam,
) (types.Type, error) {
	switch source := source.(type) {
	case *types.Basic:
		return source, nil
	case *types.TypeParam:
		target := replacements[source]
		if target == nil {
			return source, nil
		}
		return target, nil
	case *types.Pointer:
		element, err := substituteType(source.Elem(), replacements)
		return types.NewPointer(element), err
	case *types.Slice:
		element, err := substituteType(source.Elem(), replacements)
		return types.NewSlice(element), err
	case *types.Array:
		element, err := substituteType(source.Elem(), replacements)
		return types.NewArray(element, source.Len()), err
	case *types.Map:
		key, err := substituteType(source.Key(), replacements)
		if err != nil {
			return nil, err
		}
		element, err := substituteType(source.Elem(), replacements)
		if err != nil {
			return nil, err
		}
		return types.NewMap(key, element), nil
	case *types.Chan:
		element, err := substituteType(source.Elem(), replacements)
		return types.NewChan(source.Dir(), element), err
	case *types.Named:
		return substituteNamed(source, replacements)
	case *types.Alias:
		return substituteAlias(source, replacements)
	case *types.Tuple:
		return substituteTuple(source, replacements)
	case *types.Signature:
		return substituteSignature(source, replacements)
	case *types.Struct:
		return substituteStruct(source, replacements)
	case *types.Interface:
		return substituteInterface(source, replacements)
	case *types.Union:
		return substituteUnion(source, replacements)
	default:
		return nil, &api.InvariantError{
			Role:   api.RoleCallCallee,
			Reason: "generic operation contains an unsupported type form",
		}
	}
}

func substituteNamed(
	source *types.Named,
	replacements map[*types.TypeParam]*types.TypeParam,
) (types.Type, error) {
	if source.TypeArgs().Len() == 0 {
		return source, nil
	}
	arguments, err := substituteTypeList(source.TypeArgs(), replacements)
	if err != nil {
		return nil, err
	}
	return types.Instantiate(nil, source.Origin(), arguments, false)
}

func substituteAlias(
	source *types.Alias,
	replacements map[*types.TypeParam]*types.TypeParam,
) (types.Type, error) {
	if source.TypeArgs().Len() == 0 {
		return source, nil
	}
	arguments, err := substituteTypeList(source.TypeArgs(), replacements)
	if err != nil {
		return nil, err
	}
	return types.Instantiate(nil, source.Origin(), arguments, false)
}

func substituteSignature(
	source *types.Signature,
	replacements map[*types.TypeParam]*types.TypeParam,
) (types.Type, error) {
	if source.TypeParams().Len() != 0 ||
		source.RecvTypeParams().Len() != 0 {
		return nil, &api.InvariantError{
			Role:   api.RoleCallCallee,
			Reason: "nested generic operation signature is invalid",
		}
	}
	parameters, err := substituteTuple(source.Params(), replacements)
	if err != nil {
		return nil, err
	}
	results, err := substituteTuple(source.Results(), replacements)
	if err != nil {
		return nil, err
	}
	var receiver *types.Var
	if source.Recv() != nil {
		receiverType, receiverErr := substituteType(
			source.Recv().Type(),
			replacements,
		)
		if receiverErr != nil {
			return nil, receiverErr
		}
		receiver = types.NewVar(
			source.Recv().Pos(),
			source.Recv().Pkg(),
			source.Recv().Name(),
			receiverType,
		)
	}
	return types.NewSignatureType(
		receiver,
		nil,
		nil,
		parameters,
		results,
		source.Variadic(),
	), nil
}

func substituteStruct(
	source *types.Struct,
	replacements map[*types.TypeParam]*types.TypeParam,
) (types.Type, error) {
	fields := make([]*types.Var, 0, source.NumFields())
	tags := make([]string, 0, source.NumFields())
	for index := range source.NumFields() {
		field := source.Field(index)
		fieldType, err := substituteType(field.Type(), replacements)
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

func substituteInterface(
	source *types.Interface,
	replacements map[*types.TypeParam]*types.TypeParam,
) (types.Type, error) {
	methods := make([]*types.Func, 0, source.NumExplicitMethods())
	for index := range source.NumExplicitMethods() {
		method := source.ExplicitMethod(index)
		methodType, err := substituteType(method.Type(), replacements)
		if err != nil {
			return nil, err
		}
		signature, ok := methodType.(*types.Signature)
		if !ok {
			return nil, &api.InvariantError{
				Role:   api.RoleCallCallee,
				Reason: "substituted interface method is not callable",
			}
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
		embedded, err := substituteType(
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

func substituteUnion(
	source *types.Union,
	replacements map[*types.TypeParam]*types.TypeParam,
) (types.Type, error) {
	terms := make([]*types.Term, 0, source.Len())
	for index := range source.Len() {
		sourceTerm := source.Term(index)
		termType, err := substituteType(sourceTerm.Type(), replacements)
		if err != nil {
			return nil, err
		}
		terms = append(terms, types.NewTerm(sourceTerm.Tilde(), termType))
	}
	return types.NewUnion(terms), nil
}

func substituteTypeList(
	source *types.TypeList,
	replacements map[*types.TypeParam]*types.TypeParam,
) ([]types.Type, error) {
	result := make([]types.Type, 0, source.Len())
	for index := range source.Len() {
		target, err := substituteType(source.At(index), replacements)
		if err != nil {
			return nil, err
		}
		result = append(result, target)
	}
	return result, nil
}
