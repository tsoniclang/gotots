package definedmap

type Values map[int32]int32
type Other map[int32]int32
type Count int32
type Alias = Values
type PlainAlias = map[int32]int32

func ZeroState() (bool, bool) {
	var values Values
	pointer := new(Values)
	return values == nil, *pointer == nil
}

func MakeAliases() (int32, int32, int, bool) {
	values := make(Values, 2)
	alias := values
	alias[1] = 7
	var semanticAlias Alias = alias
	semanticAlias[2] = 9
	first := values[1]
	second := values[2]
	return first, second, len(values), alias != nil
}

func Conversions() (bool, bool, int32) {
	var nilValues Values
	unnamedNil := map[int32]int32(nilValues)
	otherNil := Other(nilValues)
	unnamed := make(map[int32]int32)
	values := Values(unnamed)
	other := Other(values)
	other[3] = 11
	found := values[3]
	return unnamedNil == nil, otherNil == nil, found
}

func NilOperations() (int32, bool, int) {
	var values Values
	value, ok := values[1]
	delete(values, 1)
	return value, ok, len(values)
}

func NilWrite() {
	var values Values
	values[1] = 2
}

func PlainAliasBehavior() (int32, bool) {
	values := make(PlainAlias)
	values[4] = 13
	return values[4], values != nil
}

func DefinedKeyBehavior() (int32, int32) {
	values := map[Count]int32{1: 5}
	key := Count(2)
	values[key] = 7
	found := values[key]
	values[3] = 9
	found += values[3]
	var total Count
	for current := range values {
		total += current
	}
	return found, int32(total)
}

func DefinedKeyLookup(values map[Count]int32, key Count) int32 {
	return values[key]
}

func DefinedKeyZero() bool {
	var values map[Count]int32
	return values == nil
}
