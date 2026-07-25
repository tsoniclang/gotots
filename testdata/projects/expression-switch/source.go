package expressionswitch

func Classify(value int) int {
	result := 0
	switch current := value; current {
	case 0:
		branch := 10
		result = branch
	case 1, 2:
		branch := 20
		result = branch
	default:
		branch := 30
		result = branch
	}
	return result
}
