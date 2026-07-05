# 06. 当前实现进度

更新时间：2026-07-05

## 本轮已完成并纳入 v0.1.11

- 已克隆并分析 `mg-chao/snow-shot`：其智能选择核心是平台窗口矩形、Windows UI Automation 子元素矩形和前端 Flatbush/RTree 选择层组合，而不是单纯图像边缘检测；分析结果已落地到 `15-smart-screenshot-region-assist.md`。
- 新增智能区域候选合同：`RegionSmartCandidate`、`RegionSelectionSession.candidates` 和 `AssistRegionSelection()` 已接入截图、滚动截图、录制中标注画板和自定义区域录制选择流程。
- 自定义区域与截图区域选择新增图像边缘吸附：拖选结束后会围绕选择区域截取局部屏幕图像，寻找附近强边缘，并在置信度足够时把区域吸附到目标元素边界。
- 截图菜单已补齐区域、全屏、窗口、焦点窗口和滚动截图；焦点窗口截图不再复用普通窗口截图逻辑，Windows 使用前台 HWND，macOS 使用 CoreGraphics 前台可见窗口，其他平台保持枚举窗口降级。
- 普通画板手动保存已改为保存 Excalidraw scene + PNG 快照，并把快照写入截图历史，模式为 `whiteboard`。
- 区域截图和录制中画板共用的标注 overlay 保存状态已修复：点击保存并写入截图历史或录制标注包后，胶囊保存按钮会稳定显示“已保存 / Saved”，不会被 Excalidraw 后续重复 `onChange` 重新标回“未保存 / Unsaved”。
- 标注 overlay 保存状态机新增“最后已保存场景”基准：只有当前场景 JSON 和最后成功保存的场景不同，才进入未保存状态；保存成功后同一场景的重复变更事件会被忽略。
- 保存动作新增明确的“保存中 / Saving”文案，保存失败时保持 dirty 状态，方便用户再次点击保存。
- `annotation-overlay.spec.ts` 已覆盖区域截图保存后截图历史新增、保存状态从 `Unsaved` 变为 `Saved`，并在短暂等待后仍保持 `Saved`。

本轮发布前验证：

```text
npm run build
npm run test:e2e -- annotation-overlay.spec.ts --workers=1
npm run test:e2e -- --workers=1
git diff --check
```

## 本轮已完成并纳入 v0.1.10

- 截图历史与钉图状态解耦：历史列表不再把旧的 `pinned=true` 当成“已钉图”展示；钉图窗口状态由独立 pin state 管理，历史项只保留截图本身和 `fixed` 固定状态，避免截图历史一打开就显示已钉图。
- 钉图窗口改为读取真实截图图片数据并通过 `screenshot.pin` 事件同步 `dataUrl`，避免钉图窗口空白或只显示占位状态。
- 截图进入画板能力补齐：从截图历史点击进入画板时，会把截图作为 Excalidraw image 元素和 file 数据导入普通画板；画板已打开时也能通过 `screenshot.whiteboard` 事件热导入，不再需要关闭重开。
- 普通画板保存保护已补齐：截图导入时不会被启动阶段的空场景覆盖，导入后的场景会写入白板持久化存储。
- 胶囊拖动和列表闪烁收敛：只有真实拖动结束或真实窗口移动结束才触发稳定/吸附；程序化 resize/move、列表弹出、按钮点击产生的窗口事件会被短时间忽略，降低胶囊高度抖动和闪烁。
- 胶囊可拖动区域收窄：`rf-shell`、`recorder-stage`、按钮、输入控件、下拉菜单、设置面板和确认面板都标记为 `no-drag`；只有胶囊主体和拖拽把手保留 `drag`，避免点击菜单时被 Wails 当成拖动窗口。
- 多屏吸附规则固定：吸附阈值统一为 32px，只允许吸附到当前工作区的外侧边缘；两块屏幕相接的内侧缝隙不会触发左右/上下吸附，减少跨屏边界误吸附。
- 本轮新增和更新测试覆盖：
  - 截图历史不会展示过期 pinned 状态。
  - 截图钉图会生成带真实 `data:image/*` 的 pin state。
  - 截图初始导入画板、画板已打开后的热导入。
  - 多显示器外侧边缘吸附、屏幕相接缝隙不吸附。
  - 胶囊/舞台/按钮/弹层的 Wails drag/no-drag 合同。

本轮发布前验证：

```text
npm run build
npm run test:e2e -- --workers=1
go test ./...
git diff --check
wails3 task windows:build ARCH=amd64
```

## 已完成

