package emit

import (
	"fmt"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	environmentcontract "github.com/tsoniclang/gotots/internal/emit/environmentcontract"
	builtinexpression "github.com/tsoniclang/gotots/internal/emit/expression/builtin"
	targetplacement "github.com/tsoniclang/gotots/internal/emit/placement"
	"github.com/tsoniclang/gotots/internal/load"
	targetoutput "github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type EnvironmentContractError struct {
	Object types.Object
	Cause  error
}

func (e *EnvironmentContractError) Error() string {
	name := "<unknown>"
	if e.Object != nil {
		name = e.Object.Name()
		if e.Object.Pkg() != nil {
			name = e.Object.Pkg().Path() + "." + name
		}
		if e.Object.Type() != nil {
			name += " " + e.Object.Type().String()
		}
	}
	return fmt.Sprintf("emit environment contract %q: %v", name, e.Cause)
}

func (e *EnvironmentContractError) Unwrap() error {
	return e.Cause
}

func environmentContractError(
	object types.Object,
	err error,
) error {
	if err == nil {
		return nil
	}
	return &EnvironmentContractError{Object: object, Cause: err}
}

type environmentDeclaration struct {
	object          types.Object
	name            string
	statements      []tsgo.Statement
	reconstructions uint64
}

type environmentBuiltinOverload struct {
	signature  *types.Signature
	statements []tsgo.Statement
}

type environmentContractBuilder struct {
	sourcePackage    *load.Package
	outputPath       string
	emitter          *emitter
	context          api.Context
	placement        *targetplacement.Owner
	declarations     map[types.Object]environmentDeclaration
	stateFields      map[*types.Var]tsgo.TypeElement
	projections      map[string]tsgo.Statement
	requiredBuiltins map[*types.Builtin]struct{}
	builtinOverloads map[*types.Builtin][]environmentBuiltinOverload
	requirements     map[types.Object]map[api.DeclarationRequirement]struct{}
}

func (s *programSession) emitEnvironmentObject(object types.Object) error {
	sourcePackage := s.source.EnvironmentForTypes(object.Pkg())
	if sourcePackage == nil {
		return &ScheduleError{
			Object: object.Name(),
			Reason: "scheduled object lost its declaration",
		}
	}
	builder, err := s.requireEnvironmentPackage(sourcePackage)
	if err != nil {
		return err
	}
	if variable, ok := object.(*types.Var); ok {
		field, requests, err := environmentcontract.StateField(
			builder.context,
			builder.emitter,
			variable,
		)
		if err != nil {
			return environmentContractError(object, err)
		}
		if err := s.applyRootRequests(builder.placement, requests); err != nil {
			return environmentContractError(object, err)
		}
		if _, duplicate := builder.stateFields[variable]; duplicate {
			return &ScheduleError{
				Object: variable.Name(),
				Reason: "environment state field was emitted more than once",
			}
		}
		builder.stateFields[variable] = field
		return nil
	}
	if builtin, ok := builtinexpression.FromObject(object); ok {
		if _, duplicate := builder.requiredBuiltins[builtin]; duplicate {
			return &ScheduleError{
				Object: builtin.Name(),
				Reason: "environment builtin was emitted more than once",
			}
		}
		builder.requiredBuiltins[builtin] = struct{}{}
		return nil
	}
	if _, duplicate := builder.declarations[object]; duplicate {
		return &ScheduleError{
			Object: object.Name(),
			Reason: "environment declaration was emitted more than once",
		}
	}
	target, err := s.buildEnvironmentDeclaration(
		builder,
		object,
		builder.environmentRequirements(object),
	)
	if err != nil {
		return err
	}
	builder.declarations[object] = target
	return nil
}

