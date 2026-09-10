package complexvalues

type Named complex128

type Record struct {
	Small complex64
	Large Named
}

func StorageCopies() complex128 {
	records := []Record{{Small: 1 + 2i, Large: 3 + 4i}}
	value := records[0]
	copied := value
	value.Small = 5 + 6i
	copied.Large = 7 + 8i
	return complex128(value.Small) + complex128(value.Large) + complex128(copied.Small) + complex128(copied.Large)
}

func StorageContainers() complex128 {
	array := [2]complex128{1 + 2i, 3 + 4i}
	array[0] = 5 + 6i
	values := []complex64{7 + 8i, 9 + 10i}
	copyOfValues := append([]complex64(nil), values...)
	values[0] = 11 + 12i
	lookup := map[int]complex128{1: array[0], 2: complex128(copyOfValues[0])}
	return lookup[1] + lookup[2] + complex128(values[0])
}
