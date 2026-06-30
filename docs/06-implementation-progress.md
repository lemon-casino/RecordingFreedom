# 06. 当前实现进度

更新时间：2026-07-01

## 已完成

- 使用官方 Wails v3 CLI `v3.0.0-alpha2.109` 生成 React + TypeScript + Vite 工程。
- 工程位置已整理为 `RecordingFreedom/app/`。
- Go module 为 `github.com/lemon-casino/RecordingFreedom/app`，Wails 依赖锁定为 `github.com/wailsapp/wails/v3 v3.0.0-alpha2.109`。
- 实现第一版胶囊录制工具窗口：
  - 源选择：Screen / Window / Program。
  - 音频：系统声音、麦克风、RNNoise 开关、麦克风设备、电平显示。
  - 摄像头：sidecar 开关、设备选择、PIP preset 选择。
  - 控制：开始、暂停/继续、结束、状态、计时器。
  - 语言：简体中文、English。
  - 设置：独立 Wails window，列出 storage、package、backend、audio、camera、recovery、capability、release。
- 实现 Wails v3 系统托盘入口：
  - 托盘图标点击显示/隐藏胶囊录制窗口。
  - 托盘右键菜单提供 Show Recorder、Show Settings、Hide Recorder、Quit RecordingFreedom。
  - 胶囊窗口关闭、Escape、失焦均隐藏窗口，不直接退出应用。
  - 设置窗口使用普通可调整大小的 `/settings` Wails window；关闭和 Escape 只隐藏窗口，不退出应用。
  - Windows 下胶囊窗口隐藏任务栏入口；macOS 下使用 accessory activation policy。
