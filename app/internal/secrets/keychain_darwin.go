//go:build darwin

package secrets

/*
#cgo LDFLAGS: -framework Security -framework CoreFoundation
#include <Security/Security.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>
*/
import "C"

import (
	"errors"
	"fmt"
	"strings"
	"unsafe"
)

const (
	keychainServiceName  = "RecordingFreedom OCR Translation"
	keychainFallbackName = "macos-keychain+local-file-0600-fallback"

	errSecUserCanceled          = -128
	errSecNotAvailable          = -25291
	errSecAuthFailed            = -25293
	errSecNoDefaultKeychain     = -25307
	errSecInteractionNotAllowed = -25308
)

func backendStatus(s *Store) (Status, error) {
	fallbackDir, err := diskDir(s)
	if err != nil {
		return Status{}, err
	}
	return Status{Backend: keychainFallbackName, Dir: "macOS Keychain; fallback=" + fallbackDir}, nil
}

func backendName() string {
	return "macos-keychain"
}

func backendSave(s *Store, name string, secret string) error {
	account := safeName(name)
	item, found, err := findKeychainItem(account)
	if err != nil {
		if isKeychainUnavailable(err) {
			return diskSave(s, name, secret)
		}
		return err
	}
	secretC := C.CString(secret)
	defer C.free(unsafe.Pointer(secretC))
	if found {
		defer C.CFRelease(C.CFTypeRef(item))
		status := C.SecKeychainItemModifyAttributesAndData(item, nil, C.UInt32(len(secret)), unsafe.Pointer(secretC))
		if status != C.errSecSuccess {
			err := keychainError("update secret", status)
			if isKeychainUnavailable(err) {
				return diskSave(s, name, secret)
			}
			return err
		}
		_ = diskDelete(s, name)
		return nil
	}
	serviceC := C.CString(keychainServiceName)
	accountC := C.CString(account)
	defer C.free(unsafe.Pointer(serviceC))
	defer C.free(unsafe.Pointer(accountC))
	status := C.SecKeychainAddGenericPassword(
		nil,
		C.UInt32(len(keychainServiceName)),
		serviceC,
		C.UInt32(len(account)),
		accountC,
		C.UInt32(len(secret)),
		unsafe.Pointer(secretC),
		nil,
	)
	if status != C.errSecSuccess {
		err := keychainError("save secret", status)
		if isKeychainUnavailable(err) {
			return diskSave(s, name, secret)
		}
		return err
	}
	_ = diskDelete(s, name)
	return nil
}

func backendLoad(s *Store, name string) (string, bool, error) {
	account := safeName(name)
	serviceC := C.CString(keychainServiceName)
	accountC := C.CString(account)
	defer C.free(unsafe.Pointer(serviceC))
	defer C.free(unsafe.Pointer(accountC))
	var passwordLength C.UInt32
	var passwordData unsafe.Pointer
	status := C.SecKeychainFindGenericPassword(
		nil,
		C.UInt32(len(keychainServiceName)),
		serviceC,
		C.UInt32(len(account)),
		accountC,
		&passwordLength,
		&passwordData,
		nil,
	)
	if status == C.errSecItemNotFound {
		return diskLoad(s, name)
	}
	if status != C.errSecSuccess {
		err := keychainError("load secret", status)
		if isKeychainUnavailable(err) {
			return diskLoad(s, name)
		}
		return "", false, err
	}
	defer C.SecKeychainItemFreeContent(nil, passwordData)
	secret := strings.TrimSpace(string(C.GoBytes(passwordData, C.int(passwordLength))))
	return secret, secret != "", nil
}

func backendDelete(s *Store, name string) error {
	item, found, err := findKeychainItem(safeName(name))
	if err != nil {
		if isKeychainUnavailable(err) {
			return diskDelete(s, name)
		}
		return err
	}
	if !found {
		return diskDelete(s, name)
	}
	defer C.CFRelease(C.CFTypeRef(item))
	status := C.SecKeychainItemDelete(item)
	if status == C.errSecItemNotFound {
		return diskDelete(s, name)
	}
	if status != C.errSecSuccess {
		err := keychainError("delete secret", status)
		if isKeychainUnavailable(err) {
			return diskDelete(s, name)
		}
		return err
	}
	return diskDelete(s, name)
}

func findKeychainItem(account string) (C.SecKeychainItemRef, bool, error) {
	serviceC := C.CString(keychainServiceName)
	accountC := C.CString(account)
	defer C.free(unsafe.Pointer(serviceC))
	defer C.free(unsafe.Pointer(accountC))
	var item C.SecKeychainItemRef
	status := C.SecKeychainFindGenericPassword(
		nil,
		C.UInt32(len(keychainServiceName)),
		serviceC,
		C.UInt32(len(account)),
		accountC,
		nil,
		nil,
		&item,
	)
	if status == C.errSecItemNotFound {
		return nil, false, nil
	}
	if status != C.errSecSuccess {
		return nil, false, keychainError("find secret", status)
	}
	return item, true, nil
}

type keychainStatusError struct {
	operation string
	status    int
}

func (e keychainStatusError) Error() string {
	return fmt.Sprintf("macOS Keychain %s failed with OSStatus %d", e.operation, e.status)
}

func keychainError(operation string, status C.OSStatus) error {
	return keychainStatusError{operation: operation, status: int(status)}
}

func isKeychainUnavailable(err error) bool {
	if err == nil {
		return false
	}
	var statusErr keychainStatusError
	if !errors.As(err, &statusErr) {
		return false
	}
	switch statusErr.status {
	case errSecNotAvailable,
		errSecNoDefaultKeychain,
		errSecInteractionNotAllowed,
		errSecAuthFailed,
		errSecUserCanceled:
		return true
	default:
		return false
	}
}
