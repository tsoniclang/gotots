package rune

const Star = '★'

func ASCII() rune {
	return 'A'
}

func EscapedASCII() rune {
	return '\u0041'
}

func Newline() rune {
	return '\n'
}

func Accented() rune {
	return 'é'
}

func CJK() rune {
	return '世'
}

func Emoji() rune {
	return '🎉'
}

func Constant() rune {
	return Star
}

func Widened() int32 {
	return 'Z'
}
