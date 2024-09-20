//go:build !darwin

package remoted

// As remoted is present only on Darwin systems, default implementation is empty

func stopRemoted() error {
	return nil
}

func continueRemoted() error {
	return nil
}
