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
	"fmt"
	"strings"
	"unsafe"
)

const keychainServiceName = "RecordingFreedom OCR Translation"

func backendStatus(s *Store) (Status, error) {
	if _, err := s.rootDir(); err != nil {
		return Status{}, err
	}
	return Status{Backend: backendName(), Dir: "macOS Keychain"}, nil
}

func backendName() string {
	return "macos-keychain"
}

func backendSave(s *Store, name string, secret string) error {
	account := safeName(name)
	item, found, err := findKeychainItem(account)
	if err != nil {
		return err
	}
	secretC := C.CString(secret)
	defer C.free(unsafe.Pointer(secretC))
	if found {
		defer C.CFRelease(C.CFTypeRef(item))
		status := C.SecKeychainItemModifyAttributesAndData(item, nil, C.UInt32(len(secret)), unsafe.Pointer(secretC))
		if status != C.errSecSuccess {
			return keychainError("update secret", status)
		}
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
		return keychainError("save secret", status)
	}
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
		return "", false, nil
	}
	if status != C.errSecSuccess {
		return "", false, keychainError("load secret", status)
	}
	defer C.SecKeychainItemFreeContent(nil, passwordData)
	secret := strings.TrimSpace(string(C.GoBytes(passwordData, C.int(passwordLength))))
	return secret, secret != "", nil
}

func backendDelete(s *Store, name string) error {
	item, found, err := findKeychainItem(safeName(name))
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	defer C.CFRelease(C.CFTypeRef(item))
	status := C.SecKeychainItemDelete(item)
	if status == C.errSecItemNotFound {
		return nil
	}
	if status != C.errSecSuccess {
		return keychainError("delete secret", status)
	}
	return nil
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

func keychainError(operation string, status C.OSStatus) error {
	return fmt.Errorf("macOS Keychain %s failed with OSStatus %d", operation, int(status))
}
