package api

import (
	"go/ast"
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

type GenericCallableResolver interface {
	ResolveGenericOperationSet(
		types.Object,
		GenericOperationConsumer,
	) (GenericOperationSet, bool, error)
	ResolveGenericOperation(
		types.Object,
		GenericOperationConsumer,
		GenericOperationSelection,
		*types.Signature,
	) (*GenericOperationContract, error)
	ResolveGenericCallableProfile(
		*types.Func,
		GenericCallableProfileSelection,
	) (*GenericCallableProfile, error)
	ResolveGenericRepresentationProfile(
		types.Object,
	) (GenericRepresentationProfile, bool, error)
}

func (c Context) WithGenericCallableResolver(
	resolver GenericCallableResolver,
) Context {
	if resolver == nil {
		panic("generic callable resolver is nil")
	}
	c.genericResolver = resolver
	return c
}

func (c Context) ProjectGenericOperation(
	source ast.Node,
	origin *GenericOperationContract,
	signature *types.Signature,
) (GenericOperationReference, error) {
	owner, ownerOK := c.genericSourceOwner()
	if !ownerOK ||
		!c.genericConsumer.Valid() ||
		c.genericResolver == nil ||
		source == nil ||
		!origin.Valid() ||
		!validGenericOperationSignature(signature) {
		return GenericOperationReference{}, &ContextError{
			Reason: "projected generic operation is unavailable",
		}
	}
	contract, err := c.genericResolver.ResolveGenericOperation(
		owner,
		c.genericConsumer,
		origin.Selection(),
		signature,
	)
	if err != nil {
		return GenericOperationReference{}, err
	}
	request, err := NewGenericOperationRequest(owner, contract)
	if err != nil {
		return GenericOperationReference{}, err
	}
	return NewGenericOperationReference(
		contract,
		contract.TargetName(),
		request,
	)
}

func (c Context) WithGenericParameters(
	owner types.Object,
	names map[*types.TypeParam]string,
) (Context, error) {
	owner = GenericDeclarationOrigin(owner)
	sourceOwner, sourceOwned := c.artifactOwner.Source()
	if owner == nil || !sourceOwned || sourceOwner != owner {
		return Context{}, &ContextError{
			Reason: "generic parameter owner differs from source artifact owner",
		}
	}
	switch owner.(type) {
	case *types.Func:
		c.genericConsumer = GenericFunctionOperationConsumer()
	case *types.TypeName:
		c.genericConsumer = GenericOperationConsumerInvalid
	}
	c.genericParameters = make(map[*types.TypeParam]string, len(names))
	c.genericParameterOwner = owner
	for parameter, name := range names {
		if parameter == nil ||
			name == "" ||
			!genericParameterBelongsTo(owner, parameter) {
			return Context{}, &ContextError{
				Reason: "generic parameter binding is invalid",
			}
		}
		c.genericParameters[parameter] = name
	}
	return c, nil
}

func (c Context) WithEnvironmentGenericParameters(
	owner types.Object,
	names map[*types.TypeParam]string,
) (Context, error) {
	owner = GenericDeclarationOrigin(owner)
	if owner == nil ||
		owner.Pkg() == nil ||
		owner.Pkg() != c.typesPackage ||
		c.artifactOwner.Valid() {
		return Context{}, &ContextError{
			Reason: "environment generic parameter owner is invalid",
		}
	}
	c.genericParameters = make(map[*types.TypeParam]string, len(names))
	c.genericParameterOwner = owner
	for parameter, name := range names {
		if parameter == nil ||
			name == "" ||
			!genericParameterBelongsTo(owner, parameter) {
			return Context{}, &ContextError{
				Reason: "environment generic parameter binding is invalid",
			}
		}
		c.genericParameters[parameter] = name
	}
	return c, nil
}

func (c Context) WithGenericNamedStructOperation(
	operation NamedStructOperation,
) Context {
	owner, ownerOK := c.genericSourceOwner()
	if !ownerOK {
		panic("generic named-struct operation has no source owner")
	}
	if _, ok := owner.(*types.TypeName); !ok {
		panic("generic named-struct operation has no type owner")
	}
	consumer, err := GenericNamedStructOperationConsumer(operation)
	if err != nil {
		panic(err)
	}
	c.genericConsumer = consumer
	return c
}

func (c Context) GenericOperation(
	source ast.Node,
	operation GenericOperation,
	signature *types.Signature,
) (GenericOperationReference, error) {
	selection, err := SelectGenericOperation(operation)
	if err != nil {
		return GenericOperationReference{}, err
	}
	return c.genericOperation(source, selection, signature)
}

func (c Context) GenericConstraintMethod(
	source ast.Node,
	method *types.Func,
	signature *types.Signature,
) (GenericOperationReference, error) {
	selection, err := SelectGenericConstraintMethod(method)
	if err != nil {
		return GenericOperationReference{}, err
	}
	return c.genericOperation(source, selection, signature)
}

func (c Context) genericOperation(
	source ast.Node,
	selection GenericOperationSelection,
	signature *types.Signature,
) (GenericOperationReference, error) {
	owner, ownerOK := c.genericSourceOwner()
	switch {
	case !ownerOK:
		return GenericOperationReference{}, &ContextError{
			Reason: "generic operation has no source artifact owner",
		}
	case !c.genericConsumer.Valid():
		return GenericOperationReference{}, &ContextError{
			Reason: "generic operation has no target consumer",
		}
	case c.genericResolver == nil:
		return GenericOperationReference{}, &ContextError{
			Reason: "generic operation has no resolver",
		}
	case source == nil &&
		selection.Operation() == GenericOperationConstraintMethod:
		return GenericOperationReference{}, &ContextError{
			Reason: "generic constraint-method operation has no source construct",
		}
	case !selection.Valid():
		return GenericOperationReference{}, &ContextError{
			Reason: "generic operation selection is invalid",
		}
	case !validGenericOperationSignature(signature):
		return GenericOperationReference{}, &ContextError{
			Reason: "generic operation signature is invalid",
		}
	}
	contract, err := c.genericResolver.ResolveGenericOperation(
		owner,
		c.genericConsumer,
		selection,
		signature,
	)
	if err != nil {
		return GenericOperationReference{}, err
	}
	request, err := NewGenericOperationRequest(owner, contract)
	if err != nil {
		return GenericOperationReference{}, err
	}
	return NewGenericOperationReference(
		contract,
		contract.TargetName(),
		request,
	)
}

func (c Context) ResolveGenericCallable(
	function *types.Func,
) (GenericOperationSet, bool, error) {
	if c.genericResolver == nil {
		return GenericOperationSet{}, false, &ContextError{
			Reason: "generic callable resolver is unavailable",
		}
	}
	if function == nil {
		return GenericOperationSet{}, false, nil
	}
	return c.genericResolver.ResolveGenericOperationSet(
		function.Origin(),
		GenericFunctionOperationConsumer(),
	)
}

func (c Context) ResolveGenericCallableProfile(
	function *types.Func,
	selection GenericCallableProfileSelection,
) (*GenericCallableProfile, error) {
	if c.genericResolver == nil {
		return nil, &ContextError{
			Reason: "generic callable profile resolver is unavailable",
		}
	}
	if function == nil || !selection.Valid() || !selection.Cooperative() {
		return nil, &ContextError{
			Reason: "generic callable profile selection is invalid",
		}
	}
	return c.genericResolver.ResolveGenericCallableProfile(
		function.Origin(),
		selection,
	)
}

func (c Context) ResolveGenericNamedStructOperation(
	owner *types.TypeName,
	operation NamedStructOperation,
) (GenericOperationSet, bool, error) {
	if c.genericResolver == nil {
		return GenericOperationSet{}, false, &ContextError{
			Reason: "generic named-struct operation resolver is unavailable",
		}
	}
	consumer, err := GenericNamedStructOperationConsumer(operation)
	if err != nil {
		return GenericOperationSet{}, false, err
	}
	return c.genericResolver.ResolveGenericOperationSet(owner, consumer)
}

func (c Context) GenericParameterName(
	parameter *types.TypeParam,
) (string, bool) {
	name, ok := c.genericParameters[parameter]
	return name, ok
}

func genericParameterBelongsTo(
	owner types.Object,
	parameter *types.TypeParam,
) bool {
	for _, selected := range GenericDeclarationParameters(owner) {
		if selected == parameter {
			return true
		}
	}
	return false
}

func (c Context) genericSourceOwner() (types.Object, bool) {
	source, ok := c.artifactOwner.Source()
	if !ok {
		return nil, false
	}
	owner := GenericDeclarationOrigin(source)
	return owner, owner != nil && owner == source
}
