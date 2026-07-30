package interfaceadapter

import (
	"fmt"
	"go/types"
)

type MethodStage uint8

const (
	MethodStageInvalid    MethodStage = 0
	MethodStageABI        MethodStage = 1
	MethodStageReceiver   MethodStage = 2
	MethodStageContract   MethodStage = 3
	MethodStageInvocation MethodStage = 4
)

func (s MethodStage) String() string {
	switch s {
	case MethodStageABI:
		return "ABI"
	case MethodStageReceiver:
		return "receiver"
	case MethodStageContract:
		return "contract"
	case MethodStageInvocation:
		return "invocation"
	default:
		return "unknown"
	}
}

type MethodError struct {
	Method *types.Func
	Stage  MethodStage
	Cause  error
}

func (e *MethodError) Error() string {
	name := "<unknown>"
	if e.Method != nil {
		name = e.Method.FullName()
	}
	return fmt.Sprintf(
		"emit interface-adapter method %s (%s): %v",
		name,
		e.Stage,
		e.Cause,
	)
}

func (e *MethodError) Unwrap() error {
	return e.Cause
}

func methodStageError(stage MethodStage, cause error) error {
	if cause == nil {
		return nil
	}
	return &MethodError{
		Stage: stage,
		Cause: cause,
	}
}
