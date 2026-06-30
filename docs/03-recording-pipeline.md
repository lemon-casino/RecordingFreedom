# 03. 录制、降噪、摄像头和录制包管线

## 录制包结构

新录制默认写入：

```text
<RecordingFreedomAppData>/data/video/
  recording-YYYY-MM-DD-HH-mm-ss-SSS.rfrec/
    manifest.json
    screen.mp4
    webcam.mov | webcam.mp4 | webcam.webm
    audio-diagnostics.json
    video-diagnostics.json
    recovery.log
    cache/
```

`.rfrec` 是目录包，不是 zip。目录包允许录制中持续写盘，崩溃后也能保留可检查文件。

用户可以在设置窗口修改应用数据根目录，但录制包目录结构不变：新根目录只替换 `<RecordingFreedomAppData>`，录制仍写入 `<NewRoot>/data/video/recording-*.rfrec/`。录制中不能切换数据根，避免 ScreenCaptureKit、音频 writer 和摄像头 sidecar 写入路径在同一 session 内发生漂移。

开始录制前会通过 `AppDataService.StorageStatus()` 探测当前 `data/video`：目录不可创建或不可写时 preflight 返回 `blocked`；可用空间未知或低于建议阈值时返回 `warning`。这一步只保证写盘入口健康，不替代真实 native writer 的持续增长监测。

## manifest v1

```json
{
  "schemaVersion": 1,
  "app": "RecordingFreedom",
  "createdAt": "2026-06-30T13:45:22.123+08:00",
  "status": "recording",
  "media": {
    "screenVideoPath": "screen.mp4",
    "systemAudioPath": "system-audio.wav",
    "microphoneAudioPath": "microphone.wav",
    "webcamVideoPath": "webcam.mov",
    "webcamStartOffsetMs": 120
  },
  "source": {
    "type": "screen",
    "id": "display:1",
    "name": "Built-in Display"
  },
  "recording": {
    "quality": "balanced",
    "fps": 30,
    "captureCursor": true,
    "countdownSeconds": 0
  },
  "audio": {
    "system": true,
    "systemDeviceId": "system-audio:default",
    "microphone": true,
    "microphoneDeviceId": "microphone:default",
    "microphoneNoiseSuppression": "rnnoise",
    "sampleRate": 48000
  },
  "camera": {
    "enabled": true,
    "deviceId": "camera:default",
    "pipPreset": "bottom-right"
  },
  "diagnostics": {
    "sync": {
      "timelineBase": "media-timestamp",
      "timelineStartUnixNano": 1782817522123000000,
      "audioDiagnosticsPath": "audio-diagnostics.json",
      "videoDiagnosticsPath": "video-diagnostics.json",
      "screen": {
        "enabled": true,
        "path": "screen.mp4",
        "clock": "media-timestamp",
        "startOffsetMs": 0,
        "endOffsetMs": 120000,
        "durationMs": 120000,
        "droppedFrames": 0,
        "appendFailures": 0,
        "frameRate": 30
      },
      "systemAudio": {
        "enabled": true,
        "path": "system-audio.wav",
        "clock": "media-timestamp",
        "startOffsetMs": 0,
        "endOffsetMs": 120000,
        "durationMs": 120000,
        "droppedSamples": 0,
        "appendFailures": 0,
        "sampleRate": 48000
      },
      "microphone": {
        "enabled": true,
        "path": "microphone.wav",
        "clock": "media-timestamp",
        "startOffsetMs": 0,
        "endOffsetMs": 120000,
        "durationMs": 120000,
        "droppedSamples": 0,
        "appendFailures": 0,
        "sampleRate": 48000
      },
      "webcam": {
        "enabled": true,
        "path": "webcam.mov",
        "clock": "media-timestamp",
        "startOffsetMs": 120,
        "endOffsetMs": 120120,
        "durationMs": 120000,
        "droppedFrames": 0,
        "appendFailures": 0,
        "frameRate": 30
      },
      "pauseSegments": []
    }
  }
}
```

