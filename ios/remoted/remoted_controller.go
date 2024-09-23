package remoted

// Sends SIGSTOP to remoted process.
// Implemented only on Darwin systems, as remoted service is present only on Darwin systems
func StopRemoted() error {
	return stopRemoted()
}

// Sends SIGCONT to remoted process.
// Implemented only on Darwin systems, as remoted service is present only on Darwin systems
func ContinueRemoted() error {
	return continueRemoted()
}
