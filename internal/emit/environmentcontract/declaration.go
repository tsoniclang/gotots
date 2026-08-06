package environmentcontract

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	constantbinding "github.com/tsoniclang/gotots/internal/emit/constant"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Declaration(
	context api.Context,
	children api.ChildEmitter,
	object types.Object,
	requirements []api.DeclarationRequirement,
) (api.DeclarationEmission, error) {
	switch selected := object.(type) {
	case *types.Func:
		return callableDeclaration(
			context,
			children,
			selected.Origin(),
			requirements,
		)
	case *types.Const:
		if len(requirements) != 0 {
			return api.DeclarationEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "environment constant acquired declaration requirements",
			}
		}
		if constantbinding.IsUntyped(selected.Type()) {
			return api.CoverageOnlyDeclarationEmission(), nil
		}
		return typedConstantDeclaration(context, children, selected)
	case *types.TypeName:
		return typeDeclaration(context, children, selected, requirements)
	default:
		return api.DeclarationEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "environment contract object kind is unsupported",
		}
	}
}

func callableDeclaration(
	context api.Context,
	children api.ChildEmitter,
	function *types.Func,
	requirements []api.DeclarationRequirement,
) (api.DeclarationEmission, error) {
	signature, ok := function.Type().(*types.Signature)
	if !ok {
		return api.DeclarationEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "environment callable has no signature",
		}
	}
	recovery, err := requiresRecoveryAuthority(function, requirements)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	declarations, requests, err := callableContractDeclaration(
		context,
		children,
		function,
		signature,
		requirements,
		recovery,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	return api.NewDeclarationEmission(
		declarations,
		api.CombineRequests(requests),
	)
}

func callableContractDeclaration(
	context api.Context,
	children api.ChildEmitter,
	function *types.Func,
	signature *types.Signature,
	requirements []api.DeclarationRequirement,
	recovery bool,
) ([]tsgo.Statement, []api.RootRequest, error) {
	generic, err := enterGeneric(context, function, requirements)
	if err != nil {
		return nil, nil, err
	}
	context = generic.context
	target, err := callable.EmitEnvironmentContract(
		context,
		children,
		signature,
	)
	if err != nil {
		return nil, nil, err
	}
	parameters := target.Parameters()
	requests := target.Requests()
	if signature.Recv() != nil {
		receiver, err := children.RepresentedType(
			context.WithRole(api.RoleReceiverType),
			nil,
			signature.Recv().Type(),
		)
		if err != nil {
			return nil, nil, err
		}
		parameters = append(
			[]tsgo.ParameterDeclaration{parameter(
				context,
				"$receiver",
				receiver.Value(),
			)},
			parameters...,
		)
		requests = api.CombineRequests(receiver.Requests(), requests)
	}
	name, err := context.Names().Declare(function)
	if err != nil {
		return nil, nil, err
	}
	declarations := []tsgo.Statement{context.Factory().FunctionDeclaration(
		exportDeclare(context),
		nil,
		context.Factory().Identifier(name),
		generic.parameters,
		parameters,
		target.Result(),
		nil,
	)}
	if recovery {
		recoveryParameter, recoveryRequests, recoveryErr :=
			callable.RecoveryAuthorityParameter(context)
		if recoveryErr != nil {
			return nil, nil, recoveryErr
		}
		declarations = append(
			declarations,
			context.Factory().FunctionDeclaration(
				exportDeclare(context),
				nil,
				context.Factory().Identifier(name+api.DeferredEntrySuffix),
				generic.parameters,
				append(
					[]tsgo.ParameterDeclaration{recoveryParameter},
					parameters...,
				),
				target.Result(),
				nil,
			),
		)
		requests = api.CombineRequests(requests, recoveryRequests)
	}
	return declarations, requests, nil
}

func requiresRecoveryAuthority(
	function *types.Func,
	requirements []api.DeclarationRequirement,
) (bool, error) {
	selected := false
	for _, requirement := range requirements {
		if requirement.Kind() ==
			api.DeclarationRequirementGenericRepresentation {
			continue
		}
		owner, enclosing, callable, control, ok :=
			requirement.CallableControl()
		source, sourceOwned := owner.Source()
		if !ok ||
			!sourceOwned ||
			source != function ||
			enclosing != nil ||
			callable != nil ||
			control != api.CallableControlRecovery {
			return false, &api.InvariantError{
				Reason: "environment callable requirement is invalid",
			}
		}
		if selected {
			return false, &api.InvariantError{
				Reason: "environment recovery requirement is duplicated",
			}
		}
		selected = true
	}
	return selected, nil
}

func typedConstantDeclaration(
	context api.Context,
	children api.ChildEmitter,
	selected *types.Const,
) (api.DeclarationEmission, error) {
	context, err := context.WithSourceArtifactOwner(
		api.MustSourceArtifactOwner(selected),
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	target, err := children.RepresentedType(
		context.WithRole(api.RolePackageConstantType),
		nil,
		selected.Type(),
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	name, err := context.Names().Declare(selected)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	return api.DirectDeclaration(
		ambientConstant(context, name, target.Value()),
		target.Requests()...,
	), nil
}

func ConstantProjection(
	context api.Context,
	children api.ChildEmitter,
	selected *types.Const,
	projection types.BasicKind,
) (tsgo.Statement, []api.RootRequest, error) {
	targetType, ok := api.ConstantProjectionType(projection)
	if selected == nil || !ok {
		return nil, nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "environment constant projection is invalid",
		}
	}
	target, err := children.RepresentedType(
		context.WithRole(api.RolePackageConstantType),
		nil,
		targetType,
	)
	if err != nil {
		return nil, nil, err
	}
	base, err := context.Names().Declare(selected)
	if err != nil {
		return nil, nil, err
	}
	name, err := api.ConstantProjectionName(base, projection)
	if err != nil {
		return nil, nil, err
	}
	return ambientConstant(context, name, target.Value()),
		target.Requests(),
		nil
}

func ambientConstant(
	context api.Context,
	name string,
	targetType tsgo.TypeNode,
) tsgo.VariableStatement {
	return context.Factory().VariableStatement(
		exportDeclare(context),
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				context.Factory().VariableDeclaration(
					context.Factory().Identifier(name),
					nil,
					targetType,
					nil,
				),
			},
			tsgo.NodeFlagsConst,
		),
	)
}

func exportDeclare(context api.Context) []tsgo.ModifierLike {
	return []tsgo.ModifierLike{
		context.Factory().ExportKeyword(),
		context.Factory().DeclareKeyword(),
	}
}

func parameter(
	context api.Context,
	name string,
	targetType tsgo.TypeNode,
) tsgo.ParameterDeclaration {
	return context.Factory().ParameterDeclaration(
		nil,
		nil,
		context.Factory().Identifier(name),
		nil,
		targetType,
		nil,
	)
}
