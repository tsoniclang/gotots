package sourcefact

import (
	"go/types"
	"slices"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/emit/api"
	constantbinding "github.com/tsoniclang/gotots/internal/emit/constant"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const constantProjectionSchema = "gotots-go-constant-projection-fact-v1"

func DeclarationWithRequirements(
	context api.Context,
	object types.Object,
	origin DeclarationOrigin,
	statements []tsgo.Statement,
	requirements []api.DeclarationRequirement,
	additionalBindings []string,
) (api.StatementEmission, error) {
	constant, contextual := object.(*types.Const)
	if !contextual || !constantbinding.IsUntyped(constant.Type()) {
		return Declaration(context, object, origin, statements)
	}
	baseName, err := context.Names().Declare(constant)
	if err != nil {
		return api.StatementEmission{}, err
	}
	bindings := make([]string, 0, len(requirements))
	projections := make(map[string]types.BasicKind, len(requirements))
	for _, requirement := range requirements {
		selected, projection, ok := requirement.ConstantProjection()
		if !ok || selected != constant {
			return api.StatementEmission{}, &Error{
				Subject: constant.Name(),
				Reason:  "contextual constant has a foreign declaration requirement",
			}
		}
		name, nameErr := api.ConstantProjectionName(baseName, projection)
		if nameErr != nil {
			return api.StatementEmission{}, nameErr
		}
		if _, duplicate := projections[name]; duplicate {
			return api.StatementEmission{}, &Error{
				Subject: constant.Name(),
				Reason:  "contextual constant projection is duplicated",
			}
		}
		bindings = append(bindings, name)
		projections[name] = projection
	}
	slices.Sort(bindings)
	if !slices.Equal(bindings, additionalBindings) {
		return api.StatementEmission{}, &Error{
			Subject: constant.Name(),
			Reason:  "contextual constant fact targets differ from emitted bindings",
		}
	}
	var facts []api.StatementEmission
	for _, name := range bindings {
		fact, factErr := ConstantProjection(
			context,
			constant,
			name,
			projections[name],
			origin,
			statements,
		)
		if factErr != nil {
			return api.StatementEmission{}, factErr
		}
		facts = append(facts, fact)
	}
	return combine(facts)
}

func ConstantProjection(
	context api.Context,
	constant *types.Const,
	name string,
	projection types.BasicKind,
	origin DeclarationOrigin,
	statements []tsgo.Statement,
) (api.StatementEmission, error) {
	if constant == nil || name == "" || !origin.Valid() || len(statements) == 0 {
		return api.StatementEmission{}, &Error{Reason: "constant projection fact is invalid"}
	}
	targetType, ok := api.ConstantProjectionType(projection)
	if !ok {
		return api.StatementEmission{}, &Error{Subject: constant.Name(), Reason: "constant projection type is invalid"}
	}
	typeTarget, err := exactDeclarationTarget(
		context.Factory(),
		[]string{name},
		artifactTargetValue,
		statements,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	target, err := NewTarget(typeTarget)
	if err != nil {
		return api.StatementEmission{}, err
	}
	contract, err := environmentcontract.Describe(constant)
	if err != nil {
		return api.StatementEmission{}, err
	}
	arguments := []tsgo.Expression{
		text(context.Factory(), constantProjectionSchema),
		text(context.Factory(), contract.Identity()),
		text(context.Factory(), name),
		text(context.Factory(), environmentcontract.StableTypeString(targetType)),
		text(context.Factory(), constant.Val().ExactString()),
	}
	arguments = append(arguments, origin.arguments(context.Factory())...)
	declaration, err := target.apply(
		context,
		api.RuntimeSourceDeclarationFact,
		arguments...,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	basic, err := target.apply(
		context,
		api.RuntimeSourceBasicFact,
		text(context.Factory(), constantProjectionSchema),
		text(context.Factory(), contract.Identity()),
		text(context.Factory(), "contextual-constant"),
		text(context.Factory(), targetType.Name()),
		count(context.Factory(), int(projection)),
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	return combine([]api.StatementEmission{declaration, basic})
}
