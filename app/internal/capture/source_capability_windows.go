//go:build windows

package capture

func sourceEnumerationCapability() Capability {
	return available("source-enumeration", "Source Enumeration", "Win32 display/window APIs", PermissionNotRequired, "Windows displays, visible windows, and process groups are enumerated.")
}
