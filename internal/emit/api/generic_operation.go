package api

import (
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
	owner      *types.Func
	key        string
	targetName string
	selection  GenericOperationSelection
	signature  *types.Signature
}

func NewGenericOperationContract(
	owner *types.Func,
	key string,
	targetName string,
	selection GenericOperationSelection,
	signature *types.Signature,
) (*GenericOperationContract, error) {
	if owner == nil ||
		owner.Origin() != owner ||
		len(genericTypeParameters(owner)) == 0 ||
		key == "" ||
		targetName == "" ||
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
		selection:  selection,
		signature:  signature,
	}, nil
}

func (c *GenericOperationContract) Owner() *types.Func {
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
		c.owner.Origin() == c.owner &&
		len(genericTypeParameters(c.owner)) != 0 &&
		c.key != "" &&
		c.targetName != "" &&
		c.selection.Valid() &&
		validGenericOperationSignature(c.signature) &&
		genericOperationParametersBelongTo(c.owner, c.signature)
}

func validGenericOperationSignature(signature *types.Signature) bool {
	return signature != nil &&
		signature.Recv() == nil &&
		signature.RecvTypeParams().Len() == 0 &&
		signature.TypeParams().Len() == 0
}

func genericOperationParametersBelongTo(
	owner *types.Func,
	signature *types.Signature,
) bool {
	owned := make(map[*types.TypeParam]struct{})
	for _, parameter := range genericTypeParameters(owner) {
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

type GenericCallable struct {
	owner      *types.Func
	parameters []*types.TypeParam
	operations []*GenericOperationContract
}

func NewGenericCallable(
	owner *types.Func,
	parameters []*types.TypeParam,
	operations []*GenericOperationContract,
) (GenericCallable, error) {
	if owner == nil || owner.Origin() != owner || len(parameters) == 0 {
		return GenericCallable{}, &InvariantError{
			Role:   RoleCallCallee,
			Reason: "generic callable identity is invalid",
		}
	}
	parameters = slices.Clone(parameters)
	expectedParameters := genericTypeParameters(owner)
	if len(parameters) != len(expectedParameters) {
		return GenericCallable{}, &InvariantError{
			Role:   RoleCallCallee,
			Reason: "generic callable parameters do not match their owner",
		}
	}
	for index, parameter := range parameters {
		if parameter == nil || parameter != expectedParameters[index] {
			return GenericCallable{}, &InvariantError{
				Role:   RoleCallCallee,
				Reason: "generic callable parameter identity is inconsistent",
			}
		}
	}
	operations = slices.Clone(operations)
	for index, operation := range operations {
		if !operation.Valid() ||
			operation.Owner() != owner ||
			index != 0 &&
				operations[index-1].Key() >= operation.Key() {
			return GenericCallable{}, &InvariantError{
				Role:   RoleCallCallee,
				Reason: "generic callable operations are not canonical",
			}
		}
	}
	return GenericCallable{
		owner:      owner,
		parameters: parameters,
		operations: operations,
	}, nil
}

func (c GenericCallable) Owner() *types.Func {
	return c.owner
}

func (c GenericCallable) Parameters() []*types.TypeParam {
	return slices.Clone(c.parameters)
}

func (c GenericCallable) Operations() []*GenericOperationContract {
	return slices.Clone(c.operations)
}

type GenericCallableResolver interface {
	ResolveGenericCallable(*types.Func) (GenericCallable, bool, error)
	ResolveGenericOperation(
		*types.Func,
		GenericOperationSelection,
		*types.Signature,
	) (*GenericOperationContract, error)
}
