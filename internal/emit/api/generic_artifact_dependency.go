package api

import (
	"fmt"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	"go/types"
	"slices"
)

type GenericOperationConsumer uint8

const (
	GenericOperationConsumerInvalid GenericOperationConsumer = iota
	GenericOperationConsumerFunction
	GenericOperationConsumerNamedStructZero
	GenericOperationConsumerNamedStructCopy
	GenericOperationConsumerNamedStructEqual
	GenericOperationConsumerNamedStructHash
	GenericOperationConsumerNamedStructConvert
	GenericOperationConsumerNamedStructStorage
	GenericOperationConsumerNamedStructAssign
)

func GenericFunctionOperationConsumer() GenericOperationConsumer {
	return GenericOperationConsumerFunction
}

func GenericNamedStructOperationConsumer(
	operation NamedStructOperation,
) (GenericOperationConsumer, error) {
	var consumer GenericOperationConsumer
	switch operation {
	case NamedStructOperationZero:
		consumer = GenericOperationConsumerNamedStructZero
	case NamedStructOperationCopy:
		consumer = GenericOperationConsumerNamedStructCopy
	case NamedStructOperationEqual:
		consumer = GenericOperationConsumerNamedStructEqual
	case NamedStructOperationHash:
		consumer = GenericOperationConsumerNamedStructHash
	case NamedStructOperationConvert:
		consumer = GenericOperationConsumerNamedStructConvert
	case NamedStructOperationStorage:
		consumer = GenericOperationConsumerNamedStructStorage
	case NamedStructOperationAssign:
		consumer = GenericOperationConsumerNamedStructAssign
	default:
		return GenericOperationConsumerInvalid, &InvariantError{
			Role:   RoleFileDeclaration,
			Reason: "generic named-struct operation consumer is invalid",
		}
	}
	return consumer, nil
}

func (c GenericOperationConsumer) Valid() bool {
	return c >= GenericOperationConsumerFunction &&
		c <= GenericOperationConsumerNamedStructAssign
}

func (c GenericOperationConsumer) NamedStructOperation() (
	NamedStructOperation,
	bool,
) {
	switch c {
	case GenericOperationConsumerNamedStructZero:
		return NamedStructOperationZero, true
	case GenericOperationConsumerNamedStructCopy:
		return NamedStructOperationCopy, true
	case GenericOperationConsumerNamedStructEqual:
		return NamedStructOperationEqual, true
	case GenericOperationConsumerNamedStructHash:
		return NamedStructOperationHash, true
	case GenericOperationConsumerNamedStructConvert:
		return NamedStructOperationConvert, true
	case GenericOperationConsumerNamedStructStorage:
		return NamedStructOperationStorage, true
	case GenericOperationConsumerNamedStructAssign:
		return NamedStructOperationAssign, true
	default:
		return NamedStructOperationInvalid, false
	}
}

func (c GenericOperationConsumer) Identity() string {
	if c == GenericOperationConsumerFunction {
		return "function"
	}
	if operation, ok := c.NamedStructOperation(); ok {
		return "named-struct-" + operation.String()
	}
	return ""
}

type GenericOperationContract struct {
	owner      types.Object
	key        string
	targetName string
	consumer   GenericOperationConsumer
	selection  GenericOperationSelection
	signature  *types.Signature
}

func NewGenericOperationContract(
	owner types.Object,
	key string,
	targetName string,
	consumer GenericOperationConsumer,
	selection GenericOperationSelection,
	signature *types.Signature,
) (*GenericOperationContract, error) {
	owner = GenericDeclarationOrigin(owner)
	if owner == nil ||
		len(GenericDeclarationParameters(owner)) == 0 ||
		key == "" ||
		targetName == "" ||
		!consumer.Valid() ||
		!genericConsumerMatchesOwner(owner, consumer) ||
		!selection.Valid() ||
		!validGenericOperationSignature(signature) ||
		!genericOperationParametersBelongTo(owner, signature) {
		return nil, &InvariantError{
			Role:   RoleCallCallee,
			Reason: "generic operation contract is invalid",
		}
	}
	return &GenericOperationContract{
		owner:      owner,
		key:        key,
		targetName: targetName,
		consumer:   consumer,
		selection:  selection,
		signature:  signature,
	}, nil
}