manifest 里所有媒体路径必须是包内相对路径。

当前 `internal/recpackage` 已强制校验媒体路径不能是绝对路径，也不能通过 `..` 逃逸录制包目录。

当前 `internal/recpackage` 还定义并校验了 `diagnostics.sync` 合同。真实后端停止时必须写入 screen、system audio、microphone、webcam 的 track 起止、duration、drop count、append failure、offset 和 pause segments。`mock-package` 只写 `timelineBase: "mock"` 和明确的 mock message，不代表真实音画同步已经完成。

当前 `recording.NormalizeStartRequest()` 和 `recpackage` 的归一化会保证：关闭的系统声音、麦克风、摄像头不会把旧 device id 写入 manifest；RNNoise 只在麦克风开启时可能写为 `rnnoise`；摄像头关闭时 PIP preset 固定为 `off`。

`recpackage.WriteManifest()` 会在摄像头关闭时同时清空 `webcamVideoPath`、`webcamStartOffsetMs` 和 webcam sync track，防止关闭摄像头后保留旧 sidecar 或旧 offset。

当前 `internal/recordingprofile` 会保证 manifest 中 `recording.quality`、`recording.fps`、`recording.captureCursor`、`recording.countdownSeconds` 使用同一套默认值和归一化规则。真实编码后端必须消费这些值，不能在平台层另起一套不兼容的质量/FPS 规则。

## 平台录制后端

所有平台后端都通过 `internal/recording.Backend` 接入。`RecordingService` 统一管理状态机和 manifest 状态，平台后端只负责采集、编码、flush 和平台资源释放。

开始录制前必须先通过 `internal/preflight`：

- `ready`：请求和平台能力满足当前 backend，可以开始。
- `warning`：允许 UI shell/mock package 继续，但必须显示 native 能力尚未就绪。
- `blocked`：源、设备、权限或 backend 能力不满足，不能开始。
- storage health 是预检的一部分：`data/video` 不可写时必须阻止开始，空间不足时必须让 UI 显示 warning。

预检不会创建录制包，也不会写文件；它只为 UI、日志和后续真实 backend 提供一致的可用性判断。

当前默认实现：

- `mock-package`：生成 `.rfrec` 包和 `screen.mock.txt`，manifest 标记 `diagnostics.mock = true`、`audio.mockPipeline = true`。它只用于 UI shell 和包结构验证，不代表真实录制能力。
- backend registry：`SelectBackend()` 先解析 `auto`、`mock-package`、`native`、`sck`、`wgc` 和 `pipewire`，再从 registry 查找已注册的真实平台 factory；未注册时回退到 queued native backend，避免 UI 或服务层误报可录制。
- queued native backend：通过 `RECORDINGFREEDOM_RECORDING_BACKEND=native` 或显式 backend ID 选择，按平台暴露 `screencapturekit`、`windows-graphics-capture` 或 `pipewire-portal` ID；未注册真实实现时只用于预检阻断和后续替换点，不创建包、不写媒体。

真实平台实现接入时应调用 `recording.RegisterNativeBackend("<backend-id>", factory)` 注册 factory。factory 返回的 backend 仍必须遵守同一个 `Backend` 接口、`CreateNativeWritePlan()`、`BackendStopResult.SyncDiagnostics` 和 ready 前媒体门禁；不能绕过 `RecordingService` 直接写 `ready`。

真实 native backend 开始采样前必须调用 `recording.CreateNativeWritePlan()` 初始化包和写盘计划；该 helper 内部统一归一化 `StartRequest` 并调用 `recpackage.CreateNative()`：

