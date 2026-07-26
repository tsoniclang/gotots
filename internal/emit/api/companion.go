package api

import (
	"fmt"
)

type CompanionOperation uint8

const (
	CompanionInvalid CompanionOperation = iota
	CompanionZero
	CompanionCopy
	CompanionEqual
)

func (o CompanionOperation) Valid() bool {
	return o == CompanionZero || o == CompanionCopy || o == CompanionEqual
}

func (o CompanionOperation) String() string {
	switch o {
	case CompanionZero:
		return "zero"
	case CompanionCopy:
		return "copy"
	case CompanionEqual:
		return "equal"
	default:
		return fmt.Sprintf("companion-operation(%d)", o)
	}
}

func CompanionExportName(
	typeName string,
	operation CompanionOperation,
) (string, error) {
	if typeName == "" {
		return "", &NameError{Reason: "companion type name is empty"}
	}
	if !operation.Valid() {
		return "", &NameError{
			Name:   typeName,
			Reason: "companion operation is invalid",
		}
	}
	return typeName + "$" + operation.String(), nil
}
