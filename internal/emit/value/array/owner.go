package array

import (
	"go/ast"
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	arraymember "github.com/tsoniclang/gotots/internal/emit/runtime/array/member"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type RuntimeArray struct {
	sourceType types.Type
	source     *types.Array
	defined    definedtype.Model
	nominal    bool
	aggregate  bool
}

func Resolve(
	context api.Context,
	sourceType types.Type,
) (RuntimeArray, bool) {
	var source *types.Array
	defined, nominal := definedtype.ResolveArray(sourceType)
	if nominal {
		source, _ = defined.Array()
	} else {
		source, _ = types.Unalias(sourceType).(*types.Array)
	}
	if source == nil || source.Len() < 0 {
		return RuntimeArray{}, false
	}
	return RuntimeArray{
		sourceType: sourceType,
		source:     source,
		defined:    defined,
		nominal:    nominal,
		aggregate: context.Values().RequiresStructuralCopy(
			context,
			source.Elem(),
		),
	}, true
}

func (a RuntimeArray) ElementType() types.Type {
	return a.source.Elem()
}

func (a RuntimeArray) SourceType() types.Type {
	return a.sourceType
}

func (a RuntimeArray) Length() int64 {
	return a.source.Len()
}

func (a RuntimeArray) Aggregate() bool {
	return a.aggregate
}

func (a RuntimeArray) EmitType(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
) (api.TypeEmission, error) {
	if a.source == nil {
		return api.TypeEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "runtime array has no Go array type",
		}
	}
	if a.nominal {
		reference, err := context.Names().TypeReference(a.defined.TypeName())
		if err != nil {
			return api.TypeEmission{}, err
		}
		return api.DirectType(
			context.Factory().TypeReferenceNode(
				context.Factory().Identifier(reference.Name()),
				nil,
			),
			reference.Requests()...,
		), nil
	}
	element, err := context.ContainerStorage().ContainerStorageType(
		context,
		source,
		a.source.Elem(),
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	reference, err := context.Names().Runtime(
		api.RuntimeArray,
		api.ImportPhaseType,
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return api.DirectType(
		context.Factory().TypeReferenceNode(
			context.Factory().Identifier(reference.Name()),
			[]tsgo.TypeNode{
				element.Value(),
				context.Factory().LiteralTypeNode(a.lengthLiteral(context)),
			},
		),
		api.CombineRequests(
			element.Requests(),
			reference.Requests(),
		)...,
	), nil
}

func (a RuntimeArray) storage(
	context api.Context,
	value tsgo.Expression,
) tsgo.Expression {
	if !a.nominal {
		return value
	}
	return a.defined.Unwrap(context.Factory(), value)
}

func (a RuntimeArray) wrap(
	context api.Context,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if !a.nominal {
		return value, nil
	}
	return a.defined.Wrap(context, value)
}

func (a RuntimeArray) lengthLiteral(context api.Context) tsgo.NumericLiteral {
	return context.Factory().NumericLiteral(
		strconv.FormatInt(a.source.Len(), 10),
		tsgo.TokenFlagsNone,
	)
}

func (a RuntimeArray) runtime(
	context api.Context,
	phase api.ImportPhase,
) (api.NameReference, error) {
	return context.Names().Runtime(api.RuntimeArray, phase)
}

func (a RuntimeArray) runtimeOperation(
	context api.Context,
	children api.ChildEmitter,
	symbol api.RuntimeSymbol,
	arguments ...tsgo.Expression,
) (tsgo.CallExpression, []api.RootRequest, error) {
	typeArguments, typeRequests, err := a.targetTypeArguments(
		context,
		children,
	)
	if err != nil {
		return nil, nil, err
	}
	reference, err := context.Names().Runtime(symbol, api.ImportPhaseValue)
	if err != nil {
		return nil, nil, err
	}
	return context.Factory().CallExpression(
		context.Factory().Identifier(reference.Name()),
		nil,
		typeArguments,
		arguments,
		tsgo.NodeFlagsNone,
	), api.CombineRequests(typeRequests, reference.Requests()), nil
}

func (a RuntimeArray) callStatic(
	context api.Context,
	member arraymember.Identity,
	typeArguments []tsgo.TypeNode,
	arguments ...tsgo.Expression,
) (tsgo.CallExpression, []api.RootRequest, error) {
	reference, err := a.runtime(context, api.ImportPhaseValue)
	if err != nil {
		return nil, nil, err
	}
	return context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			context.Factory().Identifier(reference.Name()),
			nil,
			context.Factory().Identifier(member.Name()),
			tsgo.NodeFlagsNone,
		),
		nil,
		typeArguments,
		arguments,
		tsgo.NodeFlagsNone,
	), reference.Requests(), nil
}

func (a RuntimeArray) targetTypeArguments(
	context api.Context,
	children api.ChildEmitter,
) ([]tsgo.TypeNode, []api.RootRequest, error) {
	element, err := context.ContainerStorage().ContainerStorageType(
		context.WithRole(api.RoleArrayElement),
		nil,
		a.ElementType(),
	)
	if err != nil {
		return nil, nil, err
	}
	return []tsgo.TypeNode{
		element.Value(),
		context.Factory().LiteralTypeNode(a.lengthLiteral(context)),
	}, element.Requests(), nil
}

func callMember(
	context api.Context,
	receiver tsgo.Expression,
	member arraymember.Identity,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			receiver,
			nil,
			context.Factory().Identifier(member.Name()),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
}

func memberProperty(
	context api.Context,
	receiver tsgo.Expression,
	member arraymember.Identity,
) tsgo.PropertyAccessExpression {
	return context.Factory().PropertyAccessExpression(
		receiver,
		nil,
		context.Factory().Identifier(member.Name()),
		tsgo.NodeFlagsNone,
	)
}