- 实现 Go 后端服务骨架：
  - `AppDataService` 保证应用托管目录和 `data/video` 存在。
  - `AppDataService` 已支持可指定应用数据根目录，并通过默认数据目录下的 `data-root.json` 指针保证下次启动仍能找到自定义根目录。
  - 切换数据根时会创建并探测目标目录可写性；录制包仍固定写入 `<DataRoot>/data/video`。
  - `AppDataService` 已新增 `StorageStatus()`：探测 `<DataRoot>/data/video` 可写性、平台可用空间、1 GiB 长录制建议阈值，并返回 `ready` / `warning` / `blocked`。
  - `SettingsService` 已落地为 `internal/settings`，读写 `<AppData>/settings.json`。
  - `settings.json` v1 持久化语言、最近源、音频、麦克风设备、RNNoise、摄像头、PIP preset 和托盘窗口行为。
  - `settings.json` v1 现已持久化 storage data root；Wails `SetDataRoot()` 会在录制中拒绝切换路径，避免持续写盘断裂。
  - `settings.json` v1 现已持久化系统声音设备 `systemDeviceId`，与麦克风设备、摄像头设备采用同一设备选择模型。
  - `settings.json` v1 现已持久化录制 profile：质量、FPS、录制光标、倒计时。
  - `DeviceService` 已从 `RecordingService` 中拆出，成为独立的屏幕、窗口、程序源枚举入口。
  - Windows 当前环境下使用 Win32 API 枚举显示器、可见顶层窗口，并按进程聚合程序源。
  - macOS 已新增 `darwin && cgo` CoreGraphics 源枚举：`CGGetActiveDisplayList` 显示器、`CGWindowListCopyWindowInfo` 可见窗口、按 PID 聚合程序源。
  - macOS `darwin && !cgo` 会返回明确错误，避免误报 native source enumeration 已可用。
  - `DeviceService` 新增 `MediaInventory` 合同，统一返回系统声音、麦克风、摄像头和 RNNoise 能力状态。
  - `DeviceService` 已新增 `MediaDeviceProvider` 替换边界；默认平台 provider 当前返回 queued inventory，测试可注入 provider 验证真实枚举、空结果和错误回退不影响前端合同。
  - Windows `MediaDeviceProvider` 已通过 MMDevice API 枚举 WASAPI render/capture endpoint：系统声音和麦克风返回真实 endpoint `NativeID`、friendly name、默认设备和 `enumerated` capability。
  - macOS/Linux 媒体设备当前仍返回明确的 `native-backend-queued` capability，预留给 CoreAudio/PipeWire 与 AVFoundation/PipeWire 后续接入。
  - Linux 当前返回明确的 `native-backend-queued` capability，占位给 XDG Portal 后端接入，不伪装真实枚举完成。
  - `CaptureService` 已新增平台能力矩阵，覆盖 source enumeration、screen/window/program recording、system audio、microphone、RNNoise、camera sidecar、PIP export 和 package recovery。
  - 能力状态统一为 `available`、`queued`、`blocked`、`unsupported`，并带 backend、permission 和 reason，避免 UI 或文档误报真实录制已完成。
  - 新增 `internal/preflight` 录制预检合同：开始录制前统一检查 source、媒体设备、平台 capability、目标 backend 和 storage health，返回 `ready` / `warning` / `blocked`。
  - `mock-package` backend 遇到真实录制能力仍为 `queued` 时返回 `warning` 并允许 UI 继续 mock 验证；真实 backend 遇到 `queued`、`blocked` 或 `unsupported` 时返回 `blocked`，避免误入不可录制流程。
  - 新增 `internal/audio` 音频增强合同：48kHz、10ms、480 samples RNNoise frame，麦克风 mono PCM 才可进入 suppressor，系统声音始终 bypass。
  - `internal/audio.Enhancer` 已新增可审计统计：processed frames、processed samples、pending samples、reset count、bypassed samples、rejected frames 和 last error。
  - 新增 `internal/audio.Pipeline` 音频处理边界：平台采集层推入 `TimedPCMBuffer`，pipeline 按配置处理 system audio、microphone、RNNoise，并输出 `ProcessedBuffer`。
  - 新增 `internal/audio.Diagnostics` 和 `WriteDiagnostics()`：定义 `audio-diagnostics.json` 写盘合同，覆盖 target format、system audio、microphone、enhancement 和 mixer 统计。
  - 新增 `recording.CreateAudioCaptureConfig()`：从 `StartRequest + RecordingWritePlan` 生成统一音频采集配置，真实后端复用同一份 system/mic/RNNoise/gain/diagnostics 路径合同。
  - 新增 `internal/audio.WAVSink`、`audio.CaptureSession` 和 `audio.NewNativeCaptureSession()`：把平台 source、pipeline、WAV sidecar 写盘和 `audio-diagnostics.json` 串成可复用运行时。
  - 新增 `internal/video` 视频采集合约：`CaptureConfig`、`Session`、`video-diagnostics.json` 和默认 unsupported 平台入口，为 ScreenCaptureKit/WGC/PipeWire writer 提供同一生命周期目标。
  - 新增 `recording.CreateVideoCaptureConfig()`：从 `StartRequest + RecordingWritePlan` 生成统一视频采集配置，真实后端复用同一份 source、profile、`screen.mp4` 和 `video-diagnostics.json` 路径合同。
  - 新增 `recording.NativeBackendRuntime`：把 native `.rfrec` 写盘计划、视频 session 生命周期和音频 session 生命周期串起来，提供 `Start()`、`Pause()`、`Resume()`、`Stop()`、单独 video/audio 控制、RNNoise suppressor 生命周期和启动失败标记 `failed` 的统一入口。
  - 新增 `NativeBackendRuntime.SyncDiagnostics()`：把 video/audio diagnostics 转成 manifest `diagnostics.sync`，统一输出 screen、system audio、microphone track 起止、duration、drop count、append failure、sample rate 和 diagnostics 相对路径。
  - 新增 `recording.NativeRuntimeBackend`：把 runtime 包装成可注册的真实 `Backend`，平台后端只需要提供 video/audio session factory，不需要重复实现包创建、状态转换、失败标记和 sync diagnostics。
  - 新增 macOS ScreenCaptureKit display/window/program video session：`screen:display-<CGDirectDisplayID>` 通过 `SCDisplay` 采集，`window:<CGWindowID>` 通过 `SCWindow` 采集，`application:<pid>` 选择该 PID 当前最大可见 `SCWindow`，三者都由 `SCStream` 输出真实 screen sample buffer，`AVAssetWriter` 写入包内 `screen.mp4`，停止时写 `video-diagnostics.json`；开启系统声音时，ScreenCaptureKit audio sample 写入同一个 `screen.mp4` 的 AAC 音轨，manifest 记录 `systemAudioStorage: "muxed"`。macOS `native`/`sck` backend 已注册到 `NativeRuntimeBackend`。当前仍需 macOS 真机录制 smoke 验证，麦克风 mux 还未完成。
  - 新增 `cmd/video-smoke`：无 UI 真实视频录制验收入口，默认使用 `native` backend、自动选择第一个可用屏幕源，也支持 `-source-type=window` 和 `-source-type=application` 验证窗口/程序源；默认关闭音频和摄像头，停止后校验 `.rfrec` 包、`screen.mp4` 非 0 字节、`video-diagnostics.json`、manifest `ready` 和 `diagnostics.sync`。
  - 正式录制策略调整为 mux 优先：默认目标是把屏幕视频、系统声音和麦克风写入同一个主媒体 `screen.mp4`；包内 WAV sidecar 继续作为 smoke、fallback、恢复和诊断路径。
  - `internal/recpackage` 已新增音频存储形态合同：系统声音和麦克风分别通过 `systemAudioStorage` / `microphoneAudioStorage` 标记为 `sidecar` 或 `muxed`；默认 fallback 写 `system-audio.wav` / `microphone.wav`，未来 mux writer 可把路径指向 `screen.mp4`。
  - `PackageService.ValidateReady()` 已在非 mock 包中校验已启用音频：`sidecar` 模式要求对应 WAV 存在、可读且大于空 header，`muxed` 模式会解析 `screen.mp4` 的 MP4 box 并要求存在 `soun` 音轨。
  - `internal/recpackage` 已新增 audio-only 包合同：manifest 使用 `recordingMode: "audio-only"` 和 `audioPath`，`CreateAudioOnly()` 不创建 `screen.mp4`，ready 门禁会校验主音频媒体的 `soun` 音轨，或校验明确声明的 WAV fallback sidecar；真实 `audio.m4a` writer、混音/mux、UI 模式入口和三平台 smoke 仍按后续任务推进。
  - Windows 已新增纯 Go WASAPI capture source：麦克风采集会 downmix/resample 为 `48kHz / mono`，系统声音使用 loopback source，二者都走同一 audio pipeline 和 WAV sink。
  - 新增 `internal/audio/rnnoise`：迁移 RNNoise C 源码和旧项目 `LikelyVoiceEnhancement` 为 cgo native wrapper；RNNoise C/H 源码已隔离到 `internal/audio/rnnoise/native` 子包，默认构建返回明确 unavailable，不做假降噪，带 `rnnoise_native` 标签的 cgo 构建才启用原生 DSP。
  - RNNoise UI/preflight capability 当前仍保持 queued；native wrapper 只在 `audio-smoke` 或后续真实 audio backend 显式使用时编译，避免设备枚举和能力矩阵路径过早耦合 native DSP。
  - 新增 `cmd/audio-smoke`：可在不启动 Wails UI 的情况下真实启动平台音频 source，当前通过 `recpackage.CreateAudioOnly()` 写入 `<DataRoot>/data/video/recording-*.rfrec/`；单流 fallback 写 `audio.wav`，双流 fallback 写 `system-audio.wav` / `microphone.wav` 分轨 sidecar，停止后写入 `audio-diagnostics.json`、manifest sync diagnostics，并通过 `ValidateReady()` 后标记 `ready`。
  - 新增 `internal/pip` 画中画 preset 合同：`off`、`bottom-right`、`bottom-left`、`free`，并提供基础 overlay layout 计算。
  - 新增 `internal/recordingprofile` 录制参数合同：`standard/balanced/high`、`24/30/60 FPS`、`captureCursor`、`countdownSeconds`，供 settings、recording request 和 manifest 共用。
  - 新增 `recording.NormalizeStartRequest()`，统一校验 `sourceId/sourceType`，归一化系统声音设备、麦克风设备、RNNoise、摄像头设备和 PIP preset。
  - `RecordingService` 支持 mock start/pause/resume/stop。
  - `RecordingService` 已拆出 `recording.Backend` 接口，默认 backend 为 `mock-package`；真实 ScreenCaptureKit/WGC/PipeWire 后端后续按同一接口接入。
  - `recording.Backend.Stop()` 已升级为返回 `BackendStopResult`；真实后端可以返回 `SyncDiagnostics`，由 `RecordingService` 统一写入 manifest，再把包标记为 `ready`。
  - `RecordingService.Stop()` 已在写入 `ready` 前接入 `PackageService.ValidateReady()`，非 mock/native 包缺失非 0 字节 screen media 时会失败并把 manifest 标为 `failed`；摄像头开启时 webcam sidecar 也必须通过同一可读性门禁。
  - 新增 `recording.CreateNativeWritePlan()`，真实 ScreenCaptureKit/WGC/PipeWire 后端后续应通过它统一创建 native `.rfrec` 写盘计划，避免各平台重复手写 source/audio/camera/recording manifest 映射。
  - 新增 backend selector 合同：默认/`auto`/`mock-package` 选择 `mock-package`；`RECORDINGFREEDOM_RECORDING_BACKEND=native` 按平台选择 queued native backend。
  - backend selector 已升级为 native backend registry：真实平台后端可通过 `recording.RegisterNativeBackend()` 注册 factory；`native`、`sck`、`wgc`、`pipewire` 会优先选择已注册实现，未注册时才回退 queued backend。
  - queued native backend 当前 ID 为 `screencapturekit`、`windows-graphics-capture`、`pipewire-portal` 或 `native-unsupported`；它不会创建包或写媒体，`Start()` 返回明确 queued error。
  - `Bootstrap()` 已返回当前 backend 和 storage health，前端启动后底部状态条不必等第一次录制也能显示当前后端。
  - `Session` 新增 `backend` 字段，前端底部状态条显示当前录制后端，便于验证 mock 与真实平台后端切换。
  - `recording.StatusEvent` 已扩展为状态事件载荷，包含 `status`、`sessionId`、`packageDir`、`manifest`、`backend` 和 `message`。
  - `RecordingFreedomService` 已通过 Wails `recording.status` 事件发出 preparing、recording、paused、stopping、ready、failed 状态；`StartMockRecording()` 兼容入口也转发到同一事件流。
  - `internal/recpackage` 已实现 typed `.rfrec` 包服务：创建唯一目录、写入 manifest、更新状态、校验相对媒体路径、扫描 recoverable 包、恢复 recoverable 包。
  - `internal/recpackage` 已新增 `CreateNative()` 原生写盘计划合同：为真实 ScreenCaptureKit/WGC/PipeWire 后端初始化 ready-to-write `.rfrec` 包、`screen.mp4` / `webcam.mov` 相对路径、diagnostics 路径、`cache/` 和 `exports/` 目录，但不会创建假媒体文件或伪造 sync diagnostics。
  - `internal/recpackage` 已新增 `ValidateReady()` ready 前媒体探测门禁：mock 包只接受非空 `screen.mock.txt` marker；非 mock 包拒绝 mock marker，并要求 screen media 可读且非 0 字节；摄像头开启时要求 webcam sidecar 可读且非 0 字节。
  - 恢复动作只接受 app-managed `data/video` 内部 `.rfrec` 包；恢复已有 manifest 时写入 `ready` / `completedAt`，缺失 manifest 但存在非 0 字节 `screen.*` 时重建最小 manifest。
  - mock start 通过 `recpackage.CreateMock()` 创建 `data/video/recording-*.rfrec/manifest.json` 和 `screen.mock.txt`。
  - stop 会先把 manifest 写为 `finalizing`，完成后写为 `ready` 并记录 `completedAt`。
  - mock manifest 会记录 `microphoneDeviceId` 和 camera `deviceId`，让后续真实管线接入不改变包结构。
  - manifest 明确标记 `diagnostics.mock = true`，不伪装为真实录制。
  - manifest 已新增 `diagnostics.sync` 音画同步诊断合同，覆盖 screen、system audio、microphone、webcam 的时间线基准、track offset、duration、drop count、append failure 和 pause segments。
  - `PackageService.PatchSyncDiagnostics()` 已落地，后续真实 ScreenCaptureKit/WGC/PipeWire 后端必须通过它写入同步诊断。
  - `recpackage.WriteManifest()` 会校验 sync 诊断路径不能是绝对路径或逃逸 `.rfrec` 包目录，并在摄像头关闭时清理 `webcamVideoPath`、`webcamStartOffsetMs` 和 webcam sync track。
  - 新增 `internal/exportplan` 导出计划合同：只接受 app-managed `data/video` 内 ready `.rfrec` 包，默认输出到包内 `exports/recording.mp4`，拒绝包外路径、mock 包、缺失 media、缺失 sync diagnostics 和缺失 webcam sidecar 的可见 PIP 导出。
