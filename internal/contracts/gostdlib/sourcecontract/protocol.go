package sourcecontract

import (
	"fmt"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

type ProtocolError struct {
	Field  string
	Reason string
}

func (e *ProtocolError) Error() string {
	return fmt.Sprintf("resolve provider protocol %s: %s", e.Field, e.Reason)
}

func ResolveProviderProtocolInterface(
	source gostdlib.ProviderProtocolInterfaceDocument,
	owner *types.Signature,
) (*types.Interface, error) {
	if owner == nil {
		return nil, &ProtocolError{
			Field:  "owner",
			Reason: "callable signature is nil",
		}
	}
	canonical, err := gostdlib.CanonicalProviderProtocolInterface(source)
	if err != nil {
		return nil, err
	}
	typeParameters := callableTypeParameters(owner)
	resolveType := func(
		reference gostdlib.ContractTypeDocument,
	) (types.Type, error) {
		return resolveContractType(reference, owner, typeParameters)
	}
	methods := make([]*types.Func, 0, len(canonical.Methods))
	for _, method := range canonical.Methods {
		parameters, err := resolveContractTuple(method.Parameters, resolveType)
		if err != nil {
			return nil, err
		}
		results, err := resolveContractTuple(method.Results, resolveType)
		if err != nil {
			return nil, err
		}
		methods = append(methods, types.NewFunc(
			token.NoPos,
			nil,
			method.Name,
			types.NewSignatureType(nil, nil, nil, parameters, results, false),
		))
	}
	return types.NewInterfaceType(methods, nil).Complete(), nil
}

func callableTypeParameters(signature *types.Signature) []*types.TypeParam {
	result := make(
		[]*types.TypeParam,
		0,
		signature.RecvTypeParams().Len()+signature.TypeParams().Len(),
	)
	for index := range signature.RecvTypeParams().Len() {
		result = append(result, signature.RecvTypeParams().At(index))
	}
	for index := range signature.TypeParams().Len() {
		result = append(result, signature.TypeParams().At(index))
	}
	return result
}

func resolveContractType(
	reference gostdlib.ContractTypeDocument,
	owner *types.Signature,
	typeParameters []*types.TypeParam,
) (types.Type, error) {
	switch reference.Kind {
	case gostdlib.ContractTypeParameter:
		if reference.TypeParameter != nil &&
			*reference.TypeParameter >= 0 &&
			*reference.TypeParameter < len(typeParameters) {
			return typeParameters[*reference.TypeParameter], nil
		}
	case gostdlib.ContractTypeCallableParameter:
		if reference.CallableParameter != nil && owner.Params() != nil &&
			*reference.CallableParameter >= 0 &&
			*reference.CallableParameter < owner.Params().Len() {
			return owner.Params().At(*reference.CallableParameter).Type(), nil
		}
	case gostdlib.ContractTypeBool:
		return types.Typ[types.Bool], nil
	case gostdlib.ContractTypeInt:
		return types.Typ[types.Int], nil
	case gostdlib.ContractTypeSlice:
		if reference.Element != nil {
			element, err := resolveContractType(
				*reference.Element,
				owner,
				typeParameters,
			)
			if err != nil {
				return nil, err
			}
			return types.NewSlice(element), nil
		}
	case gostdlib.ContractTypeMap:
		if reference.Key != nil && reference.Element != nil {
			key, err := resolveContractType(*reference.Key, owner, typeParameters)
			if err != nil {
				return nil, err
			}
			element, err := resolveContractType(
				*reference.Element,
				owner,
				typeParameters,
			)
			if err != nil {
				return nil, err
			}
			return types.NewMap(key, element), nil
		}
	}
	return nil, &ProtocolError{
		Field:  "type",
		Reason: "type expression is outside its callable declaration",
	}
}

func resolveContractTuple(
	references []gostdlib.ContractTypeDocument,
	resolve func(gostdlib.ContractTypeDocument) (types.Type, error),
) (*types.Tuple, error) {
	variables := make([]*types.Var, 0, len(references))
	for _, reference := range references {
		selected, err := resolve(reference)
		if err != nil {
			return nil, err
		}
		variables = append(variables, types.NewVar(token.NoPos, nil, "", selected))
	}
	return types.NewTuple(variables...), nil
}
