# 02. Go + React + Wails v3 技术架构

## 目标目录

后续新仓库和当前仓库内的代码根目录保持一致：

```text
RecordingFreedom/
  docs/
  app/
    go.mod
    Taskfile.yml
    main.go
    internal/
      appdata/
      capture/
      devices/
      recording/
      audio/
      package/
      settings/
      diagnostics/
      platform/
    frontend/
      package.json
      src/
        app/
        components/
        features/
        i18n/
        services/
        styles/
  native/
    macos/
    windows/
    linux/
  .github/
    workflows/
```

`docs/` 先落地在 `RecordingFreedom/docs/`。后续 scaffold 时，代码放入 `RecordingFreedom/app/`。

## Wails v3 使用边界

Wails v3 负责：

- 应用生命周期。
- 系统托盘。
- 胶囊窗口、设置窗口、后续编辑/导出窗口。
- Go 服务到 React 的方法绑定。
- Go 后端向前端发送录制状态事件。
- 平台构建入口。

Wails v3 当前仍应按 alpha 风险处理：

- 在 `go.mod` 和构建脚本中锁定版本。
- 每次升级 Wails 前单独开升级任务。
- 所有前端调用必须经过 `services/` 适配层，避免 Wails 绑定 API 变化扩散到组件。

## Go 服务划分

核心服务：

- `AppService`：启动、托盘、窗口、平台信息。
- `SettingsService`：设置读取、保存、迁移、默认值。
- `AppDataService`：应用内部数据目录、`data/video`、缓存目录、空间检查。
- `DeviceService`：屏幕、窗口、程序、麦克风、系统音频、摄像头枚举。
- `CaptureService`：平台能力探测、权限状态、源预检。
- `RecordingService`：录制状态机、开始、暂停、继续、停止、丢弃。
- `AudioService`：麦克风降噪、混音、音频诊断。
- `PackageService`：录制包创建、manifest 更新、恢复扫描和保守恢复。
- `ExportPlanService`：导出前校验 `.rfrec`、screen/webcam sidecar、PIP layout 和 sync diagnostics，生成流式导出计划。
- `DiagnosticsService`：日志、音画同步指标、构建与运行时信息。

前端不直接访问文件系统，不直接做录制编码，不直接维护录制包状态。

## 音频增强边界

当前代码已新增 `internal/audio`，用于承载麦克风增强合同：

- `RNNoiseSampleRate = 48000`。
- `RNNoiseFrameDuration = 10ms`。
- `RNNoiseFrameSamples = 480`。
- `NoiseSuppressor` 接口按 480-sample mono frame 处理。
- `Enhancer` 只允许对 `microphone` stream 应用 `rnnoise`。
- `system-audio` stream 即使请求 `rnnoise` 也会被明确 bypass，不会进入 suppressor。
- 暂停/继续时后续录制后端必须调用 `Reset()` 清理 pending frame 和 RNNoise 状态。

这一层是 native RNNoise 接入前的长期合同，不是 fake DSP。真实 CoreAudio/WASAPI/PipeWire 麦克风采集完成后，把麦克风 PCM 转成 48kHz mono，再接入 `NoiseSuppressor` 实现。

## PIP 布局合同

当前代码已新增 `internal/pip`，用于承载摄像头画中画 preset：

- `off`：不参与画中画导出。
- `bottom-right`：默认右下角。
- `bottom-left`：左下角。
- `free`：为后续自由拖拽保留；在自定义坐标字段落地前使用默认右下角确定性布局。

`pip.Layout()` 根据画布尺寸计算安全矩形，保证 overlay 不越界。设置服务会 normalize 非法 preset，录制包 manifest 会在摄像头关闭时写 `off`。PIP 仍属于预览/导出层能力，不会把摄像头烘焙进原始 `screen.mp4`。

## 导出计划合同

当前代码已新增 `internal/exportplan`，用于承载后续导出器的入口合同：

