package declaration

import (
	"go/ast"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Parameters struct {
	context      api.Context
	typeNodes    []tsgo.TypeParameterDeclaration
	capabilities []tsgo.ParameterDeclaration
	requests     []api.RootRequest
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
	signature, ok := owner.Type().(*types.Signature)
	if !ok {
		return Parameters{}, api.Unsupported(
			context,
			api.CategoryDeclaration,
			source,
		)
	}
	parameters := signatureTypeParameters(signature)
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
	context = context.WithGenericParameters(owner, names)
	operations, err := appliedOperations(owner, requirements)
	if err != nil {
		return Parameters{}, err
	}
	var capabilities []tsgo.ParameterDeclaration
	var requests []api.RootRequest
	for _, operation := range operations {
		target, targetErr := callable.EmitNonNilType(
			context.WithRole(api.RoleParameterType),
			children,
			source.Type,
			operation.Signature(),
		)
		if targetErr != nil {
			return Parameters{}, targetErr
		}
		capabilities = append(
			capabilities,
			context.Factory().ParameterDeclaration(
				nil,
				nil,
				context.Factory().Identifier(operation.TargetName()),
				nil,
				target.Value(),
				nil,
			),
		)
		requests = append(requests, target.Requests()...)
	}
	return Parameters{
		context:      context,
		typeNodes:    typeNodes,
		capabilities: capabilities,
		requests:     requests,
	}, nil
}

func (p Parameters) Context() api.Context {
	return p.context
}

func (p Parameters) TypeNodes() []tsgo.TypeParameterDeclaration {
	return append([]tsgo.TypeParameterDeclaration(nil), p.typeNodes...)
}

func (p Parameters) Capabilities() []tsgo.ParameterDeclaration {
	return append([]tsgo.ParameterDeclaration(nil), p.capabilities...)
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

func signatureTypeParameters(
	signature *types.Signature,
) []*types.TypeParam {
	var parameters []*types.TypeParam
	for _, list := range []*types.TypeParamList{
		signature.RecvTypeParams(),
		signature.TypeParams(),
	} {
		for index := range list.Len() {
			parameters = append(parameters, list.At(index))
		}
	}
	return parameters
}
