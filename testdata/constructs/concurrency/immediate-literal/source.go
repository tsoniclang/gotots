package immediateliteral

var immediateLiteralPackageValue = func() string {
	return "literal"
}()

func cooperativeStringProvider() string {
	values := make(chan string, 1)
	values <- "provider"
	return <-values
}

func immediateCooperativeLiteral() string {
	return (func() string {
		values := make(chan string, 1)
		values <- "immediate"
		return <-values
	})()
}

func cooperativeVoidProvider() {
	values := make(chan struct{}, 1)
	values <- struct{}{}
	<-values
}

func deferredLiteralABIIsolation() (result string) {
	provider := cooperativeVoidProvider
	provider()
	result = "body"
	defer func() {
		result += "deferred"
	}()
	return
}

func ImmediateLiteralABIIsolation() string {
	provider := cooperativeStringProvider
	return immediateLiteralPackageValue +
		immediateCooperativeLiteral() +
		provider() +
		deferredLiteralABIIsolation()
}
