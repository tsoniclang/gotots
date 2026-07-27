package emit

import (
	"fmt"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Root struct {
	object types.Object
}

type RootError struct {
	Object string
	Reason string
}

func (e *RootError) Error() string {
	if e.Object == "" {
		return "create emission root: " + e.Reason
	}
	return fmt.Sprintf("create emission root for %q: %s", e.Object, e.Reason)
}

func NewRoot(object types.Object) (Root, error) {
	if object == nil {
		return Root{}, &RootError{Reason: "object is nil"}
	}
	if object.Pkg() == nil ||
		(object.Parent() != object.Pkg().Scope() && !isDeclaredMethod(object)) {
		return Root{}, &RootError{
			Object: object.Name(),
			Reason: "object is not a package declaration",
		}
	}
	return Root{object: object}, nil
}

func isDeclaredMethod(object types.Object) bool {
	method, ok := object.(*types.Func)
	if !ok {
		return false
	}
	signature, ok := method.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return false
	}
	receiverType := signature.Recv().Type()
	if pointer, ok := types.Unalias(receiverType).(*types.Pointer); ok {
		receiverType = pointer.Elem()
	}
	named, ok := types.Unalias(receiverType).(*types.Named)
	return ok && named.Obj() != nil && named.Obj().Pkg() == object.Pkg()
}

func (r Root) Object() types.Object {
	return r.object
}

func ExportedAPIRoots(source *load.Package) ([]Root, error) {
	if source == nil || source.Types() == nil {
		return nil, &RootError{Reason: "source package is nil"}
	}
	scope := source.Types().Scope()
	names := scope.Names()
	sort.Strings(names)
	roots := make([]Root, 0, len(names))
	for _, name := range names {
		object := scope.Lookup(name)
		if object == nil || !object.Exported() {
			continue
		}
		root, err := NewRoot(object)
		if err != nil {
			return nil, err
		}
		roots = append(roots, root)
		typeName, ok := object.(*types.TypeName)
		if !ok {
			continue
		}
		named, ok := types.Unalias(typeName.Type()).(*types.Named)
		if !ok {
			continue
		}
		for index := range named.NumMethods() {
			method := named.Method(index)
			if !method.Exported() {
				continue
			}
			methodRoot, err := NewRoot(method)
			if err != nil {
				return nil, err
			}
			roots = append(roots, methodRoot)
		}
	}
	sort.Slice(roots, func(left, right int) bool {
		return compareObjects(roots[left].object, roots[right].object) < 0
	})
	return roots, nil
}

type scheduler struct {
	queue   []types.Object
	pending map[types.Object]struct{}
	emitted map[types.Object]struct{}
}

type ScheduleError struct {
	Object string
	Reason string
}

func (e *ScheduleError) Error() string {
	if e.Object == "" {
		return "schedule declaration: " + e.Reason
	}
	return fmt.Sprintf("schedule declaration %q: %s", e.Object, e.Reason)
}

func newScheduler() *scheduler {
	return &scheduler{
		pending: make(map[types.Object]struct{}),
		emitted: make(map[types.Object]struct{}),
	}
}

func (s *scheduler) enqueue(object types.Object) {
	if _, done := s.emitted[object]; done {
		return
	}
	if _, queued := s.pending[object]; queued {
		return
	}
	s.pending[object] = struct{}{}
	s.queue = append(s.queue, object)
}

func (s *scheduler) next() (types.Object, bool) {
	if len(s.queue) == 0 {
		return nil, false
	}
	object := s.queue[0]
	s.queue = s.queue[1:]
	delete(s.pending, object)
	s.emitted[object] = struct{}{}
	return object, true
}

func (s *scheduler) hasPending() bool {
	return len(s.queue) != 0 || len(s.pending) != 0
}

type declarationRequirementScheduler struct {
	pending map[api.DeclarationRequirement]struct{}
	applied map[api.DeclarationRequirement]struct{}
}

func newDeclarationRequirementScheduler() *declarationRequirementScheduler {
	return &declarationRequirementScheduler{
		pending: make(map[api.DeclarationRequirement]struct{}),
		applied: make(map[api.DeclarationRequirement]struct{}),
	}
}

