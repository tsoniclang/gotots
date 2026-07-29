package api

import (
	"go/ast"
	"go/types"
	"slices"
)

func GenericTypeParameter(sourceType types.Type) (*types.TypeParam, bool) {
	if sourceType == nil {
		return nil, false
	}
	parameter, ok := types.Unalias(sourceType).(*types.TypeParam)
	return parameter, ok
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

func ContainsGenericTypeParameter(sourceType types.Type) bool {
	return len(genericTypeParametersIn(sourceType)) != 0
}

func genericTypeParametersIn(
	sourceType types.Type,
) map[*types.TypeParam]struct{} {
	found := make(map[*types.TypeParam]struct{})
	collectGenericTypeParameters(
		sourceType,
		make(map[types.Type]bool),
		found,
	)
	return found
}

func collectGenericTypeParameters(
	sourceType types.Type,
	visiting map[types.Type]bool,
	found map[*types.TypeParam]struct{},
) {
	if sourceType == nil || visiting[sourceType] {
		return
	}
	sourceType = types.Unalias(sourceType)
	if parameter, ok := sourceType.(*types.TypeParam); ok {
		found[parameter] = struct{}{}
		return
	}
	visiting[sourceType] = true
	defer delete(visiting, sourceType)
	switch source := sourceType.(type) {
	case *types.Named:
		for index := range source.TypeArgs().Len() {
			collectGenericTypeParameters(
				source.TypeArgs().At(index),
				visiting,
				found,
			)
		}
	case *types.Pointer:
		collectGenericTypeParameters(source.Elem(), visiting, found)
	case *types.Slice:
		collectGenericTypeParameters(source.Elem(), visiting, found)
	case *types.Array:
		collectGenericTypeParameters(source.Elem(), visiting, found)
	case *types.Map:
		collectGenericTypeParameters(source.Key(), visiting, found)
		collectGenericTypeParameters(source.Elem(), visiting, found)
	case *types.Chan:
		collectGenericTypeParameters(source.Elem(), visiting, found)
	case *types.Struct:
		for index := range source.NumFields() {
			collectGenericTypeParameters(
				source.Field(index).Type(),
				visiting,
				found,
			)
		}
	case *types.Tuple:
		for index := range source.Len() {
			collectGenericTypeParameters(
				source.At(index).Type(),
				visiting,
				found,
			)
		}
	case *types.Signature:
		collectGenericTypeParameters(source.Params(), visiting, found)
		collectGenericTypeParameters(source.Results(), visiting, found)
	case *types.Interface:
		for index := range source.NumMethods() {
			collectGenericTypeParameters(
				source.Method(index).Type(),
				visiting,
				found,
			)
		}
		for index := range source.NumEmbeddeds() {
			collectGenericTypeParameters(
				source.EmbeddedType(index),
				visiting,
				found,
			)
		}
	case *types.Union:
		for index := range source.Len() {
			collectGenericTypeParameters(
				source.Term(index).Type(),
				visiting,
				found,
			)
		}
	}
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
	for index, operation := range operations {
		if !operation.Valid() ||
			operation.Owner() != owner ||
			operation.Consumer() != consumer ||
			index != 0 &&
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
) (NameReference, error) {
	owner, ownerOK := c.genericSourceOwner()
	if !ownerOK ||
		!c.genericConsumer.Valid() ||
		c.genericResolver == nil ||
		source == nil ||
		!origin.Valid() ||
		!validGenericOperationSignature(signature) {
		return NameReference{}, &ContextError{
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
		return NameReference{}, err
	}
	request, err := NewGenericOperationRequest(owner, contract)
	if err != nil {
		return NameReference{}, err
	}
	return NewNameReference(contract.TargetName(), request)
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
) (NameReference, error) {
	selection, err := SelectGenericOperation(operation)
	if err != nil {
		return NameReference{}, err
	}
	return c.genericOperation(source, selection, signature)
}

func (c Context) GenericConstraintMethod(
	source ast.Node,
	method *types.Func,
	signature *types.Signature,
) (NameReference, error) {
	selection, err := SelectGenericConstraintMethod(method)
	if err != nil {
		return NameReference{}, err
	}
	return c.genericOperation(source, selection, signature)
}

func (c Context) genericOperation(
	source ast.Node,
	selection GenericOperationSelection,
	signature *types.Signature,
) (NameReference, error) {
	owner, ownerOK := c.genericSourceOwner()
	switch {
	case !ownerOK:
		return NameReference{}, &ContextError{
			Reason: "generic operation has no source artifact owner",
		}
	case !c.genericConsumer.Valid():
		return NameReference{}, &ContextError{
			Reason: "generic operation has no target consumer",
		}
	case c.genericResolver == nil:
		return NameReference{}, &ContextError{
			Reason: "generic operation has no resolver",
		}
	case source == nil:
		return NameReference{}, &ContextError{
			Reason: "generic operation has no source construct",
		}
	case !selection.Valid():
		return NameReference{}, &ContextError{
			Reason: "generic operation selection is invalid",
		}
	case !validGenericOperationSignature(signature):
		return NameReference{}, &ContextError{
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
		return NameReference{}, err
	}
	request, err := NewGenericOperationRequest(owner, contract)
	if err != nil {
		return NameReference{}, err
	}
	return NewNameReference(contract.TargetName(), request)
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
