package emit

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"go/token"
	"go/types"
	"sort"
)

type declarationRequirementLedger struct {
	byOwner map[api.ArtifactOwner]map[api.DeclarationRequirement]struct{}
}

func newDeclarationRequirementLedger() declarationRequirementLedger {
	return declarationRequirementLedger{
		byOwner: make(
			map[api.ArtifactOwner]map[api.DeclarationRequirement]struct{},
		),
	}
}

func (l declarationRequirementLedger) contains(
	requirement api.DeclarationRequirement,
) bool {
	requirements := l.byOwner[requirement.Owner()]
	if requirements == nil {
		return false
	}
	_, ok := requirements[requirement]
	return ok
}

func (l declarationRequirementLedger) containsOwner(
	owner api.ArtifactOwner,
) bool {
	return len(l.byOwner[owner]) != 0
}

func (l declarationRequirementLedger) add(
	requirement api.DeclarationRequirement,
) {
	owner := requirement.Owner()
	requirements := l.byOwner[owner]
	if requirements == nil {
		requirements = make(map[api.DeclarationRequirement]struct{})
		l.byOwner[owner] = requirements
	}
	requirements[requirement] = struct{}{}
}

func (l declarationRequirementLedger) remove(
	requirement api.DeclarationRequirement,
) {
	owner := requirement.Owner()
	requirements := l.byOwner[owner]
	delete(requirements, requirement)
	if len(requirements) == 0 {
		delete(l.byOwner, owner)
	}
}

func (l declarationRequirementLedger) replaceOwner(
	owner api.ArtifactOwner,
	requirements []api.DeclarationRequirement,
) {
	delete(l.byOwner, owner)
	for _, requirement := range requirements {
		l.add(requirement)
	}
}

func (l declarationRequirementLedger) forOwner(
	owner api.ArtifactOwner,
) ([]api.DeclarationRequirement, int) {
	selected := l.byOwner[owner]
	requirements := make([]api.DeclarationRequirement, 0, len(selected))
	for requirement := range selected {
		requirements = append(requirements, requirement)
	}
	sortDeclarationRequirements(requirements)
	return requirements, len(selected)
}

func (l declarationRequirementLedger) takeOwner(
	owner api.ArtifactOwner,
) []api.DeclarationRequirement {
	requirements, _ := l.forOwner(owner)
	delete(l.byOwner, owner)
	return requirements
}

func (l declarationRequirementLedger) empty() bool {
	return len(l.byOwner) == 0
}

func sortDeclarationRequirements(requirements []api.DeclarationRequirement) {
	sort.Slice(requirements, func(left, right int) bool {
		return compareDeclarationRequirements(
			requirements[left],
			requirements[right],
		) < 0
	})
}

type artifactOwnerPriorityQueue struct {
	owners  []api.ArtifactOwner
	compare func(api.ArtifactOwner, api.ArtifactOwner) int
}

func (q *artifactOwnerPriorityQueue) push(owner api.ArtifactOwner) {
	q.owners = append(q.owners, owner)
	index := len(q.owners) - 1
	for index > 0 {
		parent := (index - 1) / 2
		if q.compare(q.owners[parent], owner) <= 0 {
			break
		}
		q.owners[index] = q.owners[parent]
		index = parent
	}
	q.owners[index] = owner
}

func (q *artifactOwnerPriorityQueue) pop() (api.ArtifactOwner, bool) {
	if len(q.owners) == 0 {
		return api.ArtifactOwner{}, false
	}
	selected := q.owners[0]
	lastIndex := len(q.owners) - 1
	last := q.owners[lastIndex]
	q.owners = q.owners[:lastIndex]
	if len(q.owners) == 0 {
		return selected, true
	}
	index := 0
	for {
		left := index*2 + 1
		if left >= len(q.owners) {
			break
		}
		right := left + 1
		next := left
		if right < len(q.owners) &&
			q.compare(
				q.owners[right],
				q.owners[left],
			) < 0 {
			next = right
		}
		if q.compare(last, q.owners[next]) <= 0 {
			break
		}
		q.owners[index] = q.owners[next]
		index = next
	}
	q.owners[index] = last
	return selected, true
}

