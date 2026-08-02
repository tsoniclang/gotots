package emit

import (
	"crypto/sha256"
	"encoding/hex"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

func (s *programSession) providerGenericOperationSet(
	owner types.Object,
	consumer api.GenericOperationConsumer,
) (api.GenericOperationSet, bool, error) {
	function, callable := owner.(*types.Func)
	if !callable {
		operationSet, err := api.NewGenericOperationSet(
			owner,
			consumer,
			nil,
		)
		return operationSet, err == nil, err
	}
	documents, providerOwned, err :=
		s.registry.ProviderGenericOperations(function)
	if err != nil {
		return api.GenericOperationSet{}, false, err
	}
	if !providerOwned {
		operationSet, setErr := api.NewGenericOperationSet(
			function,
			consumer,
			nil,
		)
		return operationSet, setErr == nil, setErr
	}
	if consumer != api.GenericFunctionOperationConsumer() {
		return api.GenericOperationSet{}, false, &ScheduleError{
			Object: function.Name(),
			Reason: "provider generic operations have a non-function consumer",
		}
	}
	operations := make(
		[]*api.GenericOperationContract,
		0,
		len(documents),
	)
	for _, document := range documents {
		selection, selectionErr := providerGenericOperationSelection(
			document.Kind,
		)
		if selectionErr != nil {
			return api.GenericOperationSet{}, false, selectionErr
		}
		signature, signatureErr := providerGenericOperationSignature(
			function,
			document,
		)
		if signatureErr != nil {
			return api.GenericOperationSet{}, false, signatureErr
		}
		operation, operationErr := s.internGenericOperation(
			function,
			consumer,
			selection,
			signature,
		)
		if operationErr != nil {
			return api.GenericOperationSet{}, false, operationErr
		}
		operations = append(operations, operation)
	}
	operationSet, err := api.NewGenericOperationABISet(
		function,
		consumer,
		operations,
	)
	return operationSet, err == nil, err
}

func providerGenericOperationSelection(
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
		return api.GenericOperationSelection{}, &ScheduleError{
			Reason: "provider generic operation kind is invalid",
		}
	}
	return api.SelectGenericOperation(operation)
}

func providerGenericOperationSignature(
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
				return nil, &ScheduleError{
					Object: owner.Name(),
					Reason: "provider generic operation type parameter is invalid",
				}
			}
			return parameters[*reference.TypeParameter], nil
		case gostdlib.ContractTypeCallableParameter:
			if declaration == nil || reference.CallableParameter == nil ||
				*reference.CallableParameter < 0 ||
				*reference.CallableParameter >= declaration.Params().Len() {
				return nil, &ScheduleError{
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
		return nil, &ScheduleError{
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

func (s *programSession) internGenericOperation(
	owner types.Object,
	consumer api.GenericOperationConsumer,
	selection api.GenericOperationSelection,
	signature *types.Signature,
) (*api.GenericOperationContract, error) {
	key, err := s.genericOperationKey(
		owner,
		consumer,
		selection,
		signature,
	)
	if err != nil {
		return nil, err
	}
	identity := genericOperationIdentity{
		owner:    owner,
		consumer: consumer,
		key:      key,
	}
	if existing := s.genericOperations[identity]; existing != nil {
		if existing.Consumer() != consumer ||
			existing.Selection() != selection ||
			!types.Identical(existing.Signature(), signature) {
			return nil, &ScheduleError{
				Object: owner.Name(),
				Reason: "generic operation identity changed semantic contract",
			}
		}
		return existing, nil
	}
	digest := sha256.Sum256([]byte(key))
	targetName := "$go$" + selection.Operation().Identifier() + "_" +
		hex.EncodeToString(digest[:10])
	contract, err := api.NewGenericOperationContract(
		owner,
		key,
		targetName,
		consumer,
		selection,
		signature,
	)
	if err != nil {
		return nil, err
	}
	s.genericOperations[identity] = contract
	return contract, nil
}
