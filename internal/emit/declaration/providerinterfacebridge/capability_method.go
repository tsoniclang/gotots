package providerinterfacebridge

import (
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type selectedCapabilityMethod struct {
	fieldName string
	emission  methodEmission
}

func emitCapabilityMethods(
	context api.Context,
	children api.ChildEmitter,
	bridgeName string,
	bridgeType *types.Named,
	capabilities []capabilitySelection,
) ([]tsgo.ClassElement, []api.RootRequest, error) {
	groups := make(map[string][]selectedCapabilityMethod)
	var requests []api.RootRequest
	for _, capability := range capabilities {
		for _, method := range capability.methods {
			emission, err := prepareMethod(
				context,
				children,
				bridgeName,
				bridgeType,
				capability.profile,
				method.method,
				method.certificate,
				context.Factory().Identifier(capability.fieldName),
			)
			if err != nil {
				return nil, nil, err
			}
			groups[emission.name] = append(
				groups[emission.name],
				selectedCapabilityMethod{
					fieldName: capability.fieldName,
					emission:  emission,
				},
			)
			requests = append(requests, emission.requests...)
		}
	}
	return emitCapabilityMethodGroups(
		context,
		bridgeName,
		groups,
		requests,
	)
}

func emitCapabilityMethodGroups(
	context api.Context,
	bridgeName string,
	groups map[string][]selectedCapabilityMethod,
	requests []api.RootRequest,
) ([]tsgo.ClassElement, []api.RootRequest, error) {
	if len(groups) == 0 {
		return nil, api.CombineRequests(requests), nil
	}
	panicReference, err := context.Names().Runtime(
		api.RuntimePanic,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, nil, err
	}
	requests = append(requests, panicReference.Requests()...)
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sort.Strings(names)
	var members []tsgo.ClassElement
	for _, name := range names {
		selected, err := capabilityMethodGroup(
			context,
			bridgeName,
			groups[name],
			panicReference.Name(),
		)
		if err != nil {
			return nil, nil, err
		}
		members = append(members, selected...)
	}
	return members, api.CombineRequests(requests), nil
}

func capabilityMethodGroup(
	context api.Context,
	bridgeName string,
	methods []selectedCapabilityMethod,
	panicName string,
) ([]tsgo.ClassElement, error) {
	if len(methods) == 0 || panicName == "" {
		return nil, shapeError(bridgeName, "capability method group is empty")
	}
	first := methods[0].emission
	for _, selected := range methods[1:] {
		if !sameMethodInputs(first.signature, selected.emission.signature) ||
			first.cooperative != selected.emission.cooperative {
			return nil, shapeError(
				bridgeName,
				"same-name capability methods have incompatible call inputs",
			)
		}
	}
	unique := make([]methodEmission, 0, len(methods))
	for _, selected := range methods {
		duplicate := false
		for _, existing := range unique {
			if types.Identical(
				existing.signature,
				selected.emission.signature,
			) {
				duplicate = true
				break
			}
		}
		if !duplicate {
			unique = append(unique, selected.emission)
		}
	}
	members := make([]tsgo.ClassElement, 0, len(unique)+1)
	if len(unique) > 1 {
		for _, overload := range unique {
			members = append(members, overload.declarationWithoutBody(
				context.Factory(),
			))
		}
	}
	body := make([]tsgo.Statement, 0, len(methods)*2+1)
	for _, selected := range methods {
		field := context.Factory().Identifier(selected.fieldName)
		body = append(body, context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					context.Factory().VariableDeclaration(
						field,
						nil,
						nil,
						capabilityField(
							context.Factory(),
							selected.fieldName,
						),
					),
				},
				tsgo.NodeFlagsConst,
			),
		))
		branch := append(
			[]tsgo.Statement(nil),
			selected.emission.body...,
		)
		if selected.emission.signature.Results().Len() == 0 {
			branch = append(branch, context.Factory().ReturnStatement(nil))
		}
		body = append(body, context.Factory().IfStatement(
			isDefined(context.Factory(), field),
			context.Factory().Block(branch, true),
			nil,
		))
	}
	body = append(body, context.Factory().ReturnStatement(
		panicruntime.Call(
			context.Factory(),
			panicName,
			context.Factory().StringLiteral(
				"provider interface capability is absent",
				tsgo.TokenFlagsNone,
			),
		),
	))
	implementation := first
	implementation.body = body
	implementation.result = capabilityMethodResult(
		context.Factory(),
		unique,
		first.cooperative,
	)
	members = append(members, implementation.declaration(context.Factory()))
	return members, nil
}

func capabilityMethodResult(
	factory tsgo.Factory,
	methods []methodEmission,
	cooperative bool,
) tsgo.TypeNode {
	if len(methods) == 1 {
		return methods[0].result
	}
	values := make([]tsgo.TypeNode, 0, len(methods))
	for _, method := range methods {
		values = append(values, method.resultValue)
	}
	result := factory.UnionTypeNode(values)
	if cooperative {
		return callable.PromiseResult(factory, result)
	}
	return result
}

func sameMethodInputs(left *types.Signature, right *types.Signature) bool {
	return left != nil && right != nil &&
		left.Variadic() == right.Variadic() &&
		types.Identical(left.Params(), right.Params())
}
