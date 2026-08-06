package externallinked

import "golang.org/x/sys/unix"

func Run() uintptr {
	result, _, _ := unix.Syscall(0, 0, 0, 0)
	return result
}
