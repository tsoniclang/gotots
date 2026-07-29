package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitGeneric(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	discarded bool,
) (api.ExpressionEmission, bool, error) {
	owner, instance, ok := genericFunctionInstance(
		context.TypesInfo(),
		source.Fun,
	)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	signature, ok := instance.Type.(*types.Signature)
	if !ok || signature.Recv() != nil {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	callable, ok, err := context.ResolveGenericCallable(owner)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if !ok ||
		instance.TypeArgs == nil ||
		instance.TypeArgs.Len() != len(callable.Parameters()) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if err := validateResults(context, source, signature, discarded); err != nil {
		return api.ExpressionEmission{}, true, err
	}
	reference, err := context.Names().Reference(owner)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	typeArguments, typeRequests, err := emitGenericTypeArguments(
		context,
		children,
		source,
		instance.TypeArgs,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	capabilities, capabilityRequests, err := emitGenericCapabilities(
		context,
		source,
		callable,
		instance.TypeArgs,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	arguments, before, argumentRequests, err := emitArguments(
		context,
		children,
		source,
		signature,
		false,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	arguments = append(capabilities, arguments...)
	result, err := api.NewExpressionEmission(
		before,
		context.Factory().CallExpression(
			context.Factory().Identifier(reference.Name()),
			nil,
			typeArguments,
			arguments,
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			reference.Requests(),
			typeRequests,
			capabilityRequests,
			argumentRequests,
		),
	)
	return result, true, err
}

func genericFunctionInstance(
	info *types.Info,
	source ast.Expr,
) (*types.Func, types.Instance, bool) {
	if info == nil || source == nil {
		return nil, types.Instance{}, false
	}
	base := source
	switch selected := source.(type) {
	case *ast.IndexExpr:
		base = selected.X
	case *ast.IndexListExpr:
		base = selected.X
	}
	var identifier *ast.Ident
	switch selected := base.(type) {
	case *ast.Ident:
		identifier = selected
	case *ast.SelectorExpr:
		if info.Selections[selected] != nil {
			return nil, types.Instance{}, false
		}
		identifier = selected.Sel
	default:
		return nil, types.Instance{}, false
	}
	instance, instantiated := info.Instances[identifier]
	owner, function := info.Uses[identifier].(*types.Func)
	if !instantiated || !function {
		return nil, types.Instance{}, false
	}
	return owner.Origin(), instance, true
}

func emitGenericTypeArguments(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	arguments *types.TypeList,
) ([]tsgo.TypeNode, []api.RootRequest, error) {
	targets := make([]tsgo.TypeNode, 0, arguments.Len())
	var requests []api.RootRequest
	for index := range arguments.Len() {
		target, err := children.RepresentedType(
			context.WithRole(api.RoleCallArgumentType),
			source,
			arguments.At(index),
		)
		if err != nil {
			return nil, nil, err
		}
		targets = append(targets, target.Value())
		requests = append(requests, target.Requests()...)
	}
	return targets, requests, nil
}

func emitGenericCapabilities(
	context api.Context,
	source ast.Node,
	callable api.GenericCallable,
	arguments *types.TypeList,
) ([]tsgo.Expression, []api.RootRequest, error) {
	targets := make([]tsgo.Expression, 0, len(callable.Operations()))
	var requests []api.RootRequest
	for _, operation := range callable.Operations() {
		signature, err := instantiateOperation(
			callable,
			operation.Signature(),
			arguments,
		)
		if err != nil {
			return nil, nil, err
		}
		var reference api.NameReference
		if api.ContainsGenericTypeParameter(signature) {
			reference, err = context.ProjectGenericOperation(
				source,
				operation,
				signature,
			)
		} else {
			reference, err = context.Names().GenericCapability(
				operation.Selection(),
				signature,
			)
		}
		if err != nil {
			return nil, nil, err
		}
		targets = append(
			targets,
			context.Factory().Identifier(reference.Name()),
		)
		requests = append(requests, reference.Requests()...)
	}
	return targets, requests, nil
}

func instantiateOperation(
	callable api.GenericCallable,
	operation *types.Signature,
	arguments *types.TypeList,
) (*types.Signature, error) {
	parameters := callable.Parameters()
	if operation == nil || arguments == nil ||
		len(parameters) != arguments.Len() {
		return nil, &api.InvariantError{
			Role:   api.RoleCallCallee,
			Reason: "generic operation instantiation is inconsistent",
		}
	}
	replacements := make(
		map[*types.TypeParam]*types.TypeParam,
		len(parameters),
	)
	fresh := make([]*types.TypeParam, 0, len(parameters))
	for _, parameter := range parameters {
		object := parameter.Obj()
		clone := types.NewTypeParam(
			types.NewTypeName(
				object.Pos(),
				object.Pkg(),
				object.Name(),
				nil,
			),
			types.NewInterfaceType(nil, nil).Complete(),
		)
		replacements[parameter] = clone
		fresh = append(fresh, clone)
	}
	params, err := substituteTuple(operation.Params(), replacements)
	if err != nil {
		return nil, err
	}
	results, err := substituteTuple(operation.Results(), replacements)
	if err != nil {
		return nil, err
	}
	generic := types.NewSignatureType(
		nil,
		nil,
		fresh,
		params,
		results,
		operation.Variadic(),
	)
	typeArguments := make([]types.Type, 0, arguments.Len())
	for index := range arguments.Len() {
		typeArguments = append(typeArguments, arguments.At(index))
	}
	instantiated, err := types.Instantiate(
		nil,
		generic,
		typeArguments,
		false,
	)
	if err != nil {
		return nil, err
	}
	signature, ok := instantiated.(*types.Signature)
	if !ok {
		return nil, &api.InvariantError{
			Role:   api.RoleCallCallee,
			Reason: "generic operation did not instantiate to a signature",
		}
	}
	return signature, nil
}

func substituteTuple(
	source *types.Tuple,
	replacements map[*types.TypeParam]*types.TypeParam,
) (*types.Tuple, error) {
	if source == nil {
		return nil, nil
	}
	variables := make([]*types.Var, 0, source.Len())
	for index := range source.Len() {
		variable := source.At(index)
		target, err := substituteType(variable.Type(), replacements)
		if err != nil {
			return nil, err
		}
		variables = append(
			variables,
			types.NewVar(
				variable.Pos(),
				variable.Pkg(),
				variable.Name(),
				target,
			),
		)
	}
	return types.NewTuple(variables...), nil
}

func substituteType(
	source types.Type,
	replacements map[*types.TypeParam]*types.TypeParam,
) (types.Type, error) {
	switch source := source.(type) {
	case *types.Basic:
		return source, nil
	case *types.TypeParam:
		target := replacements[source]
		if target == nil {
			return source, nil
		}
		return target, nil
	case *types.Pointer:
		element, err := substituteType(source.Elem(), replacements)
		return types.NewPointer(element), err
	case *types.Slice:
		element, err := substituteType(source.Elem(), replacements)
		return types.NewSlice(element), err
	case *types.Array:
		element, err := substituteType(source.Elem(), replacements)
		return types.NewArray(element, source.Len()), err
	case *types.Map:
		key, err := substituteType(source.Key(), replacements)
		if err != nil {
			return nil, err
		}
		element, err := substituteType(source.Elem(), replacements)
		if err != nil {
			return nil, err
		}
		return types.NewMap(key, element), nil
	case *types.Chan:
		element, err := substituteType(source.Elem(), replacements)
		return types.NewChan(source.Dir(), element), err
	case *types.Named:
		if source.TypeArgs().Len() == 0 {
			return source, nil
		}
		arguments, err := substituteTypeList(
			source.TypeArgs(),
			replacements,
		)
		if err != nil {
			return nil, err
		}
		return types.Instantiate(nil, source.Origin(), arguments, false)
	case *types.Alias:
		if source.TypeArgs().Len() == 0 {
			return source, nil
		}
		arguments, err := substituteTypeList(
			source.TypeArgs(),
			replacements,
		)
		if err != nil {
			return nil, err
		}
		return types.Instantiate(nil, source.Origin(), arguments, false)
	case *types.Tuple:
		return substituteTuple(source, replacements)
	case *types.Signature:
		if source.TypeParams().Len() != 0 ||
			source.RecvTypeParams().Len() != 0 {
			return nil, &api.InvariantError{
				Role:   api.RoleCallCallee,
				Reason: "nested generic operation signature is unsupported",
			}
		}
		parameters, err := substituteTuple(source.Params(), replacements)
		if err != nil {
			return nil, err
		}
		results, err := substituteTuple(source.Results(), replacements)
		if err != nil {
			return nil, err
		}
		return types.NewSignatureType(
			nil,
			nil,
			nil,
			parameters,
			results,
			source.Variadic(),
		), nil
	case *types.Struct:
		fields := make([]*types.Var, 0, source.NumFields())
		tags := make([]string, 0, source.NumFields())
		for index := range source.NumFields() {
			field := source.Field(index)
			fieldType, err := substituteType(
				field.Type(),
				replacements,
			)
			if err != nil {
				return nil, err
			}
			fields = append(fields, types.NewField(
				field.Pos(),
				field.Pkg(),
				field.Name(),
				fieldType,
				field.Embedded(),
			))
			tags = append(tags, source.Tag(index))
		}
		return types.NewStruct(fields, tags), nil
	default:
		return nil, &api.InvariantError{
			Role:   api.RoleCallCallee,
			Reason: "generic operation contains an unsupported type form",
		}
	}
}

func substituteTypeList(
	source *types.TypeList,
	replacements map[*types.TypeParam]*types.TypeParam,
) ([]types.Type, error) {
	result := make([]types.Type, 0, source.Len())
	for index := range source.Len() {
		target, err := substituteType(source.At(index), replacements)
		if err != nil {
			return nil, err
		}
		result = append(result, target)
	}
	return result, nil
}