- 生成 Wails TypeScript bindings：
  - `RecordingFreedomService.Bootstrap`
  - `RecordingFreedomService.GetCaptureCapabilities`
  - `RecordingFreedomService.GetSettings`
  - `RecordingFreedomService.SaveSettings`
  - `RecordingFreedomService.SetDataRoot`
  - `RecordingFreedomService.ShowSettingsWindow`
  - `RecordingFreedomService.HideSettingsWindow`
  - `RecordingFreedomService.ListSources`
  - `RecordingFreedomService.ListMediaDevices`
  - `RecordingFreedomService.PreflightRecording`
  - `RecordingFreedomService.ScanRecordingPackages`
  - `RecordingFreedomService.RecoverRecordingPackage`
  - `RecordingFreedomService.StartRecording`
  - `RecordingFreedomService.StartMockRecording`
  - `RecordingFreedomService.PauseRecording`
  - `RecordingFreedomService.ResumeRecording`
  - `RecordingFreedomService.StopRecording`
  - `recording.status` event type，载荷包含 status、sessionId、packageDir、manifest、backend、message
- 前端服务层已改为优先调用 Wails generated bindings：
  - 新增 `loadBootstrap()` 适配层，一次读取 app data、录制状态、源列表、媒体设备、恢复扫描、设置和能力矩阵。
  - 新增 `showSettingsWindow()` / `hideSettingsWindow()` 适配层；Wails 桌面环境调用原生窗口，浏览器预览打开 `/settings` fallback。
  - 新增 `subscribeRecordingStatus()` 适配层；Wails 桌面环境订阅 `recording.status`，浏览器预览使用 no-op fallback。
  - `loadBootstrap()` 已读取后端返回的当前 backend，初始化胶囊底部 `Backend: ...` 状态。
  - `loadSettings()` / `saveSettings()` 已接入 Go `SettingsService`；浏览器预览使用 `localStorage` fallback。
  - 胶囊语言、源、系统声音、麦克风、RNNoise、摄像头设置会 debounce 保存。
  - 语言切换已接入全局前端 i18n 显示层：点击简体中文或 English 后，胶囊主控、弹窗、独立设置窗口、状态条、预检摘要、存储状态和能力矩阵会立即使用对应语言；后端 ID、manifest 字段、路径和诊断字符串保持稳定不翻译。
  - 独立设置窗口已接入质量、FPS、录制光标、倒计时控件；开始录制会把当前录制 profile 传给后端并写入 manifest。
  - 摄像头菜单已接入 PIP preset 下拉；开始录制会把当前 preset 传给后端，摄像头关闭时 manifest 使用 `off`。
  - 音频菜单已接入系统声音设备下拉；开始录制会把当前 `systemDeviceId` 传给后端，系统声音关闭时 manifest 不保留旧 device id。
  - 独立设置窗口已接入 `CaptureService` 能力矩阵，按后端返回展示平台、backend、权限和 `Ready/Queued/Blocked/Unsupported` 状态。
  - 独立设置窗口新增 Recording backend 行，启动后显示当前 backend，并说明 queued native backend 仍会被 preflight 阻止。
  - 独立设置窗口新增 Preflight 行，显示最近一次录制预检状态、backend 和原因。
  - 独立设置窗口已接入 `Bootstrap().appData`，Storage 显示真实 `videoDir`，App data 显示真实数据根目录。
  - 独立设置窗口新增 Storage health 行，显示 `data/video` 可写性、可用空间和建议最低空间。
  - 独立设置窗口新增 Data root 输入和 Apply 操作；修改的是应用数据根，录制仍进入其下的 `data/video`。
  - 独立设置窗口新增 Recovery 行，存在 recoverable `.rfrec` 包时可以触发恢复并重新扫描。
  - `loadSources()` 现在读取 Go `DeviceService` 返回的 `devices.CaptureSource`。
  - `loadMediaDevices()` 现在读取 Go `DeviceService` 返回的 `devices.MediaInventory`。
  - 源模型包含 `available`、`capability` 和 `unavailableReason`，用于区分已枚举源和待原生后端源。
  - 麦克风和摄像头下拉已从后端媒体设备合同读取；浏览器预览仍有 mock fallback。
  - `scanRecordingPackages()` 已接入恢复扫描，胶囊底部状态条显示 recoverable 包数量或 clean。
  - Wails 桌面环境中开始/暂停/继续/停止会调用 Go `RecordingFreedomService`；开始录制已切到长期入口 `StartRecording()`。
  - 胶囊窗口已使用统一 `applyRecordingStatus()` 入口消费 Wails status event 和后端 session 返回值，底部状态条显示 `Status: ...`。
  - 开始录制前会先调用 `PreflightRecording()`；`blocked` 时前端停在失败状态并展示原因，`ready` / `warning` 才继续进入 `StartRecording()`。
  - 浏览器 `vite preview` 环境中自动 fallback 到前端 mock，方便纯 UI 开发。