- 只接受 app-managed `data/video` 内部的 `.rfrec` 包。
- 只接受 `status = ready` 的录制包。
- 默认输出为包内相对路径 `exports/recording.mp4`，拒绝绝对路径或 `..` 逃逸路径。
- `screenVideoPath`、`webcamVideoPath`、`audioDiagnosticsPath`、`videoDiagnosticsPath` 都必须留在 `.rfrec` 包内。
- 默认拒绝 `diagnostics.mock = true` 或 `timelineBase = mock` 的包，避免把 UI shell mock 当成真实视频导出。
- `RequireSync` 模式下必须存在 `diagnostics.sync`，为 screen、system audio、microphone 和 webcam offset 对齐提供依据。
- PIP 可见时必须存在非 0 字节 webcam sidecar，并且调用方必须提供 canvas 尺寸来计算 `pip.Rect`。

这一层不执行 FFmpeg、不写导出文件；真实导出器后续只能消费该计划，不能绕过它自行读取任意路径。

## 录制 Profile 合同

当前代码已新增 `internal/recordingprofile`，用于承载真实编码后端会消费的录制参数：

- `quality`：`standard`、`balanced`、`high`。
- `fps`：`24`、`30`、`60`。
- `captureCursor`：是否录制鼠标光标。
- `countdownSeconds`：开始录制前倒计时，当前规范化上限为 10 秒。

`SettingsService`、`RecordingService` 和 `PackageService` 共用同一套 profile 默认值与归一化逻辑。独立设置窗口可以修改这些值；开始录制请求会携带 `recording`，manifest 也会写入同一结构。当前这只是合同和 UI 控制，真实平台后端接入后再把这些参数映射到 ScreenCaptureKit、Windows.Graphics.Capture 或 PipeWire 编码配置。

## 录制后端扩展边界

当前代码已在 `internal/recording` 中拆出 `Backend` 接口：

- `Start()`：创建或接管 `.rfrec` 录制包，并开始写入媒体。
- `Pause()`：暂停平台采集，并让后续 RNNoise 状态能够 reset。
- `Resume()`：继续平台采集。
- `Stop()`：flush writer、关闭平台资源，返回 `BackendStopResult`；真实后端把 sync diagnostics 放进结果，由 `RecordingService` 统一写 manifest 后再进入 `ready`。

`RecordingService` 只负责：

- 校验状态流转。
- 创建 app-managed `data/video` 根目录。
- 调用当前 backend。
- 写入 `finalizing`、`ready`、`failed` 等 manifest 状态。
- 在 stop 阶段读取 `BackendStopResult.SyncDiagnostics`，通过 `PackageService.PatchSyncDiagnostics()` 写入音画同步诊断合同，而不是让平台层手写 manifest。
- 在写入 `ready` 前调用 `PackageService.ValidateReady()`，确保非 mock 包至少有可读、非 0 字节的 screen media；摄像头开启时 webcam sidecar 也必须可读且非 0 字节；音频按 manifest 存储形态校验，`sidecar` 要求非空 WAV，`muxed` 要求主 `screen.mp4` 存在 MP4 `soun` 音轨。
- 保持 `Session.backend`，让 UI 和诊断能区分 `mock-package`、`screencapturekit`、`windows-graphics-capture`、`pipewire-portal` 等实现。

当前默认 backend 是 `mock-package`，只用于 UI shell 和包结构验证。后续真实后端必须实现同一个接口，不能绕过 `RecordingService` 或 `PackageService`，也不能把视频写到 `data/video` 之外。

## 原生写盘计划合同

当前 `internal/recording` 已新增 `CreateNativeWritePlan()`，作为真实 ScreenCaptureKit/WGC/PipeWire 后端的统一入口。它会归一化 `StartRequest`，把 source、recording profile、system audio、microphone、RNNoise、camera 和 PIP preset 映射到 `recpackage.CreateNative()`，用于后端开始采样前初始化 `.rfrec` 包：

