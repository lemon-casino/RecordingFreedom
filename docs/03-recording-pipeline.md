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
  "recordingMode": "screen",
  "media": {
    "screenVideoPath": "screen.mp4",
    "systemAudioPath": "system-audio.wav",
    "systemAudioStorage": "sidecar",
    "microphoneAudioPath": "microphone.wav",
    "microphoneAudioStorage": "sidecar",
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

当前 `internal/recpackage` 还定义并校验了 `recordingMode`、主媒体路径和 `diagnostics.sync` 合同。`recordingMode: "screen"` 的主媒体是 `screen.mp4`；`recordingMode: "audio-only"` 的主媒体是 `audio.m4a`，且不能带 `screenVideoPath`。真实后端停止时必须写入 screen、system audio、microphone、webcam 的 track 起止、duration、drop count、append failure、offset 和 pause segments。`mock-package` 只写 `timelineBase: "mock"` 和明确的 mock message，不代表真实音画同步已经完成。

当前 `recording.NormalizeStartRequest()` 和 `recpackage` 的归一化会保证：关闭的系统声音、麦克风、摄像头不会把旧 device id 写入 manifest；RNNoise 只在麦克风开启时可能写为 `rnnoise`；摄像头关闭时 PIP preset 固定为 `off`。

默认设置保持系统声音、麦克风和 RNNoise 关闭，只保留默认设备 ID 作为用户后续开启时的选择记忆。这样 preview 首次启动不会把用户尚未开启的音频/降噪链路显示成正在生效；从当前发布门禁开始，artifact 本身必须带 `rnnoise_dynamic` 构建、随包携带对应平台 RNNoise 动态模块，并通过 `desktop-doctor -require-rnnoise`。

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

- `mock-package`：生成 `.rfrec` 包和 `screen.mock.txt`，manifest 标记 `diagnostics.mock = true`、`audio.mockPipeline = true`。它只用于显式 preview smoke、UI shell 和包结构验证，不代表真实录制能力，也不是桌面默认录制路径。
- backend registry：`SelectBackend()` 先解析 `auto`、`native`、`mock-package`、`sck`、`wgc` 和 `pipewire`，再从 registry 查找已注册的真实平台 factory；默认值、`auto` 和 `native` 都按当前平台选择 native backend，`mock`/`mock-package` 才显式选择 mock。
- queued native backend：通过默认值、`auto`、`native` 或显式 backend ID 选择，按平台暴露 `screencapturekit`、`ffmpeg-desktop-capture`、`windows-graphics-capture` 或 `pipewire-portal` ID；未注册真实实现时只用于预检阻断和后续替换点，不创建包、不写媒体。

真实平台实现接入时应调用 `recording.RegisterNativeBackend("<backend-id>", factory)` 注册 factory。factory 返回的 backend 仍必须遵守同一个 `Backend` 接口、`CreateNativeWritePlan()`、`BackendStopResult.SyncDiagnostics` 和 ready 前媒体门禁；不能绕过 `RecordingService` 直接写 `ready`。推荐平台实现直接返回 `recording.NewNativeRuntimeBackend()`，只提供 ScreenCaptureKit/FFmpeg/PipeWire 的 video/audio session factory。

真实 native backend 开始采样前必须调用 `recording.CreateNativeWritePlan()` 初始化包和写盘计划；该 helper 内部统一归一化 `StartRequest` 并调用 `recpackage.CreateNative()`：

- manifest 先写 `recording`，`screenVideoPath` 固定为 `screen.mp4`。
- 正式方向是优先把屏幕视频、系统声音和麦克风写进同一个主媒体 `screen.mp4`，让用户拿到的默认录制文件天然音画合一。
- `system-audio.wav` / `microphone.wav` 保留为过渡、诊断、恢复或平台 fallback sidecar；`systemAudioStorage` / `microphoneAudioStorage` 记录 `sidecar` 或 `muxed`，当平台 writer 已经把对应音轨 mux 进 `screen.mp4` 后，manifest 和 ready 门禁不会强迫重复写两份音频。
- 单独录音使用 `recpackage.CreateAudioOnly()` 初始化 `recordingMode: "audio-only"` 包，默认主媒体为 `audio.m4a`；停止阶段会通过 FFmpeg 把包内 WAV sidecar 封装为 AAC/M4A 主音频。系统声音和麦克风声明为 `muxed` 时会指向同一个 `audioPath`，ready 门禁会探测 `audio.m4a` 的 `soun` 音轨，而不是要求 `screen.mp4`。单流采集期间先写 `audio.wav` sidecar，双流采集期间保留 `system-audio.wav` / `microphone.wav` 分轨 sidecar，最终 manifest 指向 `audio.m4a`，sidecar 继续作为恢复和诊断证据留在包内。
- 摄像头开启时 `webcamVideoPath` 按后端写入：macOS/通用默认 `webcam.mov`，Windows FFmpeg DirectShow sidecar 使用 `webcam.mp4`；摄像头关闭时不写 webcam 路径。
- `cache/` 和 `exports/` 目录会提前创建。
- 返回的绝对路径只允许指向当前 `.rfrec` 包内的 screen、webcam、audio diagnostics、video diagnostics 和 cache/export 目录。
- `CreateNativeWritePlan()` 不创建媒体文件，也不写 mock marker；真实后端必须在采样后让 `screen.mp4` 持续增长。
- `diagnostics.sync` 必须等真实 timestamp 可用后通过 `PatchSyncDiagnostics()` 写入，不能用墙钟或 mock timeline 代替。
- 真实后端停止时把 `diagnostics.sync` 放进 `BackendStopResult.SyncDiagnostics`，由 `RecordingService` 写入 manifest；平台后端不能绕过服务层直接把包标记为 `ready`。
- `RecordingService` 在写 `ready` 前会调用 `recpackage.ValidateReady()`：非 mock 包必须有非 0 字节 screen media；摄像头开启时必须有非 0 字节 webcam sidecar；`sidecar` 音频必须有非空 WAV sidecar；`muxed` 音频必须能在 `screen.mp4` 的 MP4 box 中探测到 `soun` 音轨；mock 包只能凭 `diagnostics.mock = true` 和 `screen.mock.txt` marker 通过。

输出形态约束：

- 屏幕、窗口和程序录制的正式主产物是 `screen.mp4`，视频轨和已启用音频轨应在同一个容器内完成同步封装。首版可以把系统声音和麦克风混成一个 AAC 音轨；后续如果平台能力稳定，再升级为系统声音、麦克风独立音轨，但不能改变用户拿到的默认文件。
- 单独录音必须作为 `audio-only` 录制模式实现，不创建假的 `screen.mp4`，默认输出 `audio.m4a`，平台受限时才使用 `audio.wav` fallback。
- fallback sidecar 只用于 smoke、恢复、平台限制和诊断场景。manifest 使用 `systemAudioStorage` / `microphoneAudioStorage` 表达每条音频是 `muxed` 还是 `sidecar`，ready 门禁据此检查 `screen.mp4` 内音轨或包内 sidecar，避免长期重复写盘。
- 用户可见的默认视频文件必须可以直接播放出图像和声音；不能要求用户在普通播放场景手动再合并 `screen.mp4` 与音频 sidecar。
- 后期单独录制音频是正式能力，不是从屏幕录制里裁掉画面的替代方案；它复用同一录制包、音频管线、降噪和诊断合同，但 manifest 必须标明 `recordingMode: "audio-only"`。

后续真实实现命名建议：

- `screencapturekit`：macOS ScreenCaptureKit。
- `ffmpeg-desktop-capture`：Windows FFmpeg gdigrab desktop/window writer。
- `windows-graphics-capture`：Windows.Graphics.Capture 兼容 alias，保留给后续原生 WGC/Media Foundation writer。
- `pipewire-portal`：XDG Desktop Portal + PipeWire。

macOS：

- 屏幕和窗口录制使用 ScreenCaptureKit。
- 源预枚举当前使用 CoreGraphics：`CGGetActiveDisplayList` 枚举显示器，`CGWindowListCopyWindowInfo` 枚举可见窗口，并按进程聚合程序源。
- 系统声音优先使用 ScreenCaptureKit 支持的系统音频流。
- 麦克风使用 AVFoundation 或 ScreenCaptureKit 可用能力采集，并进入统一音频管线。
- 摄像头使用 AVFoundation 录 sidecar。
- 编码优先 H.264/AAC，使用系统硬件编码能力。
- 当前已接入 ScreenCaptureKit display/window/region video session：`screen:display-<CGDirectDisplayID>` 会映射为 `SCDisplay`，`window:<CGWindowID>` 会映射为 `SCWindow`，单显示器 `region:custom` 会绑定显示器 `NativeID` 并使用 ScreenCaptureKit `sourceRect` crop。它们均通过 `SCStream` 接收 screen sample buffer，并用 `AVAssetWriter` 持续写入包内 `screen.mp4`；系统声音开启时，ScreenCaptureKit audio sample 会写入同一个 `screen.mp4` 的 AAC 音轨，manifest 使用 `systemAudioStorage: "muxed"`。`application:<pid>` 解析代码保留给后续演进，但当前 capability 为 queued，不在初版验收范围内。`Stop()` 会写入 `video-diagnostics.json` 并返回 manifest sync diagnostics。麦克风 mux 仍按后续任务推进。
- 当前已新增 `cmd/video-smoke` 无 UI 验收入口，默认走 `native` backend、自动选择可用屏幕源；也可用 `-source-type=window` 验证窗口录制，或用 `-source-type=region` 验证 macOS 单显示器区域录制。系统声音真机验收使用 `-system`。命令会验证真实 `.rfrec` 包、`screen.mp4`、`video-diagnostics.json` 和 manifest sync diagnostics。

Windows：

- 默认视频录制使用 `ffmpeg-desktop-capture`，屏幕、绑定到单显示器的区域和多屏“全部屏幕”优先通过 FFmpeg `ddagrab` / Windows Desktop Duplication API 录制并保留鼠标；锁定窗口仍通过 FFmpeg `gdigrab hwnd=` 写入包内 `screen.mp4`。`windows-graphics-capture` 作为兼容 alias 保留，后续可替换为原生 WGC/Media Foundation writer。
- 系统声音使用 WASAPI loopback。
- 麦克风使用 WASAPI capture。
- 当前已通过 MMDevice API 枚举 Windows WASAPI render/capture endpoint，并保留 `system-audio:default` / `microphone:default` 作为稳定默认设备 ID；真实 endpoint id 写入 `NativeID`。
- 当前已新增纯 Go WASAPI capture source：麦克风流会 downmix/resample 为 `48kHz / mono`，系统声音 loopback source 已在有活动系统播放时写入真实样本。Windows 录屏 runtime 会在 FFmpeg 视频旁启动 WASAPI sidecar 写盘，停止阶段再 mux 到主 `screen.mp4`，manifest 将成功 mux 的系统声音和麦克风标记为 `muxed`；长录同步、live PCM pipe、Linux PipeWire 和三平台真机 smoke 仍是后续工作。
- 当前 Windows `internal/video` 已接入 FFmpeg 自动分段 writer：`screen:<display-token>` 和单显示器 `region:custom` 使用 `ddagrab`，多屏 `all-screens:virtual-desktop` 使用多路 `ddagrab + xstack` 合成，避免 `gdigrab -draw_mouse 1` 的 GDI 光标重绘闪烁；`window:<HWND hex>` 使用 `hwnd=` 锁定窗口。录制中默认每 60 秒写一个 segment，pause 会关闭当前 segment，resume 新开 segment，stop 使用 FFmpeg concat 合并为 `screen.mp4`。缺少 `ffmpeg` 时 capability 为 blocked，并写 failed diagnostics，不写假媒体或 ready manifest。
- 摄像头 sidecar 当前使用 FFmpeg DirectShow writer：`DeviceService` 解析稳定 `camera:dshow:*` ID 和 DirectShow 原生设备名，`RecordingFreedomService` 在预检和启动前补齐 `deviceNativeId`，runtime 与屏幕/音频一起 pause/resume/stop，最终写入包内 `webcam.mp4`。PIP preset 只作为后续预览/导出布局，不阻断 sidecar 录制。
- 编码优先 Media Foundation H.264/AAC，后续导出可接 FFmpeg。

Linux：

- experimental 阶段使用 XDG Desktop Portal ScreenCast + PipeWire。
- 音频根据桌面环境使用 PipeWire/PulseAudio。
- Linux 首版必须在 UI 和 release notes 标注 experimental。

## 屏幕、窗口、程序识别

`DeviceService.ListSources()` 返回统一源模型：

- All screens source：`all-screens:virtual-desktop`，表示多显示器虚拟桌面；manifest 必须记录虚拟桌面 `x/y/width/height`。当前用户菜单只在该源被平台标记为真实可录时展示。
- Screen source：单个显示器 `display id`、名称、分辨率、虚拟桌面坐标、scale factor。
- Region source：`region:custom` 或后续 `region:<id>`，表示用户框选的自定义区域；manifest 必须记录区域 `x/y/width/height`，坐标基于虚拟桌面。
- Locked window source：window id、应用名、标题、缩略图、进程 id；录制时锁定这个原生窗口 target，而不是跟随当前前台窗口。
- Application source：应用 id、应用名、进程组、可录窗口列表。

程序录制默认选择该程序的主窗口；如果平台支持更强的 app capture，再在后端升级，不改变前端接口。

当前 `DeviceService` 已把单个显示器的虚拟桌面坐标带到前端，并暴露 `all-screens:virtual-desktop` 与 `region:custom` 源。区域十字框选 overlay 已落地：前端可拖拽红框选择范围，后端会把选择结果写入 `source.geometry`。同一份 geometry 已进入 `video.CaptureConfig` 和 `video-diagnostics.json` 的 source 节点。macOS 单显示器区域已经接入 ScreenCaptureKit `sourceRect` crop writer：区域必须落在一块显示器内，并携带该显示器 `NativeID`；Objective-C 原生层会在创建 `SCStreamConfiguration` 前用 `CGDisplayBounds` 把虚拟桌面逻辑坐标转换成该显示器本地 `sourceRect`。Windows 单显示器区域 crop 已接入 FFmpeg `ddagrab`，框选结果会转换为物理像素矩形后按显示器本地 offset 交给 writer。跨显示器区域的 macOS 多屏合成、Linux 区域 crop 和 macOS/Linux `all-screens` 多屏幕合成在真实 writer 完成前仍必须保持 queued/blocked，不能假装可录制。

`RecordingFreedomService.StartRecording()` 和 `RecordingFreedomService.StartAudioOnlyRecording()` 必须执行与 UI 相同的 preflight 门禁；当来源不存在、来源处于 queued、平台 writer capability queued/blocked、存储不可写或音频/摄像头能力 blocked 时，服务入口必须直接返回错误，不允许进入 recorder backend，也不允许创建 `.rfrec` 包。前端按钮的 preflight 只是用户体验层，不能作为唯一防线。

胶囊来源菜单当前只展示真实可启动的全部屏幕入口、单个屏幕、区域和锁定窗口；Application/Program source 合同仍保留给后端和 smoke，但当前不作为用户可选菜单项或能力矩阵项展示。单个屏幕必须按显示器编号展示为 `屏幕 1`、`屏幕 2` / `Screen 1`、`Screen 2`。鼠标移入或键盘聚焦某个单屏来源时，应用会打开独立的 `screen-indicator` 透明置顶窗口，在对应物理屏幕中央显示黑底大号编号；离开菜单项、关闭菜单或选择来源后必须隐藏该标识窗口。标识窗口定位优先按原生源携带的物理屏幕 bounds 匹配 Wails screen，匹配失败再使用 display index 兜底，避免多屏枚举顺序差异导致标识显示到错误屏幕。

区域录制交互验收：

- 点击区域录制后，打开覆盖所有显示器的透明选择 overlay。
- 光标变为十字，按下鼠标开始选择，移动时显示红色边框和尺寸浮标。
- 松开鼠标后将当前矩形写入 `source.geometry` 并回到胶囊；小于最小尺寸的矩形应被拒绝。
- overlay 必须支持 `Esc` 取消，不得创建录制包。
- 多显示器坐标以虚拟桌面为基准，允许区域跨屏；后端如果暂不支持跨屏区域，必须在 preflight 阻止并提示原因。
- 当前实现已经完成 overlay 窗口、十字光标、拖拽选框、最小尺寸拒绝、`Esc` 取消和 `capture.region.selected` 事件；框选成功前不会把占位 `region:custom` 切成当前录制源。macOS 单显示器区域会绑定到对应 `cgdisplay:<id>` 并交给 ScreenCaptureKit crop writer；Windows 单显示器区域会使用物理像素矩形交给 FFmpeg `ddagrab` crop writer。下一步是补 Linux PipeWire region crop，以及 macOS/Linux 跨显示器区域的多屏合成 writer。

锁定窗口录制验收：

- UI 选择的是具体窗口 target，不是“当前活动窗口”。
- Windows 使用 `HWND`，macOS 使用 `CGWindowID/SCWindow`，Linux 后续使用 Portal/PipeWire 可恢复的窗口 token。
- 被锁定窗口最小化、关闭或权限丢失时，录制必须停止或进入 failed diagnostics，不能 silently 改录其他窗口。

当前 macOS build-tag 已有 CoreGraphics source enumeration，并已把 `screen:display-<CGDirectDisplayID>` 和 `window:<CGWindowID>` 映射到 ScreenCaptureKit capture target。`application:<pid>` 解析代码保留给后续演进，但当前 capability 标记为 queued，菜单不展示，初版验收不把程序录制算作完成项。当前 Windows build-tag 已有 Win32 source enumeration，并已把 `screen:<display-token>`、`all-screens:virtual-desktop`、`region:custom` 和 `window:<HWND hex>` 解析到 FFmpeg desktop writer；程序来源合同保留但当前菜单不展示，建议用户选择具体锁定窗口。

## 音频管线

目标顺序：

1. 采集系统声音。
2. 采集麦克风。
3. 将麦克风转换为 48kHz PCM。
4. 可选 RNNoise 降噪。
5. 对麦克风做 gain smoothing、低 VAD attenuation、limiter。
6. 将系统声音和麦克风按统一时间线混音，或作为两个独立音轨交给平台 muxer。
7. 优先编码进主媒体 `screen.mp4` 的音轨；只有平台限制、恢复需求或 smoke 调试时才写包内 WAV sidecar。
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
- `WAVSink` 已定义首版 audio sidecar 写盘策略：系统声音写 `system-audio.wav`，麦克风写 `microphone.wav`，两者仍位于 `.rfrec` 包内。它是当前 smoke 和 fallback 路径，不是最终强制架构。
- ready 门禁会拒绝只有 44-byte WAV header 的音频 sidecar；启用 sidecar 的音频流必须写入至少一个样本，避免无 packet 的系统声音被误标为真实音频。manifest 声明 `muxed` 时，ready 门禁会解析 `screen.mp4` 的 MP4 box 并要求存在 `soun` 音轨，而不是要求重复 sidecar。
- `CaptureSession` 已把 `CaptureSource -> Pipeline -> WAVSink -> audio-diagnostics.json` 串成可复用运行时，后续 ScreenCaptureKit/FFmpeg/PipeWire backend 只需要启动同一个 session。
- `recording.CreateAudioCaptureConfig()` 已把 `StartRequest + RecordingWritePlan` 转成统一 `audio.CaptureConfig`，真实后端不需要重复拼接设备、RNNoise、gain、audio sidecar 和 diagnostics 路径。
- `recording.NativeBackendRuntime` 已把 native `.rfrec` 写盘计划与音频 session 生命周期接起来：真实平台后端创建包后可以统一启动、暂停、恢复和停止音频；暂停会走音频 `Pause()` 并 reset RNNoise；RNNoise 请求在 native suppressor 不可用时会失败并把包标记为 `failed`，不会静默降级成未降噪录制。
- `NativeBackendRuntime.SyncDiagnostics()` 已把 video/audio runtime diagnostics 转成 manifest `diagnostics.sync` 合同，真实平台后端停止时应直接返回它，让 `RecordingService.Stop()` 统一 patch 和校验。
- `recording.NativeRuntimeBackend` 已把 `NativeBackendRuntime` 包装成真正的 `recording.Backend`：Start 创建包并启动 video/audio runtime，Pause/Resume 转发到 runtime，Stop flush runtime 并返回 `SyncDiagnostics()`。后续 macOS/Windows/Linux 平台文件只需要注册该 backend factory。

RNNoise native DSP 已迁移为 `internal/audio/rnnoise` 模块，并把 C 源码隔离到 `internal/audio/rnnoise/native` 子包。当前 release 标准使用 `rnnoise_dynamic`：CI 从仓库内 RNNoise C 源按平台/架构编译 `rnnoise.dll`、`librnnoise.dylib` 或 `librnnoise.so`，再随包放入 `tools/`。Go/Wails 主程序运行时通过 `rnnoise.Available()` 加载动态模块；如果模块缺失、架构不匹配或符号绑定失败，`CaptureService` 和 `DeviceService` 会显示 queued/blocked reason，预检不会假装降噪已生效。当前 CI/release 的 validate 和全平台 build job 都要求动态 DSP smoke 与 `desktop-doctor -require-rnnoise` 通过；Windows ARM64 不再是无 RNNoise 的例外架构。

## 单独音频录制

audio-only 录制模式用于只录系统声音、麦克风或麦克风 + RNNoise。audio-only 不伪装成屏幕录制，也不创建假的 `screen.mp4`：

- 包类型仍使用 `.rfrec`，路径仍位于 `<DataRoot>/data/video/`，manifest 使用 `recordingMode: "audio-only"` 区分屏幕录制。
- 默认输出使用可流式写入、可恢复的音频容器 `audio.m4a`；平台受限时可使用 `audio.wav` fallback，但必须仍由 manifest 明确记录，不能生成假 screen media。
- audio-only 可复用同一 `audio.Pipeline`、RNNoise suppressor、pause reset、diagnostics 和 bounded buffer 策略。
- UI 上作为录制源的一种模式呈现，不和 screen/window/program 源混在一个假 source 里。
- 当前代码已落地 `recpackage.CreateAudioOnly()`、`audioPath`、主音频媒体 ready 门禁、WAV sidecar 门禁和单元测试；`cmd/audio-smoke` 会生成带 manifest 的 audio-only `.rfrec` 包，停止后把 sidecar 封装为 `audio.m4a`，写入 `diagnostics.sync`，并通过 `ValidateReady()` 后才标记 `ready`。`RecordingService.StartAudioOnlyRecording()`、Wails `StartAudioOnlyRecording()`、`PreflightAudioOnlyRecording()` 和胶囊来源面板里的视频/音频模式入口已接入同一状态机，支持开始、暂停、继续、停止和 ready 校验。macOS ScreenCaptureKit 的系统声音 mux 只属于视频录制路径，audio-only 预检不会借用它来误报单独录音可用；macOS/Linux audio-only 源仍按后续任务推进。

## 内存与写盘策略

长录制不能整段放进内存，也不能等停止时一次性保存。降低硬盘读写的策略是减少重复写、顺序写和限额缓冲：

- 主录制优先 mux 到 `screen.mp4`，避免同时写 `screen.mp4`、`system-audio.wav`、`microphone.wav` 三份长期媒体。
- 音视频 sample 进入平台编码器或 muxer 前可以使用内存环形缓冲，但缓冲只服务于抖动吸收、暂停边界、编码队列和短时恢复，不作为最终媒体的主存储。
- 内存只保留有限环形缓冲和编码队列，例如最近几秒的音视频 sample、暂停边界和恢复用元数据；队列满时必须做背压、丢帧或失败诊断，不能无上限增长。
- 所有媒体 writer 使用顺序 append；不做频繁随机写，不在录制中反复 probe 大文件。
- `audio-diagnostics.json`、`video-diagnostics.json` 和 manifest 采用低频批量 flush 或状态切换时写入，避免每帧写 JSON。
- 缩略图、波形、代理文件和导出 cache 放到包内 `cache/`，按需生成，不能在录制时为了预览重复读写完整源媒体。
- 崩溃恢复优先依赖已经顺序写入的媒体文件和最小 manifest/checkpoint；内存缓冲丢失不能导致整段录制丢失。
- 长录验收必须看两个指标：主媒体文件在录制中持续顺序增长，进程内存不随录制时长线性增长。

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
- 真实平台后端必须调用 `NativeBackendRuntime.Stop()` 或对应的 video/audio stop 方法，flush 主媒体 `screen.mp4`、muxed 音轨或 fallback audio sidecar，以及 `audio-diagnostics.json` / `video-diagnostics.json`。
- 如果 `Backend.Stop()` 返回 sync diagnostics，则 `RecordingService` 在标记 `ready` 前写入并校验它；非法路径或非法计数会让包进入 `failed`。使用 `NativeBackendRuntime` 的后端应返回 `runtime.SyncDiagnostics()`，避免各平台重复手写 screen/system/microphone track 映射。
- 通过 `PackageService.ValidateReady()` 探测 ready 媒体：非 mock 包的 screen 文件必须存在、可读且非 0 字节；摄像头开启且 manifest 仍声明 webcam sidecar 时，webcam 文件也必须存在、可读且非 0 字节；音频 sidecar 必须写入样本，muxed 音频必须能在 `screen.mp4` 中探测到 `soun` 音轨。
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

摄像头 sidecar 和 PIP 已恢复开发并形成第一条可验证闭环。当前已落地结构化 PIP 配置合同、透明置顶 PIP 编辑 overlay、WebView 预览与 sidecar 设备的最佳匹配逻辑、Windows DirectShow / macOS AVFoundation / Linux v4l2 的 FFmpeg sidecar writer，以及导出阶段的 FFmpeg PIP 合成器。跨平台真机 smoke、预览匹配实际效果、暂停片段精确同步和后续原生 writer 替换仍按后续任务推进。

v1 录制阶段只保证 sidecar 和 offset。后续 PIP 功能分两步：

1. 预览层：录制时通过透明 `pip-overlay` 编辑布局；编辑器中再把 `webcam.*` 按 manifest 布局叠加显示。
2. 导出层：使用原始 screen + webcam sidecar 重新合成 MP4；当前 `internal/exporter` 已实现 FFmpeg 合成，默认写入包内 `exports/recording.mp4`。

当前已落地 `internal/pip` 合同：

- `off`、`bottom-right`、`bottom-left`、`free` 均为合法 preset。
- `camera.pip` 会记录 `preset`、`shape`、`mirror`、`position`、`scale` 和 `edgeFeather`。
- `settings.json` 会持久化 PIP 配置。
- 开始录制请求会带上当前 PIP 配置。
- 摄像头关闭时 manifest 写 `off`。
- `pip.Place()` 为导出层提供确定性 PIP rect、shape、mirror 和透明边缘参数。

不要在原始录制阶段把摄像头烘焙进 `screen.mp4`，否则无法后期调整位置、形状和大小。

## 导出计划与导出器

当前已落地 `internal/exportplan` 合同和 `internal/exporter` 执行器。`exportplan` 负责在导出前生成可验证计划，`exporter` 使用 FFmpeg 把干净的 `screen.mp4` 与 `webcam.*` sidecar 合成为最终 MP4：

- 输入必须是当前 `data/video` 内部的 `*.rfrec` 包。
- manifest 必须为 `ready`，并且默认不能是 mock 包。
- 默认输出路径为包内 `exports/recording.mp4`，仍然保持在 `.rfrec` 包内。
- screen media 必须存在且非 0 字节。
- PIP 可见时，webcam sidecar 必须存在且非 0 字节，`webcamStartOffsetMs` 会进入导出计划。
- `RequireSync` 模式要求 `diagnostics.sync` 存在，并拒绝 mock timeline。
- 导出计划会带出 `timelineBase`、diagnostics 文件路径、pause segments 和 `pip.Rect`，供导出器消费。
- 导出器支持圆形/方形、镜像、边缘羽化、位置、大小和 `webcamStartOffsetMs`，并拒绝把输出写回原始 `screen.mp4` 或 `webcam.*`。
- 导出器会先写包内临时 MP4，确认文件非空并通过 FFmpeg 解码首个视频帧后，再原子安装到 `exports/recording.mp4`；失败时不会替换已有导出文件。
- Wails 服务提供 `ExportRecordingPackage()`，设置面板可以对最近录制包导出 `exports/recording.mp4`。
- `cmd/pip-export-smoke` 可直接对 `.rfrec` 包执行无 UI 导出验收。本机已用临时真实 MP4 素材跑通圆形镜像 PIP 合成。

后续导出增强必须继续按计划读取 screen/webcam/audio diagnostics，并以流式方式写出，不能把 30 分钟项目聚合成最终内存 Blob。下一步重点是暂停片段的精确时间线压缩、真实录制包导出 smoke、ffprobe 音轨/时长门禁和长时长同步回归。