- 使用官方 Wails v3 CLI `v3.0.0-alpha2.109` 生成 React + TypeScript + Vite 工程。
- 工程位置已整理为 `RecordingFreedom/app/`。
- Go module 为 `github.com/lemon-casino/RecordingFreedom/app`，Wails 依赖锁定为 `github.com/wailsapp/wails/v3 v3.0.0-alpha2.109`。
- 实现第一版胶囊录制工具窗口：
  - 源选择：All screens / Screen 1..N / Region / Locked window。Program/Application 后端合同保留，但当前胶囊菜单不展示程序来源。
  - 音频：系统声音、麦克风、RNNoise 开关、麦克风设备、电平显示。
  - 摄像头：Windows sidecar writer 已接入；PIP 设置入口、结构化数据合同和透明置顶 PIP 编辑 overlay 已恢复开发，支持位置、圆形/方形、镜像、大小、透明边缘、拖动、缩放和录制中 manifest patch。
  - 控制：开始、暂停/继续、结束、状态、计时器。
  - 语言：简体中文、English；胶囊语言菜单和独立设置窗口都可切换，并通过 `settings.changed` 同步到其它窗口。
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
  - `DeviceService` 已从 `RecordingService` 中拆出，成为独立的视频源枚举入口：全部屏幕、单个屏幕、区域、锁定窗口和程序源。
  - Windows 当前环境下使用 Win32 API 枚举显示器、可见顶层窗口，并按进程聚合程序源。
  - macOS 已新增 `darwin && cgo` CoreGraphics 源枚举：`CGGetActiveDisplayList` 显示器、`CGWindowListCopyWindowInfo` 可见窗口、按 PID 聚合程序源。
  - macOS `darwin && !cgo` 会返回明确错误，避免误报 native source enumeration 已可用。
  - `DeviceService.ListSources()` 已把显示器虚拟桌面坐标带到前端，并新增 queued 的 `all-screens:virtual-desktop` 和 `region:custom` 源；`StartRequest.sourceGeometry` 与 manifest `source.geometry` 已落地。
  - 区域录制选择 overlay 已落地：`ShowRegionSelector()` 打开覆盖虚拟桌面的透明置顶窗口，前端支持十字光标、拖拽红框、尺寸浮标、最小尺寸拒绝和 `Esc` 取消；`CompleteRegionSelection()` 会把相对拖拽矩形转换为虚拟桌面逻辑坐标，生成带 geometry 和显示器 native id 的 `region:custom` source，并通过 `capture.region.selected` 事件回填胶囊来源。框选完成且尚未开始录制时，继续使用同一个透明 `region-overlay` 进入编辑态，只绘制红色边框、中心移动十字、八个缩放热区和右上尺寸/取消控件，不再创建窄条 WebView 编辑窗口。macOS 原生层会在设置 ScreenCaptureKit `sourceRect` 前把虚拟桌面逻辑坐标转换为显示器本地坐标。
  - `DeviceService` 新增 `MediaInventory` 合同，统一返回系统声音、麦克风、摄像头和 RNNoise 能力状态。
  - `DeviceService` 已新增 `MediaDeviceProvider` 替换边界；默认平台 provider 当前返回 queued inventory，测试可注入 provider 验证真实枚举、空结果和错误回退不影响前端合同。
  - Windows `MediaDeviceProvider` 已通过 MMDevice API 枚举 WASAPI render/capture endpoint：系统声音和麦克风返回真实 endpoint `NativeID`、friendly name、默认设备和 `enumerated` capability。
  - Windows `MediaDeviceProvider` 已通过 FFmpeg DirectShow 设备列表解析摄像头：存在 `tools/ffmpeg.exe` / PATH / `RECORDINGFREEDOM_FFMPEG_PATH` 时返回真实 camera device name、稳定 `camera:dshow:*` ID、原始 DirectShow `NativeID` 和 `sidecarEligible`；缺 FFmpeg 或无摄像头时返回明确 unavailable reason。Windows 摄像头 sidecar writer 已接入 FFmpeg DirectShow，开启摄像头时写入包内 `webcam.mp4`，缺少真实 sidecar 或 0 字节 sidecar 仍会被 ready 门禁拦截。
  - macOS 媒体设备已返回 CoreAudio 麦克风枚举和 FFmpeg AVFoundation 摄像头枚举；Linux 摄像头已按 `/dev/video*` 暴露 v4l2 sidecar 设备，系统声音/麦克风仍保持后续 PipeWire/PulseAudio 接入边界。
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
  - 新增 `internal/video` 视频采集合约：`CaptureConfig`、`Session`、`video-diagnostics.json` 和默认 unsupported 平台入口，为 ScreenCaptureKit/FFmpeg/PipeWire writer 提供同一生命周期目标。
  - 新增 Windows FFmpeg desktop writer：默认 Windows backend 为 `ffmpeg-desktop-capture`，旧 `windows-graphics-capture` 保留为兼容 alias。当前屏幕、单显示器区域和多屏全部屏幕优先使用 FFmpeg `ddagrab` / Windows Desktop Duplication API 录制鼠标，绕过 `gdigrab -draw_mouse 1` 的 GDI 光标重绘路径；多屏全部屏幕通过多路 `ddagrab + xstack` 合成虚拟桌面，锁定窗口仍使用 `gdigrab hwnd=`。录制中默认每 60 秒写入一个 segment，pause 会收尾当前 segment，resume 新开 segment，stop 合并为包内 `screen.mp4` 并写 `video-diagnostics.json`。启用系统声音或麦克风时，Windows 先用 WASAPI 写包内恢复/诊断 sidecar，再在停止阶段用 FFmpeg mux 到主 `screen.mp4`，manifest 把对应音频标记为 `muxed`；缺少 `ffmpeg` 时 preflight blocked，绕过 preflight 直接启动也会得到 failed package 和失败诊断，不写假媒体。
  - 新增 `recording.CreateVideoCaptureConfig()`：从 `StartRequest + RecordingWritePlan` 生成统一视频采集配置，真实后端复用同一份 source、profile、`screen.mp4` 和 `video-diagnostics.json` 路径合同；`sourceGeometry` 会进入 `video.CaptureConfig.SourceGeometry` 和 `video-diagnostics.json` 的 source geometry，供后续区域裁剪、多屏合成和平台 writer 诊断消费。
  - 新增 `recording.NativeBackendRuntime`：把 native `.rfrec` 写盘计划、视频 session、音频 session 和摄像头 sidecar session 生命周期串起来，提供 `Start()`、`Pause()`、`Resume()`、`Stop()`、单独 video/audio/camera 控制、RNNoise suppressor 生命周期和启动失败标记 `failed` 的统一入口。
  - 新增 `NativeBackendRuntime.SyncDiagnostics()`：把 video/audio/camera diagnostics 转成 manifest `diagnostics.sync`，统一输出 screen、system audio、microphone、webcam track 起止、duration、drop count、append failure、sample rate/frame rate 和 diagnostics 相对路径；webcam offset 会同步写入 `media.webcamStartOffsetMs`。
  - 新增 `recording.NativeRuntimeBackend`：把 runtime 包装成可注册的真实 `Backend`，平台后端只需要提供 video/audio/camera session factory，不需要重复实现包创建、状态转换、失败标记和 sync diagnostics。
  - 新增 macOS ScreenCaptureKit display/window/region video session：`screen:display-<CGDirectDisplayID>` 通过 `SCDisplay` 采集，`window:<CGWindowID>` 通过 `SCWindow` 采集，单显示器 `region:custom` 通过 `SCStreamConfiguration.sourceRect` 裁剪，三者都由 `SCStream` 输出真实 screen sample buffer，`AVAssetWriter` 写入包内 `screen.mp4`，停止时写 `video-diagnostics.json`；开启系统声音时，ScreenCaptureKit audio sample 写入同一个 `screen.mp4` 的 AAC 音轨，manifest 记录 `systemAudioStorage: "muxed"`。`application:<pid>` 解析代码保留给后续演进，但当前 capability 为 queued，不在初版验收范围内。macOS `native`/`sck` backend 已注册到 `NativeRuntimeBackend`。当前仍需 macOS 真机录制 smoke 验证，麦克风 mux 还未完成。
  - 新增 Windows FFmpeg DirectShow camera sidecar session：使用枚举到的 DirectShow 原生设备名启动 FFmpeg，pause/resume 走分段录制，stop 合并为包内 `webcam.mp4`；`RecordingFreedomService` 会在 preflight/start 前用 `MediaInventory.Cameras` 补齐 `deviceNativeId`，`video-smoke -camera` 同样走真实设备名。
  - 新增 macOS FFmpeg AVFoundation camera sidecar session 和 Linux FFmpeg v4l2 camera sidecar session：macOS 使用 AVFoundation 设备索引写 `webcam.mov`，Linux 使用 `/dev/video*` 写 `webcam.mov`。三平台 sidecar 均依赖 FFmpeg capability，缺 FFmpeg 时 preflight blocked。
  - 新增 `internal/exporter` FFmpeg PIP 导出器、Wails `ExportRecordingPackage()` 和 `cmd/pip-export-smoke`：按 `camera.pip` 的圆形/方形、镜像、位置、大小、边缘羽化和 `webcamStartOffsetMs` 合成包内 `exports/recording.mp4`，并保护原始 `screen.mp4` / `webcam.*` 不被覆盖。导出器会在安装输出前用 FFmpeg 解码首个视频帧，`pip-export-smoke` 返回 `outputVerified`。本机已用临时真实 MP4 素材跑通圆形镜像 PIP 与方形羽化 PIP 导出。
  - 新增 `cmd/video-smoke`：无 UI 真实视频录制验收入口，默认使用 `native` backend、自动选择第一个可用屏幕源，也支持 `-source-type=window` 和 `-source-type=region` 验证窗口/区域源；默认关闭音频和摄像头，停止后校验 `.rfrec` 包、`screen.mp4` 非 0 字节、`video-diagnostics.json`、manifest `ready` 和 `diagnostics.sync`。
  - 正式录制策略调整为 mux 优先：默认目标是把屏幕视频、系统声音和麦克风写入同一个主媒体 `screen.mp4`；包内 WAV sidecar 继续作为 smoke、fallback、恢复和诊断路径。
  - `internal/recpackage` 已新增音频存储形态合同：系统声音和麦克风分别通过 `systemAudioStorage` / `microphoneAudioStorage` 标记为 `sidecar` 或 `muxed`；fallback 写 `system-audio.wav` / `microphone.wav`，Windows 停止阶段 mux 成功后会把路径切到 `screen.mp4`。
  - `PackageService.ValidateReady()` 已在非 mock 包中校验已启用音频：`sidecar` 模式要求对应 WAV 存在、可读且大于空 header，`muxed` 模式会解析 `screen.mp4` 的 MP4 box 并要求存在 `soun` 音轨。
  - `internal/recpackage` 已新增 audio-only 包合同：manifest 使用 `recordingMode: "audio-only"` 和 `audioPath`，`CreateAudioOnly()` 不创建 `screen.mp4`，ready 门禁会校验主音频媒体的 `soun` 音轨，或校验明确声明的 WAV fallback sidecar。
  - 新增 `recording.AudioOnlyRuntime`、`AudioOnlyRuntimeBackend` 和 `RecordingService.StartAudioOnlyRecording()`：audio-only 录制不启动 video session，复用 `audio.CaptureSession`、RNNoise suppressor 生命周期、pause/resume/reset、sync diagnostics、`ValidateReady()` 和统一状态机；停止阶段会通过 FFmpeg 把 `audio.wav` 或 `system-audio.wav` + `microphone.wav` sidecar 封装为包内主音频 `audio.m4a`，并把 manifest 中已启用音轨标记为 `muxed`；Wails 服务已暴露 `PreflightAudioOnlyRecording()` 和 `StartAudioOnlyRecording()`，前端胶囊来源面板已接入视频/音频模式切换。
  - Windows 已新增纯 Go WASAPI capture source：麦克风采集会 downmix/resample 为 `48kHz / mono`，系统声音使用 loopback source，二者都走同一 audio pipeline 和 WAV sink。
  - 新增 `internal/audio/rnnoise`：迁移 RNNoise C 源码和旧项目 `LikelyVoiceEnhancement` 为可复用 C ABI；RNNoise C/H 源码已隔离到 `internal/audio/rnnoise/native` 子包，默认构建返回明确 unavailable，不做假降噪。
  - RNNoise capability 和 media enhancement 状态已改为读取 `rnnoise.Available()`：当前 release 标准为 `rnnoise_dynamic`，CI 按平台/架构编译 `tools/rnnoise.dll`、`tools/librnnoise.dylib` 或 `tools/librnnoise.so`，主程序运行时加载模块。模块缺失、架构不匹配或符号绑定失败时会显示 queued/blocked reason，不会假装降噪已生效。当前 CI/release artifact 要求动态 DSP smoke 与 `desktop-doctor -require-rnnoise` 全平台通过；Windows ARM64 不再是无 RNNoise 的例外架构。
  - 新增 `cmd/audio-smoke`：可在不启动 Wails UI 的情况下真实启动平台音频 source，当前通过 `RecordingService.StartAudioOnlyRecording()` 写入 `<DataRoot>/data/video/recording-*.rfrec/`；采集期间单流写 `audio.wav` sidecar，双流写 `system-audio.wav` / `microphone.wav` 分轨 sidecar，停止后用 FFmpeg 生成 `audio.m4a` 主音频、写入 `audio-diagnostics.json` 和 manifest sync diagnostics，并通过 `ValidateReady()` 后标记 `ready`。
  - 新增 `internal/pip` 画中画 preset 合同：`off`、`bottom-right`、`bottom-left`、`free`，并提供基础 overlay layout 计算。
  - 新增 `internal/recordingprofile` 录制参数合同：`standard/balanced/high`、`24/30/60 FPS`、`captureCursor`、`countdownSeconds`，供 settings、recording request 和 manifest 共用。
  - 新增 `recording.NormalizeStartRequest()`，统一校验 `sourceId/sourceType`，支持 `screen`、`all-screens`、`region`、`window`、`application`，归一化系统声音设备、麦克风设备、RNNoise、摄像头设备和 PIP preset。
  - `RecordingService` 支持 mock start/pause/resume/stop，但 mock 需要显式选择，只用于 preview smoke 和包结构验证。
  - `RecordingService` 已拆出 `recording.Backend` 接口，默认 backend 为平台 native/queued；macOS 走 `screencapturekit`，Windows 走 `ffmpeg-desktop-capture`，Linux 走 `pipewire-portal`。真实 ScreenCaptureKit/FFmpeg/PipeWire 后端按同一接口接入。
  - `recording.Backend.Stop()` 已升级为返回 `BackendStopResult`；真实后端可以返回 `SyncDiagnostics`，由 `RecordingService` 统一写入 manifest，再把包标记为 `ready`。
  - `RecordingService.Stop()` 已在写入 `ready` 前接入 `PackageService.ValidateReady()`，非 mock/native 包缺失非 0 字节 screen media 时会失败并把 manifest 标为 `failed`；摄像头开启时 webcam sidecar 也必须通过同一可读性门禁。
  - 新增 `recording.CreateNativeWritePlan()`，真实 ScreenCaptureKit/FFmpeg/PipeWire 后端后续应通过它统一创建 native `.rfrec` 写盘计划，避免各平台重复手写 source/audio/camera/recording manifest 映射。
  - 新增 backend selector 合同：默认/`auto`/`native` 按平台选择 native backend ID；`mock`/`mock-package` 才显式选择 mock。
  - backend selector 已升级为 native backend registry：真实平台后端可通过 `recording.RegisterNativeBackend()` 注册 factory；`native`、`sck`、`wgc`、`pipewire` 会优先选择已注册实现，未注册时才回退 queued backend。
  - queued native backend 当前用于未注册的平台 backend，如 `pipewire-portal` 或 `native-unsupported`；它不会创建包或写媒体，`Start()` 返回明确 queued error。已注册但 writer 未完成的 backend 必须通过 preflight 阻断，并在被直接调用时失败而不写假媒体。
  - `Bootstrap()` 已返回当前 backend 和 storage health，前端启动后底部状态条不必等第一次录制也能显示当前后端。
  - `Session` 新增 `backend` 字段，前端底部状态条显示当前录制后端，便于验证 mock 与真实平台后端切换。
  - `recording.StatusEvent` 已扩展为状态事件载荷，包含 `status`、`sessionId`、`packageDir`、`manifest`、`backend` 和 `message`。
  - `RecordingFreedomService` 已通过 Wails `recording.status` 事件发出 preparing、recording、paused、stopping、ready、failed 状态；`StartMockRecording()` 兼容入口也转发到同一事件流。
  - `internal/recpackage` 已实现 typed `.rfrec` 包服务：创建唯一目录、写入 manifest、更新状态、校验相对媒体路径、扫描 recoverable 包、恢复 recoverable 包。
  - `internal/recpackage` 已新增 `CreateNative()` 原生写盘计划合同：为真实 ScreenCaptureKit/FFmpeg/PipeWire 后端初始化 ready-to-write `.rfrec` 包、`screen.mp4` / `webcam.mov` 相对路径、diagnostics 路径、`cache/` 和 `exports/` 目录，但不会创建假媒体文件或伪造 sync diagnostics。
  - `internal/recpackage` 已新增 `ValidateReady()` ready 前媒体探测门禁：mock 包只接受非空 `screen.mock.txt` marker；非 mock 包拒绝 mock marker，并要求 screen media 可读且非 0 字节；摄像头开启时要求 webcam sidecar 可读且非 0 字节。
  - 恢复动作只接受 app-managed `data/video` 内部 `.rfrec` 包；恢复已有 manifest 时写入 `ready` / `completedAt`，缺失 manifest 但存在非 0 字节 `screen.*` 时重建最小 manifest。
  - mock start 通过 `recpackage.CreateMock()` 创建 `data/video/recording-*.rfrec/manifest.json` 和 `screen.mock.txt`。
  - stop 会先把 manifest 写为 `finalizing`，完成后写为 `ready` 并记录 `completedAt`。
  - mock manifest 会记录 `microphoneDeviceId` 和 camera `deviceId`，让后续真实管线接入不改变包结构。
  - manifest 明确标记 `diagnostics.mock = true`，不伪装为真实录制。
  - manifest 已新增 `diagnostics.sync` 音画同步诊断合同，覆盖 screen、system audio、microphone、webcam 的时间线基准、track offset、duration、drop count、append failure 和 pause segments。
  - `PackageService.PatchSyncDiagnostics()` 已落地，后续真实 ScreenCaptureKit/FFmpeg/PipeWire 后端必须通过它写入同步诊断。
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
  - `RecordingFreedomService.ShowPIPOverlay`
  - `RecordingFreedomService.UpdatePIPOverlay`
  - `RecordingFreedomService.HidePIPOverlay`
  - `RecordingFreedomService.ShowRegionSelector`
  - `RecordingFreedomService.CompleteRegionSelection`
  - `RecordingFreedomService.CancelRegionSelection`
  - `RecordingFreedomService.ListSources`
  - `RecordingFreedomService.ListMediaDevices`
  - `RecordingFreedomService.PreflightRecording`
  - `RecordingFreedomService.PreflightAudioOnlyRecording`
  - `RecordingFreedomService.ScanRecordingPackages`
  - `RecordingFreedomService.RecoverRecordingPackage`
  - `RecordingFreedomService.StartRecording`
  - `RecordingFreedomService.StartMockRecording`
  - `RecordingFreedomService.StartAudioOnlyRecording`
  - `RecordingFreedomService.PauseRecording`
  - `RecordingFreedomService.ResumeRecording`
  - `RecordingFreedomService.StopRecording`
  - `recording.status` event type，载荷包含 status、sessionId、packageDir、manifest、backend、message
  - `capture.region.selected` event type，载荷包含 selected source、geometry、cancelled 和 error
  - `settings.changed` event type，载荷为保存后的 settings；用于同步胶囊、设置窗口和区域 overlay 的语言状态
