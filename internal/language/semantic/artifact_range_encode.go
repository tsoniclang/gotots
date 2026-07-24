package semantic

import "fmt"

func relationSlice[Value any](
	values []Value,
	start uint64,
	count uint64,
) ([]Value, error) {
	if start > uint64(len(values)) ||
		count > uint64(len(values))-start {
		return nil, fmt.Errorf(
			"semantic relation range %d+%d exceeds %d",
			start, count, len(values),
		)
	}
	return values[start : start+count], nil
}

func encodeReferenceRange[
	Stored ~uint64,
	Wire ~uint64,
](
	values []Stored,
	start uint64,
	count uint64,
	cursor *uint64,
	encode func(Stored) (Wire, error),
) (wireReferenceRange[Wire], error) {
	selected, err := relationSlice(values, start, count)
	if err != nil {
		return wireReferenceRange[Wire]{}, err
	}
	out := wireReferenceRange[Wire]{
		Start:  *cursor,
		Count:  count,
		Values: make([]Wire, 0, len(selected)),
	}
	for _, value := range selected {
		encoded, encodeErr := encode(value)
		if encodeErr != nil {
			return wireReferenceRange[Wire]{}, encodeErr
		}
		out.Values = append(out.Values, encoded)
	}
	*cursor += count
	return out, nil
}

func encodeIntegerRange(
	values []int,
	start uint64,
	count uint64,
	cursor *uint64,
) (wireIntegerRange, error) {
	selected, err := relationSlice(values, start, count)
	if err != nil {
		return wireIntegerRange{}, err
	}
	out := wireIntegerRange{
		Start:  *cursor,
		Count:  count,
		Values: append([]int(nil), selected...),
	}
	*cursor += count
	return out, nil
}
