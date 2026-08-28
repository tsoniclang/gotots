package array

import (
	"go/ast"
	"go/constant"
	"go/types"
	"strconv"
	"strings"

	"github.com/tsoniclang/gotots/internal/emit/api"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const maxReadableStaticArrayEntries = 4096

func (a RuntimeArray) emitPackedLiteral(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CompositeLit,
	elements []literalSourceElement,
) (api.ExpressionEmission, bool, error) {
	payload, selected := a.packedLiteralPayload(context, elements)
	if !selected {
		return api.ExpressionEmission{}, false, nil
	}
	elementZero, err := a.zeroElement(
		context.WithRole(api.RoleCompositeElement),
		source,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if len(elementZero.Before()) != 0 {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	target, runtimeRequests, err := a.runtimeOperation(
		context,
		children,
		api.RuntimeArrayPacked,
		a.lengthLiteral(context),
		elementZero.Value(),
		context.Factory().NumericLiteral(
			strconv.Itoa(len(elements)),
			tsgo.TokenFlagsNone,
		),
		context.Factory().StringLiteral(payload, tsgo.TokenFlagsNone),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	result, err := api.NewExpressionEmission(
		nil,
		target,
		api.CombineRequests(
			elementZero.Requests(),
			runtimeRequests,
		),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	result, err = a.wrap(context, result)
	return result, true, err
}

func (a RuntimeArray) packedLiteralPayload(
	context api.Context,
	elements []literalSourceElement,
) (string, bool) {
	if a.aggregate ||
		len(elements) <= maxReadableStaticArrayEntries ||
		a.Length() > 1<<53-1 {
		return "", false
	}
	basic, ok := types.Unalias(a.ElementType()).(*types.Basic)
	if !ok || !packedBasicKind(basic.Kind()) {
		return "", false
	}
	carrier, ok := integervalue.Describe(context.TypesSizes(), basic)
	if !ok || carrier.Width() > 32 {
		return "", false
	}
	representation, ok := integervalue.CarrierRepresentation(
		context.IntegerRepresentation(),
		carrier,
	)
	if !ok || representation != api.IntegerCarrierNumber {
		return "", false
	}
	var payload strings.Builder
	payload.Grow(len(elements) * 10)
	for ordinal, element := range elements {
		facts, ok := context.TypesInfo().TypeAndValue(element.value)
		if !ok || facts.Value == nil {
			return "", false
		}
		encoded, ok := packedIntegerConstant(
			context.IntegerRepresentation(),
			carrier,
			facts.Value,
		)
		if !ok {
			return "", false
		}
		if ordinal != 0 {
			payload.WriteByte(',')
		}
		payload.WriteString(strconv.FormatInt(element.index, 36))
		payload.WriteByte(',')
		payload.WriteString(encoded)
	}
	return payload.String(), true
}

func packedBasicKind(kind types.BasicKind) bool {
	switch kind {
	case types.Int8,
		types.Int16,
		types.Int32,
		types.Uint8,
		types.Uint16,
		types.Uint32:
		return true
	default:
		return false
	}
}

func packedIntegerConstant(
	representation api.IntegerRepresentation,
	carrier integervalue.Carrier,
	value constant.Value,
) (string, bool) {
	magnitude, negative, ok := integervalue.FormatConstant(
		representation,
		carrier,
		value,
	)
	if !ok {
		return "", false
	}
	parsed, err := strconv.ParseUint(magnitude, 10, 64)
	if err != nil {
		return "", false
	}
	encoded := strconv.FormatUint(parsed, 36)
	if negative {
		encoded = "-" + encoded
	}
	return encoded, true
}
