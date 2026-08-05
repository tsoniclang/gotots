package emit

import (
	"fmt"
	"go/types"
	"sort"

	environmentidentity "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	environmentcontract "github.com/tsoniclang/gotots/internal/emit/environmentcontract"
	emitordering "github.com/tsoniclang/gotots/internal/emit/ordering"
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

type environmentContractBuilder struct {
	sourcePackage *load.Package
	outputPath    string
	emitter       *emitter
	context       api.Context
	placement     *targetplacement.Owner
	declarations  map[types.Object]environmentDeclaration
	stateFields   map[*types.Var]environmentStateField
	projections   map[string]environmentConstantProjection
}

// RequireUse schedules the canonical declaration owner of one reference on
// the root scheduling record and joins its closed use demand and typed
// provider selection. The implementation route of an environment
// declaration is derived from the canonical binding evidence; a selection
// route cannot return a target without this observation.
func (s *programSession) RequireUse(
	object types.Object,
	demand environmentidentity.UseDemand,
	selection gostdlib.UseSelection,
) error {
	if s.sealed {
		objectName := ""
		if object != nil {
			objectName = object.Name()
		}
		return &ScheduleError{
			Object: objectName,
			Reason: "declaration requested after target files were sealed",
		}
	}
	if object == nil {
		return &ScheduleError{Reason: "referenced object is nil"}
	}
	if !demand.Valid() {
		return &ScheduleError{
			Object: object.Name(),
			Reason: "environment use demand is invalid",
		}
	}
	if function, ok := object.(*types.Func); ok {
		object = function.Origin()
	}
	sourcePackage := s.source.PackageForTypes(object.Pkg())
	environmentPackage := s.source.EnvironmentForTypes(object.Pkg())
	if _, ok := s.sites[object]; !ok && environmentPackage == nil {
		return &ScheduleError{
			Object: object.Name(),
			Reason: "object has no supported source declaration",
		}
	}
	if sourcePackage == nil && environmentPackage == nil {
		return &ScheduleError{
			Object: object.Name(),
			Reason: "object package has no declaration owner",
		}
	}
	if sourcePackage != nil {
		if selection.Kind() != gostdlib.UseSelectionNone {
			return &ScheduleError{
				Object: object.Name(),
				Reason: "provider selection was recorded for a source declaration",
			}
		}
		if err := s.requirePackage(sourcePackage); err != nil {
			return err
		}
	} else {
		route, environmentOwned := s.registry.EnvironmentSelectionRoute(object)
		if !environmentOwned {
			return &ScheduleError{
				Object: object.Name(),
				Reason: "environment declaration has no indexed binding",
			}
		}
		if !route.Valid() {
			// A selected constant's value is projected exactly from checker
			// evidence; it never demands a provider body, so a missing
			// provider binding selects its generated projection.
			if _, constant := object.(*types.Const); constant {
				route = environmentidentity.RouteGeneratedFacet
			} else {
				return &ScheduleError{
					Object: object.Name(),
					Reason: "selected standard-library declaration has no provider binding",
				}
			}
		}
		record := s.scheduler.record(object)
		if err := record.joinRoute(object, route); err != nil {
			return err
		}
		record.joinDemand(demand)
		record.joinSelection(selection)
		if _, err := s.requireEnvironmentPackage(environmentPackage); err != nil {
			return err
		}
	}
	s.scheduler.enqueue(object)
	return nil
}

// ObserveImplementation records a compiler-intrinsic or generated
// runtime-facet implementation route on the root scheduling record without
// scheduling an environment declaration artifact. Such declarations retain
// their existing compiler artifact owners and must not also demand a
// provider or boundary declaration.
func (s *programSession) ObserveImplementation(
	object types.Object,
	demand environmentidentity.UseDemand,
	route environmentidentity.ImplementationRoute,
) error {
	if s.sealed {
		objectName := ""
		if object != nil {
			objectName = object.Name()
		}
		return &ScheduleError{
			Object: objectName,
			Reason: "environment implementation observed after target files were sealed",
		}
	}
	if object == nil {
		return &ScheduleError{Reason: "observed environment object is nil"}
	}
	if !demand.Valid() {
		return &ScheduleError{
			Object: object.Name(),
			Reason: "environment use demand is invalid",
		}
	}
	if route != environmentidentity.RouteIntrinsic &&
		route != environmentidentity.RouteGeneratedFacet {
		return &ScheduleError{
			Object: object.Name(),
			Reason: "observed implementation route is not intrinsic or generated",
		}
	}
	if function, ok := object.(*types.Func); ok {
		object = function.Origin()
	}
	if s.source.EnvironmentForTypes(object.Pkg()) == nil {
		return &ScheduleError{
			Object: object.Name(),
			Reason: "observed implementation owner is not an environment declaration",
		}
	}
	record := s.scheduler.record(object)
	if err := record.joinRoute(object, route); err != nil {
		return err
	}
	record.joinDemand(demand)
	return nil
}

// environmentProfile derives the complete exact identity of the settled
// environment evidence for this compilation.
func (s *programSession) environmentProfile(
	options Options,
) (EnvironmentProfile, error) {
	return environmentcontract.NewProfile(
		s.source,
		options.IntegerRepresentation,
		options.EvaluationOrder,
		options.ConcurrencySemantics,
		options.StandardLibrary,
		options.ExternalProvider,
	)
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
	if s.standardLibrary != nil &&
		s.registry.HasProviderCoverageOwner(object) {
		if _, duplicate := builder.declarations[object]; duplicate {
			return &ScheduleError{
				Object: object.Name(),
				Reason: "provider coverage owner was emitted more than once",
			}
		}
		target, err := s.buildProviderCoverageDeclaration(object)
		if err != nil {
			return err
		}
		owner := api.MustSourceArtifactOwner(object)
		if err := s.commitArtifactRevision(
			owner,
			target.contract,
			target.dependencies,
			target.requirements,
		); err != nil {
			return err
		}
		builder.declarations[object] = target
		return nil
	}
	if variable, ok := object.(*types.Var); ok {
		if _, duplicate := builder.stateFields[variable]; duplicate {
			return &ScheduleError{
				Object: variable.Name(),
				Reason: "environment state field was emitted more than once",
			}
		}
		return s.emitEnvironmentStateField(builder, variable)
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
		s.requirements.appliedFor(api.MustSourceArtifactOwner(object)),
	)
	if err != nil {
		return err
	}
	if err := s.commitArtifactRevision(
		api.MustSourceArtifactOwner(object),
		target.contract,
		target.dependencies,
		target.requirements,
	); err != nil {
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
	if s.standardLibrary != nil &&
		sourcePackage.Kind() == load.PackageStandardLibraryContract {
		outputPath, err = targetoutput.StandardLibraryConstantProjectionPath(
			sourcePackage,
		)
		if err != nil {
			return nil, err
		}
	}
	targetEmitter := newEmitter(
		sourcePackage,
		s.factory,
		s.registry,
		s.scalar,
		s.providerScalar,
		s.evaluationOrder,
		s.concurrency,
		s,
		s,
		s,
		s,
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
		sourcePackage: sourcePackage,
		outputPath:    outputPath,
		emitter:       targetEmitter,
		context:       context,
		placement:     targetplacement.New(),
		declarations:  make(map[types.Object]environmentDeclaration),
		stateFields:   make(map[*types.Var]environmentStateField),
		projections:   make(map[string]environmentConstantProjection),
	}
	s.environmentBuilders[sourcePackage] = builder
	return builder, nil
}

func (s *programSession) environmentTargetFiles(
	requirements *targetRequirements,
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
		placement, err := builder.committedPlacement()
		if err != nil {
			return nil, err
		}
		requirements.observe(placement)
		statements := placement.Statements(s.factory)
		declarations := make(
			[]environmentDeclaration,
			0,
			len(builder.declarations),
		)
		for _, declaration := range builder.declarations {
			declarations = append(declarations, declaration)
		}
		sort.Slice(declarations, func(left, right int) bool {
			return emitordering.CompareObjects(
				declarations[left].object,
				declarations[right].object,
			) < 0
		})
		for _, declaration := range declarations {
			statements = append(statements, declaration.statements...)
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
			statements = append(
				statements,
				builder.projections[name].statement,
			)
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
				fields = append(fields, builder.stateFields[variable].field)
			}
			statements = append(
				statements,
				environmentcontract.StateDeclaration(
					builder.context,
					fields,
				),
			)
		}
		if s.standardLibrary != nil &&
			builder.sourcePackage.Kind() == load.PackageStandardLibraryContract &&
			len(builder.projections) == 0 {
			continue
		}
		kind := TargetFileEnvironmentContract
		if s.standardLibrary != nil &&
			builder.sourcePackage.Kind() == load.PackageStandardLibraryContract {
			kind = TargetFileStandardLibraryConstantProjection
		}
		file, err := s.sourceFile(
			builder.outputPath,
			builder.sourcePackage.Name(),
			kind,
			statements,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func (s *programSession) applyEnvironmentRequirementSet(
	object types.Object,
	requirements []api.DeclarationRequirement,
) error {
	sourcePackage := s.source.EnvironmentForTypes(object.Pkg())
	if sourcePackage == nil {
		return &ScheduleError{
			Object: object.Name(),
			Reason: "environment requirement owner lost its package",
		}
	}
	builder := s.environmentBuilders[sourcePackage]
	if builder == nil {
		return &ScheduleError{
			Object: object.Name(),
			Reason: "environment requirement owner was not emitted",
		}
	}
	if selected, ok := object.(*types.Const); ok {
		return s.replaceEnvironmentConstantProjections(
			builder,
			selected,
			requirements,
		)
	}
	if target, ok := builder.declarations[object]; ok && target.providerCoverage {
		if _, err := s.environmentDeclarationRequirements(
			object,
			requirements,
		); err != nil {
			return err
		}
		return s.reconstructEnvironmentDeclaration(builder, object)
	}
	if _, err := s.environmentDeclarationRequirements(
		object,
		requirements,
	); err != nil {
		return err
	}
	return s.reconstructEnvironmentDeclaration(builder, object)
}

func (s *programSession) environmentDeclarationRequirements(
	object types.Object,
	requirements []api.DeclarationRequirement,
) ([]api.DeclarationRequirement, error) {
	selected := make([]api.DeclarationRequirement, 0, len(requirements))
	for _, requirement := range requirements {
		if owner, _, ok := requirement.NamedStructOperation(); ok {
			if owner != object {
				return nil, &ScheduleError{
					Object: object.Name(),
					Reason: "environment named-struct operation requirement is foreign",
				}
			}
			continue
		}
		if representationOwner, _, _, ok :=
			requirement.GenericRepresentation(); ok {
			if representationOwner != object {
				return nil, &ScheduleError{
					Object: object.Name(),
					Reason: "environment generic representation requirement is foreign",
				}
			}
			selected = append(selected, requirement)
			continue
		}
		if representationOwner, artifact, _, ok :=
			requirement.TypeRepresentation(); ok {
			if representationOwner != object || artifact != nil {
				return nil, &ScheduleError{
					Object: object.Name(),
					Reason: "environment type representation requirement is foreign",
				}
			}
			selected = append(selected, requirement)
			continue
		}
		owner, enclosing, callable, control, ok :=
			requirement.CallableControl()
		source, sourceOwned := owner.Source()
		if ok &&
			sourceOwned &&
			source == object &&
			enclosing == nil &&
			callable == nil &&
			control == api.CallableControlRecovery {
			selected = append(selected, requirement)
			continue
		}
		return nil, &ScheduleError{
			Object: object.Name(),
			Reason: fmt.Sprintf(
				"environment declaration requirement kind %d is unsupported",
				requirement.Kind(),
			),
		}
	}
	return selected, nil
}