func (c *GenericOperationContract) Owner() types.Object {
	if c == nil {
		return nil
	}
	return c.owner
}

func (c *GenericOperationContract) Key() string {
	if c == nil {
		return ""
	}
	return c.key
}

func (c *GenericOperationContract) TargetName() string {
	if c == nil {
		return ""
	}
	return c.targetName
}

func (c *GenericOperationContract) Operation() GenericOperation {
	if c == nil {
		return GenericOperationInvalid
	}
	return c.selection.Operation()
}

func (c *GenericOperationContract) Consumer() GenericOperationConsumer {
	if c == nil {
		return GenericOperationConsumerInvalid
	}
	return c.consumer
}

func (c *GenericOperationContract) Selection() GenericOperationSelection {
	if c == nil {
		return GenericOperationSelection{}
	}
	return c.selection
}

func (c *GenericOperationContract) Signature() *types.Signature {
	if c == nil {
		return nil
	}
	return c.signature
}

func (c *GenericOperationContract) Valid() bool {
	return c != nil &&
		c.owner != nil &&
		GenericDeclarationOrigin(c.owner) == c.owner &&
		len(GenericDeclarationParameters(c.owner)) != 0 &&
		c.key != "" &&
		c.targetName != "" &&
		c.consumer.Valid() &&
		genericConsumerMatchesOwner(c.owner, c.consumer) &&
		c.selection.Valid() &&
		validGenericOperationSignature(c.signature) &&
		genericOperationParametersBelongTo(c.owner, c.signature)
}

func genericConsumerMatchesOwner(
	owner types.Object,
	consumer GenericOperationConsumer,
) bool {
	switch owner.(type) {
	case *types.Func:
		return consumer == GenericOperationConsumerFunction
	case *types.TypeName:
		_, ok := consumer.NamedStructOperation()
		return ok
	default:
		return false
	}
}

func validGenericOperationSignature(signature *types.Signature) bool {
	return signature != nil &&
		signature.Recv() == nil &&
		signature.RecvTypeParams().Len() == 0 &&
		signature.TypeParams().Len() == 0
}

func genericOperationParametersBelongTo(
	owner types.Object,
	signature *types.Signature,
) bool {
	owned := make(map[*types.TypeParam]struct{})
	for _, parameter := range GenericDeclarationParameters(owner) {
		owned[parameter] = struct{}{}
	}
	for parameter := range genericTypeParametersIn(signature) {
		if _, ok := owned[parameter]; !ok {
			return false
		}
	}
	return true
}

type GenericOperationSet struct {
	owner      types.Object
	consumer   GenericOperationConsumer
	parameters []*types.TypeParam
	operations []*GenericOperationContract
}

func NewGenericOperationSet(
	owner types.Object,
	consumer GenericOperationConsumer,
	operations []*GenericOperationContract,
) (GenericOperationSet, error) {
	return newGenericOperationSet(owner, consumer, operations, true)
}

func NewGenericOperationABISet(
	owner types.Object,
	consumer GenericOperationConsumer,
	operations []*GenericOperationContract,
) (GenericOperationSet, error) {
	return newGenericOperationSet(owner, consumer, operations, false)
}

