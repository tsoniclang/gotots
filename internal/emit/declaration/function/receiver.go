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
		len(source.Recv.List[0].Names) > 1 ||
		signature.Recv() == nil {
		return nil, nil,
			api.Unsupported(context, api.CategoryDeclaration, source)
	}
	field := source.Recv.List[0]
	if field.Tag != nil {
		return nil, nil,
			api.Unsupported(context, api.CategoryDeclaration, field)
	}
	receiverType := signature.Recv().Type()
	baseType := receiverType
	if pointer, ok := types.Unalias(baseType).(*types.Pointer); ok {
		baseType = pointer.Elem()
	}
	named, ok := types.Unalias(baseType).(*types.Named)
	if !ok ||
		named.Obj() == nil ||
		named.Origin().Obj().Pkg() != signature.Recv().Pkg() ||
		(named.TypeParams().Len() != 0 &&
			named.TypeArgs().Len() != named.TypeParams().Len()) {
		return nil, nil,
			api.Unsupported(
				context.WithRole(api.RoleReceiverType),
				api.CategoryType,
				field.Type,
			)
	}
	if !types.Identical(
		context.TypesInfo().TypeOf(field.Type),
		receiverType,
	) {
		return nil, nil,
			api.Unsupported(context, api.CategoryDeclaration, field)
	}
	switch len(field.Names) {
	case 0:
		if signature.Recv().Name() != "" {
			return nil, nil,
				api.Unsupported(context, api.CategoryDeclaration, field)
		}
	case 1:
		if context.TypesInfo().Defs[field.Names[0]] != signature.Recv() {
			return nil, nil,
				api.Unsupported(context, api.CategoryDeclaration, field)
		}
	default:
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
	name, err := context.Names().Parameter(
		signature.Recv(),
		signature.Params().Len(),
	)
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
