//go:build darwin

package remoted

import (
	"syscall"

	"github.com/pbar1/pkill-go"
)

func stopRemoted() error {
	_, err := pkill.Pkill("remoted", syscall.SIGSTOP)
	return err
}

func continueRemoted() error {
	_, err := pkill.Pkill("remoted", syscall.SIGCONT)
	return err
}
