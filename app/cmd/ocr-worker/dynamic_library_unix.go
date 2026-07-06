//go:build darwin || linux

package main

import "github.com/ebitengine/purego"

func openDynamicLibrary(path string) (uintptr, error) {
	return purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_LOCAL)
}

func closeDynamicLibrary(handle uintptr) {
	if handle == 0 {
		return
	}
	_ = purego.Dlclose(handle)
}
