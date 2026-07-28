package complex

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimecomplex "github.com/tsoniclang/gotots/internal/emit/runtime/complex"
	floatvalue "github.com/tsoniclang/gotots/internal/emit/value/float"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Carrier struct {
	bits       uint8
	typeSymbol api.RuntimeSymbol
}

func Describe(sourceType types.Type) (Carrier, bool) {
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	if !ok {
		return Carrier{}, false
	}
	switch basic.Kind() {
	case types.Complex64:
		return Carrier{bits: 64, typeSymbol: api.RuntimeComplex64}, true
	case types.Complex128:
		return Carrier{bits: 128, typeSymbol: api.RuntimeComplex128}, true
	default:
		return Carrier{}, false
	}
}

func (c Carrier) Bits() uint8 {
	return c.bits
}

func (c Carrier) TypeSymbol() api.RuntimeSymbol {
	return c.typeSymbol
}

func (c Carrier) ComponentType() *types.Basic {
	switch c.bits {
	case 64:
		return types.Typ[types.Float32]
	case 128:
		return types.Typ[types.Float64]
	default:
		return nil
	}
}

func EmitType(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, error) {
	carrier, ok := Describe(sourceType)
	if !ok {
		return api.TypeEmission{},
			api.Unsupported(context, api.CategoryType, source)
	}
	reference, err := context.Names().Runtime(
		carrier.TypeSymbol(),
		api.ImportPhaseType,
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return api.DirectType(
		context.Factory().TypeReferenceNode(
			context.Factory().Identifier(reference.Name()),
			nil,
		),
		reference.Requests()...,
	), nil
}

func Construct(
	context api.Context,
	carrier Carrier,
	realPart tsgo.Expression,
	imaginaryPart tsgo.Expression,
	requests ...api.RootRequest,
) (api.ExpressionEmission, error) {
	if carrier.ComponentType() == nil ||
		realPart == nil ||
		imaginaryPart == nil {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "complex construction input is invalid",
		}
	}
	reference, err := context.Names().Runtime(
		carrier.TypeSymbol(),
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.DirectExpression(
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				context.Factory().Identifier(reference.Name()),
				nil,
				context.Factory().Identifier(runtimecomplex.MakeMember),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			[]tsgo.Expression{realPart, imaginaryPart},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(requests, reference.Requests())...,
	), nil
}

func EmitConstant(
	context api.Context,
	source ast.Node,
	targetType types.Type,
	value constant.Value,
) (api.ExpressionEmission, error) {
	carrier, ok := Describe(targetType)
	if !ok || value == nil || value.Kind() != constant.Complex {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	componentType := carrier.ComponentType()
	realPart, err := floatvalue.EmitConstant(
		context,
		source,
		componentType,
		constant.Real(value),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	imaginaryPart, err := floatvalue.EmitConstant(
		context,
		source,
		componentType,
		constant.Imag(value),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if len(realPart.Before()) != 0 || len(imaginaryPart.Before()) != 0 {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "complex constant component emitted prerequisites",
		}
	}
	return Construct(
		context,
		carrier,
		realPart.Value(),
		imaginaryPart.Value(),
		api.CombineRequests(
			realPart.Requests(),
			imaginaryPart.Requests(),
		)...,
	)
}

func BinarySymbol(
	carrier Carrier,
	operator token.Token,
) (api.RuntimeSymbol, bool) {
	switch carrier.bits {
	case 64:
		switch operator {
		case token.ADD:
			return api.RuntimeComplex64Add, true
		case token.SUB:
			return api.RuntimeComplex64Sub, true
		case token.MUL:
			return api.RuntimeComplex64Mul, true
		case token.QUO:
			return api.RuntimeComplex64Div, true
		}
	case 128:
		switch operator {
		case token.ADD:
			return api.RuntimeComplex128Add, true
		case token.SUB:
			return api.RuntimeComplex128Sub, true
		case token.MUL:
			return api.RuntimeComplex128Mul, true
		case token.QUO:
			return api.RuntimeComplex128Div, true
		}
	}
	return api.RuntimeInvalid, false
}

func NegateSymbol(carrier Carrier) (api.RuntimeSymbol, bool) {
	switch carrier.bits {
	case 64:
		return api.RuntimeComplex64Neg, true
	case 128:
		return api.RuntimeComplex128Neg, true
	default:
		return api.RuntimeInvalid, false
	}
}

func EqualSymbol(carrier Carrier) (api.RuntimeSymbol, bool) {
	switch carrier.bits {
	case 64:
		return api.RuntimeComplex64Equal, true
	case 128:
		return api.RuntimeComplex128Equal, true
	default:
		return api.RuntimeInvalid, false
	}
}

func Call(
	context api.Context,
	symbol api.RuntimeSymbol,
	arguments []tsgo.Expression,
	requests ...api.RootRequest,
) (api.ExpressionEmission, error) {
	contract, err := api.RuntimeContract(symbol)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if contract.Module() != api.RuntimeModuleComplex ||
		symbol == api.RuntimeComplex64 ||
		symbol == api.RuntimeComplex128 ||
		symbol == api.RuntimeComplexDivide {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "complex call symbol is not a public value operation",
		}
	}
	reference, err := context.Names().Runtime(symbol, api.ImportPhaseValue)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.DirectExpression(
		context.Factory().CallExpression(
			context.Factory().Identifier(reference.Name()),
			nil,
			nil,
			arguments,
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(requests, reference.Requests())...,
	), nil
}
