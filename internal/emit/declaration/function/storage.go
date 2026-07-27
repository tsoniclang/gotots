package function

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantbinding "github.com/tsoniclang/gotots/internal/emit/constant"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func applyAddressableStorage(
	context api.Context,
	source *ast.FuncDecl,
	owner *types.Func,
	requirements []api.DeclarationRequirement,
) (api.Context, error) {
	if source == nil || owner == nil {
		return api.Context{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "addressable-storage artifact identity is nil",
		}
	}
	storageNames := make(map[*types.Var]string, len(requirements))
	for _, requirement := range requirements {
		if requirement.Kind() != api.DeclarationRequirementAddressableStorage {
			continue
		}
		requirementOwner, variable, ok := requirement.AddressableStorage()
		if !ok ||
			requirementOwner != owner ||
			variable.Pkg() != owner.Pkg() ||
			variable.Pos() < source.Pos() ||
			variable.Pos() > source.End() {
			return api.Context{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "function received foreign addressable-storage requirement",
			}
		}
		if _, duplicate := storageNames[variable]; duplicate {
			return api.Context{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "function received duplicate addressable-storage requirement",
			}
		}
		name, err := context.Names().Declare(variable)
		if err != nil {
			return api.Context{}, err
		}
		storageNames[variable] = name + "$storage"
	}
	return context.WithAddressableStorage(owner, storageNames), nil
}

// localConstantProjectionPrologue materializes this function's demanded
// function-local untyped-constant projections as prologue `const` bindings, one
// per (constant, representation) requirement. Each projection is placed at the
// function prologue — a deterministic, in-scope location that dominates every
// use — and referenced by the same derived name the use sites compute, so uses
// stay constant-size. A function with no such requirement contributes nothing.
func localConstantProjectionPrologue(
	context api.Context,
	children api.ChildEmitter,
	source *ast.FuncDecl,
	owner *types.Func,
	requirements []api.DeclarationRequirement,
) ([]tsgo.Statement, []api.RootRequest, error) {
	var statements []tsgo.Statement
	var requests []api.RootRequest
	for _, requirement := range requirements {
		if requirement.Kind() != api.DeclarationRequirementLocalConstantProjection {
			continue
		}
		requirementOwner, constant, projection, ok := requirement.LocalConstantProjection()
		if !ok ||
			requirementOwner != owner ||
			constant.Pkg() != owner.Pkg() ||
			constant.Pos() < source.Pos() ||
			constant.Pos() > source.End() {
			return nil, nil, &api.InvariantError{
				Role:   context.Role(),
				Reason: "function received foreign local-constant projection requirement",
			}
		}
		base, err := context.Names().Declare(constant)
		if err != nil {
			return nil, nil, err
		}
		emission, err := constantbinding.EmitProjection(
			context,
			children,
			source.Name,
			constant,
			api.ConstantProjectionName(base, projection),
			projection,
			api.RoleLocalConstantType,
			api.RoleLocalConstantValue,
		)
		if err != nil {
			return nil, nil, err
		}
		statements = append(
			statements,
			context.Factory().VariableStatement(
				nil,
				context.Factory().VariableDeclarationList(
					[]tsgo.VariableDeclaration{emission.Declaration()},
					tsgo.NodeFlagsConst,
				),
			),
		)
		requests = append(requests, emission.Requests()...)
	}
	return statements, requests, nil
}
