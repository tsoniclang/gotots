package targetbinding

const Before Number = 7

type Number float64

func globalThis() int64 {
	return 2
}

func Audit(value int64) float64 {
	return float64(value) + float64(Before) + float64(globalThis())
}
