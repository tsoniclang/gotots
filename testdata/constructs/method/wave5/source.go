package wave5methods

type Score int32

func (score Score) Double() int32 {
	return int32(score) * 2
}

func (score *Score) Add(delta int32) {
	*score += Score(delta)
}

type Base struct {
	Value int32
}

func (base Base) Read() int32 {
	return base.Value
}

func (base *Base) Add(delta int32) {
	base.Value += delta
}

func (base *Base) NilSafe() bool {
	return base == nil
}

func (base *Base) Name() int32 {
	return 100 + base.Value
}

func (base *Base) CallName() int32 {
	return base.Name()
}

func (base Base) Sum(values ...int32) int32 {
	result := base.Value
	for _, value := range values {
		result += value
	}
	return result
}

type Derived struct {
	Base  `json:"base"`
	Extra int32
}

func (derived *Derived) Name() int32 {
	return 200 + derived.Extra
}

type PointerOuter struct {
	*Base
}

type BasePointer *Base

type Level struct {
	*Derived
}

type constructor struct {
	Value int32
}

func promotedFieldsAndCalls() []int32 {
	value := Derived{
		Base:  Base{Value: 3},
		Extra: 4,
	}
	result := []int32{
		value.Value,
		value.Read(),
		value.Name(),
		value.CallName(),
	}
	value.Value = 8
	address := &value.Value
	*address += 2
	value.Add(5)
	return append(result, value.Value, value.CallName())
}

func nestedPromotion() []int32 {
	value := Level{Derived: &Derived{
		Base:  Base{Value: 6},
		Extra: 7,
	}}
	value.Add(2)
	value.Value++
	return []int32{value.Value, value.Name(), value.CallName()}
}

func methodValues() []int32 {
	value := Base{Value: 3}
	read := value.Read
	value.Value = 9

	pointer := Base{Value: 4}
	add := pointer.Add
	pointer.Value = 8
	add(3)

	first := &Base{Value: 5}
	current := first
	addFirst := current.Add
	current = &Base{Value: 20}
	addFirst(2)

	derived := Derived{Base: Base{Value: 6}}
	promotedRead := derived.Read
	promotedAdd := derived.Add
	derived.Value = 9
	promotedAdd(2)

	variadic := derived.Sum
	return []int32{
		read(),
		value.Value,
		pointer.Value,
		first.Value,
		current.Value,
		promotedRead(),
		derived.Value,
		variadic(1, 2, 3),
	}
}

func methodExpressions() []int32 {
	read := Base.Read
	add := (*Base).Add
	promotedRead := Derived.Read
	promotedAdd := (*Derived).Add
	promotedSum := Derived.Sum

	base := Base{Value: 10}
	derived := Derived{Base: Base{Value: 20}}
	add(&base, 2)
	promotedAdd(&derived, 3)
	return []int32{
		read(base),
		promotedRead(derived),
		base.Value,
		derived.Value,
		promotedSum(derived, 1, 2),
	}
}

func definedReceiverMethods() []int32 {
	score := Score(7)
	double := score.Double
	add := score.Add
	add(3)
	valueDouble := Score.Double
	pointerAdd := (*Score).Add
	pointerAdd(&score, 2)
	return []int32{double(), score.Double(), valueDouble(score)}
}

func anonymousEmbedding() int32 {
	value := struct {
		Base
	}{
		Base: Base{Value: 13},
	}
	value.Add(2)
	return value.Read()
}

func anonymousConstructorEmbedding() int32 {
	value := struct {
		constructor
	}{
		constructor: constructor{Value: 14},
	}
	return value.Value
}

func definedPointerField() int32 {
	value := Base{Value: 15}
	pointer := BasePointer(&value)
	pointer.Value = 16
	return pointer.Value
}

func NilPointerMethodValue() bool {
	var value *Base
	method := value.NilSafe
	return method()
}

func NilPromotedPointerMethod() bool {
	var value PointerOuter
	return value.NilSafe()
}

func PanicValueMethod() int32 {
	var value *Base
	method := value.Read
	return method()
}

func PanicPromotedField() int32 {
	var value PointerOuter
	return value.Value
}

func Audit() []int32 {
	result := promotedFieldsAndCalls()
	result = append(result, nestedPromotion()...)
	result = append(result, methodValues()...)
	result = append(result, methodExpressions()...)
	result = append(result, definedReceiverMethods()...)
	result = append(result, anonymousEmbedding())
	result = append(result, anonymousConstructorEmbedding())
	result = append(result, definedPointerField())
	return result
}
