//go:build rnnoise_native && !cgo

package rnnoise

// Building with rnnoise_native must never silently fall back to the no-op implementation.
var _ = rnnoise_native_requires_CGO_ENABLED_1