- 前端服务层已改为优先调用 Wails generated bindings：
  - 新增 `loadBootstrap()` 适配层，一次读取 app data、录制状态、源列表、媒体设备、恢复扫描、设置和能力矩阵。
  - 新增 `showSettingsWindow()` / `hideSettingsWindow()` 适配层；Wails 桌面环境调用原生窗口，浏览器预览打开 `/settings` fallback。
  - 新增 `subscribeRecordingStatus()` 适配层；Wails 桌面环境订阅 `recording.status`，浏览器预览使用 no-op fallback。
  - `loadBootstrap()` 已读取后端返回的当前 backend，初始化胶囊底部 `Backend: ...` 状态。
  - `loadSettings()` / `saveSettings()` 已接入 Go `SettingsService`；浏览器预览使用 `localStorage` fallback。
  - 胶囊语言、源、系统声音、麦克风、RNNoise、摄像头设置会 debounce 保存。
  - 语言切换已接入全局前端 i18n 显示层，且只保留简体中文和 English：点击语言后，胶囊主控、弹窗、独立设置窗口、区域 overlay、状态条、预检摘要、存储状态和能力矩阵会通过 `settings.changed` 同步到对应语言；后端 ID、manifest 字段、路径和诊断字符串保持稳定不翻译。
  - 独立设置窗口已接入质量、FPS、录制光标、倒计时控件；开始录制会把当前录制 profile 传给后端并写入 manifest。
  - 摄像头菜单已接入 PIP preset 下拉和“编辑画中画”入口；开始录制会把当前 PIP 配置传给后端，摄像头关闭时 manifest 使用 `off`。
  - PIP 编辑 overlay 已落地为独立透明 `/#/pip-overlay` Wails window，支持中心拖动、八方向缩放、圆形/方形切换、镜像切换、关闭 PIP，并尝试使用 WebView `getUserMedia()` 显示实时摄像头预览；overlay 请求会携带所选摄像头的 device id、native id 和显示名，预览会在授权后枚举 `videoinput` 并优先匹配同一摄像头，匹配失败才回退默认摄像头。拖动过程只预览位置，松手后通过 `UpdatePIPOverlay()` 写回 settings 和活动录制 manifest，避免高频写盘。
  - PIP overlay 使用 Wails `SetContentProtection()` 尝试排除窗口捕获：Windows 映射到 `SetWindowDisplayAffinity`，macOS 映射到 `NSWindowSharingNone`，Linux 当前为不支持返回。原始 `screen.mp4` 仍保持干净，PIP 合成留给导出阶段。
  - 音频菜单已接入系统声音设备下拉；开始录制会把当前 `systemDeviceId` 传给后端，系统声音关闭时 manifest 不保留旧 device id。
  - 独立设置窗口已接入 `CaptureService` 能力矩阵，按后端返回展示平台、backend、权限和 `Ready/Queued/Blocked/Unsupported` 状态。
  - 独立设置窗口新增 Recording backend 行，启动后显示当前 backend，并说明 queued native backend 仍会被 preflight 阻止。
  - 独立设置窗口新增 Preflight 行，显示最近一次录制预检状态、backend 和原因。
  - 独立设置窗口已接入 `Bootstrap().appData`，Storage 显示真实 `videoDir`，App data 显示真实数据根目录。
  - 独立设置窗口新增 Storage health 行，显示 `data/video` 可写性、可用空间和建议最低空间。
  - 独立设置窗口新增 Data root 输入和 Apply 操作；修改的是应用数据根，录制仍进入其下的 `data/video`。
  - 独立设置窗口的 Storage 行新增打开目录操作，调用后端 `OpenVideoDirectory()` 打开当前应用管理的 `data/video`，便于用户直接查验录制包位置。
  - 独立设置窗口的 Recording package 行新增打开包操作，调用后端 `OpenRecordingPackage()` 打开最近一次真实 `.rfrec` 包目录，便于查看 `manifest.json`、`screen.mp4` / `audio.m4a`、`audio-diagnostics.json` 和 `video-diagnostics.json`；后端会拒绝 `data/video` 外部路径和非 `.rfrec` 目录，缺失 manifest 的崩溃包仍可打开用于诊断。
  - 独立设置窗口新增 Recovery 行，存在 recoverable `.rfrec` 包时可以触发恢复并重新扫描。
  - `loadSources()` 现在读取 Go `DeviceService` 返回的 `devices.CaptureSource`。
  - `loadMediaDevices()` 现在读取 Go `DeviceService` 返回的 `devices.MediaInventory`。
  - 源模型包含 `available`、`capability`、`unavailableReason`、虚拟桌面坐标和 native id，用于区分已枚举源、待原生后端源、全部屏幕和区域录制入口。
  - 胶囊来源面板的视频模式已显示真实可用的 `All screens / 全部屏幕`、按显示器编号的 `Screen 1..N / 屏幕 1..N`、`Region / 区域` 和 `Locked window / 锁定窗口`；不可启动的 all-screens 不再作为用户入口展示。Program/Application 后端合同保留给后续演进，但当前菜单和能力矩阵都不展示程序来源。鼠标移入或键盘聚焦某个单屏来源时，会调用 `ShowScreenIndicator()` 在对应物理屏幕中央显示黑底大号编号标识，离开或选择后调用 `HideScreenIndicator()` 隐藏；定位优先按源物理 bounds 匹配 Wails screen，匹配失败再按 display index 兜底。区域选择 overlay 已接通，只有框选成功后才会把 `region:custom` 写为当前录制源；框选后未录制时使用透明 overlay 编辑选区，可移动、缩放或取消，录制中同一个透明 overlay 切到鼠标穿透 recording 模式，只绘制红色边框持续标识录制范围，不再创建四个窄条 WebView 边框窗口。macOS 单显示器区域会绑定显示器 native id 并进入 ScreenCaptureKit `sourceRect` crop writer，Windows 区域会使用物理像素矩形进入 FFmpeg `gdigrab` crop，Linux 区域 crop 和 macOS/Linux 多屏合成 writer 在真实平台实现完成前保持 queued。
  - 锁定窗口入口已改为窗口列表视图，用户选择的是枚举出来的具体 window source，而不是“当前活动窗口”。
  - 麦克风和摄像头下拉已从后端媒体设备合同读取；浏览器预览仍有 mock fallback。
  - `scanRecordingPackages()` 已接入恢复扫描，胶囊底部状态条显示 recoverable 包数量或 clean。
  - Wails 桌面环境中开始/暂停/继续/停止会调用 Go `RecordingFreedomService`；开始录制已切到长期入口 `StartRecording()`。
  - 胶囊窗口已使用统一 `applyRecordingStatus()` 入口消费 Wails status event 和后端 session 返回值，底部状态条显示 `Status: ...`。
  - 开始录制前会先调用 `PreflightRecording()`；`blocked` 时前端停在失败状态并展示原因，`ready` / `warning` 才继续进入 `StartRecording()`。
  - Wails `StartRecording()` 服务入口也会强制执行同一套 `PreflightRecording()` 门禁；blocked 时直接返回错误，不进入 recorder backend，也不会创建 `.rfrec` 包，避免绕过前端流程产生假录制尝试。
  - 胶囊来源面板新增视频/音频模式切换：视频模式保留屏幕、区域、锁定窗口来源；音频模式显示 `Audio only` / `单独录音`，开始前调用 `PreflightAudioOnlyRecording()`，通过后调用 `StartAudioOnlyRecording()`，并禁用摄像头/PIP 入口。Wails `StartAudioOnlyRecording()` 服务入口也会强制执行同一套 audio-only preflight，blocked 时不进入 backend、不创建 `.rfrec` 包。
  - 默认设置已调整为系统声音、麦克风和 RNNoise 全部关闭，只保留默认设备 ID。用户显式打开对应音频开关后才进入 preflight；preview 不再把用户尚未开启的音频/降噪链路显示为已启用。
  - 浏览器 `vite preview` 环境中自动 fallback 到前端 mock，方便纯 UI 开发。
