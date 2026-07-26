package expressionswitch

func Classify(value int32) int32 {
	var result int32 = 0
	switch current := value; current {
	case 0:
		var branch int32 = 10
		result = branch
	case 1, 2:
		var branch int32 = 20
		result = branch
	default:
		var branch int32 = 30
		result = branch
	}
	return result
}
