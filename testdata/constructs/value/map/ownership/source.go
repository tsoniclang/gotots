package mapownership

type Key struct {
	Value int32
}

type Named map[Key]int32

func ReassignAggregate(next map[Key]int32) int32 {
	var values map[Key]int32
	values = next
	return values[Key{Value: 1}]
}

func ReassignScalar(next map[string]bool) bool {
	values := make(map[string]bool)
	values = next
	return values["yes"]
}

func Project[M ~map[K]V, K comparable, V any](value M) map[K]V {
	return value
}

func DeclarationLifecycle() (int32, bool) {
	aggregate := map[Key]int32{{Value: 1}: 7}
	scalar := map[string]bool{"yes": true}
	return ReassignAggregate(aggregate), ReassignScalar(scalar)
}

func ProjectionLifecycle() int32 {
	values := Named{{Value: 2}: 7}
	projected := Project(values)
	projected[Key{Value: 2}] = 19
	plain := map[Key]int32{{Value: 3}: 5}
	Project(plain)[Key{Value: 3}] = 11
	scalar := map[string]int32{"value": 2}
	Project(scalar)["value"] = 3
	return values[Key{Value: 2}] +
		plain[Key{Value: 3}] +
		scalar["value"]
}