- `build/config.yml` 已替换模板占位信息为 RecordingFreedom 元数据。
- 新增新仓库可用的 GitHub Actions：
  - `RecordingFreedom/.github/workflows/ci.yml`
  - `RecordingFreedom/.github/workflows/release.yml`
  - workflow 假定 `RecordingFreedom/` 是新仓库根目录，因此 `APP_DIR=app`。
  - CI 包含 bindings 检查、frontend build、Go test、preview smoke、macOS native contract、Windows/macOS/Linux Wails build。
  - Release 在 tag `v*` 时先执行 release gate（bindings 检查、frontend build、Go test、RNNoise dynamic module build/test、preview smoke、release config check、`rnnoise_dynamic` doctor），通过后用 `rnnoise_dynamic` 构建全平台 artifacts、生成 SHA256SUMS 并发布 GitHub Release。
  - 新增 `cmd/release-config-check`：轻量检查 `.github/workflows/ci.yml` 和 `.github/workflows/release.yml` 是否仍保留 RNNoise native artifact gate、Windows FFmpeg bootstrap、Windows portable zip verifier 和 release notes 能力边界，避免后续 workflow 改动把验收门禁悄悄删掉。
  - 新增无 GUI `cmd/preview-smoke` 验证入口：
  - 默认使用临时 data root，不污染开发目录；可通过 `-keep` 保留生成包。
  - 验证 settings 持久化、`data/video` storage health、preflight、mock start/pause/resume/stop、ready manifest、`screen.mock.txt` 非空、RNNoise/camera/PIP manifest 合同和恢复扫描。
