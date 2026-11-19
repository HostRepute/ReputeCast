//go:build unix

package main

import (
	"log"
	"syscall"
)

// raiseFileLimit lifts the open files limit to n when it is lower (macOS
// starts shells at 256, under 200 streams and their messages).
func raiseFileLimit(n int) {
	var l syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &l); err != nil || l.Cur >= uint64(n) {
		return
	}
	want := uint64(n)
	if l.Max < want {
		want = l.Max
	}
	cur := l.Cur
	l.Cur = want
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &l); err != nil {
		log.Printf("open files limit %d, wanted %d: %v (run ulimit -n %d first)", cur, n, err, n)
	}
}
