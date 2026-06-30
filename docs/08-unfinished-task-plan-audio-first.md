# 08. 未完成任务计划：真实音频与 RNNoise 优先

更新时间：2026-06-30

本文档把初版 preview 之后尚未完成的工作拆成可执行任务。当前策略调整为：先推进真实音频采集与 RNNoise 降噪，再继续屏幕录制、摄像头画中画、导出和正式发布链路。

当前 `v0.1.0-preview.6` 只证明 UI Shell、设置、语言、图标、mock `.rfrec` 包、`data/video` 写盘结构和全平台 preview build 可用。后续代码已新增 Windows WASAPI 麦克风真实采集 smoke 和 RNNoise native wrapper，但尚未接入完整 screen recording backend，因此不能对外宣称完整真实录制已完成。

## P0-AUDIO：真实音频与 RNNoise 降噪

目标：

- 接入真实系统声音采集。
- 接入真实麦克风采集。
- 把麦克风 PCM 归一化为 RNNoise 可处理的 `48kHz / mono / 10ms / 480 samples` frame。
- RNNoise 只处理麦克风，不处理系统声音。
- 停止录制时写入音频同步诊断和 RNNoise 诊断。
- 所有真实音频输出仍然写入软件数据根下的 `data/video/recording-*.rfrec/`。

非目标：

- 不先做最终混音导出。
- 不把摄像头 PIP 烘焙进屏幕视频。
- 不把 mock 音频或静音音轨伪装成真实音频。

### A0 音频后端边界

状态：已完成基础代码合同，并新增 `CaptureSession`、`WAVSink`、音频 sidecar 写盘路径、Windows WASAPI source 复用入口和 `recording.NativeBackendRuntime`。

交付：

- 新增 `internal/audio` 或 `internal/recording` 下的 native audio capture 接口，区分 system audio、microphone、enhancer、mixer、diagnostics。
- 后端必须能被 ScreenCaptureKit/WASAPI/PipeWire 复用，不把某个平台 API 写进 `RecordingService` 主流程。
- 已落地 `audio.Pipeline`、`audio.Diagnostics`、`audio.WriteDiagnostics()`、`recording.CreateAudioCaptureConfig()` 和对应单元测试。
- 已落地 `audio.CaptureSession`、`audio.NewNativeCaptureSession()`、`audio.WAVSink`、`system-audio.wav` / `microphone.wav` sidecar 合同和 ready 前音频 sidecar 校验。
- 已落地 `recording.NativeBackendRuntime`：真实平台 backend 后续可统一复用 native write plan、audio session 生命周期、RNNoise suppressor 生命周期和启动失败标记 `failed` 的处理。

验收：

- Go 测试已覆盖：打开/关闭系统声音、打开/关闭麦克风、打开/关闭 RNNoise 时，请求合同、输出路径和 diagnostics 路径稳定。
- Go 测试已覆盖：有音频时 `NativeBackendRuntime` 会创建并控制 audio session；无音频时不创建；RNNoise 不可用时失败且不会假装降噪可用。
- `RecordingService` 仍只负责状态机、包状态和 ready 前门禁；音频采集和处理合同位于 `internal/audio`、`recording.CreateAudioCaptureConfig()` 与 `recording.NativeBackendRuntime`。

### A1 真实音频设备枚举

状态：Windows WASAPI system audio / microphone endpoint 枚举已完成；macOS CoreAudio 与 Linux PipeWire/PulseAudio 待接入。

交付：

- macOS：CoreAudio 枚举输出设备、输入设备、默认设备和权限状态。
- Windows：WASAPI 枚举 loopback/system audio endpoint 和 capture/microphone endpoint。已完成。
- Linux：PipeWire/PulseAudio 枚举输入/输出设备，无法可靠枚举时返回 experimental/unavailable 状态。
- `DeviceService.ListMediaDevices()` 从 queued fallback 升级为真实 provider，失败时保留明确原因。

验收：

- UI 音频按钮打开后能看到真实系统声音设备和真实麦克风设备。
- 无权限或平台不支持时显示 blocked/queued/unsupported，不出现空列表假成功。
- Go 测试覆盖 provider 成功、空结果、权限失败、平台失败回退。

### A2 系统声音采集

状态：Windows WASAPI loopback source 已实现，并已在有活动系统播放时通过真实样本 smoke；长录同步、ready 包集成和完整 app recording backend 接入仍待完成。macOS/Linux 待接入。

交付：