- 新增无 GUI `cmd/desktop-doctor` 依赖检查入口：
  - 输出 JSON，检查 app data、`data/video`、storage health、当前 backend、能力矩阵和 Windows FFmpeg 依赖状态。
  - 默认只报告状态，不阻断 preview CI/release；传 `-require-video` 时会把当前平台真实 screen/window video 能力作为硬门禁，缺 FFmpeg 或平台 writer queued 时退出非零。
- 新增 Windows FFmpeg 依赖准备脚本 `scripts/ensure-windows-ffmpeg.ps1`：下载 BtbN FFmpeg-Builds static Windows zip、按 `checksums.sha256` 校验、写入 `app/tools/ffmpeg.exe` / `ffprobe.exe` 和 `THIRD_PARTY_FFMPEG.txt`；Windows CI/release build 会先执行该脚本，再用 `desktop-doctor -require-video` 阻断缺依赖的 artifact。
- Release Windows preview artifact 已调整为 portable zip，包含 `recordingfreedom.exe` 和 `tools/ffmpeg.exe`；app 运行时会按现有解析顺序从 exe 同级 `tools/` 找到 FFmpeg。
- 新增 `RecordingFreedom/README.md`，用于新仓库根目录说明、开发命令、验证命令和 roadmap。
- 已在 GitHub runner 上验证 preview release 链路：
  - `v0.1.0-preview.4` 的 Release Gate、Windows build、macOS build、Linux build 和 Publish GitHub Release 均已通过。
  - GitHub Release 已产出 Windows x64、macOS arm64、Linux x64 三个平台 preview 二进制和对应 SHA256SUMS。
  - `v0.1.0-preview.9` 已发布为 GitHub prerelease：Release Gate、Windows x64、macOS arm64、Linux x64 和 Publish GitHub Release 均通过。Release URL：`https://github.com/lemon-casino/RecordingFreedom/releases/tag/v0.1.0-preview.9`。
  - `v0.1.0-preview.9` 产物包含 `RecordingFreedom-windows-x64-v0.1.0-preview.9-portable.zip`、`RecordingFreedom-macos-arm64-v0.1.0-preview.9`、`RecordingFreedom-linux-x64-v0.1.0-preview.9` 和三个平台 SHA256SUMS。
  - `v0.1.0-preview.10` 已发布为 GitHub prerelease：Release Gate、Windows x64、macOS arm64、Linux x64 和 Publish GitHub Release 均通过。Release URL：`https://github.com/lemon-casino/RecordingFreedom/releases/tag/v0.1.0-preview.10`。
  - `v0.1.0-preview.10` 产物包含 `RecordingFreedom-windows-x64-v0.1.0-preview.10-portable.zip`、`RecordingFreedom-macos-arm64-v0.1.0-preview.10`、`RecordingFreedom-linux-x64-v0.1.0-preview.10` 和三个平台 SHA256SUMS。
  - `v0.1.0-preview.10` 纳入真实麦克风设备枚举、真实麦克风电平事件 `audio.level`、macOS CoreAudio 输入设备/PCM 采集代码路径、Windows GUI subsystem 检查和 FFmpeg/DirectShow 子进程隐藏窗口配置。
  - `v0.1.0-preview.11` 已发布为 GitHub prerelease：Release Gate、Windows x64、macOS arm64、Linux x64 和 Publish GitHub Release 均通过。Release URL：`https://github.com/lemon-casino/RecordingFreedom/releases/tag/v0.1.0-preview.11`。
  - `v0.1.0-preview.11` 产物包含 `RecordingFreedom-windows-x64-v0.1.0-preview.11-portable.zip`、`RecordingFreedom-macos-arm64-v0.1.0-preview.11`、`RecordingFreedom-linux-x64-v0.1.0-preview.11` 和三个平台 SHA256SUMS。
  - `v0.1.0-preview.11` 修复 Windows 默认麦克风不再退化为 `microphone:default`，录制中锁定来源/音频/摄像头配置，并新增区域录制持久边框窗口。
  - `v0.1.0-preview.12` 已发布为 GitHub prerelease：Release Gate、Windows x64、macOS arm64、Linux x64 和 Publish GitHub Release 均通过。Release URL：`https://github.com/lemon-casino/RecordingFreedom/releases/tag/v0.1.0-preview.12`。
  - `v0.1.0-preview.13` 已发布为 GitHub prerelease：Release Gate、Windows x64、macOS arm64、Linux x64 和 Publish GitHub Release 均通过。Release URL：`https://github.com/lemon-casino/RecordingFreedom/releases/tag/v0.1.0-preview.13`。
  - `v0.1.0-preview.13` 修复胶囊窗口周围透明 WebView 灰底/阴影伪影、单屏标识窗口编号居中与圆角尺寸、区域录制框选编辑态改为透明 overlay，避免窄条 WebView 黑块/白条和编辑遮挡。
  - `v0.1.0-preview.14` 修复区域录制开始后的持久边框，录制态改为鼠标穿透透明 overlay，只保留红色边框，避免四个窄条 WebView 窗口露出浅色背景和关闭按钮；同时清理 macOS CoreAudio 麦克风枚举中的 deprecated property element annotation。
  - `v0.1.0-preview.15` 已发布为 GitHub prerelease：Release Gate、Windows x64、macOS arm64、Linux x64 和 Publish GitHub Release 均通过。Release URL：`https://github.com/lemon-casino/RecordingFreedom/releases/tag/v0.1.0-preview.15`。
  - `v0.1.0-preview.15` 把 Windows clean-machine 验收工具纳入 portable zip：`tools/desktop-doctor.exe`、`tools/video-smoke.exe`、`tools/audio-smoke.exe` 和 `tools/run-windows-portable-smoke.ps1`，并把 CI/release 门禁和 portable zip 校验扩展到这些工具。
  - `v0.1.0-preview.7` 曾因 Linux Wails build tag 使用空格拼接 `gtk3 rnnoise_native` 导致 Linux build 失败，已在 `v0.1.0-preview.8` 前修正为 `gtk3,rnnoise_native` 并纳入 `cmd/release-config-check` 门禁。
  - `v0.1.0-preview.8` 曾因 Windows runner 的 FFmpeg 准备脚本下载链路失败而未发布，已在 `v0.1.0-preview.9` 前改为 BtbN FFmpeg-Builds GitHub 源、`curl.exe` 重试下载和按 asset name 校验 `checksums.sha256`。
  - 最新 `main` CI 已通过 Bindings/Frontend/Go、macOS Native Capture Contracts、Windows/macOS/Linux Wails Build；其中 Windows Wails Build 已通过 FFmpeg bootstrap、RNNoise gate 和 video gate。
  - 后续 preview tag 会自动标记为 GitHub prerelease。

## 验证结果

已通过：

```bash
go test ./internal/...
GOOS=darwin CGO_ENABLED=0 go test -c ./internal/devices
wails3 generate bindings -ts -i
npm run build
go test ./...
go test -tags gtk3 ./...
go run ./cmd/desktop-doctor
go run ./cmd/preview-smoke
go run ./cmd/audio-smoke -duration=1s
go run ./cmd/video-smoke -duration=3s -keep
go run ./cmd/video-smoke -source-type=all-screens -duration=3s -keep
go run ./cmd/video-smoke -source-type=region -duration=3s -keep
go run ./cmd/video-smoke -source-type=window -duration=3s -keep
go run ./cmd/video-smoke -duration=5s -pause-after=2s -pause-duration=1s -keep
go run ./cmd/video-smoke -duration=3s -system -keep
go run ./cmd/video-smoke -duration=3s -microphone -keep
go run ./cmd/video-smoke -duration=3s -system -microphone -keep
go run ./cmd/video-smoke -duration=5s -pause-after=2s -pause-duration=1s -system -microphone -keep
git diff --check
wails3 build
```

本机 Windows FFmpeg 依赖准备：

```powershell
.\scripts\ensure-windows-ffmpeg.ps1
```

该脚本本机已下载并校验 FFmpeg `N-125365-g9a01c1cb6a-20260630` BtbN static build，写入 `RecordingFreedom/app/tools/ffmpeg.exe`；随后 `go run ./cmd/desktop-doctor -require-video` 返回 `ok: true`，screen/window/ffmpeg required checks 均为 `ready`。GitHub Windows runner 也已在 `v0.1.0-preview.12` release workflow 中通过该脚本准备 FFmpeg，并通过 portable zip 验证。

本机 Windows 未准备 FFmpeg 时的预期阻断：

```bash
go run ./cmd/desktop-doctor -require-video
go run ./cmd/video-smoke -duration=1s
```

