package emit

import (
	"crypto/sha256"
	"encoding/hex"
	"go/token"
	"go/types"
	"sort"

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
	sort.Slice(operations, func(left, right int) bool {
		return operations[left].Key() < operations[right].Key()
	})
	operationSet, err := api.NewGenericOperationSet(
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
	resolve := func(
		references []gostdlib.GenericOperationTypeDocument,
	) (*types.Tuple, error) {
		variables := make([]*types.Var, 0, len(references))
		for _, reference := range references {
			if reference.TypeParameter < 0 ||
				reference.TypeParameter >= len(parameters) {
				return nil, &ScheduleError{
					Object: owner.Name(),
					Reason: "provider generic operation type parameter is invalid",
				}
			}
			variables = append(variables, types.NewVar(
				token.NoPos,
				owner.Pkg(),
				"",
				parameters[reference.TypeParameter],
			))
		}
		return types.NewTuple(variables...), nil
	}
	params, err := resolve(document.Parameters)
	if err != nil {
		return nil, err
	}
	results, err := resolve(document.Results)
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
