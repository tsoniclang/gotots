package lexicalmap

var packageLiteral = func() int32 {
	type Key [2]int32
	values := map[Key]int32{{1, 2}: 12}
	return values[Key{1, 2}]
}()

func PackageVariableLiteral() int32 {
	return packageLiteral
}

func NestedBlock() int32 {
	var result int32
	{
		type Key [2]int32
		values := map[Key]int32{{3, 4}: 34}
		result = values[Key{3, 4}]
	}
	return result
}

func NestedFunctionLiteral() int32 {
	run := func() int32 {
		type Key [2]int32
		values := map[Key]int32{{5, 6}: 56}
		return values[Key{5, 6}]
	}
	return run()
}
