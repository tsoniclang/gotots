package packageinit

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.FuncDecl,
	targetName string,
) (api.DeclarationEmission, error) {
	if source == nil {
		return api.DeclarationEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "package init declaration is nil",
		}
	}
	function, ok := context.TypesInfo().Defs[source.Name].(*types.Func)
	var signature *types.Signature
	if ok {
		signature, _ = function.Type().(*types.Signature)
	}
	if source.Name == nil ||
		source.Name.Name != "init" ||
		source.Recv != nil ||
		source.Type == nil ||
		source.Type.TypeParams != nil ||
		source.Type.Params == nil ||
		len(source.Type.Params.List) != 0 ||
		(source.Type.Results != nil && len(source.Type.Results.List) != 0) ||
		source.Body == nil ||
		targetName == "" ||
		!ok ||
		function.Name() != "init" ||
		function.Pkg() != context.TypesPackage() ||
		function.Parent() != context.TypesPackage().Scope() ||
		context.TypesPackage().Scope().Lookup("init") != nil ||
		signature == nil ||
		signature.Recv() != nil ||
		signature.Params().Len() != 0 ||
		signature.Results().Len() != 0 ||
		signature.Variadic() ||
		signature.TypeParams().Len() != 0 {
		return api.DeclarationEmission{},
			api.Unsupported(context, api.CategoryDeclaration, source)
	}
	body, err := children.Block(
		context.
			WithRole(api.RolePackageInitBody).
			EnterFunction(types.NewTuple()),
		source.Body,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	return api.DirectDeclaration(
		context.Factory().FunctionDeclaration(
			[]tsgo.ModifierLike{context.Factory().ExportKeyword()},
			nil,
			context.Factory().Identifier(targetName),
			nil,
			nil,
			context.Factory().KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindVoidKeyword,
			),
			body.Value(),
		),
		body.Requests()...,
	), nil
}
