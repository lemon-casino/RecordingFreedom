package main

import "syscall"

func openDynamicLibrary(path string) (uintptr, error) {
	handle, err := syscall.LoadLibrary(path)
	return uintptr(handle), err
}

func closeDynamicLibrary(handle uintptr) {
	if handle == 0 {
		return
	}
	_ = syscall.FreeLibrary(syscall.Handle(handle))
}
