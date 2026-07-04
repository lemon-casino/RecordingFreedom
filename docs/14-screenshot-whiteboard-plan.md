# Screenshot, Pin, and Whiteboard Plan

## Goal

Add a durable screenshot tool group to the capsule recorder. When recording is not active, screenshot and whiteboard live under one secondary capsule menu. During video recording, the existing recording annotation whiteboard stays unchanged.

## Implemented Slice

- Region screenshot uses the native Go desktop screenshot backend and the existing region selector overlay.
- Screenshots are stored under `data/screenshots`.
- `history.json` persists the screenshot history with id, file paths, dimensions, capture mode, region, pinned state, and fixed state.
- Screenshot history is visible in the capsule screenshot/whiteboard menu.
- A screenshot can be opened from history.
- A screenshot can be opened as a whiteboard background for annotation.
- A screenshot can be pinned into its own always-on-top transparent window.
- Fixed state is persisted with the history and forces a pinned screenshot to remain pinned.
- Screenshot and whiteboard both have configurable global shortcuts.
- Hover tooltips for shortcut-backed buttons include their shortcuts.

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
- `items[].pinned`
- `items[].fixed`

## Native Capture

The first native backend uses `github.com/kbinani/screenshot` and captures a selected desktop rectangle as PNG. The region selector tracks both the Wails overlay bounds and the native screenshot capture bounds, then maps the selected overlay rectangle to native capture coordinates before saving.

## Scrolling Screenshot

Scrolling screenshot is intentionally not faked. The UI and API entry exist, but the desktop build reports it as unavailable until platform scroll automation is implemented.

Required native path:

- Windows: UI Automation scroll pattern + repeated region capture + image stitching.
- macOS: Accessibility API scroll actions + ScreenCaptureKit or native screenshot region capture + image stitching.
- Linux: AT-SPI scroll actions where available, with X11/Wayland capability detection.

Acceptance:

- User can select a scrollable target area.
- The app scrolls the target in controlled increments.
- Duplicate overlapping content is removed during stitching.
- The final image is saved as one PNG and appears in screenshot history.
- If a platform cannot control the target window, the app returns a clear unsupported reason without creating a fake history item.