func (s *declarationRequirementScheduler) enqueue(
	requirement api.DeclarationRequirement,
) {
	if _, done := s.applied[requirement]; done {
		return
	}
	if _, queued := s.pending[requirement]; queued {
		return
	}
	s.pending[requirement] = struct{}{}
}

func (s *declarationRequirementScheduler) nextBatch() (
	[]api.DeclarationRequirement,
	bool,
) {
	if len(s.pending) == 0 {
		return nil, false
	}
	var selected api.DeclarationRequirement
	first := true
	for requirement := range s.pending {
		if first || compareDeclarationRequirements(requirement, selected) < 0 {
			selected = requirement
			first = false
		}
	}
	requirements := make([]api.DeclarationRequirement, 0, len(s.pending))
	for requirement := range s.pending {
		if requirement.Owner() != selected.Owner() {
			continue
		}
		requirements = append(requirements, requirement)
	}
	sort.Slice(requirements, func(left, right int) bool {
		return compareDeclarationRequirements(
			requirements[left],
			requirements[right],
		) < 0
	})
	for _, requirement := range requirements {
		delete(s.pending, requirement)
		s.applied[requirement] = struct{}{}
	}
	return requirements, true
}

func (s *declarationRequirementScheduler) hasPending() bool {
	return len(s.pending) != 0
}

func (s *declarationRequirementScheduler) appliedFor(
	owner types.Object,
) []api.DeclarationRequirement {
	requirements := make([]api.DeclarationRequirement, 0)
	for requirement := range s.applied {
		if requirement.Owner() == owner {
			requirements = append(requirements, requirement)
		}
	}
	sort.Slice(requirements, func(left, right int) bool {
		return compareDeclarationRequirements(
			requirements[left],
			requirements[right],
		) < 0
	})
	return requirements
}

func compareDeclarationRequirements(
	left api.DeclarationRequirement,
	right api.DeclarationRequirement,
) int {
	if order := compareObjects(left.Owner(), right.Owner()); order != 0 {
		return order
	}
	if left.Kind() < right.Kind() {
		return -1
	}
	if left.Kind() > right.Kind() {
		return 1
	}
	if left.Kind() == api.DeclarationRequirementAddressableStorage {
		_, leftVariable, leftOK := left.AddressableStorage()
		_, rightVariable, rightOK := right.AddressableStorage()
		switch {
		case !leftOK && rightOK:
			return -1
		case leftOK && !rightOK:
			return 1
		case !leftOK:
			return 0
		default:
			return compareObjects(leftVariable, rightVariable)
		}
	}
	_, leftOperation, _ := left.NamedStructOperation()
	_, rightOperation, _ := right.NamedStructOperation()
	switch {
	case leftOperation < rightOperation:
		return -1
	case leftOperation > rightOperation:
		return 1
	default:
		return 0
	}
}

func (s *programSession) scheduleDeclarationRequirement(
	requirement api.DeclarationRequirement,
) error {
	if s.sealed {
		return &ScheduleError{
			Object: requirementOwnerName(requirement),
			Reason: "declaration requirement requested after target files were sealed",
		}
	}
	if !requirement.Valid() {
		return &ScheduleError{Reason: "declaration requirement is invalid"}
	}
	if _, ok := s.sites[requirement.Owner()]; !ok {
		return &ScheduleError{
			Object: requirementOwnerName(requirement),
			Reason: "declaration requirement owner has no source declaration",
		}
	}
	if err := s.require(requirement.Owner()); err != nil {
		return err
	}
	s.requirements.enqueue(requirement)
	return nil
}

func (s *programSession) applyDeclarationRequirements(
	requirements []api.DeclarationRequirement,
) error {
	if s.sealed {
		return &ScheduleError{
			Reason: "declaration requirements applied after target files were sealed",
		}
	}
	if len(requirements) == 0 {
		return &ScheduleError{Reason: "declaration requirement batch is empty"}
	}
	owner := requirements[0].Owner()
	if owner == nil {
		return &ScheduleError{Reason: "declaration requirement owner is nil"}
	}
	if _, ok := s.sites[owner]; !ok {
		return &ScheduleError{
			Object: owner.Name(),
			Reason: "declaration requirement owner lost its source declaration",
		}
	}
	for _, requirement := range requirements {
		if !requirement.Valid() || requirement.Owner() != owner {
			return &ScheduleError{
				Object: owner.Name(),
				Reason: "declaration requirement batch has mixed or invalid ownership",
			}
		}
		if _, accepted := s.requirements.applied[requirement]; !accepted {
			return &ScheduleError{
				Object: owner.Name(),
				Reason: "declaration requirement was not accepted by its owner",
			}
		}
	}
	return s.reconstructArtifact(owner)
}

