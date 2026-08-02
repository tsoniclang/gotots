package api

import (
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
