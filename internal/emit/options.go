package emit

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

type IntegerRepresentation = api.IntegerRepresentation

const (
	IntegerRepresentationInvalid = api.IntegerRepresentationInvalid
	IntegerRepresentationNumber  = api.IntegerRepresentationNumber
	IntegerRepresentationBigInt  = api.IntegerRepresentationBigInt
)

type Options struct {
	IntegerRepresentation IntegerRepresentation
}

func DefaultOptions() Options {
	return Options{IntegerRepresentation: IntegerRepresentationNumber}
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

func (o Options) validate() error {
	if !o.IntegerRepresentation.Valid() {
		return &OptionsError{
			Field:  "integer representation",
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
