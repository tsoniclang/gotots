package array

import (
	"go/ast"
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	arraymember "github.com/tsoniclang/gotots/internal/emit/runtime/array/member"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type RuntimeArray struct {
	source *types.Array
}

func Resolve(
	context api.Context,
	sourceType types.Type,
) (RuntimeArray, bool) {
	source, ok := types.Unalias(sourceType).(*types.Array)
	if !ok || source.Len() < 0 {
		return RuntimeArray{}, false
	}
	array := RuntimeArray{source: source}
	alias, represented := basictype.PrimitiveAlias(
		context.TypesSizes(),
		array.ElementType(),
	)
	if !represented || alias == api.PrimitiveInvalid {
		return RuntimeArray{}, false
	}
	return array, true
}

func (a RuntimeArray) ElementType() types.Type {
	return a.source.Elem()
}

func (a RuntimeArray) Length() int64 {
	return a.source.Len()
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
	element, err := children.RepresentedType(
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
) ([]tsgo.TypeNode, []api.RootRequest, error) {
	alias, ok := basictype.PrimitiveAlias(
		context.TypesSizes(),
		a.ElementType(),
	)
	if !ok {
		return nil, nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "runtime array element lost its scalar representation",
		}
	}
	element, err := context.Names().Primitive(alias)
	if err != nil {
		return nil, nil, err
	}
	return []tsgo.TypeNode{
		context.Factory().TypeReferenceNode(
			context.Factory().Identifier(element.Name()),
			nil,
		),
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
