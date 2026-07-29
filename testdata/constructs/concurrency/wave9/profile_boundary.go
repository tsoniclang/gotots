package wave9concurrency

func RequiresPreemption() {
	started := make(chan struct{})
	go func() {
		close(started)
		for {
		}
	}()
	<-started
}