- macOS：优先使用 ScreenCaptureKit system audio；若当前 macOS 版本或权限不支持，返回 blocked reason。
- Windows：WASAPI loopback 采集系统声音。
- Linux：PipeWire/PulseAudio 采集系统声音，首版标注 experimental。
- 采集 sample 必须带 timestamp 或可映射到录制 timeline 的 sample time。

验收：

- 开启系统声音录制后，停止包内 manifest 记录 system audio track diagnostics。
- 关闭系统声音时 manifest 不保留旧 `systemDeviceId`。
- 系统声音不会进入 RNNoise。

### A3 麦克风采集

状态：Windows WASAPI 麦克风 PCM 采集已完成并本机 smoke 验证；macOS CoreAudio/AVFoundation 与 Linux PipeWire/PulseAudio 待接入。

交付：

- macOS：AVFoundation/CoreAudio 麦克风 PCM 采集。
- Windows：WASAPI capture 麦克风 PCM 采集。已完成，`go run ./cmd/audio-smoke -duration=1s -keep` 生成了非空 `microphone.wav` 和 `audio-diagnostics.json`。
- Linux：PipeWire/PulseAudio 麦克风 PCM 采集。
- 采集层输出统一 PCM frame，进入音频处理链路。

验收：

- 开启麦克风录制后，停止包内 manifest 记录 microphone track diagnostics。
- 关闭麦克风时 manifest 不保留旧 `microphoneDeviceId`，RNNoise 自动视为 off。
- 暂停/继续不会让麦克风时间线漂移。

### A4 RNNoise native DSP 接入

状态：RNNoise C 源码和旧项目 `LikelyVoiceEnhancement` 已迁移为 `internal/audio/rnnoise` cgo 包，C/H 源码已隔离到 `internal/audio/rnnoise/native` 子包；非 cgo 或未带 `rnnoise_native` 标签的构建会明确返回 unavailable。Linux cgo link 的 `-lm` 约束已修正为独立 `linux` / `darwin` LDFLAGS，CI/release gate 已恢复 RNNoise native frame 处理硬门禁。完整 app recording backend 接入前，不对用户宣称 RNNoise 已可用。本机 Windows 缺少 `gcc` 时只能验证 fallback。

交付：

- 接入 RNNoise native library 或可维护的 Go/cgo/native wrapper。
- 保持当前 `internal/audio.Enhancer` 合同：只接收 `48kHz / mono` 麦克风 PCM，每帧 480 samples。
- 平台采集层负责重采样、downmix 和格式转换，RNNoise 层不接收 stereo 或非 48k 输入。
- 暂停、继续、切换设备、停止时 reset suppressor 状态。

验收：

- Go 测试覆盖：系统声音 bypass、麦克风进入 RNNoise、非 48k 拒绝、stereo 拒绝、partial frame pending、pause reset。
- 真实录制验证：开启 RNNoise 后麦克风音轨可播放，无爆音、无明显断裂。
- `audio-diagnostics.json` 记录 RNNoise enabled、processedFrames、droppedFrames、resetCount、sampleRate、channels。

### A5 音频混音与写盘

状态：首版写盘策略已确定为包内 WAV sidecar：系统声音 `system-audio.wav`，麦克风 `microphone.wav`；`NativeBackendRuntime` 已提供平台后端可复用的音频启动/暂停/停止入口。还未完成 ScreenCaptureKit/WGC/PipeWire 视频后端调用 runtime 后的端到端 ready package，也未做最终混音/AAC/mux。

交付：

- 明确首版写盘策略：系统声音和麦克风可以先作为同一容器内独立 track，或先混成一个 AAC track，但必须记录诊断。
- 音频 writer 持续写入 `screen.mp4` 或包内约定音频 sidecar；路径必须由 `CreateNativeWritePlan()` 或后续扩展计划返回。
- 暂停段必须写入 `diagnostics.sync.pauseSegments`。

验收：

- 1 分钟、5 分钟录制停止后音频可播放。
- 开启/关闭系统声音、麦克风、RNNoise 的组合都能停止并生成 ready package。
- audio diagnostics 路径不能是绝对路径，不能逃逸 `.rfrec` 包目录。

### A6 预检、UI 和设置联动

交付：

- `preflight` 根据真实设备枚举和权限状态决定 ready/warning/blocked。
- 音频按钮保持一个入口：系统声音、麦克风、RNNoise 仍在同一个音频面板内管理。
- RNNoise 只有麦克风开启且当前麦克风满足处理条件时才显示为实际生效。

验收：

- 无麦克风权限时，开始录制会 blocked 并说明原因。
- 麦克风关闭时 RNNoise 不显示为生效。
- 设置重启后保留音频设备选择和 RNNoise 开关。

### A7 手动验证矩阵

每个平台至少保留以下验证记录：