- `build/config.yml` 已替换模板占位信息为 RecordingFreedom 元数据。
- 新增新仓库可用的 GitHub Actions：
  - `RecordingFreedom/.github/workflows/ci.yml`
  - `RecordingFreedom/.github/workflows/release.yml`
  - workflow 假定 `RecordingFreedom/` 是新仓库根目录，因此 `APP_DIR=app`。
  - CI 包含 bindings 检查、frontend build、Go test、preview smoke、macOS native contract、Windows/macOS/Linux Wails build。
  - Release 在 tag `v*` 时先执行 release gate（bindings 检查、frontend build、Go test、preview smoke），通过后构建三平台 preview artifacts、生成 SHA256SUMS 并发布 GitHub Release。
- 新增无 GUI `cmd/preview-smoke` 验证入口：
  - 默认使用临时 data root，不污染开发目录；可通过 `-keep` 保留生成包。
  - 验证 settings 持久化、`data/video` storage health、preflight、mock start/pause/resume/stop、ready manifest、`screen.mock.txt` 非空、RNNoise/camera/PIP manifest 合同和恢复扫描。
- 新增 `RecordingFreedom/README.md`，用于新仓库根目录说明、开发命令、验证命令和 roadmap。
- 已在 GitHub runner 上验证 preview release 链路：
  - `v0.1.0-preview.4` 的 Release Gate、Windows build、macOS build、Linux build 和 Publish GitHub Release 均已通过。
  - GitHub Release 已产出 Windows x64、macOS arm64、Linux x64 三个平台 preview 二进制和对应 SHA256SUMS。
  - 后续 preview tag 会自动标记为 GitHub prerelease。

