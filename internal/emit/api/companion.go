package api

import (
	"fmt"
	"go/types"
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

type CompanionOwner struct {
	typeName  *types.TypeName
	operation CompanionOperation
}

func NewCompanionOwner(
	typeName *types.TypeName,
	operation CompanionOperation,
) (CompanionOwner, error) {
	switch {
	case typeName == nil:
		return CompanionOwner{}, &PlacementRequestError{
			Reason: "companion type is nil",
		}
	case !operation.Valid():
		return CompanionOwner{}, &PlacementRequestError{
			Reason: "companion operation is invalid",
		}
	}
	return CompanionOwner{typeName: typeName, operation: operation}, nil
}

func (o CompanionOwner) TypeName() *types.TypeName {
	return o.typeName
}

func (o CompanionOwner) Operation() CompanionOperation {
	return o.operation
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
