package emit

import (
	"crypto/sha256"
	"encoding/hex"
	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/emit/api"
	externalfunction "github.com/tsoniclang/gotots/internal/emit/externalfunction"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
	"github.com/tsoniclang/gotots/internal/load"
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"sort"
	"strconv"
	"strings"
)

type genericOperationIdentity struct {
	owner    types.Object
	consumer api.GenericOperationConsumer
	key      string
}

type genericConcretizationIdentity struct {
	owner  *types.Func
	key    string
	effect api.GenericConcretizationEffect
}

func (s *programSession) GenericCallableRequiresConcretization(
	owner *types.Func,
) (bool, error) {
	if owner == nil || owner.Origin() != owner {
		return false, &ScheduleError{
			Reason: "generic concretization owner is invalid",
		}
	}
	if _, sourceOwned := s.sites[owner]; !sourceOwned {
		if s.source.EnvironmentForTypes(owner.Pkg()) != nil {
			_, kernelOwned, err := s.registry.ProviderGenericKernel(owner)
			return kernelOwned, err
		}
		return false, &ScheduleError{
			Object: owner.Name(),
			Reason: "generic concretization owner has no declaration",
		}
	}
	artifactOwner := api.MustSourceArtifactOwner(owner)
	return api.GenericKernelRequired(
		owner,
		s.requirements.selectedFor(artifactOwner),
	)
}

func (s *programSession) GenericCallableSynchronousParameters(
	owner *types.Func,
) ([]int, bool, error) {
	selected, ok, err :=
		s.registry.ProviderSynchronousGenericKernel(owner)
	if err != nil || !ok {
		return nil, ok, err
	}
	parameters := selected.CallableParameters()
	indexes := make([]int, len(parameters))
	for index, parameter := range parameters {
		indexes[index] = parameter.Parameter
	}
	return indexes, true, nil
}

func (s *programSession) ResolveGenericConcretization(
	owner *types.Func,
	arguments api.TypeArgumentList,
	signature *types.Signature,
	effect api.GenericConcretizationEffect,
	placement api.GeneratedArtifactPlacement,
	lexicalOwner api.ArtifactOwner,
	anchor *types.TypeName,
) (*api.GenericConcretization, error) {
	if owner == nil || owner.Origin() != owner || arguments.Len() == 0 ||
		signature == nil || !effect.Valid() ||
		(placement != api.GeneratedArtifactPlacementCompilation &&
			placement != api.GeneratedArtifactPlacementLexical) {
		return nil, &ScheduleError{
			Reason: "generic concretization identity is invalid",
		}
	}
	if _, sourceOwned := s.sites[owner]; !sourceOwned {
		if s.source.EnvironmentForTypes(owner.Pkg()) == nil {
			return nil, &ScheduleError{
				Object: owner.Name(),
				Reason: "generic concretization has no selected declaration",
			}
		}
		_, kernelOwned, err := s.registry.ProviderGenericKernel(owner)
		if err != nil {
			return nil, err
		}
		if !kernelOwned {
			return nil, &ScheduleError{
				Object: owner.Name(),
				Reason: "generic concretization has no certified provider kernel",
			}
		}
	}
	sourceSignature, ok := owner.Type().(*types.Signature)
	if !ok || len(api.GenericDeclarationParameters(owner)) != arguments.Len() ||
		(sourceSignature.Recv() == nil) != (signature.Recv() == nil) {
		return nil, &ScheduleError{
			Object: owner.Name(),
			Reason: "generic concretization arguments do not match the source declaration",
		}
	}
	selected := make([]types.Type, 0, arguments.Len())
	var identity strings.Builder
	ownerIdentity, err := typeidentity.SourceObjectKey(owner)
	if err != nil {
		return nil, err
	}
	identity.WriteString("generic-concretization|")
	identity.WriteString(strconv.Itoa(len(ownerIdentity)))
	identity.WriteByte(':')
	identity.WriteString(ownerIdentity)
	identity.WriteByte('|')
	if effect.Synchronous() {
		identity.WriteString("effect=synchronous|")
	}
	namedIdentity := s.genericConcretizationNamedIdentity(
		placement,
		lexicalOwner,
	)
	for index := range arguments.Len() {
		argument := arguments.At(index)
		if api.ContainsGenericTypeParameter(argument) {
			return nil, &ScheduleError{
				Object: owner.Name(),
				Reason: "generic concretization argument " +
					strconv.Itoa(index) + " remains open: " +
					types.TypeString(argument, nil),
			}
		}
		descriptor, err := typeidentity.BuildDescriptor(
			argument,
			namedIdentity,
		)
		if err != nil {
			return nil, err
		}
		selected = append(selected, argument)
		identity.WriteString(strconv.Itoa(len(descriptor)))
		identity.WriteByte(':')
		identity.WriteString(descriptor)
	}
	digest := sha256.Sum256([]byte(identity.String()))
	key := hex.EncodeToString(digest[:])
	selection := genericConcretizationIdentity{
		owner:  owner,
		key:    key,
		effect: effect,
	}
	if existing := s.genericConcretizations[selection]; existing != nil {
		if !types.Identical(existing.Signature(), signature) {
			return nil, &ScheduleError{
				Object: owner.Name(),
				Reason: "generic concretization key joined different signatures",
			}
		}
		if existing.Placement() != placement ||
			existing.LexicalOwner() != lexicalOwner ||
			existing.LexicalAnchor() != anchor {
			return nil, &ScheduleError{
				Object: owner.Name(),
				Reason: "generic concretization key joined different placements",
			}
		}
		return existing, nil
	}
	expected, err := api.InstantiateGenericCallable(owner, arguments)
	if err != nil {
		return nil, err
	}
	if !types.Identical(expected, signature) {
		return nil, &ScheduleError{
			Object: owner.Name(),
			Reason: "generic concretization signature differs from its exact instance",
		}
	}
	concretization, err := api.NewGenericConcretization(
		owner,
		selected,
		signature,
		effect,
		key,
		placement,
		lexicalOwner,
		anchor,
	)
	if err != nil {
		return nil, err
	}
	s.genericConcretizations[selection] = concretization
	return concretization, nil
}

