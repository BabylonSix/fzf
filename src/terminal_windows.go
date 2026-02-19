//go:build windows

package fzf

import (
	"os"
)

func notifyOnResize(resizeChan chan<- os.Signal) {
	// TODO
}

func notifyOnUsr1(usr1Chan chan<- os.Signal) {
	// NOOP - SIGUSR1 not available on Windows
}

func notifyStop(p *os.Process) {
	// NOOP
}
