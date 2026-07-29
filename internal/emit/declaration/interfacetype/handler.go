package interfacetype

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	declaration *ast.GenDecl,
	typeName *types.TypeName,
	requirements []api.DeclarationRequirement,
) (api.DeclarationEmission, bool, error) {
	source, interfaceType, ok := sourceInterface(
		context,
		declaration,
		typeName,
	)
	if !ok {
		return api.DeclarationEmission{}, false, nil
	}
	if len(requirements) != 0 {
		return api.DeclarationEmission{}, true, &api.InvariantError{
			Role:   context.Role(),
			Reason: "named interface received declaration requirements",
		}
	}
	name, err := context.Names().Declare(typeName)
	if err != nil {
		return api.DeclarationEmission{}, true, err
	}
	moduleExport, err := context.Names().ModuleExport(typeName)
	if err != nil {
		return api.DeclarationEmission{}, true, err
	}
	var modifiers []tsgo.ModifierLike
	if moduleExport {
		modifiers = []tsgo.ModifierLike{context.Factory().ExportKeyword()}
	}
	statements, requests, err := Build(
		context,
		children,
		source,
		name,
		interfaceType,
		modifiers,
	)
	if err != nil {
		return api.DeclarationEmission{}, true, err
	}
	target, err := api.NewDeclarationEmission(statements, requests)
	return target, true, err
}

func Build(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	name string,
	interfaceType *types.Interface,
	modifiers []tsgo.ModifierLike,
) ([]tsgo.Statement, []api.RootRequest, error) {
	if name == "" ||
		interfaceType == nil ||
		!interfaceType.Complete().IsMethodSet() {
		return nil, nil, &api.GeneratedArtifactShapeError{
			Artifact: name,
			Reason:   "interface contract is invalid",
		}
	}
	runtimeValue, err := context.Names().Runtime(
		api.RuntimeInterfaceValue,
		api.ImportPhaseType,
	)
	if err != nil {
		return nil, nil, err
	}
	members := make(
		[]tsgo.TypeElement,
		0,
		interfaceType.NumMethods(),
	)
	tokens := make(
		[]tsgo.Expression,
		0,
		interfaceType.NumMethods(),
	)
	requests := runtimeValue.Requests()
	for index := range interfaceType.NumMethods() {
		method := interfaceType.Method(index)
		signature, ok := receiverFreeSignature(method)
		if !ok {
			return nil, nil, &api.GeneratedArtifactShapeError{
				Artifact: name,
				Reason:   "interface method signature is invalid",
			}
		}
		target, err := callable.EmitAdapter(
			context,
			children,
			source,
			signature,
		)
		if err != nil {
			return nil, nil, err
		}
		memberName, err := context.Names().InterfaceMethodName(method)
		if err != nil {
			return nil, nil, err
		}
		members = append(
			members,
			context.Factory().MethodSignatureDeclaration(
				nil,
				context.Factory().Identifier(memberName),
				nil,
				nil,
				target.Parameters(),
				target.Result(),
			),
		)
		token, err := context.Names().InterfaceMethodToken(method)
		if err != nil {
			return nil, nil, err
		}
		tokens = append(
			tokens,
			context.Factory().Identifier(token.Name()),
		)
		requests = append(
			requests,
			target.Requests()...,
		)
		requests = append(requests, token.Requests()...)
	}
	statements := []tsgo.Statement{
		context.Factory().InterfaceDeclaration(
			modifiers,
			context.Factory().Identifier(name),
			nil,
			[]tsgo.HeritageClause{
				context.Factory().HeritageClause(
					tsgo.HeritageClauseTokenKindExtendsKeyword,
					[]tsgo.ExpressionWithTypeArguments{
						context.Factory().ExpressionWithTypeArguments(
							context.Factory().Identifier(
								runtimeValue.Name(),
							),
							nil,
						),
					},
				),
			},
			members,
		),
		contractDeclaration(context.Factory(), name, modifiers, tokens),
		guardDeclaration(
			context.Factory(),
			name,
			runtimeValue.Name(),
			modifiers,
		),
	}
	return statements, requests, nil
}

func sourceInterface(
	context api.Context,
	declaration *ast.GenDecl,
	typeName *types.TypeName,
) (*ast.TypeSpec, *types.Interface, bool) {
	if declaration == nil ||
		typeName == nil ||
		typeName.IsAlias() {
		return nil, nil, false
	}
	named, ok := typeName.Type().(*types.Named)
	if !ok || named.TypeParams().Len() != 0 {
		return nil, nil, false
	}
	interfaceType, ok := named.Underlying().(*types.Interface)
	if !ok || !interfaceType.Complete().IsMethodSet() {
		return nil, nil, false
	}
	for _, candidate := range declaration.Specs {
		source, ok := candidate.(*ast.TypeSpec)
		if ok &&
			source.TypeParams == nil &&
			context.TypesInfo().Defs[source.Name] == typeName {
			_, syntaxOK := source.Type.(*ast.InterfaceType)
			return source, interfaceType, syntaxOK
		}
	}
	return nil, nil, false
}

func receiverFreeSignature(
	method *types.Func,
) (*types.Signature, bool) {
	if method == nil {
		return nil, false
	}
	source, ok := method.Type().(*types.Signature)
	if !ok ||
		source.TypeParams().Len() != 0 ||
		source.RecvTypeParams().Len() != 0 {
		return nil, false
	}
	return types.NewSignatureType(
		nil,
		nil,
		nil,
		source.Params(),
		source.Results(),
		source.Variadic(),
	), true
}
