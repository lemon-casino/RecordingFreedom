//go:build darwin

package secrets

/*
#cgo LDFLAGS: -framework Security -framework CoreFoundation
#include <Security/Security.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>

static CFStringRef rf_secitem_string(UInt32 length, const char *value) {
	return CFStringCreateWithBytes(kCFAllocatorDefault, (const UInt8 *)value, (CFIndex)length, kCFStringEncodingUTF8, 0);
}

static CFMutableDictionaryRef rf_secitem_query(
	UInt32 serviceLength,
	const char *serviceName,
	UInt32 accountLength,
	const char *accountName
) {
	CFStringRef service = rf_secitem_string(serviceLength, serviceName);
	CFStringRef account = rf_secitem_string(accountLength, accountName);
	if (service == NULL || account == NULL) {
		if (service != NULL) {
			CFRelease(service);
		}
		if (account != NULL) {
			CFRelease(account);
		}
		return NULL;
	}
	CFMutableDictionaryRef query = CFDictionaryCreateMutable(
		kCFAllocatorDefault,
		0,
		&kCFTypeDictionaryKeyCallBacks,
		&kCFTypeDictionaryValueCallBacks
	);
	if (query == NULL) {
		CFRelease(service);
		CFRelease(account);
		return NULL;
	}
	CFDictionarySetValue(query, kSecClass, kSecClassGenericPassword);
	CFDictionarySetValue(query, kSecAttrService, service);
	CFDictionarySetValue(query, kSecAttrAccount, account);
	CFRelease(service);
	CFRelease(account);
	return query;
}

static OSStatus rf_secitem_add_secret(
	UInt32 serviceLength,
	const char *serviceName,
	UInt32 accountLength,
	const char *accountName,
	UInt32 secretLength,
	const void *secretData
) {
	CFMutableDictionaryRef item = rf_secitem_query(serviceLength, serviceName, accountLength, accountName);
	if (item == NULL) {
		return errSecParam;
	}
	CFDataRef data = CFDataCreate(kCFAllocatorDefault, (const UInt8 *)secretData, (CFIndex)secretLength);
	if (data == NULL) {
		CFRelease(item);
		return errSecParam;
	}
	CFDictionarySetValue(item, kSecValueData, data);
	OSStatus status = SecItemAdd(item, NULL);
	CFRelease(data);
	CFRelease(item);
	return status;
}

static OSStatus rf_secitem_update_secret(
	UInt32 serviceLength,
	const char *serviceName,
	UInt32 accountLength,
	const char *accountName,
	UInt32 secretLength,
	const void *secretData
) {
	CFMutableDictionaryRef query = rf_secitem_query(serviceLength, serviceName, accountLength, accountName);
	if (query == NULL) {
		return errSecParam;
	}
	CFMutableDictionaryRef attrs = CFDictionaryCreateMutable(
		kCFAllocatorDefault,
		0,
		&kCFTypeDictionaryKeyCallBacks,
		&kCFTypeDictionaryValueCallBacks
	);
	if (attrs == NULL) {
		CFRelease(query);
		return errSecParam;
	}
	CFDataRef data = CFDataCreate(kCFAllocatorDefault, (const UInt8 *)secretData, (CFIndex)secretLength);
	if (data == NULL) {
		CFRelease(attrs);
		CFRelease(query);
		return errSecParam;
	}
	CFDictionarySetValue(attrs, kSecValueData, data);
	OSStatus status = SecItemUpdate(query, attrs);
	CFRelease(data);
	CFRelease(attrs);
	CFRelease(query);
	return status;
}

static OSStatus rf_secitem_load_secret(
	UInt32 serviceLength,
	const char *serviceName,
	UInt32 accountLength,
	const char *accountName,
	CFDataRef *secretData
) {
	CFMutableDictionaryRef query = rf_secitem_query(serviceLength, serviceName, accountLength, accountName);
	if (query == NULL) {
		return errSecParam;
	}
	CFDictionarySetValue(query, kSecReturnData, kCFBooleanTrue);
	CFDictionarySetValue(query, kSecMatchLimit, kSecMatchLimitOne);
	CFTypeRef result = NULL;
	OSStatus status = SecItemCopyMatching(query, &result);
	CFRelease(query);
	if (status != errSecSuccess) {
		return status;
	}
	if (result == NULL || CFGetTypeID(result) != CFDataGetTypeID()) {
		if (result != NULL) {
			CFRelease(result);
		}
		return errSecInternalComponent;
	}
	*secretData = (CFDataRef)result;
	return errSecSuccess;
}

static OSStatus rf_secitem_delete_secret(
	UInt32 serviceLength,
	const char *serviceName,
	UInt32 accountLength,
	const char *accountName
) {
	CFMutableDictionaryRef query = rf_secitem_query(serviceLength, serviceName, accountLength, accountName);
	if (query == NULL) {
		return errSecParam;
	}
	OSStatus status = SecItemDelete(query);
	CFRelease(query);
	return status;
}

static CFIndex rf_cfdata_length(CFDataRef data) {
	return CFDataGetLength(data);
}

static const UInt8 *rf_cfdata_bytes(CFDataRef data) {
	return CFDataGetBytePtr(data);
}

static void rf_release_cf(CFTypeRef item) {
	if (item != NULL) {
		CFRelease(item);
	}
}
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
	status := saveKeychainSecret(safeName(name), secret)
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

	var secretData C.CFDataRef
	status := C.rf_secitem_load_secret(
		C.UInt32(len(keychainServiceName)),
		serviceC,
		C.UInt32(len(account)),
		accountC,
		&secretData,
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
	defer C.rf_release_cf(C.CFTypeRef(secretData))

	length := C.rf_cfdata_length(secretData)
	if length <= 0 {
		return "", false, nil
	}
	secret := strings.TrimSpace(string(C.GoBytes(unsafe.Pointer(C.rf_cfdata_bytes(secretData)), C.int(length))))
	return secret, secret != "", nil
}

func backendDelete(s *Store, name string) error {
	account := safeName(name)
	serviceC := C.CString(keychainServiceName)
	accountC := C.CString(account)
	defer C.free(unsafe.Pointer(serviceC))
	defer C.free(unsafe.Pointer(accountC))

	status := C.rf_secitem_delete_secret(
		C.UInt32(len(keychainServiceName)),
		serviceC,
		C.UInt32(len(account)),
		accountC,
	)
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

func saveKeychainSecret(account string, secret string) C.OSStatus {
	status := updateKeychainSecret(account, secret)
	if status == C.errSecItemNotFound {
		status = addKeychainSecret(account, secret)
		if status == C.errSecDuplicateItem {
			status = updateKeychainSecret(account, secret)
		}
	}
	return status
}

func updateKeychainSecret(account string, secret string) C.OSStatus {
	serviceC, accountC, secretC, cleanup := keychainCStringArgs(account, secret)
	defer cleanup()
	return C.rf_secitem_update_secret(
		C.UInt32(len(keychainServiceName)),
		serviceC,
		C.UInt32(len(account)),
		accountC,
		C.UInt32(len(secret)),
		unsafe.Pointer(secretC),
	)
}

func addKeychainSecret(account string, secret string) C.OSStatus {
	serviceC, accountC, secretC, cleanup := keychainCStringArgs(account, secret)
	defer cleanup()
	return C.rf_secitem_add_secret(
		C.UInt32(len(keychainServiceName)),
		serviceC,
		C.UInt32(len(account)),
		accountC,
		C.UInt32(len(secret)),
		unsafe.Pointer(secretC),
	)
}

func keychainCStringArgs(account string, secret string) (*C.char, *C.char, *C.char, func()) {
	serviceC := C.CString(keychainServiceName)
	accountC := C.CString(account)
	secretC := C.CString(secret)
	return serviceC, accountC, secretC, func() {
		C.free(unsafe.Pointer(serviceC))
		C.free(unsafe.Pointer(accountC))
		C.free(unsafe.Pointer(secretC))
	}
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
