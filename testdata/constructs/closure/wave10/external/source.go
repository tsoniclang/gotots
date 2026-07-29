package external

// Read is implemented by a selected external environment.
func Read(buffer []byte) (count int, err error)
