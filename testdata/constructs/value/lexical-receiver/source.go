package lexicalreceiver

type Mode int32

func (mode Mode) Shift(delta int32) Mode {
	mode = Mode(int32(mode) + delta)
	return mode
}

type fileRange struct {
	label     string
	fileRange Mode
}

type baseRange struct {
	label        string
	derivedRange Mode
}

type derivedRange baseRange

func Audit() int32 {
	direct := fileRange{
		label:     "direct",
		fileRange: Mode(5).Shift(2),
	}
	derived := derivedRange{
		label:        "derived",
		derivedRange: Mode(8).Shift(3),
	}
	return int32(direct.fileRange)*10 + int32(derived.derivedRange)
}