## 验证结果

已通过：

```bash
go test ./internal/...
GOOS=darwin CGO_ENABLED=0 go test -c ./internal/devices
wails3 generate bindings -ts -i
npm run build
go test ./...
go run ./cmd/preview-smoke
go run ./cmd/audio-smoke -duration=1s -keep
wails3 build
```

有 C 工具链的环境可用以下命令验证 RNNoise 原生 DSP：

```bash
go test -tags rnnoise_native ./internal/audio/rnnoise/native ./internal/recording
```

本机 Windows 当前缺少 `gcc`，因此该命令只作为有 C 工具链环境的本机验证入口；CI/release gate 会在 Linux runner 上执行 RNNoise native DSP 和 recording runtime 定向测试。

视觉检查：

- 本地预览服务：`http://127.0.0.1:9245/`
- 本轮独立设置窗口 smoke 因 `9245` 已被占用，使用 `http://127.0.0.1:9246/` 和 `http://127.0.0.1:9246/settings`。
- Playwright 检查结果：
  - 页面标题为 `RecordingFreedom`
  - `.capsule` 可见
  - `.record-button` 可见
  - 源文本显示 `Screen / Built-in Retina`
  - 状态条显示 `data/video/recording-preview.rfrec` 和 `Mic enhancement: RNNoise`
  - 引入 Wails bindings adapter 后浏览器 fallback smoke check 仍通过。
  - 点击开始录制后，浏览器 fallback 显示 `Backend: browser-mock`，录制按钮进入 `recording` 状态。