上述两条在本机未准备 `app/tools/ffmpeg.exe` 时返回非零是正确结果：doctor 报告 screen/window/ffmpeg 三个 required 检查 blocked，`video-smoke` 在 preflight 阶段阻断，不会创建假媒体或假 ready 包。执行 `.\scripts\ensure-windows-ffmpeg.ps1` 后应重新运行 `desktop-doctor -require-video` 和真实桌面 `video-smoke`。

本机 Windows FFmpeg 视频 smoke 记录：

- `screen`：`recording-2026-07-01-07-24-06-804.rfrec` 进入 `ready`，`screen.mp4` 114596 bytes，97 frames，duration 3256ms。
- `all-screens`：`recording-2026-07-01-07-25-20-533.rfrec` 进入 `ready`，`screen.mp4` 415451 bytes，102 frames，duration 3423ms。
- `region`：`recording-2026-07-01-07-24-25-451.rfrec` 进入 `ready`，`screen.mp4` 17383 bytes，101 frames，duration 3374ms。
- `window`：`recording-2026-07-01-07-24-40-729.rfrec` 进入 `ready`，`screen.mp4` 90508 bytes，96 frames，duration 3223ms。
- `pause/resume`：`recording-2026-07-01-07-25-36-234.rfrec` 进入 `ready`，分段合并后的 `screen.mp4` 231186 bytes，162 frames，duration 5429ms。
- `system audio mux`：`recording-2026-07-01-07-42-46-941.rfrec` 进入 `ready`，`systemAudioPath=screen.mp4`，`systemAudioStorage=muxed`。
- `microphone mux`：`recording-2026-07-01-07-43-00-455.rfrec` 进入 `ready`，`microphoneAudioPath=screen.mp4`，`microphoneAudioStorage=muxed`。
- `system + microphone mux`：`recording-2026-07-01-07-46-23-318.rfrec` 进入 `ready`，`systemAudioStorage=muxed`，`microphoneAudioStorage=muxed`，主媒体 `screen.mp4` 194624 bytes。
- `pause/resume + system + microphone mux`：`recording-2026-07-01-07-43-26-669.rfrec` 进入 `ready`，manifest 中系统声音和麦克风均指向 `screen.mp4`；`ffprobe` 确认主媒体存在 AAC `48000Hz / 2ch` 音轨。

本轮视频收口 smoke 记录：

- `screen + system audio`：`recording-2026-07-01-08-45-42-298.rfrec` 进入 `ready`，`screen.mp4` 77911 bytes，82 frames，duration 2764ms；`systemAudioPath=screen.mp4`，`systemAudioStorage=muxed`，`systemAudioSamples=192000`。
- `region`：`recording-2026-07-01-08-45-45-862.rfrec` 进入 `ready`，`screen.mp4` 14214 bytes，70 frames，duration 2350ms。
- `locked window`：`recording-2026-07-01-08-45-48-864.rfrec` 进入 `ready`，`screen.mp4` 101429 bytes，67 frames，duration 2237ms。
- Windows camera sidecar 代码收口后，`npm run build`、`go test . ./internal/capture ./internal/devices ./internal/preflight ./internal/recording ./internal/recpackage ./internal/video ./cmd/video-smoke ./cmd/desktop-doctor`、`go run ./cmd/desktop-doctor -require-video` 和 `go run ./cmd/video-smoke -duration=2s` 均已通过；最新 screen smoke 包为 `recording-2026-07-01-09-15-48-510.rfrec`，`screen.mp4` 212473 bytes，65 frames，duration 2199ms。
- `go run ./cmd/video-smoke -duration=2s -camera` 已执行到真实设备选择阶段，但本机 `DeviceService` 没有返回可用于 sidecar 的摄像头，结果为 `camera sidecar requested but no available sidecar camera was returned by DeviceService`。因此 Windows DirectShow camera sidecar 是代码路径完成，仍不能记录为真实摄像头 smoke 已通过。

本轮视频验收 smoke 记录（摄像头不纳入本轮）：

- `screen + system audio`：`recording-2026-07-01-09-33-01-927.rfrec` 进入 `ready`，`screen.mp4` 89952 bytes，114 frames，duration 3805ms；系统声音 mux 到 `screen.mp4`，`systemAudioSamples=288000`。
- `all-screens`：`recording-2026-07-01-09-33-22-141.rfrec` 进入 `ready`，`screen.mp4` 386410 bytes，101 frames，duration 3369ms。
- `region`：`recording-2026-07-01-09-33-39-766.rfrec` 进入 `ready`，`screen.mp4` 12964 bytes，102 frames，duration 3405ms。
- `locked window`：`recording-2026-07-01-09-33-58-145.rfrec` 进入 `ready`，`screen.mp4` 104308 bytes，96 frames，duration 3220ms。
- `pause/resume`：`recording-2026-07-01-09-34-11-710.rfrec` 进入 `ready`，分段合并后 `screen.mp4` 197483 bytes，133 frames，duration 4455ms。
- `screen + system audio + microphone`：`recording-2026-07-01-09-34-32-830.rfrec` 进入 `ready`，系统声音和麦克风均 mux 到 `screen.mp4`，`systemAudioSamples=292800`，`microphoneSamples=143520`；`ffmpeg -v error -i screen.mp4 -f null -` 通过，确认主媒体可解码。

本轮长一点视频 smoke 记录（摄像头不纳入本轮）：

- `1m screen + system audio + microphone`：`recording-2026-07-01-09-37-05-743.rfrec` 进入 `ready`，`screen.mp4` 2269237 bytes，1829 frames，duration 60993ms；系统声音和麦克风均 mux 到 `screen.mp4`，`systemAudioSamples=5707200`，`microphoneSamples=2880000`；`ffmpeg -v error -i screen.mp4 -f null -` 通过。
- `5m screen + pause/resume`：`recording-2026-07-01-09-38-35-292.rfrec` 进入 `ready`，`screen.mp4` 5161172 bytes，9017 frames，duration 300574ms；manifest sync message 为 `FFmpeg desktop capture wrote 2 segment(s).`，确认暂停/继续分段合并路径生效；`ffmpeg -v error -i screen.mp4 -f null -` 通过。
- `20m screen + system audio + microphone + pause/resume`：`recording-2026-07-01-09-46-38-233.rfrec` 进入 `ready`，`screen.mp4` 119345863 bytes，36028 frames，duration 1200935ms；系统声音和麦克风均 mux 到 `screen.mp4`，`systemAudioStorage=muxed`，`microphoneAudioStorage=muxed`，`systemAudioSamples=114340800`，`microphoneSamples=57609600`；manifest sync message 为 `FFmpeg desktop capture wrote 2 segment(s).`，确认暂停/继续分段合并路径在 20 分钟长录中生效；`ffmpeg -v error -i screen.mp4 -f null -` 通过。
- 仍未完成：外部 clean-machine 长时长真实录制 smoke、目标桌面 RNNoise 听感诊断，以及 macOS/Linux 真机录制验收。当前 Windows 桌面已从 `v0.1.0-preview.15` portable artifact 跑通 3 秒 runner 矩阵。
- 用户反馈“麦克风设备展示是假的、语音波动没有监听”后，已删除前端假麦克风列表和假波形初始化；后端新增真实麦克风电平监听入口，前端订阅 `audio.level` 事件显示 RMS/peak 推导的真实电平。无可用麦克风时 UI 显示不可用，不再展示虚构设备。
- 用户反馈 Windows 默认麦克风仍像假设备后，Windows WASAPI 枚举已改为保留真实 `microphone:wasapi:<endpoint-id>` / `system-audio:wasapi:<endpoint-id>` 作为选择 ID，默认设备只通过 `isDefault` 和 subtitle 标记；旧请求 `microphone:default` / `system-audio:default` 在 preflight 中会映射到真实默认 endpoint，避免 UI 能选真实设备但预检误挡。
- 录制开始后，胶囊 UI 会锁定来源、区域、系统声音、麦克风、RNNoise 和摄像头选择；只有结束录制后重新启用。区域录制完成框选后，未录制时由透明 `region-overlay` 显示可移动、可缩放、可取消的红色选区；开始录制后同一个 overlay 切为鼠标穿透 recording 模式，只绘制一圈红色边框，持续标识被录制范围，不再显示四条窄 WebView 窗口。
- 音频采集会话已从同步处理改为默认 `128` 帧有界异步队列：采集回调只入队，worker 顺序执行 RNNoise / sink 写入；队列满时丢弃新输入帧并写入 `audio-diagnostics.json` 的 `droppedSamples` 和 message；暂停会通过 worker flush 哨兵确认前序帧处理完成后再 reset RNNoise，避免暂停边界污染。`audio-diagnostics.json` 现在会写入 `queue.capacity`、`queue.maxDepth`、`queue.flushCount`、`queue.droppedFrames` 和 `queue.droppedSamples`，方便后续 30 分钟以上长录查看队列水位。

