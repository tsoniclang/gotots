package function

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitReceiver(
	context api.Context,
	children api.ChildEmitter,
	source *ast.FuncDecl,
	signature *types.Signature,
) (
	tsgo.ParameterDeclaration,
	[]api.RootRequest,
	error,
) {
	if source.Recv == nil ||
		len(source.Recv.List) != 1 ||
		len(source.Recv.List[0].Names) != 1 ||
		signature.Recv() == nil {
		return nil, nil,
			api.Unsupported(context, api.CategoryDeclaration, source)
	}
	field := source.Recv.List[0]
	if field.Doc != nil || field.Comment != nil || field.Tag != nil {
		return nil, nil,
			api.Unsupported(context, api.CategoryDeclaration, field)
	}
	named, ok := types.Unalias(signature.Recv().Type()).(*types.Named)
	if !ok || named.TypeParams().Len() != 0 {
		return nil, nil,
			api.Unsupported(
				context.WithRole(api.RoleReceiverType),
				api.CategoryType,
				field.Type,
			)
	}
	if _, ok := named.Underlying().(*types.Struct); !ok ||
		!types.Identical(
			context.TypesInfo().TypeOf(field.Type),
			signature.Recv().Type(),
		) ||
		context.TypesInfo().Defs[field.Names[0]] != signature.Recv() {
		return nil, nil,
			api.Unsupported(context, api.CategoryDeclaration, field)
	}
	targetType, err := children.Type(
		context.WithRole(api.RoleReceiverType),
		field.Type,
	)
	if err != nil {
		return nil, nil, err
	}
	name, err := context.Names().Declare(signature.Recv())
	if err != nil {
		return nil, nil, err
	}
	return context.Factory().ParameterDeclaration(
			nil,
			nil,
			context.Factory().Identifier(name),
			nil,
			targetType.Value(),
			nil,
		),
		targetType.Requests(),
		nil
}
