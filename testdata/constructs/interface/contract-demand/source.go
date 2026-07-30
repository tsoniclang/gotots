package interfacecontractdemand

type First interface {
	First() int32
	Shared() int32
}

type Second interface {
	Second() int32
	Shared() int32
}

type Value struct{}

func (Value) First() int32         { return 1 }
func (Value) Shared() int32        { return 2 }
func (Value) Second() int32        { return 3 }
func (Value) Unused() int32        { return 4 }
func (Value) privateUnused() int32 { return 5 }

type Other struct{}

func (Other) Second() int32 { return 6 }
func (Other) Shared() int32 { return 7 }
func (Other) Unused() int32 { return 8 }

func inspect(value First) []int32 {
	selected, ok := value.(Second)
	switch switched := value.(type) {
	case Second:
		return []int32{
			value.First(),
			selected.Second(),
			switched.Shared(),
			boolValue(ok),
		}
	default:
		return []int32{-1}
	}
}

func Audit() []int32 {
	var value First = Value{}
	var broad any = Other{}
	_, concreteOK := broad.(Other)
	return append(inspect(value), boolValue(concreteOK))
}

func boolValue(value bool) int32 {
	if value {
		return 1
	}
	return 0
}
