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

- `RegionSmartCandidate` describes screen, window, or image-edge candidates.
- `RegionSelectionSession.candidates` carries smart candidates to the overlay.
- `AssistRegionSelection()` scores candidates by overlap, edge closeness, and area ratio.
- `regionImageEdgeCandidate()` captures a padded local screenshot around the selected rectangle and snaps the selection to nearby strong vertical/horizontal edges.

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

- Windows UI Automation child-control rectangles are not yet integrated. The new `RegionSmartCandidate` contract is the stable target for that provider.
- macOS Accessibility child-element rectangles are not yet integrated. Current macOS focused-window and window candidates come from CoreGraphics window bounds.
- Linux focused-window detection currently uses the existing enumerated window fallback. A future X11/Wayland provider should feed the same smart-candidate interface.
- Image-edge snapping is intentionally conservative: it only applies when at least two nearby strong edges are found, so it should not create surprising fake precision on flat content.

## Verification

Required checks for this slice:

- Go tests proving whiteboard snapshots write screenshot history.
- Go tests proving smart image-edge snapping finds a nearby bordered element.
- Go tests proving `focused-window` mode uses the focused-window provider.
- Frontend e2e proving shortcut/settings still persist after adding screenshot/whiteboard controls.
- Frontend e2e proving whiteboard and annotation overlay save states are stable after save.
- `npm run build`
- `go test ./...`
- `git diff --check`
