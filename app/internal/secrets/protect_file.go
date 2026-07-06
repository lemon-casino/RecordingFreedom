//go:build !windows

package secrets

import "errors"

func diskBackendName() string {
	return "local-file-0600"
}

func protect(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("secret data is empty")
	}
	out := make([]byte, len(data))
	copy(out, data)
	return out, nil
}

func unprotect(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("protected secret data is empty")
	}
	out := make([]byte, len(data))
	copy(out, data)
	return out, nil
}
