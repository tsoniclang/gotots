package semantic

import "fmt"

func decodeReferenceRange[
	Wire ~uint64,
	Value any,
](
	name string,
	encoded wireReferenceRange[Wire],
	cursor *uint64,
	decode func(Wire) (Value, error),
) ([]Value, error) {
	if encoded.Start != *cursor ||
		encoded.Count != uint64(len(encoded.Values)) {
		return nil, fmt.Errorf(
			"semantic wire %s range %d+%d has %d values at cursor %d",
			name,
			encoded.Start,
			encoded.Count,
			len(encoded.Values),
			*cursor,
		)
	}
	out := make([]Value, 0, len(encoded.Values))
	for _, reference := range encoded.Values {
		value, err := decode(reference)
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	*cursor += encoded.Count
	return out, nil
}

func decodeIntegerRange(
	name string,
	encoded wireIntegerRange,
	cursor *uint64,
) ([]int, error) {
	if encoded.Start != *cursor ||
		encoded.Count != uint64(len(encoded.Values)) {
		return nil, fmt.Errorf(
			"semantic wire %s range %d+%d has %d values at cursor %d",
			name,
			encoded.Start,
			encoded.Count,
			len(encoded.Values),
			*cursor,
		)
	}
	*cursor += encoded.Count
	return append([]int(nil), encoded.Values...), nil
}

func requireSinglePayload(
	name string,
	tag uint8,
	expected uint8,
	payloads ...bool,
) error {
	active := 0
	for _, present := range payloads {
		if present {
			active++
		}
	}
	if tag != expected || active != 1 {
		return fmt.Errorf(
			"semantic wire %s payload tag %d selects %d payloads, want %d and one",
			name,
			tag,
			active,
			expected,
		)
	}
	return nil
}
