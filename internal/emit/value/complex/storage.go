package complex

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (carrier Carrier) StorageType(context api.Context) (api.TypeEmission, error) {
	symbol, err := carrier.storageSymbol(false, false)
	if err != nil {
		return api.TypeEmission{}, err
	}
	reference, err := context.Names().Runtime(symbol, api.ImportPhaseType)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return api.DirectType(context.Factory().TypeQueryNode(reference.EntityName(context.Factory()), nil), reference.Requests()...), nil
}

func (carrier Carrier) ProjectStorage(context api.Context, value api.ExpressionEmission, toStorage bool) (api.ExpressionEmission, error) {
	symbol, err := carrier.storageSymbol(true, toStorage)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	reference, err := context.Names().Runtime(symbol, api.ImportPhaseValue)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(value.Before(), context.Factory().CallExpression(reference.Expression(context.Factory()), nil, nil,
		[]tsgo.Expression{value.Value()}, tsgo.NodeFlagsNone), api.CombineRequests(value.Requests(), reference.Requests()))
}

func (carrier Carrier) storageSymbol(conversion, toStorage bool) (api.RuntimeSymbol, error) {
	switch carrier.bits {
	case 64:
		if !conversion {
			return api.RuntimeComplex64Storage, nil
		}
		if toStorage {
			return api.RuntimeComplex64ToStorage, nil
		}
		return api.RuntimeComplex64FromStorage, nil
	case 128:
		if !conversion {
			return api.RuntimeComplex128Storage, nil
		}
		if toStorage {
			return api.RuntimeComplex128ToStorage, nil
		}
		return api.RuntimeComplex128FromStorage, nil
	default:
		return api.RuntimeInvalid, &api.InvariantError{Reason: "complex storage has no component width"}
	}
}