func newGenericOperationSet(
	owner types.Object,
	consumer GenericOperationConsumer,
	operations []*GenericOperationContract,
	requireCanonicalIdentityOrder bool,
) (GenericOperationSet, error) {
	owner = GenericDeclarationOrigin(owner)
	parameters := GenericDeclarationParameters(owner)
	if owner == nil ||
		!consumer.Valid() ||
		!genericConsumerMatchesOwner(owner, consumer) ||
		len(parameters) == 0 {
		return GenericOperationSet{}, &InvariantError{
			Role:   RoleCallCallee,
			Reason: "generic operation-set identity is invalid",
		}
	}
	parameters = slices.Clone(parameters)
	operations = slices.Clone(operations)
	seen := make(map[string]struct{}, len(operations))
	for index, operation := range operations {
		if !operation.Valid() ||
			operation.Owner() != owner ||
			operation.Consumer() != consumer {
			return GenericOperationSet{}, &InvariantError{
				Role:   RoleCallCallee,
				Reason: "generic operation set is invalid",
			}
		}
		if _, duplicate := seen[operation.Key()]; duplicate {
			return GenericOperationSet{}, &InvariantError{
				Role:   RoleCallCallee,
				Reason: "generic operation set contains a duplicate",
			}
		}
		seen[operation.Key()] = struct{}{}
		if requireCanonicalIdentityOrder && index != 0 &&
			operations[index-1].Key() >= operation.Key() {
			return GenericOperationSet{}, &InvariantError{
				Role:   RoleCallCallee,
				Reason: "generic operation set is not canonical",
			}
		}
	}
	return GenericOperationSet{
		owner:      owner,
		consumer:   consumer,
		parameters: parameters,
		operations: operations,
	}, nil
}

func (s GenericOperationSet) Owner() types.Object {
	return s.owner
}

func (s GenericOperationSet) Consumer() GenericOperationConsumer {
	return s.consumer
}

func (s GenericOperationSet) Parameters() []*types.TypeParam {
	return slices.Clone(s.parameters)
}

func (s GenericOperationSet) Operations() []*GenericOperationContract {
	return slices.Clone(s.operations)
}

type ArtifactFacet uint8

const (
	ArtifactFacetInvalid             ArtifactFacet = 0
	ArtifactFacetCallableSignature   ArtifactFacet = 1
	ArtifactFacetConstructorSurface  ArtifactFacet = 2
	ArtifactFacetInstanceTypeSurface ArtifactFacet = 3
	ArtifactFacetStaticSurface       ArtifactFacet = 4
	ArtifactFacetValueSurface        ArtifactFacet = 5
	ArtifactFacetImplementation      ArtifactFacet = 6
	ArtifactFacetExportSurface       ArtifactFacet = 7
	ArtifactFacetCallableRecovery    ArtifactFacet = 8
	ArtifactFacetCount                             = 9
)

func (f ArtifactFacet) Valid() bool {
	return f >= ArtifactFacetCallableSignature &&
		f <= ArtifactFacetCallableRecovery
}

func (f ArtifactFacet) String() string {
	switch f {
	case ArtifactFacetCallableSignature:
		return "callable-signature"
	case ArtifactFacetConstructorSurface:
		return "constructor-surface"
	case ArtifactFacetInstanceTypeSurface:
		return "instance-type-surface"
	case ArtifactFacetStaticSurface:
		return "static-surface"
	case ArtifactFacetValueSurface:
		return "value-surface"
	case ArtifactFacetExportSurface:
		return "export-surface"
	case ArtifactFacetImplementation:
		return "implementation"
	case ArtifactFacetCallableRecovery:
		return "callable-recovery"
	default:
		return fmt.Sprintf("artifact-facet(%d)", f)
	}
}

type ArtifactDependency struct {
	provider ArtifactOwner
	facet    ArtifactFacet
}

func NewArtifactDependency(
	provider ArtifactOwner,
	facet ArtifactFacet,
) (ArtifactDependency, error) {
	if !provider.Valid() {
		return ArtifactDependency{},
			&RootRequestError{Reason: "artifact dependency provider is invalid"}
	}
	if !facet.Valid() {
		return ArtifactDependency{},
			&RootRequestError{Reason: "artifact dependency facet is invalid"}
	}
	return ArtifactDependency{provider: provider, facet: facet}, nil
}

func (d ArtifactDependency) Valid() bool {
	return d.provider.Valid() && d.facet.Valid()
}

func (d ArtifactDependency) Provider() ArtifactOwner {
	return d.provider
}

func (d ArtifactDependency) Facet() ArtifactFacet {
	return d.facet
}

func NewArtifactDependencyRequest(
	provider types.Object,
	facet ArtifactFacet,
) (RootRequest, error) {
	owner, err := SourceArtifactOwner(provider)
	if err != nil {
		return RootRequest{}, err
	}
	return newArtifactDependencyRequest(owner, facet)
}

