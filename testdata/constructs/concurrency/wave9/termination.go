package wave9concurrency

func PanicSendClosed() {
	values := make(chan int32)
	close(values)
	values <- 1
}

func PanicCloseNil() {
	var values chan int32
	close(values)
}

func PanicCloseClosed() {
	values := make(chan int32)
	close(values)
	close(values)
}

func DeadlockNilReceive() {
	var values chan int32
	<-values
}

func PanicGoroutine() {
	values := make(chan int32)
	close(values)
	go func() {
		values <- 1
	}()
	var blocked chan int32
	<-blocked
}

func ReturnWithBlockedGoroutine() {
	var blocked chan int32
	go func() {
		<-blocked
	}()
}