- Codex in-app browser DOM 检查结果：
  - `.capsule` 可见。
  - `/` 根路径 `.capsule` 可见，`.settings-sheet` 和 `.settings-window-panel` 不存在，确认胶囊窗口不再内嵌复杂设置。
  - `/settings` 路径 `.rf-settings-shell` 和 `.settings-window-panel` 可见，`.capsule` 不存在，确认设置已拆到独立窗口入口。
  - capability status badge 同时显示 `Ready` 和 `Queued`。
  - 独立设置窗口显示 `Platform` 和 `Recording Package Recovery`。
  - Storage 行显示 `data/video` fallback，且不再出现 `<AppData>` 占位；Wails 桌面环境通过 `Bootstrap().appData.videoDir` 显示真实路径。
  - Recovery 行显示 `Clean`；存在 recoverable 包时会显示 Recover 按钮。
  - Camera menu 显示 `PIP preset` 下拉，包含 `Bottom right`、`Bottom left`、`Free layout`、`Off`。
  - 将 PIP preset 切换为 `Bottom left` 后，菜单状态文本显示 `PIP preset: Bottom left`。
  - PIP 菜单交互后点击开始录制，浏览器 fallback 仍切换到 `Backend: browser-mock`；点击结束后进入 `SAVED` 状态。
  - 独立设置窗口显示 `Quality`、`FPS`、`Capture cursor`、`Countdown` 控件，默认值为 `Balanced`、`30 FPS`、启用光标、倒计时关闭。
  - 新增录制 profile 控件后点击开始录制，浏览器 fallback 仍切换到 `Backend: browser-mock`；点击结束后进入 `SAVED` 状态。
  - 初始底部状态条显示 `Preflight: not run`。
  - 点击开始录制后，浏览器 fallback 先返回 `Preflight: Ready`，再显示 `Backend: browser-mock` 并进入 `REC` 状态。
  - 本轮状态事件适配 smoke：初始 `Status: Waiting for recorder command`；开始后 `Status: Recording started`；暂停后 `Status: Recording paused`；继续后 `Status: Recording resumed`；结束后 `Status: Recording package ready`。
  - 独立设置窗口的 Preflight 行显示 `Ready · browser-mock`，并展示 browser mock 预检说明。

恢复测试：

- `internal/appdata` 覆盖 `data/video` 固定相对结构、自定义数据根指针持久化、`RECORDINGFREEDOM_DATA_DIR` 覆盖优先级和 env override 下禁止 UI 修改。
- `internal/appdata` 覆盖 `StorageStatus()` 创建并探测 `data/video`、可写状态、可用空间字段和建议最低空间阈值。
- `RecordingFreedomService.SetDataRoot()` 覆盖更新 `data/video` 与 settings storage 字段，以及录制中拒绝切换数据根。
- `RecordingFreedomService.Bootstrap()` 覆盖返回 app data 与 storage health，确保 Wails frontend 能显示同一份存储状态。
- `internal/recpackage` 覆盖已有 manifest 恢复为 `ready`、缺失 manifest 通过 `screen.*` 重建、拒绝恢复 `data/video` 外部包。
- `internal/recording` 覆盖通过 `RecordingService.RecoverPackage()` 恢复 app-managed `data/video` 内的包。
- `internal/recpackage` 覆盖 mock 包写入 `diagnostics.sync`、`PatchSyncDiagnostics()` 写入/拒绝逃逸路径、摄像头关闭时清理 webcam media 和 sync 诊断。
- `internal/recpackage` 覆盖 native 写盘计划初始化 manifest、screen/webcam/diagnostics/cache/exports 路径、摄像头关闭时不返回 webcam 写入路径，以及初始化阶段不创建假媒体文件。
- `internal/recpackage` 覆盖 ready 前媒体门禁：mock marker 通过、native 缺失 screen 拒绝、0 字节 screen 拒绝、非 mock marker 拒绝、摄像头开启但缺失或 0 字节 webcam sidecar 拒绝、音频 sidecar 缺失拒绝、muxed 音频缺少 MP4 `soun` 音轨拒绝、audio-only 缺失 `audio.m4a` 或缺少 `soun` 音轨拒绝、真实非 0 字节 screen/webcam 和有效音频通过。

音频合同测试：

