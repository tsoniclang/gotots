package emit

import (
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	emitordering "github.com/tsoniclang/gotots/internal/emit/ordering"
	targetplacement "github.com/tsoniclang/gotots/internal/emit/placement"
)

func (s *programSession) consumeArtifactRequests(
	consumer api.ArtifactOwner,
	requests []api.RootRequest,
) (
	*targetplacement.Owner,
	[]api.ArtifactDependency,
	[]api.DeclarationRequirement,
	error,
) {
	placement := targetplacement.New()
	dependencies := make(map[api.ArtifactDependency]struct{})
	requirements := make(map[api.DeclarationRequirement]struct{})
	err := api.WalkUniqueRootRequestPayloads(
		requests,
		func(request api.RootRequest) error {
			switch request.Kind() {
			case api.RootRequestImport:
				return placement.Apply([]api.RootRequest{request})
			case api.RootRequestDeclarationRequirement:
				requirement, ok := request.DeclarationRequirement()
				if !ok {
					return &ScheduleError{
						Object: consumer.Name(),
						Reason: "declaration requirement is invalid",
					}
				}
				if _, duplicate := requirements[requirement]; duplicate {
					return nil
				}
				if err := s.prepareDeclarationRequirement(requirement); err != nil {
					return err
				}
				requirements[requirement] = struct{}{}
			case api.RootRequestArtifactDependency:
				dependency, ok := request.ArtifactDependency()
				if !ok {
					return &ScheduleError{
						Object: consumer.Name(),
						Reason: "artifact dependency is invalid",
					}
				}
				if _, duplicate := dependencies[dependency]; duplicate {
					return nil
				}
				sourceObject, sourceProvider := dependency.Provider().Source()
				if sourceProvider {
					_, sourceProvider = s.sites[sourceObject]
					sourceProvider = sourceProvider ||
						s.environmentArtifactSource(sourceObject)
				}
				generated, generatedProvider := dependency.Provider().Generated()
				if generatedProvider {
					generatedProvider =
						s.validateGeneratedArtifact(generated) == nil &&
							(generated.Placement() ==
								api.GeneratedArtifactPlacementCompilation ||
								generated.Placement() ==
									api.GeneratedArtifactPlacementContract)
				}
				if !sourceProvider && !generatedProvider {
					return &ScheduleError{
						Object: dependency.Provider().Name(),
						Reason: "artifact dependency provider has no reconstructible declaration",
					}
				}
				if sourceProvider {
					if err := s.require(sourceObject); err != nil {
						return err
					}
				}
				dependencies[dependency] = struct{}{}
			default:
				return &ScheduleError{
					Object: consumer.Name(),
					Reason: "root request kind is invalid",
				}
			}
			return nil
		},
	)
	if err != nil {
		return nil, nil, nil, err
	}
	selectedDependencies := make(
		[]api.ArtifactDependency,
		0,
		len(dependencies),
	)
	for dependency := range dependencies {
		selectedDependencies = append(selectedDependencies, dependency)
	}
	sort.Slice(selectedDependencies, func(left, right int) bool {
		order := emitordering.CompareArtifactOwners(
			selectedDependencies[left].Provider(),
			selectedDependencies[right].Provider(),
		)
		if order != 0 {
			return order < 0
		}
		return selectedDependencies[left].Facet() <
			selectedDependencies[right].Facet()
	})
	selectedRequirements := make(
		[]api.DeclarationRequirement,
		0,
		len(requirements),
	)
	for requirement := range requirements {
		selectedRequirements = append(selectedRequirements, requirement)
	}
	sortDeclarationRequirements(selectedRequirements)
	return placement,
		selectedDependencies,
		selectedRequirements,
		nil
}

func canonicalDeclarationRequirements(
	requirements []api.DeclarationRequirement,
) []api.DeclarationRequirement {
	unique := make(map[api.DeclarationRequirement]struct{}, len(requirements))
	for _, requirement := range requirements {
		unique[requirement] = struct{}{}
	}
	result := make([]api.DeclarationRequirement, 0, len(unique))
	for requirement := range unique {
		result = append(result, requirement)
	}
	sortDeclarationRequirements(result)
	return result
}

