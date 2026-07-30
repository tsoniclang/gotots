package api

import (
	"fmt"
	"go/token"
	"go/types"
)

func GenericTypeParameter(sourceType types.Type) (*types.TypeParam, bool) {
	if sourceType == nil {
		return nil, false
	}
	parameter, ok := types.Unalias(sourceType).(*types.TypeParam)
	return parameter, ok
}

type GenericOperation uint8

const (
	GenericOperationInvalid GenericOperation = iota
	GenericOperationZero
	GenericOperationCopy
	GenericOperationEqual
	GenericOperationHash
	GenericOperationUnaryPlus
	GenericOperationUnaryMinus
	GenericOperationUnaryNot
	GenericOperationUnaryXor
	GenericOperationBinaryAdd
	GenericOperationBinarySubtract
	GenericOperationBinaryMultiply
	GenericOperationBinaryDivide
	GenericOperationBinaryRemainder
	GenericOperationBinaryAnd
	GenericOperationBinaryOr
	GenericOperationBinaryXor
	GenericOperationBinaryAndNot
	GenericOperationBinaryShiftLeft
	GenericOperationBinaryShiftRight
	GenericOperationBinaryEqual
	GenericOperationBinaryNotEqual
	GenericOperationBinaryLess
	GenericOperationBinaryLessEqual
	GenericOperationBinaryGreater
	GenericOperationBinaryGreaterEqual
	GenericOperationLength
	GenericOperationCapacity
	GenericOperationConvert
	GenericOperationIndex
	GenericOperationConstraintMethod
	GenericOperationMapConstruct
	GenericOperationInterfaceAdapt
	GenericOperationInterfaceAssert
	GenericOperationInterfaceAssertOK
	GenericOperationClear
	GenericOperationNilEqual
)

var genericOperationIdentifiers = [...]string{
	GenericOperationInvalid:            "",
	GenericOperationZero:               "zero",
	GenericOperationCopy:               "copy",
	GenericOperationEqual:              "equal",
	GenericOperationHash:               "hash",
	GenericOperationUnaryPlus:          "unary_plus",
	GenericOperationUnaryMinus:         "unary_minus",
	GenericOperationUnaryNot:           "unary_not",
	GenericOperationUnaryXor:           "unary_xor",
	GenericOperationBinaryAdd:          "binary_add",
	GenericOperationBinarySubtract:     "binary_subtract",
	GenericOperationBinaryMultiply:     "binary_multiply",
	GenericOperationBinaryDivide:       "binary_divide",
	GenericOperationBinaryRemainder:    "binary_remainder",
	GenericOperationBinaryAnd:          "binary_and",
	GenericOperationBinaryOr:           "binary_or",
	GenericOperationBinaryXor:          "binary_xor",
	GenericOperationBinaryAndNot:       "binary_and_not",
	GenericOperationBinaryShiftLeft:    "binary_shift_left",
	GenericOperationBinaryShiftRight:   "binary_shift_right",
	GenericOperationBinaryEqual:        "binary_equal",
	GenericOperationBinaryNotEqual:     "binary_not_equal",
	GenericOperationBinaryLess:         "binary_less",
	GenericOperationBinaryLessEqual:    "binary_less_equal",
	GenericOperationBinaryGreater:      "binary_greater",
	GenericOperationBinaryGreaterEqual: "binary_greater_equal",
	GenericOperationLength:             "length",
	GenericOperationCapacity:           "capacity",
	GenericOperationConvert:            "convert",
	GenericOperationIndex:              "index",
	GenericOperationConstraintMethod:   "constraint_method",
	GenericOperationMapConstruct:       "map_construct",
	GenericOperationInterfaceAdapt:     "interface_adapt",
	GenericOperationInterfaceAssert:    "interface_assert",
	GenericOperationInterfaceAssertOK:  "interface_assert_ok",
	GenericOperationClear:              "clear",
	GenericOperationNilEqual:           "nil_equal",
}

func (o GenericOperation) Valid() bool {
	return o >= GenericOperationZero &&
		o <= GenericOperationNilEqual
}

func (o GenericOperation) Identifier() string {
	if !o.Valid() {
		return ""
	}
	return genericOperationIdentifiers[o]
}

func (o GenericOperation) String() string {
	switch o {
	case GenericOperationZero:
		return "zero"
	case GenericOperationCopy:
		return "copy"
	case GenericOperationEqual:
		return "equal"
	case GenericOperationHash:
		return "hash"
	case GenericOperationUnaryPlus:
		return "unary-plus"
	case GenericOperationUnaryMinus:
		return "unary-minus"
	case GenericOperationUnaryNot:
		return "unary-not"
	case GenericOperationUnaryXor:
		return "unary-xor"
	case GenericOperationLength:
		return "length"
	case GenericOperationCapacity:
		return "capacity"
	case GenericOperationConvert:
		return "convert"
	case GenericOperationIndex:
		return "index"
	case GenericOperationConstraintMethod:
		return "constraint-method"
	case GenericOperationMapConstruct:
		return "map-construct"
	case GenericOperationInterfaceAdapt:
		return "interface-adapt"
	case GenericOperationInterfaceAssert:
		return "interface-assert"
	case GenericOperationInterfaceAssertOK:
		return "interface-assert-ok"
	case GenericOperationClear:
		return "clear"
	case GenericOperationNilEqual:
		return "nil-equal"
	default:
		if source, ok := o.BinaryToken(); ok {
			return "binary-" + source.String()
		}
		return fmt.Sprintf("generic-operation(%d)", o)
	}
}

