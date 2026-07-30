package declaration

import (
	"go/ast"
	"go/types"
	"sort"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	genericabi "github.com/tsoniclang/gotots/internal/emit/generic/abi"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Parameters struct {
	context      api.Context
	typeNodes    []tsgo.TypeParameterDeclaration
	operations   []*api.GenericOperationContract
	capabilities []genericabi.Binding[tsgo.ParameterDeclaration]
	requests     []api.RootRequest
}

type TypeParameters struct {
	context    api.Context
	nodes      []tsgo.TypeParameterDeclaration
	references []tsgo.TypeNode
}

func EnterType(
	context api.Context,
	source *ast.TypeSpec,
	owner *types.TypeName,
	requirements []api.DeclarationRequirement,
) (TypeParameters, error) {
	if source == nil || owner == nil {
		return TypeParameters{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic type declaration identity is invalid",
		}
	}
	parameters := api.GenericDeclarationParameters(owner)
	if len(parameters) == 0 {
		if source.TypeParams != nil {
			return TypeParameters{}, api.Unsupported(
				context,
				api.CategoryDeclaration,
				source.TypeParams,
			)
		}
		return TypeParameters{context: context}, nil
	}
	if source.TypeParams == nil {
		return TypeParameters{}, api.Unsupported(
			context,
			api.CategoryDeclaration,
			source,
		)
	}
	context, nodes, references, err := enterTypeParameterScope(
		context,
		owner,
		requirements,
	)
	if err != nil {
		return TypeParameters{}, err
	}
	return TypeParameters{
		context:    context,
		nodes:      nodes,
		references: references,
	}, nil
}

func (p TypeParameters) Context() api.Context {
	return p.context
}

func (p TypeParameters) Nodes() []tsgo.TypeParameterDeclaration {
	return append([]tsgo.TypeParameterDeclaration(nil), p.nodes...)
}

func (p TypeParameters) References() []tsgo.TypeNode {
	return append([]tsgo.TypeNode(nil), p.references...)
}

func Enter(
	context api.Context,
	children api.ChildEmitter,
	source *ast.FuncDecl,
	owner *types.Func,
	requirements []api.DeclarationRequirement,
) (Parameters, error) {
	if source == nil || owner == nil || owner.Origin() != owner {
		return Parameters{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic declaration identity is invalid",
		}
	}
	_, ok := owner.Type().(*types.Signature)
	if !ok {
		return Parameters{}, api.Unsupported(
			context,
			api.CategoryDeclaration,
			source,
		)
	}
	parameters := api.GenericDeclarationParameters(owner)
	if len(parameters) == 0 {
		if source.Type.TypeParams != nil {
			return Parameters{}, api.Unsupported(
				context,
				api.CategoryDeclaration,
				source.Type.TypeParams,
			)
		}
		return Parameters{context: context}, nil
	}
	context, typeNodes, _, err := enterTypeParameterScope(
		context,
		owner,
		requirements,
	)
	if err != nil {
		return Parameters{}, err
	}
	operations, err := appliedOperations(owner, requirements)
	if err != nil {
		return Parameters{}, err
	}
	capabilities, requests, err := EmitOperationParameters(
		context,
		children,
		source.Type,
		operations,
	)
	if err != nil {
		return Parameters{}, err
	}
	return Parameters{
		context:      context,
		typeNodes:    typeNodes,
		operations:   operations,
		capabilities: capabilities,
		requests:     requests,
	}, nil
}

func enterTypeParameterScope(
	context api.Context,
	owner types.Object,
	requirements []api.DeclarationRequirement,
) (
	api.Context,
	[]tsgo.TypeParameterDeclaration,
	[]tsgo.TypeNode,
	error,
) {
	parameters := api.GenericDeclarationParameters(owner)
	profile, err := api.SelectGenericRepresentationProfile(
		owner,
		requirements,
	)
	if err != nil {
		return api.Context{}, nil, nil, err
	}
	names := make(map[*types.TypeParam]string, len(parameters))
	for index, parameter := range parameters {
		nameParameter := parameter
		if representationOwner, selected, ok :=
			api.GenericRepresentationParameter(owner, parameter); ok &&
			representationOwner != api.GenericDeclarationOrigin(owner) {
			nameParameter = selected
		}
		name, nameErr := typeParameterName(context, nameParameter, index)
		if nameErr != nil {
			return api.Context{}, nil, nil, nameErr
		}
		names[parameter] = name
	}
	layout, err := EmitTypeParameterLayout(context, profile, names)
	if err != nil {
		return api.Context{}, nil, nil, err
	}
	context, err = context.WithGenericParameters(owner, names)
	if err != nil {
		return api.Context{}, nil, nil, err
	}
	return context, layout.Parameters(), layout.Arguments(), nil
}

func EnterClassMethod(
	context api.Context,
	children api.ChildEmitter,
	source *ast.FuncDecl,
	owner *types.Func,
	requirements []api.DeclarationRequirement,
) (Parameters, error) {
	signature, ok := owner.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return Parameters{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "class-method generic identity is invalid",
		}
	}
	parameters, err := Enter(
		context,
		children,
		source,
		owner,
		requirements,
	)
	if err != nil {
		return Parameters{}, err
	}
	if api.ValueReceiverTypeName(owner) != nil {
		parameters.typeNodes = nil
	}
	return parameters, nil
}