func (s *programSession) commitArtifactRevision(
	owner api.ArtifactOwner,
	contract artifactstate.Contract,
	dependencies []api.ArtifactDependency,
	requirements []api.DeclarationRequirement,
) error {
	if err := s.commitArtifactContract(owner, contract, dependencies); err != nil {
		return err
	}
	s.requirements.replace(owner, requirements)
	return nil
}

func (s *programSession) commitArtifactContract(
	owner api.ArtifactOwner,
	contract artifactstate.Contract,
	dependencies []api.ArtifactDependency,
) error {
	if s.requirementRemovalOwner == owner {
		return s.artifacts.CommitHistoricalReplacement(
			owner,
			contract,
			dependencies,
		)
	}
	return s.artifacts.Commit(owner, contract, dependencies)
}

func (s *programSession) reconstructArtifact(owner types.Object) error {
	if s.sealed {
		return &ScheduleError{
			Object: owner.Name(),
			Reason: "target artifact reconstructed after target files were sealed",
		}
	}
	site, ok := s.sites[owner]
	if !ok {
		return &ScheduleError{
			Object: owner.Name(),
			Reason: "dirty target artifact lost its source declaration",
		}
	}
	builder, err := s.builder(site)
	if err != nil {
		return err
	}
	artifactOwner := api.MustSourceArtifactOwner(owner)
	index, ok := builder.indexByOwner[artifactOwner]
	if !ok {
		return &ScheduleError{
			Object: owner.Name(),
			Reason: "dirty target artifact was not emitted first",
		}
	}
	declaration := &builder.declarations[index]
	revision, err := s.buildArtifactRevision(
		builder,
		site,
		owner,
		declaration.temporaryStart,
		true,
	)
	if err != nil {
		return err
	}
	if err := s.commitArtifactRevision(
		artifactOwner,
		revision.contract,
		revision.dependencies,
		revision.requirements,
	); err != nil {
		return err
	}
	s.commitClassMemberContribution(owner, revision.classContribution)
	s.artifacts.DiscardDirty(artifactOwner)
	declaration.statements = revision.statements
	declaration.placement = revision.placement
	declaration.eagerDependencies = revision.eagerDependencies
	declaration.reconstructions++
	return nil
}

func eagerDeclarationDependencies(
	consumer types.Object,
	dependencies []api.ArtifactDependency,
) []api.ArtifactOwner {
	if _, eager := consumer.(*types.Const); !eager {
		return nil
	}
	seen := make(map[api.ArtifactOwner]struct{})
	result := make([]api.ArtifactOwner, 0, len(dependencies))
	for _, dependency := range dependencies {
		provider := dependency.Provider()
		if _, sourceOwned := provider.Source(); !sourceOwned ||
			provider == api.MustSourceArtifactOwner(consumer) {
			continue
		}
		if _, duplicate := seen[provider]; duplicate {
			continue
		}
		seen[provider] = struct{}{}
		result = append(result, provider)
	}
	sort.Slice(result, func(left, right int) bool {
		return emitordering.CompareArtifactOwners(
			result[left],
			result[right],
		) < 0
	})
	return result
}

func (s *programSession) reconstructScheduledArtifact(
	owner api.ArtifactOwner,
) error {
	if _, ok := owner.PackageAssembly(); ok {
		return s.reconstructPackageExports(owner)
	}
	if generated, ok := owner.Generated(); ok {
		return s.reconstructGeneratedArtifact(generated)
	}
	if _, _, ok := owner.PackageInitializer(); ok {
		return s.reconstructPackageInitializer(owner)
	}
	source, ok := owner.Source()
	if !ok {
		return &ScheduleError{Reason: "dirty target artifact owner is invalid"}
	}
	if s.environmentArtifactSource(source) {
		return s.reconstructEnvironmentArtifact(source)
	}
	if variable, ok := source.(*types.Var); ok {
		return s.reconstructPackageStorage(owner, variable)
	}
	return s.reconstructArtifact(source)
}