- manifest 先写 `recording`，`screenVideoPath` 固定为 `screen.mp4`。
- 系统声音开启时 `systemAudioPath` 固定为 `system-audio.wav`；麦克风开启时 `microphoneAudioPath` 固定为 `microphone.wav`。
- 摄像头开启时 `webcamVideoPath` 固定为 `webcam.mov`；摄像头关闭时不写 webcam 路径。
- `cache/` 和 `exports/` 目录会提前创建。
- 返回的绝对路径只允许指向当前 `.rfrec` 包内的 screen、webcam、audio diagnostics、video diagnostics 和 cache/export 目录。
- `CreateNativeWritePlan()` 不创建媒体文件，也不写 mock marker；真实后端必须在采样后让 `screen.mp4` 持续增长。
- `diagnostics.sync` 必须等真实 timestamp 可用后通过 `PatchSyncDiagnostics()` 写入，不能用墙钟或 mock timeline 代替。
- 真实后端停止时把 `diagnostics.sync` 放进 `BackendStopResult.SyncDiagnostics`，由 `RecordingService` 写入 manifest；平台后端不能绕过服务层直接把包标记为 `ready`。
- `RecordingService` 在写 `ready` 前会调用 `recpackage.ValidateReady()`：非 mock 包必须有非 0 字节 screen media；摄像头开启时必须有非 0 字节 webcam sidecar；mock 包只能凭 `diagnostics.mock = true` 和 `screen.mock.txt` marker 通过。

后续真实实现命名建议：

- `screencapturekit`：macOS ScreenCaptureKit。
- `windows-graphics-capture`：Windows.Graphics.Capture。
- `pipewire-portal`：XDG Desktop Portal + PipeWire。

macOS：

- 屏幕和窗口录制使用 ScreenCaptureKit。
- 源预枚举当前使用 CoreGraphics：`CGGetActiveDisplayList` 枚举显示器，`CGWindowListCopyWindowInfo` 枚举可见窗口，并按进程聚合程序源。
- 系统声音优先使用 ScreenCaptureKit 支持的系统音频流。
- 麦克风使用 AVFoundation 或 ScreenCaptureKit 可用能力采集，并进入统一音频管线。
- 摄像头使用 AVFoundation 录 sidecar。
- 编码优先 H.264/AAC，使用系统硬件编码能力。

Windows：

- 屏幕、窗口、程序窗口录制使用 Windows.Graphics.Capture。
- 系统声音使用 WASAPI loopback。
- 麦克风使用 WASAPI capture。
- 当前已通过 MMDevice API 枚举 Windows WASAPI render/capture endpoint，并保留 `system-audio:default` / `microphone:default` 作为稳定默认设备 ID；真实 endpoint id 写入 `NativeID`。
- 当前已新增纯 Go WASAPI capture source：麦克风流会 downmix/resample 为 `48kHz / mono` 后写入 `microphone.wav`；系统声音 loopback source 可以启动并写入 `system-audio.wav`，本轮无活动系统播放时 smoke 未收到 system audio packet。
- 摄像头使用 Media Foundation，必要时兼容 DirectShow。
- 编码优先 Media Foundation H.264/AAC，后续导出可接 FFmpeg。

Linux：

- experimental 阶段使用 XDG Desktop Portal ScreenCast + PipeWire。
- 音频根据桌面环境使用 PipeWire/PulseAudio。
- Linux 首版必须在 UI 和 release notes 标注 experimental。

## 屏幕、窗口、程序识别

`DeviceService.ListSources()` 返回统一源模型：

- Screen source：display id、名称、分辨率、scale factor。
- Window source：window id、应用名、标题、缩略图、进程 id。
- Application source：应用 id、应用名、进程组、可录窗口列表。

程序录制默认选择该程序的主窗口；如果平台支持更强的 app capture，再在后端升级，不改变前端接口。

当前 macOS build-tag 已有 CoreGraphics source enumeration；真正开始录制时仍需把这些 `cgdisplay:*` / `cgwindow:*` native ID 映射到 ScreenCaptureKit capture target。

## 音频管线

目标顺序：

