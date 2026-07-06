//go:build windows

package secrets

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"
)

func diskBackendName() string {
	return "windows-dpapi"
}

func protect(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("secret data is empty")
	}
	name, err := windows.UTF16PtrFromString("RecordingFreedom OCR translation secret")
	if err != nil {
		return nil, err
	}
	input := windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
	var output windows.DataBlob
	if err := windows.CryptProtectData(&input, name, nil, 0, nil, 0, &output); err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(output.Data)))
	return copyBlob(output), nil
}

func unprotect(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("protected secret data is empty")
	}
	input := windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
	var output windows.DataBlob
	if err := windows.CryptUnprotectData(&input, nil, nil, 0, nil, 0, &output); err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(output.Data)))
	return copyBlob(output), nil
}

func copyBlob(blob windows.DataBlob) []byte {
	if blob.Data == nil || blob.Size == 0 {
		return nil
	}
	view := unsafe.Slice(blob.Data, int(blob.Size))
	out := make([]byte, len(view))
	copy(out, view)
	return out
}
