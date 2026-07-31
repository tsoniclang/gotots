package wave7generics

func AssertValue[T any](value any) (T, bool) {
	result, ok := value.(T)
	return result, ok
}

func MustAssertValue[T any](value any) T {
	return value.(T)
}

func TypeSwitchValue[T any](value any) bool {
	switch selected := value.(type) {
	case T:
		var typed T = selected
		_ = typed
		return true
	default:
		return false
	}
}

func AuditGenericAssertions() []int32 {
	success, successOK := AssertValue[int32](int32(23))
	failure, failureOK := AssertValue[int32]("wrong")
	required := MustAssertValue[int32](int32(24))
	if !successOK ||
		failureOK ||
		failure != 0 ||
		!TypeSwitchValue[int32](int32(25)) ||
		TypeSwitchValue[int32]("wrong") {
		return []int32{-1}
	}
	return []int32{success, failure, required}
}