func (s *programSession) requireEnvironmentPackage(
	sourcePackage *load.Package,
) (*environmentContractBuilder, error) {
	if sourcePackage == nil ||
		!sourcePackage.Kind().EnvironmentContract() ||
		sourcePackage.Types() == nil {
		return nil, &ScheduleError{
			Reason: "required environment package is invalid",
		}
	}
	if existing := s.environmentBuilders[sourcePackage]; existing != nil {
		return existing, nil
	}
	outputPath, err := targetoutput.EnvironmentContractPath(sourcePackage)
	if err != nil {
		return nil, err
	}
	targetEmitter := newEmitter(
		sourcePackage,
		s.factory,
		s.registry,
		s.integer,
		s.evaluationOrder,
		s.concurrency,
		s.require,
		s,
		s,
		s.goRuntime,
	)
	context, err := targetEmitter.targetContext(nil, outputPath)
	if err != nil {
		return nil, err
	}
	context = context.WithEnvironmentContract()
	builder := &environmentContractBuilder{
		sourcePackage:    sourcePackage,
		outputPath:       outputPath,
		emitter:          targetEmitter,
		context:          context,
		placement:        targetplacement.New(),
		declarations:     make(map[types.Object]environmentDeclaration),
		stateFields:      make(map[*types.Var]tsgo.TypeElement),
		projections:      make(map[string]tsgo.Statement),
		requiredBuiltins: make(map[*types.Builtin]struct{}),
		builtinOverloads: make(
			map[*types.Builtin][]environmentBuiltinOverload,
		),
		requirements: make(
			map[types.Object]map[api.DeclarationRequirement]struct{},
		),
	}
	s.environmentBuilders[sourcePackage] = builder
	return builder, nil
}

