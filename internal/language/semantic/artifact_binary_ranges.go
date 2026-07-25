package semantic

import "fmt"

func writeReferenceRange[Reference ~uint64](
	encoder *binaryShardEncoder,
	values []Reference,
	start uint64,
	count uint64,
) {
	if start > uint64(len(values)) ||
		count > uint64(len(values))-start {
		if encoder.err == nil {
			encoder.err = fmt.Errorf(
				"semantic binary relation %d+%d exceeds %d",
				start,
				count,
				len(values),
			)
		}
		return
	}
	encoder.unsigned(count)
	for _, value := range values[start : start+count] {
		encoder.unsigned(uint64(value))
	}
}

func readReferenceRange[Reference binaryIdentityReference](
	decoder *binaryShardDecoder,
	name string,
	values *[]Reference,
) (uint64, uint64, error) {
	count, err := decoder.count(name)
	if err != nil {
		return 0, 0, err
	}
	start := uint64(len(*values))
	for index := 0; index < count; index++ {
		value, readErr := readIdentityReference[Reference](
			decoder, name+" reference",
		)
		if readErr != nil {
			return 0, 0, readErr
		}
		*values = append(*values, value)
	}
	return start, uint64(count), nil
}

func readExpectedRecordCount(
	decoder *binaryShardDecoder,
	name string,
	expected int,
) (int, error) {
	count, err := decoder.count(name)
	if err != nil {
		return 0, err
	}
	if count != expected {
		return 0, fmt.Errorf(
			"semantic binary %s count %d disagrees with manifest %d",
			name,
			count,
			expected,
		)
	}
	return count, nil
}
