package declaration

import (
	"go/ast"
	"go/types"
	"sort"

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
	names := make(map[*types.TypeParam]string, len(parameters))
	nodes := make([]tsgo.TypeParameterDeclaration, 0, len(parameters))
	references := make([]tsgo.TypeNode, 0, len(parameters))
	for _, parameter := range parameters {
		name, err := context.Names().Declare(parameter.Obj())
		if err != nil {
			return TypeParameters{}, err
		}
		names[parameter] = name
		nodes = append(nodes, context.Factory().TypeParameterDeclaration(
			nil,
			context.Factory().Identifier(name),
			nil,
			nil,
			nil,
		))
		references = append(references, context.Factory().TypeReferenceNode(
			context.Factory().Identifier(name),
			nil,
		))
	}
	context, err := context.WithGenericParameters(owner, names)
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
	names := make(map[*types.TypeParam]string, len(parameters))
	typeNodes := make(
		[]tsgo.TypeParameterDeclaration,
		0,
		len(parameters),
	)
	for _, parameter := range parameters {
		name, err := context.Names().Declare(parameter.Obj())
		if err != nil {
			return Parameters{}, err
		}
		names[parameter] = name
		typeNodes = append(
			typeNodes,
			context.Factory().TypeParameterDeclaration(
				nil,
				context.Factory().Identifier(name),
				nil,
				nil,
				nil,
			),
		)
	}
	context, err := context.WithGenericParameters(owner, names)
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
		target, err := callable.EmitInlineNonNilType(
			context.WithRole(api.RoleParameterType),
			children,
			source,
			operation.Signature(),
			false,
		)
		if err != nil {
			return nil, nil, err
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
		requests = append(requests, target.Requests()...)
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
