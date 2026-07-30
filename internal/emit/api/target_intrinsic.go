package api

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const TargetGlobalAnchorName = "globalThis"

type TargetIntrinsic uint8

const (
	TargetIntrinsicInvalid TargetIntrinsic = iota
	TargetIntrinsicNumber
)

func (i TargetIntrinsic) String() string {
	switch i {
	case TargetIntrinsicNumber:
		return "Number"
	default:
		return fmt.Sprintf("target-intrinsic(%d)", i)
	}
}

func (i TargetIntrinsic) Expression(
	factory tsgo.Factory,
) tsgo.PropertyAccessExpression {
	if i != TargetIntrinsicNumber {
		panic("invalid target intrinsic")
	}
	return factory.PropertyAccessExpression(
		factory.Identifier(TargetGlobalAnchorName),
		nil,
		factory.Identifier(i.String()),
		tsgo.NodeFlagsNone,
	)
}
