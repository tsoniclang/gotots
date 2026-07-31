package promotedpointer

type Inner struct {
	Value int32
}

func (inner *Inner) Increment() int32 {
	inner.Value++
	return inner.Value
}

type Outer struct {
	Inner
}

type Incrementer interface {
	Increment() int32
}

func Audit() int32 {
	outer := &Outer{Inner: Inner{Value: 30}}
	var selected Incrementer = outer
	return selected.Increment()*100 + outer.Value
}
