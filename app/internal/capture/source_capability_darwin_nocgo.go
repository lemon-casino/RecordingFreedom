//go:build darwin && !cgo

package capture

func sourceEnumerationCapability() Capability {
	return queued("source-enumeration", "Source Enumeration", "CoreGraphics", PermissionScreenRecording, "macOS source enumeration requires cgo for CoreGraphics.")
}
