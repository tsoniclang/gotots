package api

type Number interface {
	~int32
}

func Twice[T Number](value T) T {
	return value + value
}

func BothEqual[A comparable, B comparable](
	leftA A,
	rightA A,
	leftB B,
	rightB B,
) bool {
	return leftA == rightA && leftB == rightB
}