有 C 工具链的环境可用以下命令验证 RNNoise 原生 DSP：

```bash
go test -tags rnnoise_dynamic ./internal/audio/rnnoise ./internal/recording
```

本机需先通过 `scripts/build-rnnoise-windows.ps1` 或 `scripts/build-rnnoise-unix.sh` 在 `app/tools/` 生成对应平台动态模块；CI/release gate 会执行 RNNoise dynamic DSP 和 recording runtime 定向测试，并在 artifact 构建时启用 `rnnoise_dynamic` 后运行 `desktop-doctor -require-rnnoise`。

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
- `internal/audio` 覆盖 WAV sidecar header/data 写入、格式变化拒绝、mono resampler、`CaptureSession` source -> pipeline -> sink -> diagnostics 运行时、有界队列满时丢弃输入帧并记录 diagnostics、queue capacity / maxDepth / flushCount / droppedFrames / droppedSamples 诊断，以及暂停前 flush 队列再 reset RNNoise。
- `internal/video` 覆盖 Windows FFmpeg audio mux 参数：单音频输入直接映射，系统声音 + 麦克风使用 `amix` 混成主媒体 AAC 音轨。
- `internal/video` 覆盖视频 capture config 归一化、source geometry 写入 diagnostics、`video-diagnostics.json` 写盘和默认平台 session 明确 unsupported。
- `internal/audio/rnnoise` 覆盖非动态模块 fallback；`rnnoise_dynamic` 构建下会运行时加载平台动态库并跑 native frame 处理测试。CI/release gate 会先从 RNNoise C 源编译平台模块，再执行动态 DSP 定向测试；Windows x64/ARM64 主程序保持 `CGO_ENABLED=0`，macOS/Linux 通过 `dlopen` 加载 dylib/so。
- `internal/recording` 覆盖 `CreateAudioCaptureConfig()`：打开/关闭系统声音、麦克风和 RNNoise 时，音频设备、sidecar 输出路径、diagnostics 路径和系统声音不降噪策略保持稳定。
- `internal/recording` 覆盖 `CreateVideoCaptureConfig()`：source、source geometry、profile、`screen.mp4` 输出路径和 `video-diagnostics.json` 路径保持稳定；区域 source 会携带用户框选的虚拟桌面矩形进入 video config。
- `internal/recording` 覆盖 `NativeBackendRuntime`：会创建并控制 video session；有音频时创建并控制 audio session；无音频时不启动 audio session；RNNoise suppressor 会传入并在停止时关闭；视频 session 或 RNNoise 不可用时初始化失败并把已创建 native 包标记为 `failed`。
- `internal/recording` 覆盖 `NativeBackendRuntime.SyncDiagnostics()`：runtime 生成的 screen/system/microphone track diagnostics 可被 `recpackage.PatchSyncDiagnostics()` 接受并写回 manifest。
- `internal/recording` 覆盖 `NativeRuntimeBackend`：Start/Pause/Resume/Stop 会驱动 runtime，Stop 返回 sync diagnostics，Start 失败会把已创建 native 包标记为 `failed`；backend registry 可选择注册后的 runtime backend。
- 本机 Windows audio-only M4A smoke 已确认默认麦克风 WASAPI capture 生成 ready 的 audio-only `.rfrec` 包：`recording-2026-07-01-09-29-58-163.rfrec`，manifest 为 `recordingMode: "audio-only"`、`status: "ready"`、`audioPath: "audio.m4a"`，麦克风 track 指向 `audio.m4a` 且 `microphoneAudioStorage=muxed`；包内保留 `audio.wav` sidecar 作为恢复/诊断证据。`ffmpeg -v error -i audio.m4a -f null -` 通过，确认 `audio.m4a` 可解码。
- 有界音频队列接入后，本机 Windows 1 秒 audio-only smoke 再次通过：`recording-2026-07-02-01-19-25-977.rfrec` 进入 `ready`，`audio.m4a` 存在；`audio-diagnostics.json` 记录麦克风 `framesReceived=100`、`samplesReceived=48000`、`samplesWritten=48000`、`droppedSamples=0`。
- 队列水位 diagnostics 接入后，本机 Windows 1 秒 audio-only smoke 通过：`recording-2026-07-02-01-30-06-400.rfrec` 进入 `ready`，`audio.m4a` 存在；`audio-diagnostics.json` 记录 `queue.capacity=128`、`queue.maxDepth=0`、`queue.flushCount=0`、`queue.droppedFrames=0`、`queue.droppedSamples=0`，麦克风 `framesReceived=100`、`samplesReceived=48000`、`samplesWritten=48000`。
- `cmd/audio-smoke` 已扩展为直接读取 `audio-diagnostics.json` 并在 JSON 输出里返回 `queue`、`microphone`、`systemAudio` 摘要；本机 Windows 1 秒 smoke `recording-2026-07-02-01-32-35-448.rfrec` 进入 `ready`，命令输出直接显示 `queue.capacity=128`、`queue.droppedFrames=0`、`queue.droppedSamples=0` 和麦克风 `framesReceived=100`。
- 本机 Windows system audio smoke 已确认 WASAPI loopback source 在有活动系统播放时可以写入真实样本：`system-audio.wav` 614444 bytes，`framesReceived=160`，`samplesReceived=153600`，`samplesWritten=153600`，`sampleRate=48000`，`channels=2`，duration 约 `1600ms`。

发布产物验收：

- `scripts/verify-windows-portable.ps1` 已增强为解压检查 portable zip：验证 `recordingfreedom.exe` 是 x64 GUI PE，`tools/ffmpeg.exe`、`tools/ffprobe.exe`、`tools/desktop-doctor.exe`、`tools/video-smoke.exe` 和 `tools/audio-smoke.exe` 是 x64 PE，并在 Windows host 上执行 FFmpeg/FFprobe `-version`。校验还会解析 `tools/run-windows-portable-smoke.ps1` 并检查 runner 内含 doctor、video-smoke、audio-smoke、FFmpeg 环境变量、区域/窗口/系统声音/麦克风/RNNoise 入口。
- Windows portable zip 的 release workflow 已新增 clean-machine 验收工具打包：`tools/desktop-doctor.exe`、`tools/video-smoke.exe`、`tools/audio-smoke.exe` 和 `tools/run-windows-portable-smoke.ps1`。该 runner 解压后即可运行，不依赖 Go/Node/Wails 源码环境，默认把 smoke 包写入 portable 根目录下的 `data-smoke/data/video`。
- 新增 `scripts/verify-windows-preview-release.ps1`：可按 tag 或最近 release 下载 GitHub Windows x64 portable zip 和 `SHA256SUMS-windows-x64*.txt`，校验 SHA256 后复用 `verify-windows-portable.ps1`。该脚本用于 release asset 完整性复验，不替代 clean-machine 真实 screen/region/window 录制 smoke。
- `v0.1.0-preview.15` Windows portable zip 已用真实 GitHub Release 下载复验通过：SHA256 `99E1EB5C425B925F0F0269EE364C95A4F0CB7278EEE73C8E6D5A31196A8CD7DD` 匹配，`recordingfreedom.exe` 为 x64 GUI PE，`tools/ffmpeg.exe` / `tools/ffprobe.exe` 为 x64 PE 且 `-version` 可执行，`tools/desktop-doctor.exe`、`tools/video-smoke.exe` 和 `tools/audio-smoke.exe` 均为 x64 console PE，`tools/run-windows-portable-smoke.ps1` 已打入 zip。
- `v0.1.0-preview.15` Windows portable artifact-run smoke 已在当前 Windows 桌面通过：解压 release zip 后运行 `tools/run-windows-portable-smoke.ps1 -Duration 3s -ContinueOnError`，12/12 step 成功，包含 `desktop-doctor`、screen/all-screens/region/window、pause/resume、system audio、microphone、system+microphone、audio-only microphone RNNoise、audio-only system 和 audio-only system+microphone RNNoise；生成 11 个 ready `.rfrec` 包。随包 `ffprobe` 确认代表性 `screen.mp4` 含 H.264 video + AAC audio，代表性 `audio.m4a` 含 AAC audio。

PIP 合同测试：

