package abi

func Method[T any](
	capabilities []T,
	receiver T,
	sourceArguments []T,
) []T {
	result := make(
		[]T,
		0,
		len(capabilities)+1+len(sourceArguments),
	)
	result = append(result, capabilities...)
	result = append(result, receiver)
	result = append(result, sourceArguments...)
	return result
}
