package provideroperation

import (
	"fmt"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

type ContractError struct {
	Object string
	Reason string
}

func (e *ContractError) Error() string {
	if e.Object == "" {
		return "provider generic operation: " + e.Reason
	}
	return fmt.Sprintf("provider generic operation %q: %s", e.Object, e.Reason)
}

func Selection(
	kind gostdlib.GenericOperationKind,
) (api.GenericOperationSelection, error) {
	var operation api.GenericOperation
	switch kind {
	case gostdlib.GenericOperationCopy:
		operation = api.GenericOperationCopy
	case gostdlib.GenericOperationZero:
		operation = api.GenericOperationZero
	case gostdlib.GenericOperationEqual:
		operation = api.GenericOperationEqual
	case gostdlib.GenericOperationBinaryLess:
		operation = api.GenericOperationBinaryLess
	case gostdlib.GenericOperationConvert:
		operation = api.GenericOperationConvert
	case gostdlib.GenericOperationMapConstruct:
		operation = api.GenericOperationMapConstruct
	case gostdlib.GenericOperationToStorage:
		operation = api.GenericOperationToStorage
	case gostdlib.GenericOperationFromStorage:
		operation = api.GenericOperationFromStorage
	case gostdlib.GenericOperationToContainerStorage:
		operation = api.GenericOperationToContainerStorage
	case gostdlib.GenericOperationFromContainerStorage:
		operation = api.GenericOperationFromContainerStorage
	case gostdlib.GenericOperationInterfaceAssertOK:
		operation = api.GenericOperationInterfaceAssertOK
	default:
		return api.GenericOperationSelection{}, &ContractError{
			Reason: "provider generic operation kind is invalid",
		}
	}
	return api.SelectGenericOperation(operation)
}

func Signature(
	owner *types.Func,
	document gostdlib.GenericOperationDocument,
) (*types.Signature, error) {
	parameters := api.GenericDeclarationParameters(owner)
	declaration, _ := owner.Type().(*types.Signature)
	var resolveType func(
		gostdlib.ContractTypeDocument,
	) (types.Type, error)
	resolveType = func(
		reference gostdlib.ContractTypeDocument,
	) (types.Type, error) {
		switch reference.Kind {
		case gostdlib.ContractTypeParameter:
			if reference.TypeParameter == nil ||
				*reference.TypeParameter < 0 ||
				*reference.TypeParameter >= len(parameters) {
				return nil, &ContractError{
					Object: owner.Name(),
					Reason: "provider generic operation type parameter is invalid",
				}
			}
			return parameters[*reference.TypeParameter], nil
		case gostdlib.ContractTypeCallableParameter:
			if declaration == nil || reference.CallableParameter == nil ||
				*reference.CallableParameter < 0 ||
				*reference.CallableParameter >= declaration.Params().Len() {
				return nil, &ContractError{
					Object: owner.Name(),
					Reason: "provider generic operation callable parameter is invalid",
				}
			}
			return declaration.Params().At(*reference.CallableParameter).Type(), nil
		case gostdlib.ContractTypeBool:
			return types.Typ[types.Bool], nil
		case gostdlib.ContractTypeInt:
			return types.Typ[types.Int], nil
		case gostdlib.ContractTypeSlice:
			if reference.Element == nil {
				break
			}
			element, err := resolveType(*reference.Element)
			if err != nil {
				return nil, err
			}
			return types.NewSlice(element), nil
		case gostdlib.ContractTypeMap:
			if reference.Key == nil || reference.Element == nil {
				break
			}
			key, err := resolveType(*reference.Key)
			if err != nil {
				return nil, err
			}
			element, err := resolveType(*reference.Element)
			if err != nil {
				return nil, err
			}
			return types.NewMap(key, element), nil
		}
		return nil, &ContractError{
			Object: owner.Name(),
			Reason: "provider generic operation type expression is invalid",
		}
	}
	resolveTuple := func(
		references []gostdlib.ContractTypeDocument,
	) (*types.Tuple, error) {
		variables := make([]*types.Var, 0, len(references))
		for _, reference := range references {
			selected, err := resolveType(reference)
			if err != nil {
				return nil, err
			}
			variables = append(variables, types.NewVar(
				token.NoPos,
				owner.Pkg(),
				"",
				selected,
			))
		}
		return types.NewTuple(variables...), nil
	}
	params, err := resolveTuple(document.Parameters)
	if err != nil {
		return nil, err
	}
	results, err := resolveTuple(document.Results)
	if err != nil {
		return nil, err
	}
	return types.NewSignatureType(
		nil,
		nil,
		nil,
		params,
		results,
		false,
	), nil
}
