# RecordingFreedom

RecordingFreedom is a new screen recorder rewrite built with Go, React, and Wails v3.

This directory is intended to become the root of the new repository:

```text
https://github.com/lemon-casino/RecordingFreedom.git
```

The legacy LikelySnap/Electron project is only a reference source. RecordingFreedom does not inherit the Electron runtime, browser-first recording pipeline, old theme, or old release workflow.

## Current Status

- Project analysis and implementation plan are in [docs](docs/README.md).
- The Wails v3 app is in [app](app/README.md).
- The first capsule recorder UI shell and system tray entry are implemented.
- The Go backend has app data, persistent settings, capture source/media device contracts, typed `.rfrec` package services, and mock recording package creation.
- The recording backend selector defaults to `mock-package`; native backend IDs are wired as queued contracts until real capture lands.
- Mock recording packages are created under the managed `data/video` structure.
- GitHub Actions are scaffolded in [.github/workflows](.github/workflows).

The current build is a UI and architecture milestone. Native capture backends are still staged work.

## Requirements

- Go `1.25.x`
- Node `24.x`
- npm
- Wails v3 CLI:

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha2.109
```

On Linux, Wails needs GTK/WebKit build dependencies. See `.github/workflows/ci.yml` for the current runner packages.

## Development

Install frontend dependencies:

```bash
cd app/frontend
npm install
```

Generate Wails bindings after changing Go services:

```bash
cd app
wails3 generate bindings -ts -i
```

Run frontend-only preview:

```bash
cd app/frontend
npm run dev
```

Run the Wails desktop app:

```bash
cd app
wails3 dev
```

The desktop shell creates a tray icon. Clicking the tray icon toggles the capsule recorder window; right-click opens a menu with show, hide, and quit actions.

## Icons

Regenerate all preview icon sizes from a source image:

```bash
cd app
go run ./cmd/icon-build -source "D:\图库\43574409.png" -sizes "16,24,32,48,64,128,256,512,1024"
```

The command writes `build/icons/icon-*.png`, updates the Wails source icon at `build/appicon.png`, updates the frontend favicon, and regenerates the Windows `.ico` and macOS `.icns` files. Use `-skip-wails` when Wails CLI is not installed and only PNG outputs are needed.

Full replacement steps are documented in [docs/07-icon-workflow.md](docs/07-icon-workflow.md).

## Verification

From `app/frontend`:

```bash
npm run build
```

From `app`:

```bash
go test ./...
go run ./cmd/preview-smoke
go run ./cmd/audio-smoke -duration=3s -keep
wails3 build
```

`go run ./cmd/preview-smoke -keep` keeps the generated temporary data root so the `.rfrec` package can be inspected manually.

`go run ./cmd/audio-smoke -duration=3s -keep` records real platform audio into a temporary `data/video/audio-smoke-*.rfrec/` folder. On Windows today, microphone capture writes `microphone.wav` and `audio-diagnostics.json`; RNNoise requires a cgo-enabled build and a C compiler.

To test and smoke the RNNoise path on a machine with a C toolchain:

```bash
go test -tags rnnoise_native ./internal/audio/rnnoise/native
go run -tags rnnoise_native ./cmd/audio-smoke -duration=3s -rnnoise -keep
```

## Preview Release

After `RecordingFreedom/` becomes the new repository root, pushing a `v*` tag publishes a GitHub Release with Windows, macOS, and Linux preview binaries plus SHA256 files:

```bash
git tag v0.1.0-preview.5
git push origin v0.1.0-preview.5
```

Preview tags are published as GitHub prereleases. This preview release is for UI shell, settings, mock package, developer audio smoke, RNNoise native build verification, and full-platform build verification. It is not a signed installer release, and it does not claim full native screen/audio/camera recording yet. See [docs/04-ci-release-plan.md](docs/04-ci-release-plan.md).

## Data Directory

All recording output must stay under:

```text
<RecordingFreedomAppData>/data/video/
```

Development can override the app data root:

```bash
RECORDINGFREEDOM_DATA_DIR=./data
```

The current mock recorder writes `.rfrec` package directories with typed `manifest.json` and `screen.mock.txt`. This is only for UI/package flow verification and must not be presented as real native recording. Startup/recovery scanning can identify unfinished or partially missing packages as recoverable candidates.

Backend selection can be exercised in development:

```bash
RECORDINGFREEDOM_RECORDING_BACKEND=native
```

This selects the platform native backend ID (`screencapturekit`, `windows-graphics-capture`, or `pipewire-portal`) but currently remains queued and blocked by preflight. It is a stable replacement point for the upcoming real capture backends, not a real recording implementation yet.

User settings are persisted in:

```text
<RecordingFreedomAppData>/settings.json
```

## Roadmap

1. Validate CI on the new GitHub repository.
2. Replace queued media-device placeholders with native macOS/Windows/Linux audio and camera enumeration.
3. Implement macOS ScreenCaptureKit recording.
4. Finish Windows WGC recording and connect the implemented WASAPI audio session to the real backend.
5. Connect the verified RNNoise native DSP into the full app recording backend and expose it through preflight/UI after real audio capture is active there.
6. Add camera sidecar recording and later PIP preview/export.
