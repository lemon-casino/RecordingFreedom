# Screenshot, Pin, and Whiteboard Plan

## Goal

Add a durable screenshot tool group to the capsule recorder. When recording is not active, screenshot and whiteboard live under one secondary capsule menu. During video recording, the existing recording annotation whiteboard stays unchanged.

## Implemented Slice

- Region screenshot uses the native Go desktop screenshot backend and the existing region selector overlay.
- Screenshots are stored under `data/screenshots`.
- `history.json` persists the screenshot history with id, file paths, dimensions, capture mode, region, and fixed state. The legacy `pinned` field is normalized to `false` and is not used as the current pin-window state.
- Screenshot history is visible in the capsule screenshot/whiteboard menu.
- A screenshot can be opened from history.
- A screenshot can be opened as a whiteboard background for annotation.
- A screenshot can be pinned into its own always-on-top transparent window with the real screenshot image data.
- Fixed state is persisted with the history and can reopen the same screenshot as a fixed pin; current pin visibility is stored separately from history so stale history rows do not appear as already pinned.
- Opening a screenshot in the normal whiteboard imports it as an Excalidraw image element and file. If the whiteboard is already open, a live `screenshot.whiteboard` event imports the new screenshot without requiring a window restart.
- Screenshot-backed whiteboard import is guarded against the startup empty-scene save path, so imported screenshots are not overwritten by an empty board.
- Screenshot and whiteboard both have configurable global shortcuts.
- Hover tooltips for shortcut-backed buttons include their shortcuts.
- Normal whiteboard manual save now saves the Excalidraw scene and also writes a PNG snapshot into screenshot history with `mode: whiteboard`.
- Screenshot capture modes now include region, full, window, focused-window, and scrolling screenshot entries from the capsule screenshot/whiteboard menu.
- Region selection sessions carry smart screen/window/edge candidates so screenshots, scrolling screenshots, annotation overlay, and custom region recording can share the same boundary detection path.

## Data Model

`data/screenshots/history.json`:

- `schemaVersion`
- `items[]`
- `items[].id`
- `items[].path`
- `items[].thumbnailPath`
- `items[].createdAt`
- `items[].width`
- `items[].height`
- `items[].mode`
- `items[].region`
- `items[].pinned`: legacy compatibility field; normalized to `false` on load/save and never used as the live pin state.
- `items[].fixed`: persisted fixed-pin preference.

Live pin-window state is kept separately from screenshot history and contains:

- `visible`
- `item`
- `dataUrl`
- `fixed`

## Native Capture

The first native backend uses `github.com/kbinani/screenshot` and captures a selected desktop rectangle as PNG. The region selector tracks both the Wails overlay bounds and the native screenshot capture bounds, then maps the selected overlay rectangle to native capture coordinates before saving.

`focused-window` is handled separately from normal `window` screenshots. Windows uses the foreground HWND and DWM frame bounds where available; macOS uses CoreGraphics visible window z-order and skips RecordingFreedom's own process; unsupported platforms fall back to the existing enumerated window source.

Smart region assist is documented in [15-smart-screenshot-region-assist.md](15-smart-screenshot-region-assist.md).

## Scrolling Screenshot

Scrolling screenshot is implemented through the same native region-selection overlay used by screenshots. The user selects a target region, the app hides the capsule/overlay, captures repeated native frames, sends controlled platform scroll input, detects the overlap between adjacent frames, stitches only the new content into a single PNG, and saves it to screenshot history with `mode: scrolling`. If the selected area does not move after scroll input, the app saves the first captured frame as a normal `mode: region` screenshot instead of failing or creating a fake long image.

Native scroll path:

- Windows: `SendInput` mouse-wheel events at the center of the selected region.
- macOS: `CGEventCreateScrollWheelEvent` posted at the selected region center. The user may need to grant Accessibility permission for controlled scrolling.
- Linux: `xdotool` on X11. Wayland or missing `xdotool` returns a clear unsupported reason and does not save a fake history item.

Acceptance:

- User can select a scrollable target area.
- The app scrolls the target in controlled increments.
- Duplicate overlapping content is removed during stitching.
- The final image is saved as one PNG and appears in screenshot history.
- If a platform cannot control the target window, the app returns a clear unsupported reason without creating a fake history item.
- If the selected content does not move after scroll input, the app saves the selected area as a normal region screenshot.

## v0.1.10 Regression Fixes

- Screenshot history starts in an unpinned state even if an older history file contains `pinned: true`.
- Pinning a screenshot opens a real image pin state with `dataUrl`, not a fake row marker.
- Fixed screenshots preserve `fixed: true` while still keeping history `pinned: false`.
- Screenshot-to-whiteboard works on initial whiteboard load and while the whiteboard window is already open.
- Browser-preview fallbacks use the same history normalization and screenshot-whiteboard event shape as the desktop backend.
