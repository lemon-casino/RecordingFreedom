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
- The Go backend has app data, persistent settings, capture source/media device contracts, typed `.rfrec` package services, native backend selection, and explicit mock package creation for preview smoke only.
- The recording backend selector defaults to platform native/queued backends; `mock-package` must be requested explicitly and never represents real capture.
- Preview mock recording packages are created under the managed `data/video` structure.
- GitHub Actions are scaffolded in [.github/workflows](.github/workflows).

The current build is a UI and recording-pipeline milestone. macOS ScreenCaptureKit screen/window/single-display region code paths are wired. Windows now has a real FFmpeg desktop writer for screen/all-screen/region/locked-window video when an `ffmpeg` executable is available; enabled WASAPI system audio and microphone tracks are muxed into the final `screen.mp4` at stop. Without FFmpeg, preflight blocks before recording starts. Linux PipeWire remains queued.

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
go run ./cmd/desktop-doctor
go run ./cmd/preview-smoke
go run ./cmd/audio-smoke -duration=3s -keep
go run ./cmd/video-smoke -duration=3s -keep
wails3 build
```

`go run ./cmd/preview-smoke -keep` keeps the generated temporary data root so the `.rfrec` package can be inspected manually.

`go run ./cmd/audio-smoke -duration=3s -keep` records real platform audio into a temporary `data/video/audio-smoke-*.rfrec/` folder. RNNoise uses a packaged native dynamic module; build it first with the platform scripts below when testing `-rnnoise`.

On Windows, `go run ./cmd/video-smoke -duration=3s -keep` requires FFmpeg. Put `ffmpeg.exe` on `PATH`, place it beside the app under `tools/`, or set:

```bash
RECORDINGFREEDOM_FFMPEG_PATH=C:\path\to\ffmpeg.exe
```

The same Windows FFmpeg layout used by CI/release can be prepared locally:

```powershell
.\scripts\ensure-windows-ffmpeg.ps1
```

That script downloads the BtbN FFmpeg-Builds static Windows archive from GitHub, verifies it against `checksums.sha256`, and writes `app/tools/ffmpeg.exe`. The app resolves the same `tools/ffmpeg.exe` path when shipped beside `recordingfreedom.exe`.

Build the RNNoise dynamic module used by release artifacts:

```powershell
.\scripts\build-rnnoise-windows.ps1 -Architecture x64
.\scripts\build-rnnoise-windows.ps1 -Architecture arm64
```

```bash
bash ./scripts/build-rnnoise-unix.sh --platform macos --arch arm64
bash ./scripts/build-rnnoise-unix.sh --platform linux --arch x64
```

The module is written under `app/tools/` as `rnnoise.dll`, `librnnoise.dylib`, or `librnnoise.so`. The app can also load a diagnostic override from `RECORDINGFREEDOM_RNNOISE_PATH`.

Verify a staged Windows portable zip before uploading it:

```powershell
.\scripts\verify-windows-portable.ps1 -ZipPath .\release-out\RecordingFreedom-windows-x64-v0.1.0-preview.9-portable.zip
```

The current Windows portable preview also carries clean-machine diagnostics under `tools/`. After unzipping on a target Windows desktop, run:

```powershell
.\tools\run-windows-portable-smoke.ps1
```

That runner uses the bundled `tools/desktop-doctor.exe`, `tools/video-smoke.exe`, `tools/audio-smoke.exe`, `tools/ffmpeg.exe`, and `tools/ffprobe.exe`. It writes all smoke output under the portable-local `data-smoke/data/video` tree unless `-DataDir` is provided.

Use the desktop doctor to inspect the same dependency gate before trying a real recording:

```bash
go run ./cmd/desktop-doctor
go run ./cmd/desktop-doctor -require-video
go run -tags rnnoise_dynamic ./cmd/desktop-doctor -require-rnnoise
```

The first command reports app data, `data/video`, backend, capabilities, RNNoise, and FFmpeg status as JSON without failing preview builds. `-require-video` exits non-zero when the current platform cannot start real screen/window video recording. `-require-rnnoise` exits non-zero unless the current binary can load and bind the native RNNoise module.

To test and smoke the RNNoise path:

```bash
go test -tags rnnoise_dynamic ./internal/audio/rnnoise
go run -tags rnnoise_dynamic ./cmd/audio-smoke -duration=3s -rnnoise -keep
```

## Preview Release

After `RecordingFreedom/` becomes the new repository root, pushing a `v*` tag publishes a GitHub Release with Windows, macOS, and Linux preview artifacts plus SHA256 files:

```bash
git tag v0.1.0-preview.9
git push origin v0.1.0-preview.9
```

Preview tags are published as GitHub prereleases. Release artifacts are built with `rnnoise_dynamic`; Windows packages include `tools/rnnoise.dll`, macOS app bundles include `Contents/MacOS/tools/librnnoise.dylib`, and Linux archives include `tools/librnnoise.so`. `desktop-doctor -require-rnnoise` and a dynamic RNNoise DSP smoke gate every desktop architecture, including Windows ARM64. See [docs/04-ci-release-plan.md](docs/04-ci-release-plan.md) and [docs/13-rnnoise-dynamic-module.md](docs/13-rnnoise-dynamic-module.md).

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

This selects the platform native backend ID (`screencapturekit`, `ffmpeg-desktop-capture`, or `pipewire-portal`). macOS ScreenCaptureKit has real screen/window/single-display-region writer code paths that still require real-device smoke validation. Windows uses FFmpeg gdigrab for real screen/all-screen/region/locked-window MP4 writing, and enabled WASAPI system audio/microphone tracks are muxed into `screen.mp4` at stop. Camera sidecar/PIP work is paused until video recording and voice/audio recording are accepted. Linux remains queued until the PipeWire writer lands.

User settings are persisted in:

```text
<RecordingFreedomAppData>/settings.json
```

## Roadmap

1. Validate CI on the new GitHub repository.
2. Replace queued media-device placeholders with native macOS/Linux audio and camera enumeration; Windows WASAPI/DirectShow enumeration is already wired.
3. Real-device smoke macOS ScreenCaptureKit recording.
4. Download the Windows portable preview artifact and run `.\tools\run-windows-portable-smoke.ps1` on a real desktop to verify screen/all-screens/region/locked-window, pause/resume, system audio, microphone, RNNoise, and audio-only smoke.
5. Smoke `audio-smoke -rnnoise` on target desktops built with the same `rnnoise_dynamic` release toolchain, then keep the release `desktop-doctor -require-rnnoise` gate green on every desktop runner.
6. After video recording and voice/audio recording are accepted, resume camera sidecar and PIP preview/export work.
