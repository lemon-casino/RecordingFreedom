//go:build darwin && cgo

package capture

func sourceEnumerationCapability() Capability {
	return available("source-enumeration", "Source Enumeration", "CoreGraphics", PermissionScreenRecording, "CoreGraphics display and visible window enumeration is implemented; ScreenCaptureKit target mapping is still queued.")
}