func (s *programSession) environmentTargetFiles(
	primitiveAliases map[api.PrimitiveAlias]struct{},
	runtimeSymbols map[api.RuntimeSymbol]struct{},
) ([]TargetFile, error) {
	builders := make(
		[]*environmentContractBuilder,
		0,
		len(s.environmentBuilders),
	)
	for _, builder := range s.environmentBuilders {
		builders = append(builders, builder)
	}
	sort.Slice(builders, func(left, right int) bool {
		return builders[left].outputPath < builders[right].outputPath
	})
	files := make([]TargetFile, 0, len(builders))
	for _, builder := range builders {
		for _, alias := range builder.placement.PrimitiveAliases() {
			primitiveAliases[alias] = struct{}{}
		}
		for _, symbol := range builder.placement.RuntimeSymbols() {
			runtimeSymbols[symbol] = struct{}{}
		}
		statements := builder.placement.Statements(s.factory)
		declarations := make(
			[]environmentDeclaration,
			0,
			len(builder.declarations),
		)
		for _, declaration := range builder.declarations {
			declarations = append(declarations, declaration)
		}
		sort.Slice(declarations, func(left, right int) bool {
			return compareObjects(
				declarations[left].object,
				declarations[right].object,
			) < 0
		})
		for _, declaration := range declarations {
			statements = append(statements, declaration.statements...)
		}
		builtins := make(
			[]*types.Builtin,
			0,
			len(builder.requiredBuiltins),
		)
		for builtin := range builder.requiredBuiltins {
			builtins = append(builtins, builtin)
		}
		sort.Slice(builtins, func(left, right int) bool {
			return compareObjects(builtins[left], builtins[right]) < 0
		})
		for _, builtin := range builtins {
			overloads := builder.builtinOverloads[builtin]
			if len(overloads) == 0 {
				return nil, &ScheduleError{
					Object: builtin.Name(),
					Reason: "environment builtin has no concrete overload",
				}
			}
			sort.Slice(overloads, func(left, right int) bool {
				return stableTypeString(overloads[left].signature) <
					stableTypeString(overloads[right].signature)
			})
			for _, overload := range overloads {
				statements = append(statements, overload.statements...)
			}
		}
		projectionNames := make(
			[]string,
			0,
			len(builder.projections),
		)
		for name := range builder.projections {
			projectionNames = append(projectionNames, name)
		}
		sort.Strings(projectionNames)
		for _, name := range projectionNames {
			statements = append(statements, builder.projections[name])
		}
		variables := make(
			[]*types.Var,
			0,
			len(builder.stateFields),
		)
		for variable := range builder.stateFields {
			variables = append(variables, variable)
		}
		sort.Slice(variables, func(left, right int) bool {
			return variables[left].Name() < variables[right].Name()
		})
		if len(variables) != 0 {
			fields := make([]tsgo.TypeElement, 0, len(variables))
			for _, variable := range variables {
				fields = append(fields, builder.stateFields[variable])
			}
			statements = append(
				statements,
				environmentcontract.StateDeclaration(
					builder.context,
					fields,
				),
			)
		}
		file, err := s.sourceFile(
			builder.outputPath,
			builder.sourcePackage.Name(),
			TargetFileEnvironmentContract,
			statements,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func (s *programSession) applyEnvironmentRequirement(
	requirement api.DeclarationRequirement,
) (bool, error) {
	owner, sourceOwned := requirement.Owner().Source()
	if !sourceOwned || owner == nil {
		return false, nil
	}
	sourcePackage := s.source.EnvironmentForTypes(owner.Pkg())
	if sourcePackage == nil {
		return false, nil
	}
	if err := s.require(owner); err != nil {
		return true, err
	}
	builder, err := s.requireEnvironmentPackage(sourcePackage)
	if err != nil {
		return true, err
	}
	if _, _, ok := requirement.NamedStructOperation(); ok {
		return true, nil
	}
	if _, ok := requirement.GenericCallableProfile(); ok {
		return true, s.applyEnvironmentGenericCallableProfile(
			builder,
			owner,
			requirement,
		)
	}
	if representationOwner, _, _, ok :=
		requirement.GenericRepresentation(); ok {
		if representationOwner != owner {
			return true, &ScheduleError{
				Object: owner.Name(),
				Reason: "environment generic representation requirement is foreign",
			}
		}
		return true, s.applyEnvironmentDeclarationRequirement(
			builder,
			owner,
			requirement,
		)
	}
	if _, _, _, _, ok := requirement.CallableControl(); ok {
		return true, s.applyEnvironmentCallableControl(
			builder,
			owner,
			requirement,
		)
	}
	if builtin, signature, ok := requirement.EnvironmentBuiltin(); ok {
		if builtin != owner {
			return true, &ScheduleError{
				Object: owner.Name(),
				Reason: "environment builtin requirement owner diverged",
			}
		}
		for _, existing := range builder.builtinOverloads[builtin] {
			if types.Identical(existing.signature, signature) {
				return true, nil
			}
		}
		target, err := environmentcontract.BuiltinDeclaration(
			builder.context,
			builder.emitter,
			builtin,
			signature,
		)
		if err != nil {
			return true, environmentContractError(owner, err)
		}
		if err := s.applyRootRequests(
			builder.placement,
			target.Requests(),
		); err != nil {
			return true, environmentContractError(owner, err)
		}
		builder.builtinOverloads[builtin] = append(
			builder.builtinOverloads[builtin],
			environmentBuiltinOverload{
				signature:  signature,
				statements: target.Declarations(),
			},
		)
		return true, nil
	}
	selected, projection, ok := requirement.ConstantProjection()
	if !ok {
		object := owner.Name()
		if owner.Pkg() != nil {
			object = owner.Pkg().Path() + "." + object
		}
		return true, &ScheduleError{
			Object: object,
			Reason: fmt.Sprintf(
				"environment declaration requirement kind %d is unsupported",
				requirement.Kind(),
			),
		}
	}
	base, ok := s.registry.Target(selected)
	if !ok {
		return true, &ScheduleError{
			Object: selected.Name(),
			Reason: "environment constant has no target binding",
		}
	}
	name, err := api.ConstantProjectionName(base.Name, projection)
	if err != nil {
		return true, err
	}
	if _, exists := builder.projections[name]; exists {
		return true, nil
	}
	statement, requests, err := environmentcontract.ConstantProjection(
		builder.context,
		builder.emitter,
		selected,
		projection,
	)
	if err != nil {
		return true, err
	}
	if err := s.applyRootRequests(builder.placement, requests); err != nil {
		return true, err
	}
	builder.projections[name] = statement
	return true, nil
}

func (b *environmentContractBuilder) environmentRequirements(
	object types.Object,
) []api.DeclarationRequirement {
	selected := b.requirements[object]
	requirements := make(
		[]api.DeclarationRequirement,
		0,
		len(selected),
	)
	for requirement := range selected {
		requirements = append(requirements, requirement)
	}
	sort.Slice(requirements, func(left, right int) bool {
		return compareDeclarationRequirements(
			requirements[left],
			requirements[right],
		) < 0
	})
	return requirements
}

func (s *programSession) buildEnvironmentDeclaration(
	builder *environmentContractBuilder,
	object types.Object,
	requirements []api.DeclarationRequirement,
) (environmentDeclaration, error) {
	target, err := environmentcontract.Declaration(
		builder.context,
		builder.emitter,
		object,
		requirements,
	)
	if err != nil {
		return environmentDeclaration{}, environmentContractError(object, err)
	}
	if err := s.applyRootRequests(
		builder.placement,
		target.Requests(),
	); err != nil {
		return environmentDeclaration{}, environmentContractError(object, err)
	}
	return environmentDeclaration{
		object:     object,
		name:       object.Name(),
		statements: target.Declarations(),
	}, nil
}

func (s *programSession) applyEnvironmentCallableControl(
	builder *environmentContractBuilder,
	object types.Object,
	requirement api.DeclarationRequirement,
) error {
	owner, enclosing, callable, control, ok :=
		requirement.CallableControl()
	source, sourceOwned := owner.Source()
	if !ok ||
		!sourceOwned ||
		source != object ||
		enclosing != nil ||
		callable != nil ||
		control != api.CallableControlRecovery {
		return &ScheduleError{
			Object: object.Name(),
			Reason: "environment callable-control requirement is invalid",
		}
	}
	return s.applyEnvironmentDeclarationRequirement(
		builder,
		object,
		requirement,
	)
}

func (s *programSession) applyEnvironmentGenericCallableProfile(
	builder *environmentContractBuilder,
	object types.Object,
	requirement api.DeclarationRequirement,
) error {
	function, callable := object.(*types.Func)
	profile, profiled := requirement.GenericCallableProfile()
	if !callable ||
		!profiled ||
		profile.Owner() != function.Origin() ||
		function != function.Origin() {
		return &ScheduleError{
			Object: object.Name(),
			Reason: "environment generic callable profile requirement is invalid",
		}
	}
	return s.applyEnvironmentDeclarationRequirement(
		builder,
		object,
		requirement,
	)
}

func (s *programSession) applyEnvironmentDeclarationRequirement(
	builder *environmentContractBuilder,
	object types.Object,
	requirement api.DeclarationRequirement,
) error {
	current := builder.requirements[object]
	if _, duplicate := current[requirement]; duplicate {
		return nil
	}
	next := make(
		map[api.DeclarationRequirement]struct{},
		len(current)+1,
	)
	for existing := range current {
		next[existing] = struct{}{}
	}
	next[requirement] = struct{}{}
	builder.requirements[object] = next
	if _, emitted := builder.declarations[object]; !emitted {
		return nil
	}
	target, err := s.buildEnvironmentDeclaration(
		builder,
		object,
		builder.environmentRequirements(object),
	)
	if err != nil {
		builder.requirements[object] = current
		return err
	}
	target.reconstructions =
		builder.declarations[object].reconstructions + 1
	builder.declarations[object] = target
	return nil
}
