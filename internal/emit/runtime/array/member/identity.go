package arraymember

type Identity uint8

const (
	Invalid Identity = 0
	Zero    Identity = 1
	Literal Identity = 2
	Copy    Identity = 3
	Get     Identity = 4
	Set     Identity = 5
	Length  Identity = 6
)

func All() []Identity {
	return []Identity{
		Zero,
		Literal,
		Copy,
		Get,
		Set,
		Length,
	}
}

func (i Identity) Valid() bool {
	return i >= Zero && i <= Length
}

func (i Identity) Name() string {
	switch i {
	case Zero:
		return "zero"
	case Literal:
		return "literal"
	case Copy:
		return "copy"
	case Get:
		return "get"
	case Set:
		return "set"
	case Length:
		return "length"
	default:
		panic("invalid array runtime member identity")
	}
}