- 在 app-managed `data/video` 下创建唯一 `recording-*.rfrec` 目录。
- 写入 `status = recording` 的 typed `manifest.json`。
- screen 默认相对路径固定为 `screen.mp4`。
- 摄像头开启时 webcam sidecar 默认相对路径固定为 `webcam.mov`；摄像头关闭时不写 webcam 路径，PIP preset 为 `off`。
- 创建 `cache/` 与 `exports/` 子目录，供持续写盘临时状态和后续导出使用。
- 返回绝对写入计划：screen、webcam、`audio-diagnostics.json`、`video-diagnostics.json`、cache 和 exports 路径。
- 不创建假媒体文件，不写 0 字节 `screen.mp4` 或 `webcam.mov`；这些文件必须由真实 native writer 在采样后持续写入。
- 不写 `diagnostics.sync`；真实后端停止或恢复时必须用真实 sample timestamp 通过 `PatchSyncDiagnostics()` 写入。

平台后端不应各自手写 manifest 映射；应调用 `recording.CreateNativeWritePlan()` 拿到 screen、webcam、diagnostics、cache 和 exports 的绝对写入路径，再由平台 writer 持续写入媒体。这个合同是 macOS ScreenCaptureKit 最小录制的前置包初始化层，不代表真实采集、编码或音画同步已经实现。

## 录制 Backend 选择合同

当前代码已新增 `internal/recording` backend selector：

- 默认值、`auto`、`mock`、`mock-package` 都选择 `mock-package`，保证 UI shell 和 `.rfrec` 包结构可持续验证。
- `RECORDINGFREEDOM_RECORDING_BACKEND=native` 会按平台选择 queued native backend：macOS `screencapturekit`、Windows `windows-graphics-capture`、Linux `pipewire-portal`。
- 也可以显式请求 `screencapturekit`、`windows-graphics-capture` 或 `pipewire-portal`，当前都会进入 queued native backend。
- queued native backend 不创建录制包、不写媒体文件，`Start()` 会返回明确错误。
- `Bootstrap()` 返回当前 `backend`，前端底部状态条和预检都使用同一个 backend ID。

这个合同用于后续替换真实后端，而不是临时开关。真实 backend 未实现前，native backend 会被 `PreflightRecording()` 阻止；只有 `mock-package` 可以在 UI shell 阶段继续生成明确标记为 mock 的 `.rfrec` 包。

## 录制预检合同

当前代码已新增 `internal/preflight`，用于在真正开始录制前统一检查请求和平台能力：

- 先调用 `recording.NormalizeStartRequest()`，拒绝非法 source、非法 profile、非法音频/摄像头请求。
- 检查当前 source 是否仍存在于 `DeviceService.ListSources()`。
- 检查系统声音、麦克风、摄像头 device id 是否仍存在于 `MediaInventory`。
- 检查 `CaptureService` 的 screen/window/program、system audio、microphone、RNNoise、camera sidecar、PIP export 能力。
- 检查 `AppDataService.StorageStatus()`，确保 `<DataRoot>/data/video` 可创建、可写，并把可用空间未知或低于建议阈值作为 warning。
- 返回 `ready`、`warning`、`blocked` 三种状态和逐项 `checks`。

`mock-package` backend 下，native capture 能力仍为 `queued` 时会返回 `warning`，允许 UI shell 创建 mock `.rfrec` 包，但不会宣称真实录制可用。queued native backend 或真实 backend 下，同样的 queued/blocked 能力会返回 `blocked`，用于阻止不可用录制开始。

Wails facade 暴露 `PreflightRecording(req)`。前端开始录制前会先调用它：`blocked` 时不启动，`warning` 时继续 UI shell/mock 流程并显示预检摘要。

## 前端结构

React 负责 UI 和轻量状态：

- `features/capsule-recorder/`：胶囊工具窗口。
- `features/source-picker/`：屏幕/窗口/程序选择。
- `features/settings/`：设置窗口。
- `features/audio/`：音频菜单、电平、降噪状态。
- `features/camera/`：摄像头菜单和 PIP 预设。
- `services/backend.ts`：Wails Go 绑定的唯一适配层。
- `services/mockBackend.ts`：第一阶段 UI Shell 的 mock adapter。
- `styles/tokens.css`：颜色、尺寸、动效 token。

React 状态原则：

