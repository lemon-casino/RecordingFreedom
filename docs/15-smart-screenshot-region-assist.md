# Smart Screenshot and Region Assist

## Goal

Improve screenshot, screenshot-backed whiteboard, recording annotation overlay, and custom region recording selection so the user can capture precise target areas instead of manually lining up every edge.

This document records the `mg-chao/snow-shot` reference analysis and the RecordingFreedom implementation boundary for this release.

## snow-shot Reference Analysis

Reference repository cloned for analysis:

- `mg-chao/snow-shot.git`
- inspected commit: `c7f2d9f feat: 编译离线版本 (#792)`

Relevant implementation areas:

- `src/pages/draw/components/selectLayer/index.tsx`
- `src-tauri/src-crates/app-os/src/ui_automation/windows.rs`
- `src/commands/screenshot.ts`
- `src/functions/screenshot.ts`

snow-shot's intelligent selection is not only an image-edge algorithm. The practical design is:

- Native screenshot/window capture prepares monitor and window bounding boxes.
- Windows has a deeper UI Automation path that can query child UI element rectangles below the pointer.
- The frontend select layer stores candidate rectangles in a Flatbush/RTree index and chooses the rectangle under the pointer.
- When child element data is unavailable, the select layer falls back to monitor/window rectangles.
- Focused-window and full-screen modes are separate commands, not just aliases of region selection.

The maintainable lesson for RecordingFreedom is to keep a single smart-candidate contract. Native platform element providers can feed that contract without changing the region overlay UI.

## Implemented In RecordingFreedom

Backend contract:

- `RegionSmartCandidate` describes screen, window, native UI element, or image-edge candidates.
- `RegionSelectionSession.candidates` carries static screen/window candidates to the overlay.
- `AssistRegionSelection()` is the shared dynamic recognition entry point for all region selectors.
- Hover recognition first asks the native element provider for the element chain under the pointer. The selected level can be cycled from child to parent with the mouse wheel, matching snow-shot's practical interaction model.
- If native element data is unavailable, hover recognition falls back to image-based point candidates inside the current window/parent bounds, then to window/screen candidates.
- Native element hits do not stop the rest of the hierarchy from being built. The hover chain is merged as native elements first, image-detected panels second, and window/static candidates last, so a user can wheel from a small control to a sidebar/panel/window when the OS provider exposes an incomplete element tree.
- Manual drag selection still uses `regionImageEdgeCandidate()` to snap nearby strong edges.
- A clean desktop screenshot is captured before the transparent overlay is shown and reused by the image recognizer. This avoids recognizing RecordingFreedom's own red border, dim layer, or crosshair.
- Multi-display coordinate conversion is display-aware through `RegionSelectionSession.displayBounds`; mixed-DPI desktops must not be converted with one virtual-desktop scale ratio.
- All selector entry points must be activated through the shared `showRegionSelectionSession()` helper. This keeps native element cache reset, clean snapshot preparation, overlay bounds, and `rf-region-session` dispatch identical for recording region, screenshot region, scrolling screenshot, and recording annotation region.
- `showRegionSelectionSession()` also records the current cursor as `RegionSelectionSession.initialPointer` when the pointer is inside the overlay bounds. The overlay immediately runs the same hover-assist path for that point, matching snow-shot's behavior where selection starts by highlighting the target under the cursor instead of waiting for the first mouse move.
- The overlay treats hover recognition as the primary path. It waits for the pending native/image assist request before showing the manual crosshair, so users see a snow-shot-style target highlight first instead of a full-screen red guide line flash.
- The overlay keeps snow-shot's auto-selection posture during small pointer jitter. A left-button press does not become a manual drag until the pointer moves more than 6 px; below that threshold, the current auto-recognized candidate stays selected.
- Right-click cancel must work during the whole recognition path, including while a hover assist request is still pending and before any candidate or manual crosshair is visible.
- Cancel paths must clear local React state and global runtime fields such as `window.__RF_REGION_SESSION__` and `window.__RF_LAST_REGION_ASSIST__`. A cancelled selector must not leave stale session data for the next selector invocation or browser fallback run.

Native providers:

- Windows uses UI Automation. It queries the top-level window under the point while filtering RecordingFreedom's own overlay process, then walks content/control/raw UIA trees to produce the smallest containing child first, followed by parent elements.
- macOS cgo builds use Accessibility/AX. They call `AXUIElementCopyElementAtPosition`, descend through `AXChildren` to find the smallest containing child under the pointer, walk parent elements for level cycling, and use CoreGraphics visible-window bounds as the window fallback.
- Linux and other non-native builds currently use the shared image/window fallback path.

