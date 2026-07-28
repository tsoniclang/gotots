package definedarray

type Element int32
type Pair [2]Element
type Other [2]Element
type PairAlias = Pair

func ElementFromInt(value int32) Element {
	return Element(value)
}

func IntFromElement(value Element) int32 {
	return int32(value)
}

func NewPair(left, right Element) Pair {
	return Pair{left, right}
}

func PairValues(value Pair) (Element, Element) {
	return value[0], value[1]
}

func RawValues(value [2]Element) (Element, Element) {
	return value[0], value[1]
}

func ZeroPair() Pair {
	var value Pair
	return value
}

func CopyPair(value Pair) (Pair, Pair) {
	copied := value
	copied[0]++
	return value, copied
}

func ConvertRaw(value [2]Element) Pair {
	return Pair(value)
}

func ConvertPair(value Pair) [2]Element {
	return [2]Element(value)
}

func ConvertOther(value Pair) Other {
	return Other(value)
}

func AliasIdentity(value PairAlias) Pair {
	return value
}

func Store(value Pair, index int, next Element) Pair {
	value[index] = next
	return value
}

func Compound(value Pair) Pair {
	value[0] += 2
	return value
}

func Equal(left, right Pair) bool {
	return left == right
}

func Length(value Pair) int {
	return len(value)
}

func PointerStore(value Pair) Pair {
	target := &value[1]
	*target = 9
	return value
}

func CallIsolation() (Pair, Pair, Pair, Pair, bool) {
	original := NewPair(3, 4)
	stored := Store(original, 1, 8)
	compounded := Compound(original)
	pointed := PointerStore(original)
	return original,
		stored,
		compounded,
		pointed,
		Equal(original, NewPair(3, 4))
}

func ConversionIsolation() (Pair, Pair, Pair, [2]Element) {
	raw := [2]Element{1, 2}
	convertedPair := Pair(raw)
	convertedPair[0] = 9

	pair := NewPair(3, 4)
	convertedRaw := [2]Element(pair)
	convertedRaw[1] = 8
	return Pair(raw), convertedPair, pair, convertedRaw
}
