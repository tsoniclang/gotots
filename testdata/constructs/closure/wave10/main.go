package main

//go:generate echo selected-directive

import (
	. "example.com/wave10closure/dependency/dot"
	_ "example.com/wave10closure/dependency/sideeffect"
	renamed "example.com/wave10closure/dependency/value"
)

// Scale proves declaration comments are source metadata.
const (
	// ScaleValue proves specification comments are source metadata.
	ScaleValue int32 = 2 // scale value
)

// Box proves callable and field comments are source metadata.
type Box struct {
	Value int32
}

// Sum proves callable comments do not alter signature ownership.
func (value Box) Sum(
	// delta is deliberately documented.
	delta int32, // trailing parameter comment
) (
	// result is deliberately documented.
	result int32,
) {
	return value.Value + delta
}

func scoped(input int32) int32 {
	// Local is owned by the enclosing lexical scope.
	type Local int32
	const (
		// localConstant keeps its checker-owned type.
		localConstant Local = 3
	)
	var (
		// localVariable keeps its checker-owned type.
		localVariable Local = 4
	)
	{
		input := Local(input)
		localVariable += input
	}
	quotient := int32(20)
	quotient /= 2
	quotient %= 7
	return int32(localConstant+localVariable) + quotient
}

func Audit() int32 {
	if false {
		goto empty
	}
empty:
	;
	box := Box{Value: renamed.Value()}
	return box.Sum(DotValue()) + scoped(1) + ScaleValue
}

func main() {
	_ = Audit()
}
