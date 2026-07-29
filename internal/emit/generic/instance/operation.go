package instance

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func InstantiateOperation(
	operationSet api.GenericOperationSet,
	operation *types.Signature,
	arguments *types.TypeList,
) (*types.Signature, error) {
	parameters := operationSet.Parameters()
	if operation == nil || arguments == nil ||
		len(parameters) != arguments.Len() {
		return nil, &api.InvariantError{
			Role:   api.RoleCallCallee,
			Reason: "generic operation instantiation is inconsistent",
		}
	}
	replacements := make(
		map[*types.TypeParam]*types.TypeParam,
		len(parameters),
	)
	fresh := make([]*types.TypeParam, 0, len(parameters))
	for _, parameter := range parameters {
		object := parameter.Obj()
		clone := types.NewTypeParam(
			types.NewTypeName(
				object.Pos(),
				object.Pkg(),
				object.Name(),
				nil,
			),
			types.NewInterfaceType(nil, nil).Complete(),
		)
		replacements[parameter] = clone
		fresh = append(fresh, clone)
	}
	params, err := substituteTuple(operation.Params(), replacements)
	if err != nil {
		return nil, err
	}
	results, err := substituteTuple(operation.Results(), replacements)
	if err != nil {
		return nil, err
	}
	generic := types.NewSignatureType(
		nil,
		nil,
		fresh,
		params,
		results,
		operation.Variadic(),
	)
	typeArguments := make([]types.Type, 0, arguments.Len())
	for index := range arguments.Len() {
		typeArguments = append(typeArguments, arguments.At(index))
	}
	instantiated, err := types.Instantiate(
		nil,
		generic,
		typeArguments,
		false,
	)
	if err != nil {
		return nil, err
	}
	signature, ok := instantiated.(*types.Signature)
	if !ok {
		return nil, &api.InvariantError{
			Role:   api.RoleCallCallee,
			Reason: "generic operation did not instantiate to a signature",
		}
	}
	return signature, nil
}