type declarationRequirementScheduler struct {
	pendingOwners artifactOwnerPriorityQueue
	pending       map[api.ArtifactOwner]struct{}
	removed       map[api.ArtifactOwner]struct{}
	orphaned      map[api.DeclarationRequirement]struct{}
	roots         declarationRequirementLedger
	byConsumer    map[api.ArtifactOwner]map[api.DeclarationRequirement]struct{}
	consumers     map[api.DeclarationRequirement]map[api.ArtifactOwner]struct{}
	active        declarationRequirementLedger
	applied       declarationRequirementLedger
}

func newDeclarationRequirementScheduler(
	compare func(api.ArtifactOwner, api.ArtifactOwner) int,
) *declarationRequirementScheduler {
	return &declarationRequirementScheduler{
		pendingOwners: artifactOwnerPriorityQueue{
			compare: compare,
		},
		pending: make(map[api.ArtifactOwner]struct{}),
		removed: make(map[api.ArtifactOwner]struct{}),
		orphaned: make(
			map[api.DeclarationRequirement]struct{},
		),
		roots: newDeclarationRequirementLedger(),
		byConsumer: make(
			map[api.ArtifactOwner]map[api.DeclarationRequirement]struct{},
		),
		consumers: make(
			map[api.DeclarationRequirement]map[api.ArtifactOwner]struct{},
		),
		active:  newDeclarationRequirementLedger(),
		applied: newDeclarationRequirementLedger(),
	}
}

func (s *declarationRequirementScheduler) enqueue(
	requirement api.DeclarationRequirement,
) {
	if s.roots.contains(requirement) {
		return
	}
	s.roots.add(requirement)
	if !s.active.contains(requirement) {
		s.active.add(requirement)
		s.enqueueOwner(requirement.Owner())
	}
}

func (s *declarationRequirementScheduler) nextBatch() (
	api.ArtifactOwner,
	[]api.DeclarationRequirement,
	bool,
	bool,
) {
	owner, ok := s.pendingOwners.pop()
	if !ok {
		return api.ArtifactOwner{}, nil, false, false
	}
	delete(s.pending, owner)
	_, removed := s.removed[owner]
	delete(s.removed, owner)
	requirements, _ := s.active.forOwner(owner)
	s.applied.replaceOwner(owner, requirements)
	return owner, requirements, removed, true
}

func (s *declarationRequirementScheduler) hasPending() bool {
	return len(s.pending) != 0 || len(s.orphaned) != 0
}

func (s *declarationRequirementScheduler) wasApplied(
	requirement api.DeclarationRequirement,
) bool {
	return s.applied.contains(requirement)
}

func (s *declarationRequirementScheduler) appliedFor(
	owner api.ArtifactOwner,
) []api.DeclarationRequirement {
	requirements, _ := s.applied.forOwner(owner)
	return requirements
}

func (s *declarationRequirementScheduler) replace(
	consumer api.ArtifactOwner,
	requirements []api.DeclarationRequirement,
) {
	next := make(
		map[api.DeclarationRequirement]struct{},
		len(requirements),
	)
	for _, requirement := range requirements {
		next[requirement] = struct{}{}
	}
	current := s.byConsumer[consumer]
	for requirement := range current {
		if _, retained := next[requirement]; retained {
			continue
		}
		owners := s.consumers[requirement]
		delete(owners, consumer)
		if len(owners) != 0 {
			continue
		}
		delete(s.consumers, requirement)
		if s.roots.contains(requirement) {
			continue
		}
		s.orphaned[requirement] = struct{}{}
	}
	for requirement := range next {
		if _, retained := current[requirement]; retained {
			continue
		}
		owners := s.consumers[requirement]
		if owners == nil {
			owners = make(map[api.ArtifactOwner]struct{})
			s.consumers[requirement] = owners
		}
		_, orphaned := s.orphaned[requirement]
		delete(s.orphaned, requirement)
		wasInactive := !orphaned &&
			!s.active.contains(requirement)
		owners[consumer] = struct{}{}
		if wasInactive {
			s.active.add(requirement)
			s.enqueueOwner(requirement.Owner())
		}
	}
	if len(next) == 0 {
		delete(s.byConsumer, consumer)
		return
	}
	s.byConsumer[consumer] = next
}