func NewGeneratedArtifactDependencyRequest(
	provider *GeneratedArtifact,
	facet ArtifactFacet,
) (RootRequest, error) {
	owner, err := GeneratedArtifactOwner(provider)
	if err != nil {
		return RootRequest{}, err
	}
	return newArtifactDependencyRequest(owner, facet)
}

func NewOwnedArtifactDependencyRequest(
	provider ArtifactOwner,
	facet ArtifactFacet,
) (RootRequest, error) {
	return newArtifactDependencyRequest(provider, facet)
}

func newArtifactDependencyRequest(
	provider ArtifactOwner,
	facet ArtifactFacet,
) (RootRequest, error) {
	dependency, err := NewArtifactDependency(provider, facet)
	if err != nil {
		return RootRequest{}, err
	}
	return RootRequest{payload: &rootRequestPayload{
		owner: RootRequestOwner{
			kind:               RootRequestArtifactDependency,
			artifactDependency: dependency,
		},
	}}, nil
}

type NameReference struct {
	qualifier string
	name      string
	requests  []RootRequest
	provider  bool
}

func NewNameReference(name string, requests ...RootRequest) (NameReference, error) {
	if name == "" {
		return NameReference{}, &NameError{Reason: "reference name is empty"}
	}
	return NameReference{name: name, requests: slices.Clone(requests)}, nil
}

func NewQualifiedNameReference(
	qualifier string,
	name string,
	requests ...RootRequest,
) (NameReference, error) {
	switch {
	case qualifier == "":
		return NameReference{}, &NameError{
			Name:   name,
			Reason: "reference qualifier is empty",
		}
	case name == "":
		return NameReference{}, &NameError{
			Reason: "reference name is empty",
		}
	}
	return NameReference{
		qualifier: qualifier,
		name:      name,
		requests:  slices.Clone(requests),
	}, nil
}

func NewProviderQualifiedNameReference(
	qualifier string,
	name string,
	requests ...RootRequest,
) (NameReference, error) {
	reference, err := NewQualifiedNameReference(
		qualifier,
		name,
		requests...,
	)
	if err != nil {
		return NameReference{}, err
	}
	reference.provider = true
	return reference, nil
}

func (r NameReference) Name() string {
	return r.name
}

func (r NameReference) Qualifier() (string, bool) {
	return r.qualifier, r.qualifier != ""
}

func (r NameReference) Requests() []RootRequest {
	return slices.Clone(r.requests)
}

func (r NameReference) ProviderBoundary() bool {
	return r.provider
}

func (r NameReference) WithRequests(
	requests ...RootRequest,
) (NameReference, error) {
	if r.name == "" {
		return NameReference{}, &NameError{Reason: "reference name is empty"}
	}
	r.requests = slices.Clone(requests)
	return r, nil
}

func (r NameReference) Expression(factory tsgo.Factory) tsgo.Expression {
	if r.qualifier == "" {
		return factory.Identifier(r.name)
	}
	return factory.PropertyAccessExpression(
		factory.Identifier(r.qualifier),
		nil,
		factory.Identifier(r.name),
		tsgo.NodeFlagsNone,
	)
}

func (r NameReference) EntityName(factory tsgo.Factory) tsgo.EntityName {
	if r.qualifier == "" {
		return factory.Identifier(r.name)
	}
	return factory.QualifiedName(
		factory.Identifier(r.qualifier),
		factory.Identifier(r.name),
	)
}

func (r NameReference) MemberExpression(
	factory tsgo.Factory,
	member string,
) (tsgo.PropertyAccessExpression, error) {
	if member == "" {
		return nil, &NameError{
			Name:   r.name,
			Reason: "reference member is empty",
		}
	}
	return factory.PropertyAccessExpression(
		r.Expression(factory),
		nil,
		factory.Identifier(member),
		tsgo.NodeFlagsNone,
	), nil
}

func operationOwnerFunction(
	operation *GenericOperationContract,
) (*types.Func, bool) {
	if operation == nil {
		return nil, false
	}
	function, ok := operation.Owner().(*types.Func)
	return function, ok && function != nil && function.Origin() == function
}