- 录制状态以后端事件为准。
- 设备列表由 `DeviceService` 返回，前端只缓存当前视图所需数据。
- 启动阶段优先通过 `RecordingFreedomService.Bootstrap()` 一次拿到 app data、录制状态、源列表、媒体设备、恢复扫描、设置和能力矩阵，减少多服务请求的竞态。
- 可推导数据不重复存 state。
- 弹窗、菜单、焦点状态只在组件局部维护。

## 应用内部数据目录

用户要求新录制视频等放在软件内部 `data/video` 下。跨平台实现采用“应用托管数据根目录 + 固定相对路径”：

```text
<RecordingFreedomAppData>/
  data/
    video/
      recording-2026-06-30-13-45-22-123.rfrec/
        manifest.json
        screen.mp4
        webcam.mov
        audio-diagnostics.json
        recovery.log
    cache/
    logs/
    settings.json
```

开发模式默认：

```text
RecordingFreedom/data/video/
```

生产模式默认：

- macOS: `~/Library/Application Support/RecordingFreedom/data/video/`
- Windows: `%APPDATA%/RecordingFreedom/data/video/`
- Linux: `~/.local/share/RecordingFreedom/data/video/`

不能写入 macOS `.app` bundle、Windows `Program Files` 或 Linux 安装目录，因为这些位置在正式安装后可能只读。

设置中可以允许用户选择新的应用数据根目录，但内部结构仍保持 `data/video`，不让录制文件散落到任意裸目录。

当前 `AppDataService` 已支持可指定数据根目录：

- `RECORDINGFREEDOM_DATA_DIR` 仍是最高优先级的开发/测试覆盖项。
- 桌面设置通过 `RecordingFreedomService.SetDataRoot(rootDir)` 修改应用数据根目录。
- 默认应用数据目录写入 `data-root.json` 指针，让下次启动能够找到用户指定的新根目录。
- `SetDataRoot()` 会创建并探测目标目录可写性，再返回新的 `<Root>/data/video`。
- `StorageStatus()` 会探测 `<Root>/data/video` 是否可写，读取平台可用空间，并返回 `ready` / `warning` / `blocked` 状态；当前建议长录制至少保留 1 GiB 可用空间。
- 录制中禁止切换数据根，避免持续写盘中的 `.rfrec` 包跨目录断裂。

当前独立设置窗口会读取 `Bootstrap().appData.videoDir` 并显示实际路径，同时提供数据根目录输入、Apply 操作和 Storage health 行。浏览器预览 fallback 显示 `data/video`，Wails 桌面运行显示平台真实路径和可写/空间状态。

## 设置服务合同

当前代码使用 `internal/settings` 实现 `SettingsService`，设置文件固定为：

```text
<RecordingFreedomAppData>/settings.json
```

`settings.json` v1 持久化：

- `locale`：`zh-CN`、`en`。旧的或非法 locale 会归一为默认 `zh-CN`。
- `source`：最近选择的源 ID 和源类型。
- `storage`：当前应用数据根目录，录制包仍固定在其下的 `data/video`。
- `recording`：质量、FPS、录制光标、倒计时。
- `audio`：系统声音、系统声音设备、麦克风、麦克风设备、RNNoise 开关、麦克风增益。
- `camera`：摄像头开关、摄像头设备、PIP preset。
- `window`：最小化到托盘等窗口行为。

Wails 暴露 `GetSettings()`、`SaveSettings()`、`SetDataRoot()`、`ShowSettingsWindow()` 和 `HideSettingsWindow()`。前端启动时加载设置，胶囊中的语言、音频、摄像头和源选择变化后会 debounce 保存。语言切换由前端 i18n 显示层即时应用到胶囊、弹窗、设置窗口、状态条、预检和能力矩阵；后端枚举、路径、manifest 字段和诊断 ID 保持稳定。胶囊窗口的设置按钮只负责打开独立 `/settings` Wails window，不再把复杂设置塞进透明胶囊窗口。

## 事件流

录制事件统一由 Go 后端发出：

