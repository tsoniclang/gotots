package packageconstant

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantbinding "github.com/tsoniclang/gotots/internal/emit/constant"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

// EmitObject emits a package-level constant. A primitive typed constant becomes
// one direct binding. A defined-basic constant becomes one ESM-cycle-safe typed
// value thunk and an assembly-owned public const. An untyped constant has no
// single runtime type, so it is projected once per required target basic
// representation, each projection scheduled by a use site through the
// constant-projection declaration requirement. An untyped constant with no
// requirements yet contributes no statements; its projections arrive by
// artifact reconstruction as uses demand them.
func EmitObject(
	context api.Context,
	children api.ChildEmitter,
	source *ast.GenDecl,
	selected *types.Const,
	requirements []api.DeclarationRequirement,
) (api.DeclarationEmission, error) {
	if selected == nil {
		return api.DeclarationEmission{},
			&api.InvariantError{
				Role:   context.Role(),
				Reason: "selected package constant is nil",
			}
	}
	if source.Tok != token.CONST || len(source.Specs) == 0 {
		return api.DeclarationEmission{},
			api.Unsupported(context, api.CategoryDeclaration, source)
	}
	sourceName, err := locateName(context, source, selected)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	if constantbinding.IsUntyped(selected.Type()) {
		return emitProjections(context, children, sourceName, selected, requirements)
	}
	if len(requirements) != 0 {
		return api.DeclarationEmission{},
			&api.InvariantError{
				Role:   context.Role(),
				Reason: "typed constant received projection requirements",
			}
	}
	return emitTypedBinding(context, children, sourceName, selected)
}

// locateName finds the declaring identifier of the selected constant among the
// declaration's specs. Every constant object has exactly one declaring name;
// its absence is a scheduling invariant violation, not source shape the handler
// tolerates.
func locateName(
	context api.Context,
	source *ast.GenDecl,
	selected *types.Const,
) (*ast.Ident, error) {
	for _, sourceSpec := range source.Specs {
		spec, ok := sourceSpec.(*ast.ValueSpec)
		if !ok {
			return nil, api.Unsupported(context, api.CategoryDeclaration, sourceSpec)
		}
		if len(spec.Names) == 0 {
			return nil, api.Unsupported(context, api.CategoryDeclaration, spec)
		}
		for _, name := range spec.Names {
			object, ok := context.TypesInfo().DefOf(name).(*types.Const)
			if ok && object == selected {
				return name, nil
			}
		}
	}
	return nil, &api.InvariantError{
		Role:   context.Role(),
		Reason: "selected package constant is absent from its declaration",
	}
}

func emitTypedBinding(
	context api.Context,
	children api.ChildEmitter,
	sourceName *ast.Ident,
	selected *types.Const,
) (api.DeclarationEmission, error) {
	if constantbinding.RequiresDeferredBinding(selected) {
		deferred, err := constantbinding.EmitDeferredBinding(
			context,
			children,
			sourceName,
			selected,
			api.RolePackageConstantType,
			api.RolePackageConstantValue,
		)
		if err != nil {
			return api.DeclarationEmission{}, err
		}
		baseName, err := context.Names().Declare(selected)
		if err != nil {
			return api.DeclarationEmission{}, err
		}
		deferredName, err := constantbinding.DeferredBindingName(baseName)
		if err != nil {
			return api.DeclarationEmission{}, err
		}
		return api.NewDeclarationEmissionWithAdditionalPackageBindings(
			[]tsgo.Statement{deferred.Declaration()},
			deferred.Requests(),
			[]string{deferredName},
		)
	}
	binding, err := constantbinding.EmitBinding(
		context,
		children,
		sourceName,
		selected,
		api.RolePackageConstantType,
		api.RolePackageConstantValue,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	moduleExport, err := context.Names().ModuleExport(selected)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	if !moduleExport {
		return api.DeclarationEmission{},
			&api.InvariantError{
				Role:   context.Role(),
				Reason: "package constant is not module-exported",
			}
	}
	statement := context.Factory().VariableStatement(
		[]tsgo.ModifierLike{context.Factory().ExportKeyword()},
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{binding.Declaration()},
			tsgo.NodeFlagsConst,
		),
	)
	return api.NewDeclarationEmission(
		[]tsgo.Statement{statement},
		binding.Requests(),
	)
}

func emitProjections(
	context api.Context,
	children api.ChildEmitter,
	sourceName *ast.Ident,
	selected *types.Const,
	requirements []api.DeclarationRequirement,
) (api.DeclarationEmission, error) {
	baseName, err := context.Names().Declare(selected)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	moduleExport, err := context.Names().ModuleExport(selected)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	if !moduleExport {
		return api.DeclarationEmission{},
			&api.InvariantError{
				Role:   context.Role(),
				Reason: "package constant projection owner is not module-exported",
			}
	}
	declarations := make([]tsgo.Statement, 0, len(requirements))
	bindings := make([]string, 0, len(requirements))
	var requests []api.RootRequest
	for _, requirement := range requirements {
		constant, projection, ok := requirement.ConstantProjection()
		if !ok || constant != selected {
			return api.DeclarationEmission{},
				&api.InvariantError{
					Role:   context.Role(),
					Reason: "constant projection requirement does not own this constant",
				}
		}
		projectionName, err := api.ConstantProjectionName(
			baseName,
			projection,
		)
		if err != nil {
			return api.DeclarationEmission{}, err
		}
		emission, err := constantbinding.EmitProjection(
			context,
			children,
			sourceName,
			selected,
			projectionName,
			projection,
			api.RolePackageConstantType,
			api.RolePackageConstantValue,
		)
		if err != nil {
			return api.DeclarationEmission{}, err
		}
		declarations = append(
			declarations,
			emission.ExportedStatement(context.Factory()),
		)
		bindings = append(bindings, projectionName)
		requests = append(requests, emission.Requests()...)
	}
	if len(declarations) == 0 {
		return api.CoverageOnlyDeclarationEmission(), nil
	}
	return api.NewDeclarationEmissionWithAdditionalPackageBindings(
		declarations,
		requests,
		bindings,
	)
}