1. 采集系统声音。
2. 采集麦克风。
3. 将麦克风转换为 48kHz PCM。
4. 可选 RNNoise 降噪。
5. 对麦克风做 gain smoothing、低 VAD attenuation、limiter。
6. 将系统声音和麦克风按统一时间线混音。
7. 编码为录制文件音轨。
8. 写入 `audio-diagnostics.json`。

继承旧项目降噪算法边界：

- RNNoise 10ms frame，48kHz 下每帧 480 samples。
- 麦克风先转 mono 进入 RNNoise，再复制到目标声道，保证播放稳定。
- 系统声音不进入降噪器。
- 暂停/继续时 reset 降噪器，避免状态跨暂停段污染。

当前已落地 `internal/audio` 合同：

- `Enhancer` 按 `NoiseSuppressor` 接口处理麦克风 mono PCM。
- 只有 `StreamMicrophone + EnhancementRNNoise` 会进入 suppressor。
- `StreamSystemAudio` 会直接 bypass，测试覆盖“系统声音请求 rnnoise 也不会进入 suppressor”。
- 非 48kHz 或非 mono 的麦克风 RNNoise 输入会被拒绝；重采样和 downmix 必须在平台采集层完成。
- 不足 480 samples 的尾部会进入 pending buffer；暂停/继续时必须 `Reset()`。
- `Enhancer` 已输出可审计统计：processed frames、processed samples、pending samples、reset count、bypassed samples、rejected frames 和 last error。
- `Pipeline` 已定义真实音频采集边界：平台后端推入 `TimedPCMBuffer`，pipeline 按配置区分 system audio、microphone 和 RNNoise，输出 `ProcessedBuffer` 并累计 diagnostics。
- `Diagnostics` / `WriteDiagnostics()` 已定义 `audio-diagnostics.json` 合同，记录 target format、system audio、microphone、enhancement 和 mixer 统计。
- `WAVSink` 已定义首版 audio sidecar 写盘策略：系统声音写 `system-audio.wav`，麦克风写 `microphone.wav`，两者仍位于 `.rfrec` 包内。
- ready 门禁会拒绝只有 44-byte WAV header 的音频 sidecar；启用的音频流必须写入至少一个样本，避免无 packet 的系统声音被误标为真实音频。
- `CaptureSession` 已把 `CaptureSource -> Pipeline -> WAVSink -> audio-diagnostics.json` 串成可复用运行时，后续 ScreenCaptureKit/WGC/PipeWire backend 只需要启动同一个 session。
- `recording.CreateAudioCaptureConfig()` 已把 `StartRequest + RecordingWritePlan` 转成统一 `audio.CaptureConfig`，真实后端不需要重复拼接设备、RNNoise、gain、audio sidecar 和 diagnostics 路径。

RNNoise native DSP 已迁移为 `internal/audio/rnnoise` cgo 包，并把 C 源码隔离到 `internal/audio/rnnoise/native` 子包：带 `rnnoise_native` 标签的 cgo 构建会链接旧项目 RNNoise + `LikelyVoiceEnhancement`，非 cgo 或未带标签构建返回明确 unavailable，不会假装降噪已生效。当前 preview artifact 仍保持默认构建；RNNoise native release toolchain 验证和完整 app recording backend 接入完成后，才能作为用户可用能力发布。本机 Windows 若缺少 `gcc`，只能验证默认 fallback。当前 UI capability 仍保持 queued。

## 音画同步规则

- 录制时间线以第一帧有效视频 sample 或平台后端确认的 start timestamp 为基准。
- 音频 sample 使用媒体 timestamp，不使用纯墙钟推算。
- 暂停段从视频、音频、摄像头 offset 中一致扣除。
- 摄像头记录 `webcamStartOffsetMs`，预览和导出都用 `screenTime - offset` 对齐。
- 停止后通过 `PackageService.PatchSyncDiagnostics()` 写入音视频 track 起止、duration、drop count、append failure、offset。
- `timelineBase` 只能使用 `mock`、`media-timestamp` 或 `platform-start-timestamp`，避免不同平台后端随意发明不可比较的时间线。
- `audioDiagnosticsPath`、`videoDiagnosticsPath` 和各 track `path` 必须是 `.rfrec` 包内相对路径，不能指向包外文件。

