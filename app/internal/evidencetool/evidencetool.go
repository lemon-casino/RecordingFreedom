// Package evidencetool holds helpers shared by the OCR and smoke evidence
// tools in app/cmd. Every exported function is a verbatim convergence of
// identical copies that previously lived in the individual tools; behavior
// and output are unchanged.
package evidencetool

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"strings"
)

// ContainsAny reports whether content contains at least one of the terms
// (each term is matched lowercased against the content).
// Source: cmd/annotation-overlay-evidence-check, cmd/ocr-desktop-evidence-check.
func ContainsAny(content string, terms []string) bool {
	for _, term := range terms {
		if strings.Contains(content, strings.ToLower(term)) {
			return true
		}
	}
	return false
}

// FileSHA256 returns the hex-encoded SHA-256 digest of the file at path.
// Source: cmd/ocr-desktop-evidence-check, cmd/ocr-desktop-evidence-export.
func FileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// ClampInt constrains value to the inclusive range [minimum, maximum].
// Source: cmd/annotation-export-smoke, cmd/pip-export-smoke, cmd/ocr-worker (identical copies).
func ClampInt(value int, minimum int, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

// AbsInt returns the absolute value of value.
// Source: cmd/annotation-export-smoke, cmd/pip-export-smoke.
func AbsInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
