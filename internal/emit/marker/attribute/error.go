package attribute

type Error struct {
	Reason string
}

func (e *Error) Error() string {
	return "apply canonical source attribute: " + e.Reason
}
