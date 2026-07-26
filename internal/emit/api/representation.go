package api

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type IntegerRepresentation uint8

const (
	IntegerRepresentationInvalid IntegerRepresentation = iota
	IntegerRepresentationNumber
	IntegerRepresentationBigInt
)

func (r IntegerRepresentation) Valid() bool {
	return r == IntegerRepresentationNumber ||
		r == IntegerRepresentationBigInt
}

func (r IntegerRepresentation) String() string {
	switch r {
	case IntegerRepresentationNumber:
		return "number"
	case IntegerRepresentationBigInt:
		return "bigint"
	default:
		return fmt.Sprintf("integer-representation(%d)", r)
	}
}

type EvaluationOrder uint8

const (
	EvaluationOrderInvalid EvaluationOrder = iota
	EvaluationOrderDirect
	EvaluationOrderPreserveGo
)

func (o EvaluationOrder) Valid() bool {
	return o == EvaluationOrderDirect ||
		o == EvaluationOrderPreserveGo
}

func (o EvaluationOrder) String() string {
	switch o {
	case EvaluationOrderDirect:
		return "direct"
	case EvaluationOrderPreserveGo:
		return "preserve-go"
	default:
		return fmt.Sprintf("evaluation-order(%d)", o)
	}
}

func IntegerLiteral(
	factory tsgo.Factory,
	representation IntegerRepresentation,
	decimal string,
) (tsgo.Expression, error) {
	if decimal == "" {
		return nil, &IntegerRepresentationError{
			Representation: representation,
		}
	}
	switch representation {
	case IntegerRepresentationNumber:
		return factory.NumericLiteral(decimal, tsgo.TokenFlagsNone), nil
	case IntegerRepresentationBigInt:
		return factory.BigIntLiteral(decimal+"n", tsgo.TokenFlagsNone), nil
	default:
		return nil, &IntegerRepresentationError{
			Representation: representation,
		}
	}
}