- `internal/audio` 覆盖系统声音绕过 RNNoise、麦克风按 480-sample frame 进入 suppressor、partial frame pending、reset 清理 pending 和 suppressor 状态、拒绝 stereo microphone RNNoise 输入。
- `internal/audio` 覆盖音频 pipeline：系统声音即使请求 RNNoise 也会 bypass、麦克风按配置进入 RNNoise、禁用流拒收、reset 清理 enhancer 状态、`audio-diagnostics.json` 可写入可读 JSON。
- `internal/audio` 覆盖 WAV sidecar header/data 写入、格式变化拒绝、mono resampler、`CaptureSession` source -> pipeline -> sink -> diagnostics 运行时。
- `internal/video` 覆盖视频 capture config 归一化、`video-diagnostics.json` 写盘和默认平台 session 明确 unsupported。
- `internal/audio/rnnoise` 覆盖非 cgo/未带标签 fallback；`rnnoise_native` cgo 构建下会编译 RNNoise C 源并跑 native frame 处理测试。Linux cgo link 的 `-lm` 约束已修正为独立 `linux` / `darwin` LDFLAGS，CI/release gate 执行 native 定向测试，本机 Windows 缺少 `gcc` 时只验证 fallback。
- `internal/recording` 覆盖 `CreateAudioCaptureConfig()`：打开/关闭系统声音、麦克风和 RNNoise 时，音频设备、sidecar 输出路径、diagnostics 路径和系统声音不降噪策略保持稳定。
- `internal/recording` 覆盖 `CreateVideoCaptureConfig()`：source、profile、`screen.mp4` 输出路径和 `video-diagnostics.json` 路径保持稳定。
- `internal/recording` 覆盖 `NativeBackendRuntime`：会创建并控制 video session；有音频时创建并控制 audio session；无音频时不启动 audio session；RNNoise suppressor 会传入并在停止时关闭；视频 session 或 RNNoise 不可用时初始化失败并把已创建 native 包标记为 `failed`。
- `internal/recording` 覆盖 `NativeBackendRuntime.SyncDiagnostics()`：runtime 生成的 screen/system/microphone track diagnostics 可被 `recpackage.PatchSyncDiagnostics()` 接受并写回 manifest。
- `internal/recording` 覆盖 `NativeRuntimeBackend`：Start/Pause/Resume/Stop 会驱动 runtime，Stop 返回 sync diagnostics，Start 失败会把已创建 native 包标记为 `failed`；backend registry 可选择注册后的 runtime backend。
- 本机 Windows audio smoke 已确认默认麦克风 WASAPI capture 生成 ready 的 audio-only `.rfrec` 包：manifest 为 `recordingMode: "audio-only"`、`status: "ready"`、`audioPath: "audio.wav"`，麦克风 track 指向 `audio.wav`，`framesReceived=99`、`samplesReceived=47520`、`samplesWritten=47520`、duration 约 `990ms`。
- 本机 Windows system audio smoke 已确认 WASAPI loopback source 在有活动系统播放时可以写入真实样本：`system-audio.wav` 614444 bytes，`framesReceived=160`，`samplesReceived=153600`，`samplesWritten=153600`，`sampleRate=48000`，`channels=2`，duration 约 `1600ms`。

PIP 合同测试：

- `internal/pip` 覆盖非法 preset normalize、右下角/左下角布局、`off` 隐藏布局、非法 preset 拒绝。
- `internal/settings` 覆盖非法 PIP preset 回退为默认 `bottom-right`。
- `internal/recpackage` 覆盖 camera enabled 但缺失/非法 PIP preset 时写入默认 `bottom-right`。
- `internal/exportplan` 覆盖 ready 包 PIP 导出计划、screen-only 隐藏 PIP、包外路径拒绝、输出路径逃逸拒绝、mock 包拒绝、可见 PIP 缺失 webcam sidecar 拒绝、手工破坏 diagnostics 路径拒绝。

录制请求合同测试：

- `internal/recording` 覆盖缺失 source id / 非法 source type 拒绝、系统声音/麦克风/摄像头设备默认值、关闭流时清理旧 device id、非法 microphone gain 拒绝。
- `internal/recpackage` 覆盖关闭系统声音、麦克风或摄像头时，manifest 不保留对应旧 device id。

录制 backend selector 测试：

- `internal/recording` 覆盖默认 backend 为 `mock-package`。
- `internal/recording` 覆盖 `native` 请求按平台映射到 `screencapturekit`、`windows-graphics-capture`、`pipewire-portal` 或 `native-unsupported`。
- `internal/recording` 覆盖 backend registry 已注册真实 native factory 时会优先返回注册实现，并把同一个 `recpackage.Service` 传入 factory。
- `internal/recording` 覆盖 backend registry 未注册或 factory 为空时回退 queued native backend，不误创建真实 backend。
- `internal/recording` 覆盖 queued native backend 不能开始录制且不会返回录制包。
- `internal/recording` 覆盖 `RECORDINGFREEDOM_RECORDING_BACKEND=native` 会影响默认 backend 选择。
- `internal/recording` 覆盖 `RecordingService` 在 native queued 模式下即使被直接调用也会失败，并且不会在 `data/video` 下创建 `.rfrec` 包。
- `internal/recording` 覆盖 backend stop 返回的 sync diagnostics 会在 `ready` 前写入 manifest；非法 sync diagnostics 会让 stop 失败，并把 manifest 标为 `failed`。
- `internal/recording` 覆盖 native-like backend 未写 screen media 时 `Stop()` 失败并把 manifest 标为 `failed`，写入非 0 字节 screen/webcam 且 sync diagnostics 合法时才能进入 `ready`。
- `internal/recording` 覆盖 `CreateNativeWritePlan()` 会归一化 source、默认系统声音设备、默认麦克风设备、RNNoise、默认摄像头和 PIP preset；关闭系统声音、麦克风或摄像头时不会保留旧 device id 或 webcam 写入路径。

录制 profile 合同测试：

- `internal/recordingprofile` 覆盖默认值、非法质量/FPS 归一化、倒计时上下限。
- `internal/settings` 覆盖录制 profile 持久化与非法 profile 归一化。
- `internal/recording` 覆盖开始录制请求默认 profile 和显式 profile。
- `internal/recpackage` 覆盖 manifest 写入录制 profile，以及非法 profile 归一化。

录制预检合同测试：