Image fallback:

- `detectImageRegionsAroundPoint()` returns multiple candidates for the point instead of one rectangle.
- Panel detection handles UI regions that have only one strong divider and one strong top/bottom edge, such as Windows Explorer/Finder sidebars whose bottom edge is the parent window boundary.
- Panel boundary scoring treats long continuous faint dividers as usable boundaries, so white Explorer/Finder-style sidebars with subtle gray separators can be selected without hard-coded app-specific coordinates.
- Image candidates are sorted by confidence plus smaller-area preference so a sidebar/panel beats the whole parent window.

Selection surfaces using the same contract:

- Custom region recording: `ShowRegionSelector()`
- Region screenshot: `ShowScreenshotRegionSelector()`
- Scrolling screenshot target region: `StartScrollingScreenshot()`
- Recording annotation / screenshot annotation region: `ShowAnnotationRegionSelector()`

Screenshot modes now exposed from the screenshot/whiteboard menu:

- Region screenshot
- Full screenshot
- Window screenshot
- Focused-window screenshot
- Scrolling screenshot

Focused-window behavior:

- Windows uses the foreground HWND, ignores RecordingFreedom's own process, and prefers DWM extended frame bounds before falling back to `GetWindowRect`.
- macOS uses CoreGraphics visible window z-order, skips RecordingFreedom's own process, and captures the first usable front window bounds.
- Other platforms fall back to the normal enumerated window source until a native active-window provider is added.

Whiteboard and history behavior:

- Normal whiteboard manual save calls `SaveWhiteboardSnapshot()`.
- `SaveWhiteboardSnapshot()` saves the Excalidraw scene and writes a PNG snapshot into screenshot history with `mode: whiteboard`.
- Region screenshot annotation save writes the annotated PNG into screenshot history.
- Recording annotation save writes to the recording annotation package and clears the unsaved state when the content signature matches the last saved content.

## Current Limitations

- macOS element recognition requires Accessibility permission for the terminal/app process. Without permission, the app must fall back to image/window candidates and should log that no AX elements were available.
- Linux focused-window and child-element detection still need a native provider. A future X11/Wayland provider should feed the same `RegionSmartCandidate` interface.
- Image-edge snapping and panel detection are intentionally conservative. They prefer no precise match over a fake precise match on flat content.
- Full release validation still requires a real macOS run with Accessibility permission and visible Finder/browser controls under the cursor.

## Verification

Required checks for this slice:

- Go tests proving whiteboard snapshots write screenshot history.
- Go tests proving smart image-edge snapping finds a nearby bordered element.
- Go tests proving image hover can recognize an Explorer-like sidebar/panel without a bottom border.
- Go tests proving image hover can recognize an Explorer-like sidebar/panel with a long faint divider.
- Go tests proving image panel candidates remain reachable by level cycling even when a native element candidate exists.
- Go tests proving per-display coordinate conversion for mixed-DPI, negative-origin, and seam points.
- Go tests proving native Windows UIA / macOS AX candidate normalization returns the smallest containing element first.
- Go tests proving `focused-window` mode uses the focused-window provider.
- Frontend e2e proving shortcut/settings still persist after adding screenshot/whiteboard controls.
- Frontend e2e proving whiteboard and annotation overlay save states are stable after save.
- Frontend e2e proving region hover candidates can cycle child/parent/image-panel levels and all selector purposes share the dynamic recognition path.
- Frontend e2e proving the region overlay recognizes `initialPointer` before the first mouse move.
- Frontend e2e proving the manual crosshair is not shown while hover recognition is pending.
- Frontend e2e proving small pointer jitter keeps auto recognition instead of drawing a manual rectangle.
- Frontend e2e proving right-click cancels while hover recognition is pending.
- Frontend e2e proving right-click cancel clears stale global region session and assist data.
- `RECORDINGFREEDOM_REGION_PROBE=cursor go test -run 'TestRegion(UIElement|Accessibility)ProbeFromEnv' -v .` on a real desktop point.
- `RECORDINGFREEDOM_REGION_PROBE=cursor go test -run TestRegionAssistDesktopProbeFromEnv -v .` on the same point; passing output should show `source=element` where the platform exposes a real UI element.
- macOS convenience verifier: `bash scripts/verify-region-recognition-macos.sh`. This verifier requires `CGO_ENABLED=1` and hard-fails unless the output proves both a native `source=accessibility:` candidate chain and `region-assist source=element`.
- `npm run build`
- `go test ./...`
- `git diff --check`
