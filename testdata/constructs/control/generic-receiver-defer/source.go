package genericreceiverdefer

type Box[T any] struct {
	Value  T
	Output *T
}

func (box *Box[T]) store() {
	*box.Output = box.Value
}

func Audit() (result int32) {
	box := Box[int32]{
		Value:  41,
		Output: &result,
	}
	defer box.store()
	box.Value = 42
	return
}