## 暂停、停止和恢复

暂停：

- 后端进入 `paused` 状态。
- 停止写入有效媒体 sample 或按平台能力写入连续时间线。
- 记录 pause range。
- reset 麦克风降噪状态。

停止：

- manifest 先写 `finalizing`。
- 调用当前 `recording.Backend.Stop()`，flush 视频、音频、摄像头 writer。
- 如果 `Backend.Stop()` 返回 sync diagnostics，则 `RecordingService` 在标记 `ready` 前写入并校验它；非法路径或非法计数会让包进入 `failed`。
- 通过 `PackageService.ValidateReady()` 探测 ready 媒体：非 mock 包的 screen 文件必须存在、可读且非 0 字节；摄像头开启且 manifest 仍声明 webcam sidecar 时，webcam 文件也必须存在、可读且非 0 字节。
- 写诊断文件。
- manifest 写 `ready`。

崩溃恢复：

- 启动时扫描 `data/video/*.rfrec`。
- `manifest.json` 缺失时，从包内非 0 字节 `screen.*` 文件重建最小 manifest。
- 只要 `screen.mp4` 可读，就不能把包判为完全失败。
- 0 字节 webcam sidecar 不写入 manifest，但保留文件和诊断。
- 恢复动作只能作用于当前 app-managed `data/video` 内的 `.rfrec` 包。
- 恢复成功后 manifest 写为 `ready`，记录 `completedAt`，并标记 `diagnostics.recovered = true`。

当前已落地第一层恢复扫描、保守恢复和 ready 前媒体门禁：`recording`、`paused`、`finalizing` 状态会标记为 recoverable；manifest 缺失但存在非 0 字节 `screen.*` 文件的包也会标记为 recoverable；停止录制时非 mock 包不能在缺失或 0 字节 screen/webcam 媒体的情况下进入 `ready`。后续真实后端接入后再补容器级媒体 probe、诊断重建和失败原因分级。

## 后续画中画摄像头

v1 录制阶段只保证 sidecar 和 offset。后续 PIP 功能分两步：

1. 预览层：在 React/Canvas 中把 `webcam.*` 按 manifest 布局叠加显示。
2. 导出层：使用原始 screen + webcam sidecar 重新合成 MP4。

当前已落地 `internal/pip` preset 合同：

- `off`、`bottom-right`、`bottom-left`、`free` 均为合法 preset。
- `settings.json` 会持久化 PIP preset。
- 开始录制请求会带上当前 PIP preset。
- 摄像头关闭时 manifest 写 `off`。
- `pip.Layout()` 为导出层提供确定性基础矩形，`free` 在自定义坐标落地前使用默认右下角布局。

不要在原始录制阶段把摄像头烘焙进 `screen.mp4`，否则无法后期调整位置、形状和大小。

## 导出计划

当前已落地 `internal/exportplan` 合同。它不会执行真实导出，也不会假装已经完成 FFmpeg/原生合成；它只负责在导出前生成可验证计划：

- 输入必须是当前 `data/video` 内部的 `*.rfrec` 包。
- manifest 必须为 `ready`，并且默认不能是 mock 包。
- 默认输出路径为包内 `exports/recording.mp4`，仍然保持在 `.rfrec` 包内。
- screen media 必须存在且非 0 字节。
- PIP 可见时，webcam sidecar 必须存在且非 0 字节，`webcamStartOffsetMs` 会进入导出计划。
- `RequireSync` 模式要求 `diagnostics.sync` 存在，并拒绝 mock timeline。
- 导出计划会带出 `timelineBase`、diagnostics 文件路径、pause segments 和 `pip.Rect`，供后续流式导出器消费。

后续真实导出器必须按计划读取 screen/webcam/audio diagnostics，并以流式方式写出，不能把 30 分钟项目聚合成最终内存 Blob。