- `devices.changed`
- `permissions.changed`
- `recording.preparing`
- `recording.started`
- `recording.paused`
- `recording.resumed`
- `recording.stopping`
- `recording.ready`
- `recording.recoverable`
- `recording.failed`
- `audio.level`
- `storage.warning`

当前代码已落地第一层录制状态事件：

- Wails `RegisterEvent[recording.StatusEvent]("recording.status")` 生成类型化前端事件合同。
- `RecordingFreedomService.StartRecording()` 会先发 `preparing`，成功后发 `recording`，失败后发 `failed`。
- `PauseRecording()` / `ResumeRecording()` / `StopRecording()` 会发 `paused`、`recording`、`stopping`、`ready` 或 `failed`。
- 事件载荷包含 `status`、`sessionId`、`packageDir`、`manifest`、`backend` 和 `message`，供胶囊窗口、设置窗口、诊断面板共享。
- 前端通过 `subscribeRecordingStatus()` 订阅 `recording.status`，并用同一个状态更新入口刷新录制状态、录制包路径、backend 和状态消息。

前端只根据后端事件和后端 session 返回值更新 UI，不凭按钮点击直接宣称录制成功。真实 ScreenCaptureKit/WGC/PipeWire 后端接入后必须继续使用同一个事件合同。

## 第一阶段接口

UI Shell 阶段就定义最终接口形状：

```ts
type CaptureSourceType = "screen" | "window" | "application";
type RecordingState = "idle" | "preparing" | "recording" | "paused" | "stopping" | "ready" | "failed";

type StartRecordingRequest = {
  sourceId: string;
  sourceType: CaptureSourceType;
  sourceName?: string;
  recording: {
    quality: "standard" | "balanced" | "high";
    fps: 24 | 30 | 60;
    captureCursor: boolean;
    countdownSeconds: number;
  };
  audio: {
    system: boolean;
    systemDeviceId?: string;
    microphone: boolean;
    microphoneDeviceId?: string;
    noiseSuppression: boolean;
    microphoneGain: number;
  };
  camera: {
    enabled: boolean;
    deviceId?: string;
    pipPreset: "off" | "bottom-right" | "bottom-left" | "free";
  };
};
```

mock adapter 必须遵守这个接口，后续真实 Go 绑定直接替换实现。

Wails facade 已提供长期入口 `StartRecording()`。`StartMockRecording()` 仅保留为兼容旧 UI shell 调用的转发入口，新前端代码应调用 `StartRecording()`。

当前后端已新增 `recording.NormalizeStartRequest()` 作为真实后端前置合同：

- `sourceId` 必填，`sourceType` 只能是 `screen`、`window`、`application`。
- `recording` 会通过 `internal/recordingprofile` 归一化，非法 quality/FPS 回退为默认值，倒计时限制在 0-10 秒。
- 系统声音开启时会保留或默认 `systemDeviceId`，关闭时清空。
- 麦克风关闭时会清空麦克风 device id、关闭 RNNoise、清零 gain。
- 摄像头关闭时会清空摄像头 device id，并把 PIP preset 写为 `off`。
- 非法 PIP preset 会回退到默认 `bottom-right`；非法麦克风 gain 会被拒绝。

## 设备枚举合同

`DeviceService` 当前分为两类合同：

- `ListSources()` 返回屏幕、窗口、程序源，类型为 `devices.CaptureSource`。
- `ListMediaDevices()` 返回系统声音、麦克风、摄像头和 RNNoise 能力状态，类型为 `devices.MediaInventory`。
- `MediaDeviceProvider` 是媒体设备枚举的替换边界。当前平台 provider 返回明确的 queued inventory；后续 macOS/Windows/Linux 真实枚举只实现 provider，不改变 Wails facade、前端设置窗口或 preflight 合同。

所有设备项必须包含：

- `id`：稳定选择 ID，后续写入录制 manifest。
- `type`：源类型或媒体设备类型。
- `name` / `subtitle`：给 UI 展示。
- `available` / `capability` / `unavailableReason`：明确区分真实可用设备、未实现 native backend、权限不可用或平台不可用。

在真实 native backend 完成前，不能用 mock 设备伪装可用能力。Queued backend 必须返回 `native-backend-queued`，并说明原因。