func (s *declarationRequirementScheduler) finalizeRemovals() bool {
	if len(s.pending) != 0 {
		panic("declaration requirement removal finalized before additions settled")
	}
	if len(s.orphaned) == 0 {
		return false
	}
	owners := make(map[api.ArtifactOwner]struct{})
	for requirement := range s.orphaned {
		delete(s.orphaned, requirement)
		if s.roots.contains(requirement) ||
			len(s.consumers[requirement]) != 0 {
			continue
		}
		s.active.remove(requirement)
		owners[requirement.Owner()] = struct{}{}
	}
	for owner := range owners {
		s.removed[owner] = struct{}{}
		s.enqueueOwner(owner)
	}
	return len(owners) != 0
}

func (s *declarationRequirementScheduler) enqueueOwner(
	owner api.ArtifactOwner,
) {
	if _, queued := s.pending[owner]; queued {
		return
	}
	s.pending[owner] = struct{}{}
	s.pendingOwners.push(owner)
}

func (s *programSession) providerGenericOperationSet(
	owner types.Object,
	consumer api.GenericOperationConsumer,
) (api.GenericOperationSet, bool, error) {
	function, callable := owner.(*types.Func)
	if !callable {
		operationSet, err := api.NewGenericOperationSet(
			owner,
			consumer,
			nil,
		)
		return operationSet, err == nil, err
	}
	documents, providerOwned, err :=
		s.registry.ProviderGenericOperations(function)
	if err != nil {
		return api.GenericOperationSet{}, false, err
	}
	if !providerOwned {
		operationSet, setErr := api.NewGenericOperationSet(
			function,
			consumer,
			nil,
		)
		return operationSet, setErr == nil, setErr
	}
	if consumer != api.GenericFunctionOperationConsumer() {
		return api.GenericOperationSet{}, false, &ScheduleError{
			Object: function.Name(),
			Reason: "provider generic operations have a non-function consumer",
		}
	}
	operations := make(
		[]*api.GenericOperationContract,
		0,
		len(documents),
	)
	for _, document := range documents {
		selection, selectionErr := providerGenericOperationSelection(
			document.Kind,
		)
		if selectionErr != nil {
			return api.GenericOperationSet{}, false, selectionErr
		}
		signature, signatureErr := providerGenericOperationSignature(
			function,
			document,
		)
		if signatureErr != nil {
			return api.GenericOperationSet{}, false, signatureErr
		}
		operation, operationErr := s.internGenericOperation(
			function,
			consumer,
			selection,
			signature,
		)
		if operationErr != nil {
			return api.GenericOperationSet{}, false, operationErr
		}
		operations = append(operations, operation)
	}
	operationSet, err := api.NewGenericOperationABISet(
		function,
		consumer,
		operations,
	)
	return operationSet, err == nil, err
}

func providerGenericOperationSelection(
	kind gostdlib.GenericOperationKind,
) (api.GenericOperationSelection, error) {
	var operation api.GenericOperation
	switch kind {
	case gostdlib.GenericOperationCopy:
		operation = api.GenericOperationCopy
	case gostdlib.GenericOperationZero:
		operation = api.GenericOperationZero
	case gostdlib.GenericOperationEqual:
		operation = api.GenericOperationEqual
	case gostdlib.GenericOperationBinaryLess:
		operation = api.GenericOperationBinaryLess
	case gostdlib.GenericOperationConvert:
		operation = api.GenericOperationConvert
	case gostdlib.GenericOperationMapConstruct:
		operation = api.GenericOperationMapConstruct
	case gostdlib.GenericOperationToStorage:
		operation = api.GenericOperationToStorage
	case gostdlib.GenericOperationFromStorage:
		operation = api.GenericOperationFromStorage
	case gostdlib.GenericOperationToContainerStorage:
		operation = api.GenericOperationToContainerStorage
	case gostdlib.GenericOperationFromContainerStorage:
		operation = api.GenericOperationFromContainerStorage
	case gostdlib.GenericOperationInterfaceAssertOK:
		operation = api.GenericOperationInterfaceAssertOK
	default:
		return api.GenericOperationSelection{}, &ScheduleError{
			Reason: "provider generic operation kind is invalid",
		}
	}
	return api.SelectGenericOperation(operation)
}

