//go:build !windows

package fzf

import (
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sys/unix"
)

func notifyOnResize(resizeChan chan<- os.Signal) {
	signal.Notify(resizeChan, syscall.SIGWINCH)
}

func notifyOnUsr1(usr1Chan chan<- os.Signal) {
	signal.Notify(usr1Chan, syscall.SIGUSR1)
}

func notifyStop(p *os.Process) {
	pid := p.Pid
	pgid, err := unix.Getpgid(pid)
	if err == nil {
		pid = pgid * -1
	}
	unix.Kill(pid, syscall.SIGTSTP)
}