func (s *programSession) genericConcretizationNamedIdentity(
	placement api.GeneratedArtifactPlacement,
	lexicalOwner api.ArtifactOwner,
) typeidentity.NamedObjectIdentity {
	return func(object *types.TypeName) (string, error) {
		if object == nil {
			return "", &api.NameError{
				Reason: "generic concretization named type is nil",
			}
		}
		if object.Pkg() == nil ||
			object.Parent() == object.Pkg().Scope() {
			return typeidentity.NamedObjectKey(object)
		}
		if placement != api.GeneratedArtifactPlacementLexical ||
			!lexicalOwner.Valid() ||
			lexicalOwner.Package() != object.Pkg() {
			return "", &api.NameError{
				Name:   object.Name(),
				Reason: "generic concretization local type has no lexical placement",
			}
		}
		return typeidentity.LexicalNamedObjectKey(
			object,
			lexicalOwner,
			object.Pkg().Scope(),
		)
	}
}

func (s *programSession) ResolveGenericOperationSet(
	declaration types.Object,
	consumer api.GenericOperationConsumer,
) (api.GenericOperationSet, bool, error) {
	owner := api.GenericDeclarationOrigin(declaration)
	if owner == nil ||
		!consumer.Valid() ||
		len(api.GenericDeclarationParameters(owner)) == 0 {
		return api.GenericOperationSet{}, false, nil
	}
	if _, ok := s.sites[owner]; !ok {
		if s.source.EnvironmentForTypes(owner.Pkg()) != nil {
			return s.providerGenericOperationSet(owner, consumer)
		}
		return api.GenericOperationSet{}, false, &ScheduleError{
			Object: owner.Name(),
			Reason: "generic operation set has no source declaration",
		}
	}
	var operations []*api.GenericOperationContract
	for _, requirement := range s.requirements.selectedFor(
		api.MustSourceArtifactOwner(owner),
	) {
		requirementOwner, operation, generic :=
			requirement.GenericOperation()
		if !generic {
			continue
		}
		if requirementOwner != owner ||
			operation.Consumer() != consumer {
			if requirementOwner == owner {
				continue
			}
			return api.GenericOperationSet{}, false, &ScheduleError{
				Object: owner.Name(),
				Reason: "generic operation has inconsistent ownership",
			}
		}
		operations = append(operations, operation)
	}
	sort.Slice(operations, func(left, right int) bool {
		return operations[left].Key() < operations[right].Key()
	})
	operationSet, err := api.NewGenericOperationSet(
		owner,
		consumer,
		operations,
	)
	return operationSet, err == nil, err
}

