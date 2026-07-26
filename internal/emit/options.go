package emit

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

type IntegerRepresentation = api.IntegerRepresentation
type EvaluationOrder = api.EvaluationOrder

const (
	IntegerRepresentationInvalid = api.IntegerRepresentationInvalid
	IntegerRepresentationNumber  = api.IntegerRepresentationNumber
	IntegerRepresentationBigInt  = api.IntegerRepresentationBigInt

	EvaluationOrderInvalid    = api.EvaluationOrderInvalid
	EvaluationOrderDirect     = api.EvaluationOrderDirect
	EvaluationOrderPreserveGo = api.EvaluationOrderPreserveGo
)

type Options struct {
	IntegerRepresentation IntegerRepresentation
	EvaluationOrder       EvaluationOrder
}

func DefaultOptions() Options {
	return Options{
		IntegerRepresentation: IntegerRepresentationNumber,
		EvaluationOrder:       EvaluationOrderDirect,
	}
}

func ParseIntegerRepresentation(value string) (IntegerRepresentation, error) {
	switch value {
	case IntegerRepresentationNumber.String():
		return IntegerRepresentationNumber, nil
	case IntegerRepresentationBigInt.String():
		return IntegerRepresentationBigInt, nil
	default:
		return IntegerRepresentationInvalid, &OptionsError{
			Field:  "integer representation",
			Reason: fmt.Sprintf("%q is not number or bigint", value),
		}
	}
}

func ParseEvaluationOrder(value string) (EvaluationOrder, error) {
	switch value {
	case EvaluationOrderDirect.String():
		return EvaluationOrderDirect, nil
	case EvaluationOrderPreserveGo.String():
		return EvaluationOrderPreserveGo, nil
	default:
		return EvaluationOrderInvalid, &OptionsError{
			Field:  "evaluation order",
			Reason: fmt.Sprintf("%q is not direct or preserve-go", value),
		}
	}
}

func (o Options) validate() error {
	if !o.IntegerRepresentation.Valid() {
		return &OptionsError{
			Field:  "integer representation",
			Reason: "value is invalid",
		}
	}
	if !o.EvaluationOrder.Valid() {
		return &OptionsError{
			Field:  "evaluation order",
			Reason: "value is invalid",
		}
	}
	return nil
}

type OptionsError struct {
	Field  string
	Reason string
}

func (e *OptionsError) Error() string {
	if e.Field == "" {
		return "validate compilation options: " + e.Reason
	}
	return fmt.Sprintf("validate compilation option %q: %s", e.Field, e.Reason)
}
