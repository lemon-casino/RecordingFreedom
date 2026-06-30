//go:build !windows && !darwin

package capture

func sourceEnumerationCapability() Capability {
	return queued("source-enumeration", "Source Enumeration", "XDG Desktop Portal + PipeWire", PermissionUnknown, "Linux source enumeration is queued behind the portal/PipeWire backend.")
}
