package classmembers

func (counter Counter) Bump() int32 {
	counter.Value++
	return counter.Value
}

func (counter Counter) Read() int32 {
	return counter.Value
}

func (counter *Counter) Reset(value int32) {
	counter.Value = value
}

func (counter Counter) unused() int32 {
	return counter.Value + 99
}
