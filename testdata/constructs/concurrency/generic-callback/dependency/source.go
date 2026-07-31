package dependency

var Value = build()

var Values = make(chan int32, 1)

func init() {
	Values <- 40
}

func build() int32 {
	return 7
}
