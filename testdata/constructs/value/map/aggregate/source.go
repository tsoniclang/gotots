package aggregatemap

type Box struct {
	Value int32
	Pair  [2]int32
}

type BoxMap map[int32]Box
type BoxMapAlias = BoxMap
type WideBoxMap map[uint64]Box

type ArrayKey [2]int32

type StructKey struct {
	Label int32
	Parts [2]int32
}

type CollisionKey struct {
	Value int64
}

type Label string

func NamedKeyLifecycle() (int32, bool, string, bool) {
	values := map[Label]int32{
		"first": 41,
	}
	found, ok := values[Label("first")]
	delete(values, Label("first"))
	values[Label("second")] = 42
	keys := make([]Label, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	_, deleted := values[Label("first")]
	return found, ok, string(keys[0]), deleted
}

func NamedValueLifecycle() (
	int32,
	int32,
	int32,
	int32,
	bool,
	bool,
	int,
	bool,
) {
	var nilValues BoxMap
	values := make(BoxMap, 2)
	original := Box{Value: 7, Pair: [2]int32{8, 9}}
	values[1] = original
	original.Value = 70
	original.Pair[0] = 80

	found, ok := values[1]
	found.Value = 71
	found.Pair[0] = 81
	again := values[1]

	missing, missingOK := values[99]
	missing.Value = 91
	missing.Pair[0] = 92
	secondMissing := values[99]

	var alias BoxMapAlias = values
	alias[2] = Box{Value: 10, Pair: [2]int32{11, 12}}
	delete(alias, 1)
	_, deletedOK := values[1]
	return again.Value,
		again.Pair[0],
		secondMissing.Value,
		secondMissing.Pair[0],
		ok,
		missingOK,
		len(values),
		!deletedOK && nilValues == nil
}

func WideKeyLifecycle() (
	int32,
	int32,
	int32,
	int32,
	int32,
	int32,
	bool,
	bool,
	bool,
	bool,
	bool,
	bool,
	int,
) {
	const lower uint64 = 9007199254740992
	const upper uint64 = 9007199254740993
	const maximum uint64 = 18446744073709551615

	values := make(WideBoxMap, 3)
	source := Box{Value: 10, Pair: [2]int32{11, 12}}
	values[lower] = source
	source.Value = 100
	source.Pair[0] = 110
	values[upper] = Box{Value: 20, Pair: [2]int32{21, 22}}
	values[maximum] = Box{Value: 30, Pair: [2]int32{31, 32}}

	lowerValue, lowerOK := values[lower]
	lowerValue.Value = 90
	lowerValue.Pair[0] = 91
	lowerAgain := values[lower]
	upperValue, upperOK := values[upper]
	maximumValue, maximumOK := values[maximum]
	_, missingOK := values[9007199254740994]
	values[upper] = Box{Value: 25, Pair: [2]int32{26, 27}}
	updated := values[upper]

	delete(values, lower)
	_, upperAfterDeleteOK := values[upper]
	delete(values, maximum)
	_, maximumAfterDeleteOK := values[maximum]
	return lowerValue.Value,
		lowerAgain.Value,
		lowerAgain.Pair[0],
		upperValue.Value,
		maximumValue.Value,
		updated.Value,
		lowerOK,
		upperOK,
		maximumOK,
		missingOK,
		upperAfterDeleteOK,
		maximumAfterDeleteOK,
		len(values)
}

func ArrayKeyLifecycle() (int32, bool, int, bool) {
	key := ArrayKey{3, 4}
	values := map[ArrayKey]int32{key: 34}
	key[0] = 30
	found, ok := values[ArrayKey{3, 4}]
	delete(values, ArrayKey{3, 4})
	_, deletedOK := values[ArrayKey{3, 4}]
	return found, ok, len(values), deletedOK
}

func StructKeyLifecycle() (int32, int32, bool, int) {
	key := StructKey{Label: 5, Parts: [2]int32{6, 7}}
	values := map[StructKey]Box{
		key: {Value: 56, Pair: [2]int32{57, 58}},
	}
	key.Label = 50
	key.Parts[0] = 60
	found, ok := values[StructKey{
		Label: 5,
		Parts: [2]int32{6, 7},
	}]
	found.Pair[0] = 99
	again := values[StructKey{
		Label: 5,
		Parts: [2]int32{6, 7},
	}]
	return found.Value, again.Pair[0], ok, len(values)
}

func AnonymousShapeLifecycle() (int32, int32, bool) {
	values := map[struct {
		Value int32
	}]struct {
		Number int32
	}{
		{Value: 8}: {Number: 89},
	}
	found, ok := values[struct {
		Value int32
	}{Value: 8}]
	found.Number = 99
	again := values[struct {
		Value int32
	}{Value: 8}]
	return found.Number, again.Number, ok
}

func CollisionEquality() (int32, int32, int) {
	values := map[CollisionKey]int32{
		{Value: 1}:          10,
		{Value: 4294967297}: 20,
	}
	return values[CollisionKey{Value: 1}],
		values[CollisionKey{Value: 4294967297}],
		len(values)
}

func LiteralOrder() int32 {
	var next int32
	step := func() int32 {
		next++
		return next
	}
	values := map[ArrayKey]int32{
		{step(), step()}: step(),
		{step(), step()}: step(),
	}
	return values[ArrayKey{1, 2}]*100 +
		values[ArrayKey{4, 5}]*10 +
		next
}