func typeParameterName(
	context api.Context,
	parameter *types.TypeParam,
	index int,
) (string, error) {
	if parameter == nil || parameter.Obj() == nil || index < 0 {
		return "", &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic type-parameter identity is invalid",
		}
	}
	if parameter.Obj().Name() == "" || parameter.Obj().Name() == "_" {
		return "$T" + strconv.Itoa(index), nil
	}
	return context.Names().Declare(parameter.Obj())
}

func EmitOperationParameters(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	operations []*api.GenericOperationContract,
) (
	[]genericabi.Binding[tsgo.ParameterDeclaration],
	[]api.RootRequest,
	error,
) {
	capabilities := make(
		[]genericabi.Binding[tsgo.ParameterDeclaration],
		0,
		len(operations),
	)
	var requests []api.RootRequest
	for _, operation := range operations {
		if !operation.Valid() {
			return nil, nil, &api.InvariantError{
				Role:   context.Role(),
				Reason: "generic operation parameter is invalid",
			}
		}
		var (
			target        api.TypeEmission
			targetErr     error
			facetRequests []api.RootRequest
		)
		target, storageOperation, targetErr := emitStorageOperationType(
			context,
			children,
			source,
			operation,
		)
		if targetErr != nil {
			return nil, nil, targetErr
		}
		handled := storageOperation
		if !handled {
			target, handled, targetErr = emitPointerOperationType(
				context,
				children,
				source,
				operation,
			)
			if targetErr != nil {
				return nil, nil, targetErr
			}
		}
		if !handled && operation.Operation() ==
			api.GenericOperationConstraintMethod {
			facet, facetErr :=
				api.NewGenericOperationCallableFacet(operation)
			if facetErr != nil {
				return nil, nil, facetErr
			}
			observation, observationErr :=
				context.ObserveCooperativeCallable(facet)
			if observationErr != nil {
				return nil, nil, observationErr
			}
			target, targetErr = callable.EmitInlineAwaitableType(
				context.WithRole(api.RoleParameterType),
				children,
				source,
				operation.Signature(),
				observation.Cooperative(),
			)
			facetRequests = observation.Requests()
		} else if !handled {
			target, targetErr = callable.EmitInlineNonNilType(
				context.WithRole(api.RoleParameterType),
				children,
				source,
				operation.Signature(),
				false,
			)
		}
		if targetErr != nil {
			return nil, nil, targetErr
		}
		parameter := context.Factory().ParameterDeclaration(
			nil,
			nil,
			context.Factory().Identifier(operation.TargetName()),
			nil,
			target.Value(),
			nil,
		)
		binding, err := genericabi.Capability(operation, parameter)
		if err != nil {
			return nil, nil, err
		}
		capabilities = append(capabilities, binding)
		requests = append(
			requests,
			api.CombineRequests(
				target.Requests(),
				facetRequests,
			)...,
		)
	}
	return capabilities, requests, nil
}

func (p Parameters) Context() api.Context {
	return p.context
}

func (p Parameters) TypeNodes() []tsgo.TypeParameterDeclaration {
	return append([]tsgo.TypeParameterDeclaration(nil), p.typeNodes...)
}

func (p Parameters) Operations() []*api.GenericOperationContract {
	return append([]*api.GenericOperationContract(nil), p.operations...)
}

func (p Parameters) Capabilities() []genericabi.Binding[tsgo.ParameterDeclaration] {
	return append(
		[]genericabi.Binding[tsgo.ParameterDeclaration](nil),
		p.capabilities...,
	)
}

func (p Parameters) Requests() []api.RootRequest {
	return append([]api.RootRequest(nil), p.requests...)
}

func appliedOperations(
	owner *types.Func,
	requirements []api.DeclarationRequirement,
) ([]*api.GenericOperationContract, error) {
	var operations []*api.GenericOperationContract
	for _, requirement := range requirements {
		if requirement.Kind() != api.DeclarationRequirementGenericOperation {
			continue
		}
		selectedOwner, operation, ok := requirement.GenericOperation()
		if !ok || selectedOwner != owner {
			return nil, &api.InvariantError{
				Role:   api.RoleFileDeclaration,
				Reason: "generic declaration received foreign operation",
			}
		}
		operations = append(operations, operation)
	}
	sort.Slice(operations, func(left, right int) bool {
		return operations[left].Key() < operations[right].Key()
	})
	for index := 1; index < len(operations); index++ {
		if operations[index-1].Key() == operations[index].Key() {
			return nil, &api.InvariantError{
				Role:   api.RoleFileDeclaration,
				Reason: "generic declaration received duplicate operation",
			}
		}
	}
	return operations, nil
}