真实 provider 接入时仍必须遵守同一模型：系统声音、麦克风、摄像头分别返回稳定 `id` 和平台 `nativeId`；权限缺失或运行时不可用时返回 `available = false` 与可读 `unavailableReason`，不能返回空列表来表示失败。

## 平台能力探测合同

`CaptureService` 当前实现为能力矩阵，不直接开始录制。它通过 `RecordingFreedomService.GetCaptureCapabilities()` 和 `Bootstrap().capabilities` 暴露给前端。

能力状态固定为：

- `available`：当前平台和代码路径已经可用。
- `queued`：合同已定义，但 native backend 尚未落地。
- `blocked`：平台支持但运行时权限或环境阻塞。
- `unsupported`：当前平台没有计划中的 backend。

当前能力项：

- `sourceEnumeration`：屏幕、窗口、程序源枚举。
- `screenRecording` / `windowRecording` / `applicationRecording`：三类录制目标。
- `systemAudio`：系统声音采集。
- `microphone`：麦克风采集。
- `microphoneEnhancement`：RNNoise 麦克风降噪。只允许处理麦克风 PCM，不能处理系统声音。
- `cameraSidecar`：摄像头 sidecar 采集。
- `pipExport`：后期导出阶段的画中画合成。
- `packageRecovery`：`.rfrec` 恢复扫描和保守恢复。

独立设置窗口显示这套能力矩阵，用于区分“已可用”“已排期”“权限阻塞”和“不支持”。录制按钮后续接入真实 backend 前，不能只凭前端开关推断 native 能力。

## 录制包服务合同

当前代码使用 `internal/recpackage` 实现文档中的 `PackageService`：

- `CreateMock()`：在 `data/video` 下创建唯一 `.rfrec` 目录、写入 mock media marker 和 typed `manifest.json`。
- `CreateNative()`：底层包服务，在 `data/video` 下创建真实 native 后端的 `.rfrec` 包和写盘计划，但不创建假媒体、不伪造同步诊断；平台录制后端应通过 `recording.CreateNativeWritePlan()` 调用它。
- `PatchStatus()`：更新 manifest 状态，停止时支持先写 `finalizing`，再写 `ready` 和 `completedAt`。
- `PatchSyncDiagnostics()`：为真实后端写入 screen、system audio、microphone、webcam 的同步诊断，包括时间线基准、track offset、duration、drop count、append failure 和 pause segments。
- `ValidateReady()`：ready 前媒体探测门禁。mock 包只接受 `diagnostics.mock = true` + 非空 `screen.mock.txt`；非 mock 包拒绝 mock marker，并要求 `screenVideoPath` 指向包内可读、非 0 字节文件；摄像头开启时同样要求 `webcamVideoPath` 可读且非 0 字节。
- `ReadManifest()` / `WriteManifest()`：统一 JSON 读写，拒绝绝对路径或逃逸包目录的媒体路径。
- `Scan()`：扫描 `*.rfrec`，把 `recording`、`paused`、`finalizing` 或缺失 manifest 但存在 screen media 的包标记为 recoverable。
- `Recover()`：只接受 `data/video` 内部的 `.rfrec` 包；对已有 recoverable manifest 的包写入 `ready` 和 `completedAt`；对缺失 manifest 但有非 0 字节 `screen.*` 的包重建最小 manifest；所有恢复结果标记 `diagnostics.recovered = true`。

`WriteManifest()` 当前还会归一化关闭的摄像头：摄像头关闭时清空 `webcamVideoPath`、`webcamStartOffsetMs` 和 webcam sync track，避免旧 sidecar/offset 污染下一次录制。所有 sync 诊断里的路径同样必须是包内相对路径，不能是绝对路径或 `..` 逃逸路径。

前端通过 `RecordingFreedomService.ScanRecordingPackages()` 获得恢复候选，胶囊状态条显示恢复扫描结果，独立设置窗口中的 Recovery 行可以调用 `RecoverRecordingPackage()` 执行恢复。真实录制后端必须复用这个包服务，不再在录制服务里手写 manifest。
