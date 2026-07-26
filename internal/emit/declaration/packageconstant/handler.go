package packageconstant

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.GenDecl,
) (api.DeclarationEmission, error) {
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
		target, targetRequests, err := emitSpec(context, children, spec)
		if err != nil {
			return api.DeclarationEmission{}, err
		}
		declarations = append(declarations, target...)
		requests = append(requests, targetRequests...)
	}
	return api.NewDeclarationEmission(declarations, requests)
}

func emitSpec(
	context api.Context,
	children api.ChildEmitter,
	source *ast.ValueSpec,
) ([]tsgo.Statement, []api.PlacementRequest, error) {
	if source.Doc != nil || source.Comment != nil || source.Type == nil ||
		len(source.Names) == 0 || len(source.Names) != len(source.Values) {
		return nil, nil,
			api.Unsupported(context, api.CategoryDeclaration, source)
	}

	declarations := make([]tsgo.Statement, 0, len(source.Names))
	var requests []api.PlacementRequest
	for index, sourceName := range source.Names {
		object, ok := context.TypesInfo().Defs[sourceName].(*types.Const)
		if !ok || sourceName.Name == "_" ||
			!types.Identical(context.TypesInfo().TypeOf(source.Type), object.Type()) {
			return nil, nil,
				api.Unsupported(context, api.CategoryDeclaration, source)
		}
		sourceValue := source.Values[index]
		typeAndValue, ok := context.TypesInfo().Types[sourceValue]
		if !ok || typeAndValue.Value == nil ||
			!constant.Compare(typeAndValue.Value, token.EQL, object.Val()) ||
			!types.AssignableTo(typeAndValue.Type, object.Type()) {
			return nil, nil,
				api.Unsupported(context, api.CategoryDeclaration, source)
		}
		value, err := children.Expression(
			context.
				WithRole(api.RolePackageConstantValue).
				WithExpectedType(object.Type()),
			sourceValue,
		)
		if err != nil {
			return nil, nil, err
		}
		if len(value.Before()) != 0 {
			return nil, nil,
				api.Unsupported(context, api.CategoryDeclaration, source)
		}
		targetType, err := children.RepresentedType(
			context.WithRole(api.RolePackageConstantType),
			sourceName,
			object.Type(),
		)
		if err != nil {
			return nil, nil, err
		}
		targetName, err := context.Names().Declare(object)
		if err != nil {
			return nil, nil, err
		}
		moduleExport, err := context.Names().ModuleExport(object)
		if err != nil {
			return nil, nil, err
		}
		if !moduleExport {
			return nil, nil,
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
	}
	return declarations, requests, nil
}
