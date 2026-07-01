//go:build rnnoise_native && !cgo

package native

// Building with rnnoise_native must never silently skip the native RNNoise wrapper.
var _ = rnnoise_native_requires_CGO_ENABLED_1
