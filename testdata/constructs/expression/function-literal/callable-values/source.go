package callablevalues

func Apply(transform func(int32) int32, value int32) int32 {
	return transform(value)
}

func Double(value int32) int32 {
	return value * 2
}

func UseNamed(value int32) int32 {
	return Apply(Double, value)
}

func Offset(delta int32) func(int32) int32 {
	return func(value int32) int32 {
		return value + delta
	}
}

func UseClosure(value, delta int32) int32 {
	transform := Offset(delta)
	return transform(value)
}

func Increment(value int32) int32 {
	return value + 1
}

func Decrement(value int32) int32 {
	return value - 1
}

func Choose(positive bool) func(int32) int32 {
	if positive {
		return Increment
	}
	return Decrement
}

func UseReturned(value int32, positive bool) int32 {
	return Choose(positive)(value)
}

func Immediate(value int32) int32 {
	return func(input int32) int32 {
		return input + 1
	}(value)
}

func Ignore(int32) int32 {
	return 7
}

func UseExplicit(value, delta int32) int32 {
	var transform func(int32) int32 = Offset(delta)
	return transform(value)
}

func Counter(start int32) func() int32 {
	value := start
	return func() int32 {
		value++
		return value
	}
}

func UseCounter(start int32) int32 {
	counter := Counter(start)
	first := counter()
	second := counter()
	return first*10 + second
}

var trace int32 = 0

func mark(step int32) {
	trace = trace*10 + step
}

func addPair(left, right int32) int32 {
	return left + right
}

func selectPair() func(int32, int32) int32 {
	mark(1)
	return addPair
}

func pair() (int32, int32) {
	mark(2)
	return 3, 4
}

func OrderedCalleeAndArguments() int32 {
	trace = 0
	result := selectPair()(pair())
	return trace*100 + result
}

type Item struct {
	Value int32
}

func IncrementItem(item Item) Item {
	item.Value++
	return item
}

func ApplyItem(transform func(Item) Item, item Item) Item {
	return transform(item)
}

func UseItem(value int32) int32 {
	original := Item{Value: value}
	changed := ApplyItem(IncrementItem, original)
	return original.Value*100 + changed.Value
}

func PairValue(value int32) (int32, bool) {
	return value + 1, value >= 0
}

func ApplyPair(
	transform func(int32) (int32, bool),
	value int32,
) (int32, bool) {
	return transform(value)
}

func UsePair(value int32) (int32, bool) {
	return ApplyPair(PairValue, value)
}