- `internal/preflight` 覆盖 mock backend 遇到 queued native capability 时返回 `warning`。
- `internal/preflight` 覆盖缺失或不可用 source 时返回 `blocked`。
- `internal/preflight` 覆盖真实 backend 遇到 queued capability 时返回 `blocked`。
- `internal/preflight` 覆盖 source、media 和 capability 均 available 时返回 `ready`。
- `internal/preflight` 覆盖 `data/video` 不可写时返回 `blocked`，可写但低空间时返回 `warning`。

设备枚举合同测试：

- `internal/devices` 覆盖 `MediaDeviceProvider` 注入路径：provider 返回真实设备时会归一化 ID/type/available，不改变 `MediaInventory` 形状。
- `internal/devices` 覆盖 provider 失败回退：系统声音、麦克风、摄像头和 RNNoise enhancement 均返回带原因的 queued fallback，避免 UI 看到空列表假成功。
- Windows 本机 smoke 已确认 `DeviceService.ListMediaDevices()` 返回真实 WASAPI system audio 和 microphone endpoint，capability 为 `enumerated`，并保留默认设备 ID。

构建产物：

```text
RecordingFreedom/app/bin/recordingfreedom.exe
```

## 当前非目标项

以下能力尚未实现，不能对外宣称可用：

- macOS ScreenCaptureKit display/window/program 录制已接入代码路径，但仍需要真机 smoke：授权屏幕录制后运行 `go run ./cmd/video-smoke -duration=1m`、`go run ./cmd/video-smoke -source-type=window -duration=1m`、`go run ./cmd/video-smoke -source-type=application -duration=1m` 和 `go run ./cmd/video-smoke -duration=5m -pause-after=10s -pause-duration=2s`，并确认 `screen.mp4` 可播放、包进入 `ready`、`video-diagnostics.json` 和 `diagnostics.sync` 正确。
- ScreenCaptureKit 系统声音 mux 已接入代码路径但仍需 macOS 真机 smoke；麦克风 mux 仍未完成。
- 真实 Windows.Graphics.Capture 录制。
- 真实 PipeWire / XDG Portal 录制。
- 真实 macOS CoreAudio 麦克风枚举和 Linux PipeWire 音频设备枚举；当前 Windows WASAPI endpoint 枚举已完成，macOS system audio 使用 ScreenCaptureKit 默认系统混音流。
- 真实 CoreAudio/PipeWire 音频采集；当前 Windows WASAPI 麦克风采集已通过 smoke，Windows system loopback 已通过有播放源真实样本 smoke，native backend 音频运行时边界已落地，但平台视频后端尚未调用该 runtime 完成端到端录制。
- 真实 audio-only 录制模式；当前已完成 `.rfrec` audio-only 包格式、`audio.m4a` 主媒体路径、WAV fallback ready 门禁和 `audio-smoke` 包级验收入口，尚未接 UI、preflight、真实 `audio.m4a` writer 和混音/mux。
- 真实 AVFoundation / Media Foundation / PipeWire 摄像头设备枚举；当前只完成 `MediaDeviceProvider` 替换边界和 sidecar eligibility 合同。
- RNNoise native DSP 的 C 源码和 Go wrapper 已迁移并隔离；CI/release gate 已恢复 native 定向测试，preview artifact 仍保持默认构建，当前 Windows 本机因缺少 `gcc` 只能验证非 cgo fallback，真实 app recording backend 仍未暴露 RNNoise capability。
- 真实摄像头 sidecar 写入。
- 真实 PIP 预览与导出。
- 真实 FFmpeg/原生流式导出执行；当前只完成导出计划和路径/同步/PIP 校验合同。
- 真实音画同步时间戳采集和容器级媒体 probe；当前只完成 manifest 诊断合同、mock 标记、ready 前非 0 字节媒体门禁。

## 已知提示

- `wails3 build` 在 Windows 上输出过 `"uname": executable file not found in $PATH`，但构建最终成功。这是 Wails 构建脚本里的跨平台探测提示，当前不是阻塞项。
- `npm run build` 输出 Tailwind content 警告；当前 UI 使用自写 CSS，没有依赖 Tailwind 生成样式，后续可移除模板残留或补 Tailwind 配置。
- 本机全局 `go version` 是 1.24.1，但 Go toolchain 自动切换并下载了 Wails v3 需要的 Go 1.25.11。

## 下一步

1. 按 `docs/08-unfinished-task-plan-audio-first.md` 继续推进真实音频采集与 RNNoise 降噪。
2. A1 已完成 Windows WASAPI system audio/microphone endpoint 枚举；继续补 macOS CoreAudio 与 Linux PipeWire/PulseAudio 枚举。
3. Windows 麦克风 PCM 采集和系统声音 loopback 样本写盘已完成 smoke，`NativeBackendRuntime` 已为真实平台后端提供统一视频和音频生命周期；下一步补有 C 工具链本机的 `audio-smoke -rnnoise`、Windows 长录同步、让 ScreenCaptureKit/WGC/PipeWire 后端实际调用 runtime，以及 macOS/Linux 音频源。
4. 通过 `recording.RegisterNativeBackend(recording.BackendScreenCaptureKit, ...)` 注册 `NativeRuntimeBackend`，接入 macOS ScreenCaptureKit video session factory，并实现最小可录制 `screen.mp4` 写盘。
5. 把 release workflow 从 preview executable 升级为正式安装包、签名和公证流水线。