func (s *programSession) ResolveGenericOperation(
	declaration types.Object,
	consumer api.GenericOperationConsumer,
	selection api.GenericOperationSelection,
	signature *types.Signature,
) (*api.GenericOperationContract, error) {
	owner := api.GenericDeclarationOrigin(declaration)
	if owner == nil {
		return nil, &ScheduleError{
			Reason: "generic operation owner is nil",
		}
	}
	if !consumer.Valid() || !selection.Valid() || signature == nil {
		return nil, &ScheduleError{
			Object: owner.Name(),
			Reason: "generic operation identity is invalid",
		}
	}
	if _, ok := s.sites[owner]; !ok ||
		len(api.GenericDeclarationParameters(owner)) == 0 {
		return nil, &ScheduleError{
			Object: owner.Name(),
			Reason: "generic operation owner has no generic declaration",
		}
	}
	return s.internGenericOperation(
		owner,
		consumer,
		selection,
		signature,
	)
}

func (s *programSession) genericOperationKey(
	owner types.Object,
	consumer api.GenericOperationConsumer,
	selection api.GenericOperationSelection,
	signature *types.Signature,
) (string, error) {
	signatureKey, err := typeidentity.BuildParameterizedKey(
		signature,
		s.genericOperationNamedIdentity(owner),
		genericOperationParameterIdentity(owner),
	)
	if err != nil {
		return "", err
	}
	prefix, err := selection.IdentityPrefix()
	if err != nil {
		return "", err
	}
	return consumer.Identity() + "|" + prefix + "|" + signatureKey, nil
}

func (s *programSession) genericOperationNamedIdentity(
	owner types.Object,
) typeidentity.NamedObjectIdentity {
	return func(object *types.TypeName) (string, error) {
		if object == nil {
			return "", &api.NameError{
				Reason: "generic operation named type is nil",
			}
		}
		if object.Pkg() == nil {
			return typeidentity.NamedObjectKey(object)
		}
		if object.Parent() == object.Pkg().Scope() {
			return typeidentity.NamedObjectKey(object)
		}
		site, ok := s.sites[owner]
		if !ok {
			return "", &api.NameError{
				Name:   object.Name(),
				Reason: "generic operation local type has no owning declaration",
			}
		}
		function, functionOwner := site.Declaration.(*ast.FuncDecl)
		var root *types.Scope
		if functionOwner {
			root = site.Source.TypesInfo().Scopes[function.Type]
		}
		if owner.Pkg() != object.Pkg() || root == nil {
			return "", &api.NameError{
				Name:   object.Name(),
				Reason: "generic operation local type has no owning declaration",
			}
		}
		return typeidentity.LexicalNamedObjectKey(
			object,
			api.MustSourceArtifactOwner(owner),
			root,
		)
	}
}

func genericOperationParameterIdentity(
	owner types.Object,
) typeidentity.TypeParameterIdentity {
	identities := make(map[*types.TypeParam]string)
	switch owner := owner.(type) {
	case *types.Func:
		signature, _ := owner.Type().(*types.Signature)
		if signature != nil {
			for index := range signature.RecvTypeParams().Len() {
				identities[signature.RecvTypeParams().At(index)] =
					"receiver|" + strconv.Itoa(index)
			}
			for index := range signature.TypeParams().Len() {
				identities[signature.TypeParams().At(index)] =
					"function|" + strconv.Itoa(index)
			}
		}
	case *types.TypeName:
		for index, parameter := range api.GenericDeclarationParameters(owner) {
			identities[parameter] = "type|" + strconv.Itoa(index)
		}
	}
	return func(parameter *types.TypeParam) (string, error) {
		identity := identities[parameter]
		if identity == "" {
			name := ""
			if parameter != nil && parameter.Obj() != nil {
				name = parameter.Obj().Name()
			}
			return "", &api.NameError{
				Name:   name,
				Reason: "generic operation type parameter is foreign to its owner",
			}
		}
		return identity, nil
	}
}

type ExternalFunctionObligation struct {
	identity      string
	function      *types.Func
	signature     *types.Signature
	modulePath    string
	moduleVersion string
	role          api.Role
	position      token.Position
	buildProfile  load.BuildProfile
}

type ExternalFunctionBindingError = externalfunction.BindingError

func (o ExternalFunctionObligation) Identity() string {
	return o.identity
}

func (o ExternalFunctionObligation) Function() *types.Func {
	return o.function
}

func (o ExternalFunctionObligation) Signature() *types.Signature {
	return o.signature
}

func (o ExternalFunctionObligation) ModulePath() string {
	return o.modulePath
}