- 系统声音 only。
- 麦克风 only。
- 系统声音 + 麦克风。
- 麦克风 + RNNoise。
- 系统声音 + 麦克风 + RNNoise。
- 暂停 2 次后继续录制并停止。
- kill app 后恢复包，不丢失已有音频文件和 manifest。

通过标准：

- 包位于 `<DataRoot>/data/video/recording-*.rfrec/`。
- manifest 为 `ready`。
- 音频可播放。
- `diagnostics.sync` 和 `audio-diagnostics.json` 存在且路径合法。
- 系统声音没有经过 RNNoise。

## P0-RECORDING：真实屏幕/窗口/程序录制

任务：

- macOS ScreenCaptureKit 最小 `screen.mp4` 写盘。
- Windows Windows.Graphics.Capture 最小 `screen.mp4` 写盘。
- Linux XDG Desktop Portal + PipeWire experimental 写盘。
- 将 CoreGraphics/Win32/PipeWire 源 ID 映射为真实 capture target。
- 录制中 `screen.mp4` 持续增长。
- ready 前缺失或 0 字节 screen media 必须失败。

验收：

- 1 分钟、5 分钟、20 分钟录制可停止并播放。
- 暂停/继续不会造成 manifest 状态错误。
- kill app 后可扫描 recoverable package。

## P1-CAMERA：摄像头 sidecar 与画中画预备

任务：

- macOS AVFoundation 摄像头 sidecar。
- Windows Media Foundation 摄像头 sidecar。
- Linux PipeWire 摄像头 sidecar experimental。
- 记录 `webcamStartOffsetMs`。
- PIP preset 继续只作为 sidecar 布局，不烘焙进原始 screen media。

验收：

- 开启摄像头后包内存在可播放 `webcam.mov` 或 `webcam.mp4`。
- 摄像头开启但 sidecar 缺失或 0 字节时，manifest 不能进入 ready。
- PIP preset 写入 manifest，导出计划可读取。

## P1-EXPORT：真实导出与音画同步

任务：

- 接入 FFmpeg 或平台原生流式导出。
- 使用 `internal/exportplan` 生成的计划读取 screen、audio diagnostics、webcam sidecar 和 PIP layout。
- 不使用最终内存 Blob。
- 输出默认写入包内 `exports/recording.mp4`。

验收：

- mock 包被拒绝导出。
- 缺失 sync diagnostics 被拒绝导出。
- 30 分钟项目导出不出现内存暴涨。
- 导出 MP4 音画同步可接受。

## P1-RELEASE：正式发布链路

任务：

- macOS Developer ID Application 签名、公证、staple。
- Windows installer/MSIX 或 portable zip 验证。
- Linux deb/rpm/AppImage 取舍和验证。
- release workflow 缺少签名密钥时必须失败或只产出明确 preview artifact。

验收：

- GitHub Actions 产出可安装包。
- SHA256SUMS 和 release notes 完整。
- macOS Gatekeeper 不阻止已公证包。
- Windows clean machine 可启动。

## P2-QUALITY：长期稳定性

任务：

- 长录制磁盘空间持续监控。
- 容器级 media probe。
- `cache/media-info.json`。
- chunked cursor/audio diagnostics。
- 崩溃恢复分级原因。
- 用户可见的“显示录制文件夹”和诊断包入口。

验收：

- 20 分钟以上录制可恢复、可播放、可诊断。
- 失败包不会被误标记 ready。
- 用户可以定位录制包和诊断文件。

## 第一执行顺序

1. A0 音频后端边界。已完成基础代码合同和 `NativeBackendRuntime`。
2. A1 真实音频设备枚举。Windows 已完成，下一步补 macOS/Linux。
3. A3 麦克风采集。Windows 已完成并 smoke 验证，下一步补 macOS/Linux。
4. A4 RNNoise native DSP。wrapper 已迁移并恢复 CI/release gate 定向验证；下一步在有 C 工具链的本机补真实 `audio-smoke -rnnoise`，并在 app recording backend 接入后再开放 UI/preflight capability。
5. A2 系统声音采集。Windows source 已实现并通过有播放源真实样本 smoke，下一步做长录同步和完整 app recording backend 接入。
6. A5 音频混音与写盘。下一步让真实平台视频后端调用 `NativeBackendRuntime`，再做长录同步和 mux 策略。
7. A6 预检、UI 和设置联动。
8. A7 三平台手动验证矩阵。

完成 A0-A7 并接入真实 screen recording backend 前，不对外宣称 RecordingFreedom 的正式录制流程已支持完整真实音频录制或真实 RNNoise 降噪。