func requirementOwnerName(requirement api.DeclarationRequirement) string {
	if requirement.Owner() == nil {
		return ""
	}
	return requirement.Owner().Name()
}

type artifactRevision struct {
	statements     []tsgo.Statement
	placement      *placementOwner
	dependencies   []api.ArtifactDependency
	contract       artifactstate.Contract
	temporaryStart temporarySnapshot
}

func (s *programSession) buildArtifactRevision(
	builder *targetFileBuilder,
	site declarationSite,
	owner types.Object,
	temporaryStart temporarySnapshot,
	reconstruction bool,
) (artifactRevision, error) {
	names, ok := builder.context.Names().(*fileNames)
	if !ok {
		return artifactRevision{}, &ScheduleError{
			Object: owner.Name(),
			Reason: "target artifact has no concrete name owner",
		}
	}
	if !reconstruction {
		temporaryStart = names.snapshotTemporaries()
	} else {
		current := names.snapshotTemporaries()
		names.restoreTemporaries(temporaryStart)
		defer names.restoreTemporaries(current)
	}
	finish, err := names.beginArtifact(owner)
	if err != nil {
		return artifactRevision{}, err
	}
	defer finish()

	result, err := builder.emitter.declarationObject(
		builder.context,
		site.declaration,
		owner,
		s.requirements.appliedFor(owner),
	)
	if err != nil {
		return artifactRevision{}, err
	}
	placement, dependencies, err := s.consumeArtifactRequests(
		owner,
		result.Requests(),
	)
	if err != nil {
		return artifactRevision{}, err
	}
	statements := result.Declarations()
	contract, err := artifactstate.ProjectContract(s.factory, statements)
	if err != nil {
		return artifactRevision{}, err
	}
	return artifactRevision{
		statements:     statements,
		placement:      placement,
		dependencies:   dependencies,
		contract:       contract,
		temporaryStart: temporaryStart,
	}, nil
}

func (s *programSession) consumeArtifactRequests(
	consumer types.Object,
	requests []api.RootRequest,
) (*placementOwner, []api.ArtifactDependency, error) {
	placement := newPlacementOwner()
	imports := make([]api.RootRequest, 0, len(requests))
	dependencies := make([]api.ArtifactDependency, 0, len(requests))
	for _, request := range requests {
		switch request.Kind() {
		case api.RootRequestImport:
			imports = append(imports, request)
		case api.RootRequestDeclarationRequirement:
			requirement, ok := request.DeclarationRequirement()
			if !ok {
				return nil, nil, &ScheduleError{
					Object: consumer.Name(),
					Reason: "declaration requirement is invalid",
				}
			}
			if err := s.scheduleDeclarationRequirement(requirement); err != nil {
				return nil, nil, err
			}
		case api.RootRequestArtifactDependency:
			dependency, ok := request.ArtifactDependency()
			if !ok {
				return nil, nil, &ScheduleError{
					Object: consumer.Name(),
					Reason: "artifact dependency is invalid",
				}
			}
			if _, ok := s.sites[dependency.Provider()]; !ok {
				return nil, nil, &ScheduleError{
					Object: dependency.Provider().Name(),
					Reason: "artifact dependency provider has no source declaration",
				}
			}
			if err := s.require(dependency.Provider()); err != nil {
				return nil, nil, err
			}
			dependencies = append(dependencies, dependency)
		default:
			return nil, nil, &ScheduleError{
				Object: consumer.Name(),
				Reason: "root request kind is invalid",
			}
		}
	}
	if err := placement.Apply(imports); err != nil {
		return nil, nil, err
	}
	return placement, dependencies, nil
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
	index, ok := builder.indexByObject[owner]
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
	if err := s.artifacts.Commit(
		owner,
		revision.contract,
		revision.dependencies,
	); err != nil {
		return err
	}
	s.artifacts.DiscardDirty(owner)
	declaration.statements = revision.statements
	declaration.placement = revision.placement
	declaration.reconstructions++
	return nil
}
