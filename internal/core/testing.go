package core

// Reset forces singletons to be recreated. Useful between unit tests.
func Reset() {
	collectionOnce.Reset()
	configOnce.Reset()
	dbClientOnce.Reset()
	dbOnce.Reset()
}
