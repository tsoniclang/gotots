package semanticname

type Error struct {
	Reason string
}

func (e *Error) Error() string {
	if e == nil {
		return "semantic name error"
	}
	return "semantic name: " + e.Reason
}

func invalid(reason string) error {
	return &Error{Reason: reason}
}
