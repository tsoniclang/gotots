package boundary

func Print() {
	print(1, "value")
	println(2, "line")
}

func DeferredPrint() {
	defer print(3)
}

func Println() {
	println(4)
}
