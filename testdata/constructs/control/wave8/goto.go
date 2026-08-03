package wave8control

func GotoForward(value int) int {
	if value < 0 {
		goto negative
	}
	value += 10
	return value
negative:
	return -value
}

func GotoLoop(limit int) int {
	total := 0
	index := 0
again:
	if index >= limit {
		return total
	}
	total += index
	index++
	goto again
}

func GotoSameLabelA(value int) int {
	goto done
	value = -1
done:
	return value + 1
}

func GotoSameLabelB(value int) int {
	goto done
	value = -2
done:
	return value + 2
}

func GotoDirectMultiple(value int) int {
	goto first
	value = -10
first:
	value++
	goto second
	value = -20
second:
	return value
}

func GotoLabeledLoop(limit int) int {
	result := 0
	if limit < 0 {
		goto outer
	}
	result++
outer:
	for index := 0; index < limit; index++ {
		if index == 3 {
			break outer
		}
		result += index
	}
	return result
}

func GotoSwitchClause(value int) int {
	switch value {
	case 1:
		goto adjust
		value = -10
	adjust:
		value += 2
	default:
		value++
	}
	return value
}

func GotoSwitchClauseLoop(value int) int {
	switch value {
	case 2:
		index := 0
	again:
		index++
		if index < value {
			goto again
		}
		value += index
	}
	return value
}

func GotoTypeSwitchClause(value any) int {
	result := 0
	switch value.(type) {
	case int:
		goto adjust
		result = -10
	adjust:
		result = 3
	case string:
		result = 4
	}
	return result
}

func GotoTypeSwitchClauseAudit() int {
	return GotoTypeSwitchClause(1)*10 + GotoTypeSwitchClause("value")
}

func GotoSwitchFallthrough(value int) int {
	switch value {
	case 1:
		goto adjust
		value = -10
	adjust:
		value += 2
		fallthrough
	case 2:
		value += 20
	}
	return value
}

func GotoState(limit int) int {
	total := 0
	index := 0
	goto check
body:
	total += index
	index++
check:
	if index < limit {
		goto body
	}
	return total
}

func GotoStateAddress(limit int) int {
	value := 1
	pointer := &value
	goto check
step:
	(*pointer)++
check:
	if value < limit {
		goto step
	}
	return value
}

func controlPair() (int, int) {
	return 3, 4
}

func GotoStateDeclarations(limit int) int {
	left, right := 1, 2
	var third, fourth = controlPair()
	goto check
body:
	left += right
	third += fourth
	limit--
check:
	if limit > 0 {
		goto body
	}
	return left*1000 + right*100 + third*10 + fourth
}

func GotoStateStatic(limit int) int {
	const typed int = 3
	const untyped = 4
	type Local int
	value := Local(typed + untyped)
	goto check
body:
	value++
	limit--
check:
	if limit > 0 {
		goto body
	}
	return int(value)
}

func GotoStateFreshAddress() bool {
	var previous *int
	count := 0
enter:
	if count >= 2 {
		return true
	}
	goto again
again:
	value := count
	current := &value
	if previous != nil {
		if previous == current {
			return false
		}
	}
	previous = current
	count++
	goto enter
}

func GotoNestedState(limit int) int {
	result := 0
	{
		index := 0
		goto check
	body:
		result += index
		index++
	check:
		if index < limit {
			goto body
		}
	}
	return result
}

func GotoStateLoopControl() int {
	result := 0
	for outer := 0; outer < 5; outer++ {
		index := 0
		goto check
	body:
		index++
		if outer == 1 {
			continue
		}
		if outer == 3 {
			break
		}
		result += outer
	check:
		if index < 1 {
			goto body
		}
	}
	return result
}

func GotoSwitchStateBreak(value int) int {
	result := 0
	switch value {
	case 1:
		index := 0
		goto check
	body:
		result++
		if result == 2 {
			break
		}
		index++
	check:
		if index < 3 {
			goto body
		}
		result = 99
	}
	return result
}

func GotoSwitchStateFallthrough(value int) int {
	result := 0
	switch value {
	case 1:
		index := 0
		goto check
	body:
		result++
		index++
	check:
		if index < 2 {
			goto body
		}
		fallthrough
	case 2:
		result += 10
	}
	return result
}

func GotoTypeSwitchStateBreak(value any) int {
	result := 0
	switch selected := value.(type) {
	case int:
		index := 0
		goto check
	body:
		result += selected
		if result == selected*2 {
			break
		}
		index++
	check:
		if index < 3 {
			goto body
		}
		result = 99
	}
	return result
}

func GotoTypeSwitchStateBreakAudit() int {
	return GotoTypeSwitchStateBreak(2)
}

func GotoRangeStateContinue(values []int) int {
	result := 0
	for _, value := range values {
		index := 0
		goto check
	body:
		index++
		if value < 0 {
			continue
		}
		result += value
	check:
		if index < 1 {
			goto body
		}
	}
	return result
}

func GotoRangeStateContinueAudit() int {
	return GotoRangeStateContinue([]int{1, -2, 3})
}

func GotoWithDefer(limit int) (result int) {
	defer func() {
		result++
	}()
	index := 0
	goto check
body:
	result += index
	index++
check:
	if index < limit {
		goto body
	}
	return
}

func GotoRepeatedDefer(limit int) (result int) {
	index := 0
again:
	defer func(value int) {
		result = result*10 + value
	}(index)
	index++
	if index < limit {
		goto again
	}
	return
}

func GotoVoid(result *int, limit int) {
	index := 0
	goto check
body:
	*result += index
	index++
check:
	if index < limit {
		goto body
	}
}

func GotoVoidAudit() int {
	result := 0
	GotoVoid(&result, 5)
	return result
}

func GotoDeferRange(values []int) (result int) {
	defer func() {
		result++
	}()
	index := 0
	goto check
body:
	for _, value := range values {
		if value < 0 {
			goto next
		}
		result += value
	}
next:
	index++
check:
	if index < 2 {
		goto body
	}
	return
}

func GotoDeferRangeAudit() int {
	return GotoDeferRange([]int{1, 2, -1, 8})
}

func GotoFromRange(values []int) int {
	result := 0
	for _, value := range values {
		if value < 0 {
			goto done
		}
		result += value
	}
done:
	return result
}

func GotoFromRangeAudit() int {
	return GotoFromRange([]int{1, 2, -1, 8})
}

func LabeledControl(limit int) int {
	result := 0
outer:
	for index := 0; index < limit; index++ {
		for inner := 0; inner < limit; inner++ {
			if inner == 1 {
				continue outer
			}
			result++
			if result > 10 {
				break outer
			}
		}
	}
	return result
}

func FallthroughControl(value int) int {
	switch value {
	case 1:
		value += 10
		fallthrough
	case 2:
		value += 20
	default:
		value += 30
	}
	return value
}

func FallthroughLoopControl(limit int) int {
	total := 0
	for index := 0; index < limit; index++ {
		switch index {
		case 0:
			total++
			fallthrough
		case 1:
			total += 10
			continue
		case 2:
			total += 100
			break
		default:
			total += 1000
			fallthrough
		case 3:
			total += 10000
		}
		total += 100000
	}
	return total
}

func FallthroughReturn(value int) int {
	switch value {
	case 0:
		fallthrough
	case 1:
		return 10
	default:
		return 20
	}
}