- `internal/pip` 覆盖非法 preset normalize、右下角/左下角布局、`off` 隐藏布局、非法 preset 拒绝。
- `internal/settings` 覆盖非法 PIP preset 回退为默认 `bottom-right`。
- `internal/recpackage` 覆盖 camera enabled 但缺失/非法 PIP preset 时写入默认 `bottom-right`。
- `internal/exportplan` 覆盖 ready 包 PIP 导出计划、screen-only 隐藏 PIP、包外路径拒绝、输出路径逃逸拒绝、mock 包拒绝、可见 PIP 缺失 webcam sidecar 拒绝、手工破坏 diagnostics 路径拒绝。

录制请求合同测试：

- `internal/recording` 覆盖缺失 source id / 非法 source type 拒绝、系统声音/麦克风/摄像头设备默认值、关闭流时清理旧 device id、非法 microphone gain 拒绝。
- `internal/recpackage` 覆盖关闭系统声音、麦克风或摄像头时，manifest 不保留对应旧 device id。

录制 backend selector 测试：

- `internal/recording` 覆盖默认 backend 为平台 native/queued。
- `internal/recording` 覆盖 `native` 请求按平台映射到 `screencapturekit`、`ffmpeg-desktop-capture`、`pipewire-portal` 或 `native-unsupported`。
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

- macOS ScreenCaptureKit display/window/region 录制已接入代码路径，但仍需要真机 smoke：授权屏幕录制后运行 `go run ./cmd/video-smoke -duration=1m`、`go run ./cmd/video-smoke -source-type=window -duration=1m`、`go run ./cmd/video-smoke -source-type=region -duration=1m` 和 `go run ./cmd/video-smoke -duration=5m -pause-after=10s -pause-duration=2s`，并确认 `screen.mp4` 可播放、包进入 `ready`、`video-diagnostics.json` 和 `diagnostics.sync` 正确。Application/Program 后端合同仍保留给 smoke 和后端演进，但不在当前用户菜单展示。
- ScreenCaptureKit 系统声音 mux 已接入代码路径但仍需 macOS 真机 smoke；麦克风 mux 仍未完成。
- Windows FFmpeg video writer 已接入代码路径，CI/release 已能准备并打包 `tools/ffmpeg.exe`；本机真实 smoke 已验证 screen、all-screens、region、locked-window、pause/resume segment merge、系统声音 mux、麦克风 mux、系统声音 + 麦克风混音 mux，并已完成 1 分钟、5 分钟和 20 分钟录制包可解码验证。GitHub `v0.1.0-preview.15` Windows portable artifact 已内置 `tools/run-windows-portable-smoke.ps1`、通过下载复验，并在当前 Windows 桌面完成 3 秒 artifact-run 矩阵；仍需外部 clean machine 执行该 runner，验证真实录制、失败诊断、空间/权限边界和更长时长。
- 真实区域录制 crop writer：当前已有 `region` 源类型、`source.geometry` 合同、跨显示器透明十字框选 overlay、拖拽红框、尺寸浮标、选择事件，以及进入 `video.CaptureConfig` / `video-diagnostics.json` 的 writer 边界 geometry。macOS 单显示器区域已接入 ScreenCaptureKit `sourceRect` 写入 `screen.mp4`；Windows 已通过 FFmpeg desktop crop 接入；Linux crop writer 仍未完成。
- 真实全部屏幕多显示器合成录制：当前已有 `all-screens` 源类型和虚拟桌面 bounds；Windows FFmpeg 可按虚拟桌面录制，macOS/Linux 多屏合成仍保持 queued。
- 真实 PipeWire / XDG Portal 录制。
- 真实 macOS CoreAudio 麦克风枚举和 Linux PipeWire 音频设备枚举；当前 Windows WASAPI endpoint 枚举已完成，macOS system audio 使用 ScreenCaptureKit 默认系统混音流。
- 真实 CoreAudio/PipeWire 音频采集；当前 Windows WASAPI 麦克风采集已通过 smoke，Windows system loopback 已通过有播放源真实样本 smoke，Windows FFmpeg 视频后端已通过 `NativeBackendRuntime` 调用 WASAPI 音频运行时，并在停止阶段把系统声音/麦克风 mux 到主 `screen.mp4`。macOS CoreAudio 麦克风枚举和 PCM 采集代码路径已完成，仍需 macOS 真机 smoke、视频麦克风 mux/sync 验收；Linux 音频源和长录同步仍未完成。
- 真实 audio-only 录制模式；当前已完成 `.rfrec` audio-only 包格式、`audio.m4a` 主媒体路径、WAV sidecar 写盘、停止阶段 FFmpeg M4A 封装、ready 前 `soun` 音轨门禁、`audio-smoke` 包级验收入口、RecordingService/Wails 后端入口、胶囊 UI 模式入口和 audio-only preflight。macOS/Linux 真实音频源、RNNoise 目标桌面实录听感和长录同步仍未完成。
- 摄像头/画中画已完成第一条可验证代码闭环：结构化 `camera.pip` 合同贯通 settings、start request、manifest、export plan、前端摄像头设置面板、透明 PIP 编辑 overlay、三平台 FFmpeg sidecar writer、Wails 导出入口和 FFmpeg PIP 导出器。仍不能把没有真机摄像头录制 smoke 的平台说成已完成真实摄像头验收。
- RNNoise native DSP 的 C 源码和 Go wrapper 已迁移并隔离；CI/release gate 已切到动态模块定向测试。当前能力矩阵会按 `rnnoise.Available()` 动态显示：带 `rnnoise_dynamic` 且能加载随包模块的 release artifact 显示可用并允许预检，未带模块或模块不可加载的本地构建仍显示 queued/blocked。全平台 artifact 构建已要求 `desktop-doctor -require-rnnoise` 通过；目标桌面的 `audio-smoke -rnnoise` 实录听感和长录诊断仍需补。
- macOS AVFoundation、Windows DirectShow、Linux v4l2 摄像头 sidecar 写入已接入 FFmpeg 实现；WebView 预览与 sidecar 设备的最佳匹配逻辑已接入。真实设备预览匹配效果、macOS/Linux 真机 sidecar smoke、PipeWire 摄像头替换和长时长同步仍保持后续项。
- 真实 FFmpeg PIP 导出执行已接入并通过本机临时包 smoke；导出后首帧解码门禁已接入。暂停片段精确同步、真实录制包矩阵导出、ffprobe 音轨/时长门禁和长录同步仍保持后续项。
- 真实音画同步时间戳采集和容器级媒体 probe；当前只完成 manifest 诊断合同、mock 标记、ready 前非 0 字节媒体门禁。

## 已知提示

- `wails3 build` 在 Windows 上输出过 `"uname": executable file not found in $PATH`，但构建最终成功。这是 Wails 构建脚本里的跨平台探测提示，当前不是阻塞项。
- `npm run build` 输出 Tailwind content 警告；当前 UI 使用自写 CSS，没有依赖 Tailwind 生成样式，后续可移除模板残留或补 Tailwind 配置。
- 本机全局 `go version` 是 1.24.1，但 Go toolchain 自动切换并下载了 Wails v3 需要的 Go 1.25.11。

## 下一步

1. 当前验收边界在语言设置、语音/音频录制和视频录制之外，已把摄像头 sidecar/PIP 恢复为进行中能力；下一步补真实设备 smoke、预览设备匹配和导出同步门禁。
2. A1 已完成 Windows WASAPI system audio/microphone endpoint 枚举；继续补 macOS CoreAudio 与 Linux PipeWire/PulseAudio 枚举。
3. Windows 麦克风 PCM 采集、系统声音 loopback 样本写盘、FFmpeg desktop video writer、runtime backend 注册、Windows portable zip FFmpeg 准备路径、停止阶段音视频 mux、音频会话有界队列和队列水位/丢帧诊断已落地；下一步补有 C 工具链本机的 `audio-smoke -rnnoise`、目标桌面 RNNoise 实录听感/诊断、30 分钟以上音频队列/内存水位记录，以及 macOS/Linux 音频源。
4. 在真实 macOS 机器补 ScreenCaptureKit screen/window/region/system-audio `video-smoke`，确认权限、可播放性、`video-diagnostics.json` 和 `diagnostics.sync`。
5. 下一版 Windows portable artifact 发布后，在真实桌面解压并执行 `.\tools\run-windows-portable-smoke.ps1`，覆盖 screen/all-screens/region/locked-window、pause/resume segment merge、音频 mux、RNNoise 和 audio-only 组合；随后补 20 分钟长录验收。
6. 把 release workflow 从 preview executable 升级为正式安装包、签名和公证流水线。
