package emit

import (
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	environmentcontract "github.com/tsoniclang/gotots/internal/emit/environmentcontract"
	"github.com/tsoniclang/gotots/internal/emit/generic/semanticname"
	emitordering "github.com/tsoniclang/gotots/internal/emit/ordering"
	targetplacement "github.com/tsoniclang/gotots/internal/emit/placement"
	provideroperation "github.com/tsoniclang/gotots/internal/emit/provideroperation"
	"github.com/tsoniclang/gotots/internal/emit/requirements"
)

type declarationRequirementScheduler = requirements.Scheduler

func newDeclarationRequirementScheduler(
	compareOwners func(api.ArtifactOwner, api.ArtifactOwner) int,
) *declarationRequirementScheduler {
	return requirements.New(compareOwners, compareDeclarationRequirements)
}

func (s *programSession) NamedStructOperationSelected(
	owner *types.TypeName,
	operation api.NamedStructOperation,
) (bool, error) {
	requirement, err := api.NewNamedStructOperationRequirement(owner, operation)
	if err != nil {
		return false, err
	}
	return s.requirements.WasSelected(requirement), nil
}

func (s *programSession) AnonymousStructDemandSelected(
	artifact *api.GeneratedArtifact,
	demand api.AnonymousStructDemand,
) (bool, error) {
	requirement, err := api.NewAnonymousStructRequirement(artifact, demand)
	if err != nil {
		return false, err
	}
	return s.requirements.WasSelected(requirement), nil
}

func (s *programSession) prepareDeclarationRequestGraph(
	consumer api.ArtifactOwner,
	requests []api.RootRequest,
) error {
	if s.preparedDeclarationRequests == nil {
		s.preparedDeclarationRequests = make(map[api.RootRequest]struct{})
	}
	for _, request := range requests {
		if err := s.prepareDeclarationRequestNode(consumer, request); err != nil {
			return err
		}
	}
	return nil
}

func (s *programSession) prepareDeclarationRequestNode(
	consumer api.ArtifactOwner,
	request api.RootRequest,
) error {
	if _, prepared := s.preparedDeclarationRequests[request]; prepared {
		return nil
	}
	if children, nested := request.NestedRequests(); nested {
		for _, child := range children {
			if err := s.prepareDeclarationRequestNode(consumer, child); err != nil {
				return err
			}
		}
		s.preparedDeclarationRequests[request] = struct{}{}
		return nil
	}
	requirement, ok := request.DeclarationRequirement()
	if !ok {
		return &ScheduleError{
			Object: consumer.Name(),
			Reason: "declaration request graph contains a non-declaration leaf",
		}
	}
	if err := s.prepareDeclarationRequirement(requirement); err != nil {
		return err
	}
	s.preparedDeclarationRequests[request] = struct{}{}
	return nil
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
		selection, selectionErr := provideroperation.Selection(document.Kind)
		if selectionErr != nil {
			return api.GenericOperationSet{}, false, selectionErr
		}
		signature, signatureErr := provideroperation.Signature(function, document)
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

func (s *programSession) internGenericOperation(
	owner types.Object,
	consumer api.GenericOperationConsumer,
	selection api.GenericOperationSelection,
	signature *types.Signature,
) (*api.GenericOperationContract, error) {
	key, err := s.genericOperationKey(owner, consumer, selection, signature)
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
	targetName, err := semanticname.OperationNameWithIdentityTokens(
		selection.Operation().Identifier(),
		genericOperationMethod(selection),
		signature,
		s.semanticNamedTypeToken,
		s.semanticPackageToken,
	)
	if err != nil {
		return nil, err
	}
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

func genericOperationMethod(
	selection api.GenericOperationSelection,
) *types.Func {
	method, _ := selection.Method()
	return method
}

func (s *programSession) semanticPackageToken(
	sourcePackage *types.Package,
) (string, error) {
	if s == nil || s.registry == nil || sourcePackage == nil {
		return "", &ScheduleError{Reason: "semantic package token owner is invalid"}
	}
	qualifier := s.registry.ImportQualifier(sourcePackage)
	if qualifier == "" {
		return "", &ScheduleError{
			Object: sourcePackage.Path(),
			Reason: "semantic package token is absent",
		}
	}
	return qualifier, nil
}

func (s *programSession) semanticNamedTypeToken(
	object *types.TypeName,
) (string, error) {
	if object == nil {
		return "", &ScheduleError{Reason: "semantic named type is nil"}
	}
	if object.Pkg() == nil {
		return semanticname.Identifier(object.Name()), nil
	}
	if object.Parent() != object.Pkg().Scope() {
		return "", &ScheduleError{
			Object: object.Name(),
			Reason: "semantic operation local type has no lexical owner",
		}
	}
	qualifier, err := s.semanticPackageToken(object.Pkg())
	if err != nil {
		return "", err
	}
	return qualifier + "$" + semanticname.Identifier(object.Name()), nil
}

func (s *programSession) consumeArtifactRequests(
	consumer api.ArtifactOwner,
	requests []api.RootRequest,
) (
	*targetplacement.Owner,
	[]api.ArtifactDependency,
	[]api.RootRequest,
	error,
) {
	placement := targetplacement.New()
	dependencies := make(map[api.ArtifactDependency]struct{})
	nonDeclarationRequests, err := api.SelectNonDeclarationRequests(requests)
	if err != nil {
		return nil, nil, nil, err
	}
	err = api.WalkUniqueRootRequestPayloads(
		nonDeclarationRequests,
		func(request api.RootRequest) error {
			switch request.Kind() {
			case api.RootRequestImport:
				return placement.Apply([]api.RootRequest{request})
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
				if err := s.prepareArtifactDependency(dependency); err != nil {
					return err
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
	selectedRequests, err := api.SelectDeclarationRequests(requests)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := s.prepareDeclarationRequestGraph(
		consumer,
		selectedRequests,
	); err != nil {
		return nil, nil, nil, err
	}
	return placement,
		selectedDependencies,
		selectedRequests,
		nil
}

func (s *programSession) prepareArtifactDependency(
	dependency api.ArtifactDependency,
) error {
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
	if !sourceProvider {
		return nil
	}
	return s.RequireUse(
		sourceObject,
		environmentcontract.ArtifactFacetUseDemand(
			dependency.Facet(),
			sourceObject,
		),
		gostdlib.NoUseSelection(),
	)
}