func (o ExternalFunctionObligation) ModuleVersion() string {
	return o.moduleVersion
}

func (o ExternalFunctionObligation) Role() api.Role {
	return o.role
}

func (o ExternalFunctionObligation) Position() token.Position {
	return o.position
}

func (o ExternalFunctionObligation) BuildProfile() load.BuildProfile {
	return o.buildProfile
}

func (o ExternalFunctionObligation) valid() bool {
	if o.identity == "" || o.function == nil ||
		o.function != o.function.Origin() || o.signature == nil ||
		o.function.Type() != o.signature || o.modulePath == "" ||
		o.role != api.RoleFileDeclaration || !o.position.IsValid() ||
		!o.buildProfile.Valid() {
		return false
	}
	contract, err := environmentcontract.Describe(o.function)
	return err == nil && contract.Identity() == o.identity
}

func newExternalFunctionObligation(
	site declarationSite,
	function *types.Func,
	profile load.BuildProfile,
) (ExternalFunctionObligation, error) {
	declaration, ok := site.Declaration.(*ast.FuncDecl)
	if !ok || declaration.Body != nil || function == nil ||
		function != function.Origin() || !profile.Valid() {
		return ExternalFunctionObligation{}, &api.InvariantError{
			Role:   api.RoleFileDeclaration,
			Reason: "external function obligation lacks a bodyless source owner",
		}
	}
	signature, ok := function.Type().(*types.Signature)
	if !ok {
		return ExternalFunctionObligation{}, &api.InvariantError{
			Role:   api.RoleFileDeclaration,
			Reason: "external function obligation lacks an exact signature",
		}
	}
	contract, err := environmentcontract.Describe(function)
	if err != nil {
		return ExternalFunctionObligation{}, err
	}
	position := site.Source.FileSet().Position(declaration.Pos())
	if contract.Identity() == "" || !position.IsValid() ||
		site.Source.ModulePath() == "" {
		return ExternalFunctionObligation{}, &api.InvariantError{
			Role:   api.RoleFileDeclaration,
			Reason: "external function obligation lacks canonical evidence",
		}
	}
	return ExternalFunctionObligation{
		identity:      contract.Identity(),
		function:      function,
		signature:     signature,
		modulePath:    site.Source.ModulePath(),
		moduleVersion: site.Source.ModuleVersion(),
		role:          api.RoleFileDeclaration,
		position:      position,
		buildProfile:  profile,
	}, nil
}

func (s *programSession) ResolveExternalFunction(
	function *types.Func,
) (api.ExternalFunctionTarget, bool, error) {
	if function == nil {
		return api.ExternalFunctionTarget{}, false, &ExternalFunctionBindingError{
			Reason: "function identity is nil",
		}
	}
	function = function.Origin()
	target, ok := s.externalFunctionBindings[function]
	return target, ok, nil
}

func (s *programSession) recordExternalFunctionObligation(
	site declarationSite,
	object types.Object,
) error {
	declaration, ok := site.Declaration.(*ast.FuncDecl)
	if !ok || declaration.Body != nil {
		return nil
	}
	function, ok := object.(*types.Func)
	if !ok {
		return &api.InvariantError{
			Role:   api.RoleFileDeclaration,
			Reason: "bodyless declaration is not a function",
		}
	}
	function = function.Origin()
	obligation, err := newExternalFunctionObligation(
		site,
		function,
		s.source.BuildProfile(),
	)
	if err != nil {
		return err
	}
	if _, linked := s.externalFunctionBindings[function]; linked {
		return nil
	}
	if existing, duplicate := s.externalFunctions[function]; duplicate {
		if existing.identity != obligation.identity ||
			existing.position != obligation.position {
			return &api.InvariantError{
				Role:   api.RoleFileDeclaration,
				Reason: "external function obligation identity is inconsistent",
			}
		}
		return nil
	}
	s.externalFunctions[function] = obligation
	return nil
}

func (s *programSession) externalFunctionObligations() []ExternalFunctionObligation {
	result := make(
		[]ExternalFunctionObligation,
		0,
		len(s.externalFunctions),
	)
	for _, obligation := range s.externalFunctions {
		result = append(result, obligation)
	}
	sort.Slice(result, func(left, right int) bool {
		if result[left].identity != result[right].identity {
			return result[left].identity < result[right].identity
		}
		return result[left].position.String() < result[right].position.String()
	})
	return slices.Clone(result)
}
