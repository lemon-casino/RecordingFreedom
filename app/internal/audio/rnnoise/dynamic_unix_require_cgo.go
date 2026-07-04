//go:build rnnoise_dynamic && !cgo && (darwin || linux)

package rnnoise

// Unix dynamic RNNoise uses dlopen/dlsym through cgo. Release builds must keep CGO_ENABLED=1.
var _ = rnnoise_dynamic_requires_CGO_ENABLED_1_on_unix