func UnaryGenericOperation(source token.Token) (GenericOperation, bool) {
	switch source {
	case token.ADD:
		return GenericOperationUnaryPlus, true
	case token.SUB:
		return GenericOperationUnaryMinus, true
	case token.NOT:
		return GenericOperationUnaryNot, true
	case token.XOR:
		return GenericOperationUnaryXor, true
	default:
		return GenericOperationInvalid, false
	}
}

func BinaryGenericOperation(source token.Token) (GenericOperation, bool) {
	switch source {
	case token.ADD:
		return GenericOperationBinaryAdd, true
	case token.SUB:
		return GenericOperationBinarySubtract, true
	case token.MUL:
		return GenericOperationBinaryMultiply, true
	case token.QUO:
		return GenericOperationBinaryDivide, true
	case token.REM:
		return GenericOperationBinaryRemainder, true
	case token.AND:
		return GenericOperationBinaryAnd, true
	case token.OR:
		return GenericOperationBinaryOr, true
	case token.XOR:
		return GenericOperationBinaryXor, true
	case token.AND_NOT:
		return GenericOperationBinaryAndNot, true
	case token.SHL:
		return GenericOperationBinaryShiftLeft, true
	case token.SHR:
		return GenericOperationBinaryShiftRight, true
	case token.EQL:
		return GenericOperationBinaryEqual, true
	case token.NEQ:
		return GenericOperationBinaryNotEqual, true
	case token.LSS:
		return GenericOperationBinaryLess, true
	case token.LEQ:
		return GenericOperationBinaryLessEqual, true
	case token.GTR:
		return GenericOperationBinaryGreater, true
	case token.GEQ:
		return GenericOperationBinaryGreaterEqual, true
	default:
		return GenericOperationInvalid, false
	}
}

func (o GenericOperation) BinaryToken() (token.Token, bool) {
	for _, candidate := range []token.Token{
		token.ADD,
		token.SUB,
		token.MUL,
		token.QUO,
		token.REM,
		token.AND,
		token.OR,
		token.XOR,
		token.AND_NOT,
		token.SHL,
		token.SHR,
		token.EQL,
		token.NEQ,
		token.LSS,
		token.LEQ,
		token.GTR,
		token.GEQ,
	} {
		if selected, _ := BinaryGenericOperation(candidate); selected == o {
			return candidate, true
		}
	}
	return token.ILLEGAL, false
}

type GenericOperationSelection struct {
	operation GenericOperation
	method    *types.Func
}

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
		c <= GenericOperationConsumerNamedStructStorage
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

func SelectGenericOperation(
	operation GenericOperation,
) (GenericOperationSelection, error) {
	if !operation.Valid() ||
		operation == GenericOperationConstraintMethod {
		return GenericOperationSelection{}, &InvariantError{
			Role:   RoleCallCallee,
			Reason: "generic operation selection is invalid",
		}
	}
	return GenericOperationSelection{operation: operation}, nil
}

func SelectGenericConstraintMethod(
	method *types.Func,
) (GenericOperationSelection, error) {
	if method == nil {
		return GenericOperationSelection{}, &InvariantError{
			Role:   RoleCallCallee,
			Reason: "generic constraint method is nil",
		}
	}
	method = method.Origin()
	if _, ok := method.Type().(*types.Signature); !ok {
		return GenericOperationSelection{}, &InvariantError{
			Role:   RoleCallCallee,
			Reason: "generic constraint method has no signature",
		}
	}
	return GenericOperationSelection{
		operation: GenericOperationConstraintMethod,
		method:    method,
	}, nil
}

func (s GenericOperationSelection) Valid() bool {
	return s.operation.Valid() &&
		((s.operation == GenericOperationConstraintMethod) ==
			(s.method != nil))
}

func (s GenericOperationSelection) Operation() GenericOperation {
	if !s.Valid() {
		return GenericOperationInvalid
	}
	return s.operation
}

func (s GenericOperationSelection) Method() (*types.Func, bool) {
	return s.method,
		s.Valid() &&
			s.operation == GenericOperationConstraintMethod
}

func (s GenericOperationSelection) IdentityPrefix() (string, error) {
	if !s.Valid() {
		return "", &InvariantError{
			Role:   RoleCallCallee,
			Reason: "generic operation selection is invalid",
		}
	}
	if s.operation != GenericOperationConstraintMethod {
		return s.operation.Identifier(), nil
	}
	identity := s.operation.Identifier() + "|"
	if s.method.Exported() {
		return identity + "exported|" + s.method.Name(), nil
	}
	if s.method.Pkg() == nil {
		return "", &InvariantError{
			Role:   RoleCallCallee,
			Reason: "unexported generic constraint method has no package",
		}
	}
	return identity + s.method.Pkg().Path() + "|" + s.method.Name(), nil
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
