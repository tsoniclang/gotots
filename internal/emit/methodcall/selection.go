package methodcall

import (
	"go/ast"
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	genericabi "github.com/tsoniclang/gotots/internal/emit/generic/abi"
	genericinstance "github.com/tsoniclang/gotots/internal/emit/generic/instance"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Selection struct {
	owner         *types.Func
	signature     *types.Signature
	facet         api.CallableFacet
	memberSuffix  string
	typeArguments []tsgo.TypeNode
	capabilities  []genericabi.Binding[tsgo.Expression]
	operations    []*api.GenericOperationContract
	requests      []api.RootRequest
}

func Resolve(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	method *types.Func,
	signature *types.Signature,
) (Selection, error) {
	if method == nil ||
		signature == nil ||
		signature.Recv() == nil ||
		signature.TypeParams().Len() != 0 {
		return Selection{}, invariant(
			context,
			"selected-method signature is invalid",
		)
	}
	owner := method.Origin()
	declaration, ok := owner.Type().(*types.Signature)
	if !ok || declaration.Recv() == nil {
		return Selection{}, invariant(
			context,
			"selected-method owner has no receiver signature",
		)
	}
	concrete, err := genericinstance.ConcreteCallableSignature(signature)
	if err != nil {
		return Selection{}, err
	}
	if declaration.RecvTypeParams().Len() == 0 {
		facet, facetErr := api.NewSourceCallableFacet(owner)
		if facetErr != nil {
			return Selection{}, facetErr
		}
		return Selection{
			owner:     owner,
			signature: concrete,
			facet:     facet,
		}, nil
	}
	operationSet, resolved, err :=
		context.ResolveGenericCallable(owner)
	if err != nil {
		return Selection{}, err
	}
	arguments := genericinstance.ReceiverTypeArguments(
		signature.Recv().Type(),
	)
	if !resolved ||
		arguments == nil ||
		arguments.Len() != len(operationSet.Parameters()) {
		return Selection{}, invariant(
			context,
			"generic selected-method representation is unresolved",
		)
	}
	memberSuffix, facet, _, selectionRequests, err :=
		cooperativecall.SelectGenericClassMethod(
			context,
			owner,
			declaration,
			concrete,
		)
	if err != nil {
		return Selection{}, err
	}
	typeArguments, typeRequests, err :=
		genericinstance.EmitTypeArguments(
			context,
			children,
			source,
			method,
			arguments,
		)
	if err != nil {
		return Selection{}, err
	}
	capabilities, capabilityRequests, err :=
		genericinstance.EmitCapabilities(
			context,
			source,
			operationSet,
			arguments,
		)
	if err != nil {
		return Selection{}, err
	}
	if api.ValueReceiverTypeName(owner) != nil {
		typeArguments = nil
	}
	return Selection{
		owner:         owner,
		signature:     concrete,
		facet:         facet,
		memberSuffix:  memberSuffix,
		typeArguments: slices.Clone(typeArguments),
		capabilities:  slices.Clone(capabilities),
		operations:    slices.Clone(operationSet.Operations()),
		requests: api.CombineRequests(
			selectionRequests,
			typeRequests,
			capabilityRequests,
		),
	}, nil
}

func (s Selection) Signature() *types.Signature {
	return s.signature
}

func (s Selection) Facet() api.CallableFacet {
	return s.facet
}

func (s Selection) Requests() []api.RootRequest {
	return slices.Clone(s.requests)
}

func (s Selection) Call(
	context api.Context,
	receiver tsgo.Expression,
	sourceArguments []tsgo.Expression,
	recovery tsgo.Expression,
) (tsgo.CallExpression, []api.RootRequest, error) {
	if s.owner == nil ||
		s.signature == nil ||
		!s.facet.Valid() ||
		receiver == nil ||
		s.signature.Params().Len() != len(sourceArguments) {
		return nil, nil, invariant(
			context,
			"selected-method invocation does not match its plan",
		)
	}
	arguments := slices.Clone(sourceArguments)
	if len(s.operations) != 0 {
		source, err := genericabi.SourceParameters(
			s.owner,
			sourceArguments,
		)
		if err != nil {
			return nil, nil, err
		}
		arguments, err = genericabi.JoinClassMethod(
			s.owner,
			s.operations,
			genericabi.Combine(s.capabilities, source),
		)
		if err != nil {
			return nil, nil, err
		}
	}
	requests := s.Requests()
	if recovery != nil {
		arguments = append(arguments, recovery)
		control, err := api.NewDirectCallableControlRequest(
			s.owner,
			api.CallableControlRecovery,
		)
		if err != nil {
			return nil, nil, err
		}
		requests = append(requests, control)
	}
	call, callRequests, err := callable.SelectedMethodCall(
		context,
		s.owner,
		s.memberSuffix,
		receiver,
		s.typeArguments,
		arguments,
	)
	return call, api.CombineRequests(requests, callRequests), err
}

func invariant(context api.Context, reason string) error {
	return &api.InvariantError{
		Role:   context.Role(),
		Reason: reason,
	}
}