func providerGenericOperationSignature(
	owner *types.Func,
	document gostdlib.GenericOperationDocument,
) (*types.Signature, error) {
	parameters := api.GenericDeclarationParameters(owner)
	declaration, _ := owner.Type().(*types.Signature)
	var resolveType func(
		gostdlib.ContractTypeDocument,
	) (types.Type, error)
	resolveType = func(
		reference gostdlib.ContractTypeDocument,
	) (types.Type, error) {
		switch reference.Kind {
		case gostdlib.ContractTypeParameter:
			if reference.TypeParameter == nil ||
				*reference.TypeParameter < 0 ||
				*reference.TypeParameter >= len(parameters) {
				return nil, &ScheduleError{
					Object: owner.Name(),
					Reason: "provider generic operation type parameter is invalid",
				}
			}
			return parameters[*reference.TypeParameter], nil
		case gostdlib.ContractTypeCallableParameter:
			if declaration == nil || reference.CallableParameter == nil ||
				*reference.CallableParameter < 0 ||
				*reference.CallableParameter >= declaration.Params().Len() {
				return nil, &ScheduleError{
					Object: owner.Name(),
					Reason: "provider generic operation callable parameter is invalid",
				}
			}
			return declaration.Params().At(*reference.CallableParameter).Type(), nil
		case gostdlib.ContractTypeBool:
			return types.Typ[types.Bool], nil
		case gostdlib.ContractTypeInt:
			return types.Typ[types.Int], nil
		case gostdlib.ContractTypeSlice:
			if reference.Element == nil {
				break
			}
			element, err := resolveType(*reference.Element)
			if err != nil {
				return nil, err
			}
			return types.NewSlice(element), nil
		case gostdlib.ContractTypeMap:
			if reference.Key == nil || reference.Element == nil {
				break
			}
			key, err := resolveType(*reference.Key)
			if err != nil {
				return nil, err
			}
			element, err := resolveType(*reference.Element)
			if err != nil {
				return nil, err
			}
			return types.NewMap(key, element), nil
		}
		return nil, &ScheduleError{
			Object: owner.Name(),
			Reason: "provider generic operation type expression is invalid",
		}
	}
	resolveTuple := func(
		references []gostdlib.ContractTypeDocument,
	) (*types.Tuple, error) {
		variables := make([]*types.Var, 0, len(references))
		for _, reference := range references {
			selected, err := resolveType(reference)
			if err != nil {
				return nil, err
			}
			variables = append(variables, types.NewVar(
				token.NoPos,
				owner.Pkg(),
				"",
				selected,
			))
		}
		return types.NewTuple(variables...), nil
	}
	params, err := resolveTuple(document.Parameters)
	if err != nil {
		return nil, err
	}
	results, err := resolveTuple(document.Results)
	if err != nil {
		return nil, err
	}
	return types.NewSignatureType(
		nil,
		nil,
		nil,
		params,
		results,
		false,
	), nil
}

func (s *programSession) internGenericOperation(
	owner types.Object,
	consumer api.GenericOperationConsumer,
	selection api.GenericOperationSelection,
	signature *types.Signature,
) (*api.GenericOperationContract, error) {
	key, err := s.genericOperationKey(
		owner,
		consumer,
		selection,
		signature,
	)
	if err != nil {
		return nil, err
	}
	identity := genericOperationIdentity{
		owner:    owner,
		consumer: consumer,
		key:      key,
	}
	if existing := s.genericOperations[identity]; existing != nil {
		if existing.Consumer() != consumer ||
			existing.Selection() != selection ||
			!types.Identical(existing.Signature(), signature) {
			return nil, &ScheduleError{
				Object: owner.Name(),
				Reason: "generic operation identity changed semantic contract",
			}
		}
		return existing, nil
	}
	digest := sha256.Sum256([]byte(key))
	targetName := "$go$" + selection.Operation().Identifier() + "_" +
		hex.EncodeToString(digest[:10])
	contract, err := api.NewGenericOperationContract(
		owner,
		key,
		targetName,
		consumer,
		selection,
		signature,
	)
	if err != nil {
		return nil, err
	}
	s.genericOperations[identity] = contract
	return contract, nil
}
