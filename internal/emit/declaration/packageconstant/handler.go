package packageconstant

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func EmitObject(
	context api.Context,
	children api.ChildEmitter,
	source *ast.GenDecl,
	selected *types.Const,
) (api.DeclarationEmission, error) {
	if selected == nil {
		return api.DeclarationEmission{},
			&api.InvariantError{
				Role:   context.Role(),
				Reason: "selected package constant is nil",
			}
	}
	if source.Doc != nil || source.Tok != token.CONST || len(source.Specs) == 0 {
		return api.DeclarationEmission{},
			api.Unsupported(context, api.CategoryDeclaration, source)
	}
	var declarations []tsgo.Statement
	var requests []api.PlacementRequest
	for _, sourceSpec := range source.Specs {
		spec, ok := sourceSpec.(*ast.ValueSpec)
		if !ok {
			return api.DeclarationEmission{},
				api.Unsupported(context, api.CategoryDeclaration, sourceSpec)
		}
		target, targetRequests, found, err := emitSpec(
			context,
			children,
			spec,
			selected,
		)
		if err != nil {
			return api.DeclarationEmission{}, err
		}
		declarations = append(declarations, target...)
		requests = append(requests, targetRequests...)
		if found {
			break
		}
	}
	if len(declarations) == 0 {
		return api.DeclarationEmission{},
			&api.InvariantError{
				Role:   context.Role(),
				Reason: "selected package constant is absent from its declaration",
			}
	}
	return api.NewDeclarationEmission(declarations, requests)
}

func emitSpec(
	context api.Context,
	children api.ChildEmitter,
	source *ast.ValueSpec,
	selected *types.Const,
) ([]tsgo.Statement, []api.PlacementRequest, bool, error) {
	if source.Doc != nil || source.Comment != nil || source.Type == nil ||
		len(source.Names) == 0 || len(source.Names) != len(source.Values) {
		return nil, nil, false,
			api.Unsupported(context, api.CategoryDeclaration, source)
	}

	declarations := make([]tsgo.Statement, 0, len(source.Names))
	var requests []api.PlacementRequest
	selectedIndex := -1
	for index, sourceName := range source.Names {
		object, ok := context.TypesInfo().Defs[sourceName].(*types.Const)
		if ok && object == selected {
			selectedIndex = index
			break
		}
	}
	if selectedIndex < 0 {
		return nil, nil, false, nil
	}
	sourceName := source.Names[selectedIndex]
	if sourceName.Name == "_" ||
		!types.Identical(context.TypesInfo().TypeOf(source.Type), selected.Type()) {
		return nil, nil, false,
			api.Unsupported(context, api.CategoryDeclaration, source)
	}
	sourceValue := source.Values[selectedIndex]
	typeAndValue, ok := context.TypesInfo().Types[sourceValue]
	if !ok || typeAndValue.Value == nil ||
		!constant.Compare(typeAndValue.Value, token.EQL, selected.Val()) ||
		!types.AssignableTo(typeAndValue.Type, selected.Type()) {
		return nil, nil, false,
			api.Unsupported(context, api.CategoryDeclaration, source)
	}
	value, err := children.Expression(
		context.
			WithRole(api.RolePackageConstantValue).
			WithExpectedType(selected.Type()),
		sourceValue,
	)
	if err != nil {
		return nil, nil, false, err
	}
	if len(value.Before()) != 0 {
		return nil, nil, false,
			api.Unsupported(context, api.CategoryDeclaration, source)
	}
	targetType, err := children.RepresentedType(
		context.WithRole(api.RolePackageConstantType),
		sourceName,
		selected.Type(),
	)
	if err != nil {
		return nil, nil, false, err
	}
	targetName, err := context.Names().Declare(selected)
	if err != nil {
		return nil, nil, false, err
	}
	moduleExport, err := context.Names().ModuleExport(selected)
	if err != nil {
		return nil, nil, false, err
	}
	if !moduleExport {
		return nil, nil, false,
			&api.InvariantError{
				Role:   context.Role(),
				Reason: "package constant is not module-exported",
			}
	}
	declaration := context.Factory().VariableDeclaration(
		context.Factory().Identifier(targetName),
		nil,
		targetType.Value(),
		value.Value(),
	)
	declarations = append(
		declarations,
		context.Factory().VariableStatement(
			[]tsgo.ModifierLike{context.Factory().ExportKeyword()},
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{declaration},
				tsgo.NodeFlagsConst,
			),
		),
	)
	requests = append(
		requests,
		api.CombineRequests(targetType.Requests(), value.Requests())...,
	)
	return declarations, requests, true, nil
}
