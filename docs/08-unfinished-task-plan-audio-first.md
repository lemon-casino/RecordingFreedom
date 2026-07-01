# 08. 未完成任务计划：真实音频与 RNNoise 优先

更新时间：2026-07-01

本文档把初版 preview 之后尚未完成的工作拆成可执行任务。当前策略调整为：先收口语言设置、真实语音/音频录制与视频录制；摄像头 sidecar、画中画、导出和正式发布链路在这三项验收后再恢复。

当前 preview 仍不能对外宣称完整真实录制已完成。已完成的是 UI Shell、设置、语言、图标、mock `.rfrec` 包、`data/video` 写盘结构、全平台 preview build、Windows WASAPI 音频采集、真实麦克风设备枚举与电平监听、RNNoise native wrapper、Windows FFmpeg desktop video writer 代码路径、Windows 停止阶段音视频 mux、Windows 20 分钟音视频长录 smoke、Windows portable zip FFmpeg 依赖准备路径、Windows preview release asset 下载/SHA256/portable zip 结构复验入口、Windows GUI subsystem/FFmpeg 子进程命令窗口隐藏门禁、Windows 默认音频设备保留真实 WASAPI endpoint、录制态来源/音频/摄像头配置锁定、区域录制持久边框、设置窗口打开 `data/video` 目录和最近 `.rfrec` 包内容入口、macOS ScreenCaptureKit video/system-audio mux 代码路径、macOS CoreAudio 麦克风枚举和 PCM 采集代码路径，以及无 GUI doctor/smoke 验收入口。`v0.1.0-preview.14` 已作为 GitHub prerelease 发布，Release Gate、Windows x64、macOS arm64、Linux x64 和 Publish GitHub Release 均通过；本轮修复区域录制开始后的持久边框和清理 macOS CoreAudio deprecated property element annotation。还缺少的是 release artifact 下载复验后的 clean-machine 真实录制 smoke、目标桌面 RNNoise 实录听感/诊断、macOS/Linux 真机视频验收和 Linux PipeWire writer。摄像头 sidecar 和 PIP 当前暂停，不计入本轮视频/语音验收。

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
- 不在视频录制和语音/音频录制验收前继续摄像头/画中画功能。
- 不把 mock 音频或静音音轨伪装成真实音频。

### A0 音频后端边界

状态：已完成基础代码合同，并新增 `CaptureSession`、`WAVSink`、音频 sidecar 写盘路径、Windows WASAPI source 复用入口和 `recording.NativeBackendRuntime`。

交付：

- 新增 `internal/audio` 或 `internal/recording` 下的 native audio capture 接口，区分 system audio、microphone、enhancer、mixer、diagnostics。
- 后端必须能被 ScreenCaptureKit/WASAPI/PipeWire 复用，不把某个平台 API 写进 `RecordingService` 主流程。
- 已落地 `audio.Pipeline`、`audio.Diagnostics`、`audio.WriteDiagnostics()`、`recording.CreateAudioCaptureConfig()` 和对应单元测试。
- 已落地 `audio.CaptureSession`、`audio.NewNativeCaptureSession()`、`audio.WAVSink`、`system-audio.wav` / `microphone.wav` sidecar 合同和 ready 前音频 sidecar 校验。
- 已落地 `recording.NativeBackendRuntime`：真实平台 backend 后续可统一复用 native write plan、audio session 生命周期、RNNoise suppressor 生命周期和启动失败标记 `failed` 的处理。
- 已落地 `NativeBackendRuntime.SyncDiagnostics()`：真实平台 backend 后续可统一把 video/audio diagnostics 写成 manifest `diagnostics.sync`，不用各平台重复手写音画同步映射。
- 已落地 `recording.NativeRuntimeBackend`：真实平台 backend 后续可直接注册 runtime backend，只实现平台 video/audio session factory。

验收：

- Go 测试已覆盖：打开/关闭系统声音、打开/关闭麦克风、打开/关闭 RNNoise 时，请求合同、输出路径和 diagnostics 路径稳定。
- Go 测试已覆盖：有音频时 `NativeBackendRuntime` 会创建并控制 audio session；无音频时不创建；RNNoise 不可用时失败且不会假装降噪可用。
- Go 测试已覆盖：`NativeBackendRuntime.SyncDiagnostics()` 输出可被 `PatchSyncDiagnostics()` 接受并写回 manifest。
- Go 测试已覆盖：`NativeRuntimeBackend` 会驱动 Start/Pause/Resume/Stop，Stop 返回 sync diagnostics，registry 可以选择注册后的 runtime backend。
- `RecordingService` 仍只负责状态机、包状态和 ready 前门禁；音频采集和处理合同位于 `internal/audio`、`recording.CreateAudioCaptureConfig()` 与 `recording.NativeBackendRuntime`。

### A1 真实音频设备枚举

状态：Windows WASAPI system audio / microphone endpoint 枚举已完成；Windows FFmpeg DirectShow 摄像头枚举已完成并已供 sidecar writer 使用；macOS CoreAudio 输入设备枚举、默认麦克风识别和设备 UID 选择代码路径已完成，系统声音仍由 ScreenCaptureKit 视频录制路径负责；Linux PipeWire/PulseAudio/PipeWire camera 待接入，macOS AVFoundation 摄像头仍待接入。

交付：

- macOS：CoreAudio 枚举输入设备、默认麦克风和设备 UID；ScreenCaptureKit 负责视频录制中的系统声音能力，后续再补更完整的输出设备/权限诊断。
- Windows：WASAPI 枚举 loopback/system audio endpoint 和 capture/microphone endpoint。已完成。
- Linux：PipeWire/PulseAudio 枚举输入/输出设备，无法可靠枚举时返回 experimental/unavailable 状态。
- `DeviceService.ListMediaDevices()` 从 queued fallback 升级为真实 provider，失败时保留明确原因。

验收：

- UI 音频按钮打开后能看到真实系统声音设备和真实麦克风设备。
- 无权限或平台不支持时显示 blocked/queued/unsupported，不出现空列表假成功。
- Go 测试覆盖 provider 成功、空结果、权限失败、平台失败回退。

### A2 系统声音采集

状态：Windows WASAPI loopback source 已实现，并已在有活动系统播放时通过真实样本 smoke；Windows 录屏 runtime 会在用户启用系统声音时启动 WASAPI 写盘、写入 sync diagnostics，并在停止阶段 mux 到主 `screen.mp4`。本机 20 分钟 `screen + system audio + microphone + pause/resume` smoke 已通过，系统声音 mux 到 `screen.mp4` 并可解码。macOS ScreenCaptureKit system audio 已接入同容器 `screen.mp4` AAC mux 代码路径，待 macOS 真机 smoke。Linux 待接入。

交付：

- macOS：优先使用 ScreenCaptureKit system audio；代码路径已接入 `screen.mp4` mux，待真机验证；若当前 macOS 版本或权限不支持，返回 blocked reason。
- Windows：WASAPI loopback 采集系统声音。
- Linux：PipeWire/PulseAudio 采集系统声音，首版标注 experimental。
- 采集 sample 必须带 timestamp 或可映射到录制 timeline 的 sample time。

验收：

- 开启系统声音录制后，停止包内 manifest 记录 system audio track diagnostics。
- 关闭系统声音时 manifest 不保留旧 `systemDeviceId`。
- 系统声音不会进入 RNNoise。

### A3 麦克风采集

状态：Windows WASAPI 麦克风 PCM 采集已完成并本机 smoke 验证；macOS CoreAudio 麦克风 PCM 采集源、设备 UID 选择、增益处理、暂停/继续和真实电平监听代码路径已完成，仍需 macOS 真机 smoke 与长录同步验证；Linux PipeWire/PulseAudio 待接入。

交付：

- macOS：CoreAudio 麦克风 PCM 采集。代码路径已完成，待 macOS 真机 `audio-smoke` / `video-smoke -microphone` 验证。
- Windows：WASAPI capture 麦克风 PCM 采集。已完成，`go run ./cmd/audio-smoke -duration=1s -keep` 生成了非空 `microphone.wav` 和 `audio-diagnostics.json`。
- Linux：PipeWire/PulseAudio 麦克风 PCM 采集。
- 采集层输出统一 PCM frame，进入音频处理链路。

验收：

- 开启麦克风录制后，停止包内 manifest 记录 microphone track diagnostics。
- 关闭麦克风时 manifest 不保留旧 `microphoneDeviceId`，RNNoise 自动视为 off。
- 暂停/继续不会让麦克风时间线漂移。

### A4 RNNoise native DSP 接入

状态：RNNoise C 源码和旧项目 `LikelyVoiceEnhancement` 已迁移为 `internal/audio/rnnoise` cgo 包，C/H 源码已隔离到 `internal/audio/rnnoise/native` 子包；非 cgo 或未带 `rnnoise_native` 标签的构建会明确返回 unavailable。Linux cgo link 的 `-lm` 约束已修正为独立 `linux` / `darwin` LDFLAGS，CI/release gate 已恢复 RNNoise native frame 处理硬门禁。`CaptureService` / `DeviceService` 已按 `rnnoise.Available()` 暴露真实能力：带 `rnnoise_native` 的 release artifact 显示 available 并允许预检，未带标签的本地开发构建仍显示 queued/blocked。当前 release/CI build job 已要求 `rnnoise_native`，并用 `desktop-doctor -require-rnnoise` 阻断缺真降噪的 preview artifact；目标桌面的 `audio-smoke -rnnoise` 实录听感和长录诊断仍需补。

交付：

- 接入 RNNoise native library 或可维护的 Go/cgo/native wrapper。
- 保持当前 `internal/audio.Enhancer` 合同：只接收 `48kHz / mono` 麦克风 PCM，每帧 480 samples。
- 平台采集层负责重采样、downmix 和格式转换，RNNoise 层不接收 stereo 或非 48k 输入。
- 暂停、继续、切换设备、停止时 reset suppressor 状态。

验收：

- Go 测试覆盖：系统声音 bypass、麦克风进入 RNNoise、非 48k 拒绝、stereo 拒绝、partial frame pending、pause reset。
- 真实录制验证：开启 RNNoise 后麦克风音轨可播放，无爆音、无明显断裂。
- `audio-diagnostics.json` 记录 RNNoise enabled、processedFrames、droppedFrames、resetCount、sampleRate、channels。

### A5 音频混音、mux 与写盘

状态：首版 smoke/fallback 写盘策略已确定为包内 WAV sidecar：系统声音 `system-audio.wav`，麦克风 `microphone.wav`；正式录制方向调整为优先把系统声音和麦克风 mux 进主媒体 `screen.mp4`。`NativeBackendRuntime` 已提供平台后端可复用的音频启动/暂停/停止入口。manifest 已新增 `systemAudioStorage` / `microphoneAudioStorage`，ready 门禁已能区分 `sidecar` 与 `muxed`，并会解析 `screen.mp4` 的 MP4 box 确认 muxed 模式存在 `soun` 音轨。Windows FFmpeg 后端已在停止阶段把 WASAPI sidecar mux 到主 `screen.mp4`，系统声音 + 麦克风会混成一个 AAC 音轨并保留 sidecar 作诊断/恢复证据；本机 20 分钟 `screen + system audio + microphone + pause/resume` 已进入 `ready` 且 `ffmpeg -v error -i screen.mp4 -f null -` 通过；macOS ScreenCaptureKit 系统声音已接入同容器 AAC mux 代码路径，CoreAudio 麦克风 sidecar/diagnostics 代码路径已可由 native audio runtime 复用，二者仍待 macOS 真机 mux/sync smoke；Linux/PipeWire mux 和三平台长录诊断仍未完成。

交付：

- 默认录制优先把音频写入 `screen.mp4`：可以是系统声音和麦克风独立 track，也可以先混成一个 AAC track，但必须记录诊断。
- 默认视频产物必须是用户可直接播放的音画合一文件，不能把“录完后再手动合并音频”作为正常路径。
- WAV sidecar 只作为 smoke、fallback、恢复或平台限制下的过渡输出；不能在最终架构里强迫所有平台重复写两份音频。
- manifest 需要记录音频存储形态：已 mux 进 `screen.mp4` 的 track 走主媒体探测；fallback sidecar 才检查 `system-audio.wav` / `microphone.wav`。代码合同已完成。
- 音频 writer 持续写入 `screen.mp4` 或包内约定 fallback sidecar；路径必须由 `CreateNativeWritePlan()` 或后续扩展计划返回。
- 暂停段必须写入 `diagnostics.sync.pauseSegments`。
- 增加 muxed audio track probe：当 manifest 声明音频已 mux 进 `screen.mp4`，ready 门禁应检查主媒体音轨，而不是要求 `system-audio.wav` / `microphone.wav`。代码合同已完成。

验收：

- 1 分钟、5 分钟录制停止后，主媒体中的音频可播放且和视频同步。
- 开启/关闭系统声音、麦克风、RNNoise 的组合都能停止并生成 ready package。
- 关闭 fallback sidecar 时，启用系统声音或麦克风不会长期重复写 WAV 文件。
- audio diagnostics 路径不能是绝对路径，不能逃逸 `.rfrec` 包目录。

### A5b 单独音频录制

状态：包结构、ready 门禁合同、`cmd/audio-smoke` 的 manifest-ready 包、RecordingService/Wails 后端入口、audio-only preflight、服务端强制 preflight 门禁和胶囊来源面板里的视频/音频模式入口已落地；默认设置保持系统声音、麦克风和 RNNoise 关闭，用户显式开启后才进入 preflight。Windows audio-only 采集期间先写包内 WAV sidecar，停止阶段通过 FFmpeg 生成主音频 `audio.m4a`，manifest 把已启用音轨标记为 `muxed` 并由 ready 门禁探测 `soun` 音轨。本机 smoke 已通过：`recording-2026-07-01-09-29-58-163.rfrec` 进入 `ready`，`audio.m4a` 可被 FFmpeg 解码，`audio.wav` sidecar 保留在包内。macOS ScreenCaptureKit 的系统声音 mux 只属于视频录制路径，audio-only 预检不会借用它来误报单独录音可用；macOS/Linux 真实 audio-only 音频源和三平台长录 smoke 仍未完成。作为后续录制模式支持，不阻塞屏幕录制主线。

交付：

- 新增 audio-only recording kind 或 source type，不创建假的 `screen.mp4`。代码已新增 `recordingMode: "audio-only"`、`audioPath` 和 `recpackage.CreateAudioOnly()`。
- 支持系统声音、麦克风、麦克风 + RNNoise 的单独录音。
- 单独录音必须走正式 `RecordingService.StartAudioOnlyRecording()` 入口和统一状态机，不能由 CLI 或 UI 绕过服务层临时拼包。胶囊 UI 当前通过来源面板里的视频/音频模式切换进入 audio-only 分支，开始前调用 `PreflightAudioOnlyRecording()`。
- 默认输出使用 `audio.m4a` 主音频；采集期间使用可持续写盘的 WAV sidecar，停止阶段由 FFmpeg 封装为 `audio.m4a`，仍写入 `<DataRoot>/data/video/<session>.rfrec/`。当前 ready 门禁已能在 audio-only 模式校验主音频媒体 `soun` 音轨，也能校验明确声明的 WAV fallback，不再要求 `screen.mp4`。
- 复用 `audio.Pipeline`、RNNoise reset、diagnostics 和 bounded buffer 策略。

验收：

- Go 测试覆盖：audio-only 包不创建 screen 路径、默认主媒体为 `audio.m4a`、没有真实音频媒体不能 ready、缺少 `soun` 音轨不能 ready。
- `cmd/audio-smoke` 使用 `RecordingService.StartAudioOnlyRecording()` 创建正式包，单流采集先产出 `audio.wav` sidecar，停止后封装 `audio.m4a`、写入 sync diagnostics，并通过 `ValidateReady()` 后标记 `ready`。
- `RecordingService.StartAudioOnlyRecording()` 使用 audio-only runtime，不创建 video session，暂停/继续/停止走同一状态机，停止后通过 `ValidateReady()` 才写 `ready`。代码已覆盖。
- 只录音 1 分钟、5 分钟可播放。
- 关闭麦克风或系统声音时不会保留旧 device id。
- audio-only 包可以在 UI 中识别为音频录制，不被误认为损坏的屏幕录制。

### A5c 内存和磁盘写入优化

状态：设计已明确，底层包合同已支持 mux 优先；Windows 已完成停止阶段 mux，能让最终 `screen.mp4` 包含音频并减少用户侧后续合并成本。具体 live PCM pipe、bounded queue、flush cadence 和长录内存水位监控仍待在各平台 writer 中实现。

交付：

- 使用有限环形缓冲和编码队列，不能把完整录制放进内存。
- 主录制优先 mux 到单个 `screen.mp4`，减少重复写长期音频 sidecar。
- 内存只用于 sample 队列、短环形缓冲、暂停边界和低频 checkpoint；最终媒体必须持续顺序落盘，避免崩溃时丢失整段录制。
- 队列需要有明确容量、背压和失败诊断；不能因为编码或磁盘变慢而无限吃内存。
- diagnostics 和 manifest 低频批量 flush，状态切换时强制落盘。
- 预览、波形、缩略图和导出 cache 按需写入包内 `cache/`，不在录制时反复读取完整媒体。

验收：

- 30 分钟录制内存占用保持稳定，不随时长线性增长。
- 录制中主媒体持续顺序增长，崩溃后保留可检查媒体。
- 关闭 fallback sidecar 后，启用音频的录制不会重复写长期 WAV 文件。

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

状态：macOS ScreenCaptureKit display/window/region video session 已接入代码路径：`screen:display-<CGDirectDisplayID>`、`window:<CGWindowID>` 和单显示器 `region:custom` 会通过 `SCStream` 写入真实 `screen.mp4`，并输出 `video-diagnostics.json`；`application:<pid>` 解析代码保留给后续演进，但 Program/Application capability 当前标记为 queued，不在用户菜单展示，也不作为初版验收完成项。macOS system audio 已接入 ScreenCaptureKit audio output，并通过 `AVAssetWriter` mux 到同一个 `screen.mp4`；macOS 麦克风由 CoreAudio native audio source 提供 PCM sidecar/diagnostics 代码路径，仍待真机验证与停止阶段 mux/sync 复核。Windows 默认 backend 已切换为 `ffmpeg-desktop-capture`：Win32 枚举的 `screen:<display-token>`、`all-screens:virtual-desktop`、`region:custom` 和 `window:<HWND hex>` 会交给 FFmpeg `gdigrab` 分段 writer，pause/resume 以 segment 方式处理，stop 合并为包内 `screen.mp4`；启用系统声音或麦克风时，WASAPI sidecar 会在停止阶段 mux 到主 `screen.mp4` 并把 manifest 标记为 `muxed`；缺少 `ffmpeg` 时 preflight blocked，直接调用也会得到 failed package 和失败诊断，不写假媒体。当前源合同已新增 `all-screens`、`region`、单屏坐标和 `source.geometry` manifest 写入；区域十字框选 overlay 已接通，可覆盖虚拟桌面、拖拽红框、尺寸浮标、Esc 取消并在成功框选后回填 selected source；同一份 geometry 已进入 `video.CaptureConfig` 与 `video-diagnostics.json`。macOS 单显示器区域已接入 ScreenCaptureKit `sourceRect` crop writer，原生层会把虚拟桌面逻辑坐标转换成显示器本地 crop rect；Windows 区域 crop 会把框选结果转换为物理像素矩形后交给 FFmpeg desktop crop；Linux 区域 crop 仍保持 queued。已新增 `cmd/video-smoke` 作为无 UI 真机验收入口。Windows 本机真实 smoke 已验证 screen、all-screens、region、locked-window、pause/resume segment merge、系统声音 mux、麦克风 mux、系统声音 + 麦克风混音 mux，以及 1 分钟、5 分钟和 20 分钟可解码录制包；20 分钟包为 `recording-2026-07-01-09-46-38-233.rfrec`，`screen.mp4` 119345863 bytes，36028 frames，duration 1200935ms，manifest 记录 `FFmpeg desktop capture wrote 2 segment(s).`。仍需 Windows clean-machine artifact 验证。macOS 仍需真机 smoke；Linux PipeWire 仍未完成。

任务：

- macOS ScreenCaptureKit 最小 display `screen.mp4` 写盘。代码已接入，待真机录制验收。
- macOS ScreenCaptureKit 最小 window `screen.mp4` 写盘。代码已接入，待真机录制验收。
- macOS ScreenCaptureKit 最小 program `screen.mp4` 写盘。代码已接入，待真机录制验收。
- macOS ScreenCaptureKit system audio mux 到 `screen.mp4`。代码已接入，待真机录制验收：`go run ./cmd/video-smoke -system -duration=1m`。
- 多屏幕识别：`DeviceService.ListSources()` 必须返回单个显示器坐标，并在多显示器时暴露 `all-screens:virtual-desktop` 源；Windows FFmpeg backend 可录虚拟桌面，macOS/Linux 多屏合成 writer 完成前仍不能进入 ready。
- 单屏选择 UI：胶囊来源菜单按 `屏幕 1..N` / `Screen 1..N` 展示每块显示器，移入或聚焦时在对应物理屏幕显示编号标识；当前已完成 `ShowScreenIndicator()` / `HideScreenIndicator()` 与前端 hover/focus 接入。
- 程序来源 UI：Application/Program 后端合同保留给后续演进，但当前胶囊来源菜单和能力矩阵不展示程序来源；preflight 也不能把它作为初版 ready 能力。
- 区域录制 overlay：点击区域后出现跨显示器透明 overlay、十字光标、拖拽红框、尺寸浮标、松开确认、Esc 取消；确认后把虚拟桌面坐标写入 `source.geometry`。框选完成但尚未录制时，同一个透明 `region-overlay` 进入编辑态，只保留红色边框、中心移动十字、八个缩放热区和右上尺寸/取消控件，支持扩大、缩小、移动和取消框选。
- 区域录制边框：开始录制后同一个透明 `region-overlay` 切为 recording 模式，设置鼠标穿透，只绘制一圈红色边框标识选中范围；录制中保持显示，切换到非区域来源或音频模式时隐藏。不再创建四个独立窄条 WebView 边框窗口，避免 Windows 打包态露出浅色背景和关闭按钮。
- 录制态配置锁定：进入 preparing/recording/paused/stopping 后，胶囊 UI 会禁用来源/区域/音频/摄像头入口并关闭已打开的对应面板；结束录制后重新启用。
- 区域录制 crop writer：macOS 单显示器区域已读取 `video.CaptureConfig.SourceGeometry`，并通过 ScreenCaptureKit `sourceRect` 输出裁剪后的 `screen.mp4`；区域 geometry 在 manifest / diagnostics 中保持虚拟桌面逻辑坐标，ScreenCaptureKit 原生层负责转换到显示器本地坐标。Windows 已读取同一 geometry 并通过 FFmpeg `offset_x/offset_y/video_size` 裁剪虚拟桌面；Linux writer 仍需要读取同一 geometry 并做真实裁剪或 crop/scaling。
- 锁定窗口录制：窗口列表选择的是固定 native target；最小化/关闭/权限丢失时必须停止或 failed diagnostics，不允许改录其他窗口。
- macOS `cmd/video-smoke` 真实录制验收：`go run ./cmd/video-smoke -duration=1m` 和 `go run ./cmd/video-smoke -duration=5m -pause-after=10s -pause-duration=2s`。
- macOS window smoke：`go run ./cmd/video-smoke -source-type=window -duration=1m`。
- macOS program smoke：`go run ./cmd/video-smoke -source-type=application -duration=1m`。
- Windows FFmpeg desktop 最小 `screen.mp4` 写盘；本机 3 秒 `screen` / `all-screens` smoke 已通过，本轮再次验证 `screen + system audio`、`all-screens`、`pause/resume` 和 `screen + system audio + microphone` 均进入 `ready`。长一点 smoke 已补：`1m screen + system audio + microphone`、`5m screen + pause/resume` 和 `20m screen + system audio + microphone + pause/resume` 均进入 `ready`，主媒体可被 FFmpeg 解码。下一步在 release portable zip 解压环境验证同一矩阵。
- Windows 区域录制和锁定窗口录制 smoke：本机 `region` / `window` smoke 已通过，pause/resume segment merge 已通过；本轮 3 秒 `region` 和 `locked-window` smoke 再次验证包进入 `ready`。系统声音、麦克风和二者组合已验证 mux 到 `screen.mp4`，最新 `screen + system audio + microphone` 包 `recording-2026-07-01-09-34-32-830.rfrec` 可被 FFmpeg 解码；最新 1 分钟音视频包 `recording-2026-07-01-09-37-05-743.rfrec` 和 20 分钟音视频包 `recording-2026-07-01-09-46-38-233.rfrec` 均可被 FFmpeg 解码。下一步做失败诊断和 clean-machine artifact 验收。
- Linux XDG Desktop Portal + PipeWire experimental 写盘。
- 将 CoreGraphics/Win32/PipeWire 源 ID 映射为真实 capture target。
- 录制中 `screen.mp4` 持续增长。
- ready 前缺失或 0 字节 screen media 必须失败。

验收：

- 1 分钟、5 分钟、20 分钟录制可停止并播放。
- 暂停/继续不会造成 manifest 状态错误。
- kill app 后可扫描 recoverable package。

## P1-CAMERA：摄像头 sidecar 与画中画预备（暂停）

状态：按当前验收策略暂停。摄像头 sidecar 和画中画不参与本轮视频录制与语音/音频录制验收，也不作为 preview 发布门槛。已有代码路径保持可追踪：Windows FFmpeg DirectShow sidecar writer 已接入 runtime，开启摄像头时使用枚举到的 DirectShow 原生设备名写入包内 `webcam.mp4`，pause/resume 会分段录制并在 stop 合并，sync diagnostics 会写入 `webcam` track 和 `webcamStartOffsetMs`。`PIPExport` 仍为 queued。`go run ./cmd/video-smoke -duration=2s -camera` 已执行到真实设备选择阶段，但当前本机没有可用 sidecar 摄像头，因此不能记录为真实摄像头 smoke 通过。等视频录制和语音/音频录制验收后，再恢复有摄像头机器的 sidecar smoke、长录同步和 clean-machine artifact 验证。macOS AVFoundation、Linux PipeWire 摄像头 sidecar 未完成。

任务：

- macOS AVFoundation 摄像头 sidecar。
- Windows DirectShow camera sidecar 真机 smoke、长录同步和 clean-machine artifact 验证；后续再评估是否替换为 Media Foundation。
- Linux PipeWire 摄像头 sidecar experimental。
- 记录 `webcamStartOffsetMs`。Windows 已写入，macOS/Linux 待接入。
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

状态：preview 发布链路已能产出可下载验收包。`v0.1.0-preview.14` 已作为 GitHub prerelease 发布，Release Gate、Windows x64、macOS arm64、Linux x64 和 Publish GitHub Release 均通过；产物包含 Windows portable zip、macOS arm64 raw preview binary、Linux x64 raw preview binary 和 SHA256SUMS。该版本把真实麦克风设备枚举/电平监听、真实 WASAPI endpoint 选择、macOS CoreAudio 麦克风代码路径、Windows GUI subsystem、FFmpeg 子进程命令窗口隐藏、录制态配置锁定、区域录制持久边框、胶囊透明背景、屏幕编号标识、区域编辑 overlay、录制态透明 overlay 边框和 CoreAudio annotation 清理纳入验收包。`scripts/verify-windows-preview-release.ps1` 已提供发布后 Windows preview asset 下载复验：下载 portable zip 和 SHA256SUMS，校验 SHA256，复用 `verify-windows-portable.ps1` 检查 x64 GUI PE、FFmpeg/FFprobe 和第三方说明。正式签名安装包、公证、Linux 包格式和 clean-machine 真实录制验收仍未完成。

任务：

- macOS Developer ID Application 签名、公证、staple。
- Windows installer/MSIX 或 portable zip 验证。preview portable zip 已通过 GitHub Windows runner 内容、x64 GUI PE、FFmpeg/FFprobe 执行校验；发布后可用 `scripts/verify-windows-preview-release.ps1 -TagName <tag>` 下载并校验 SHA256 与 portable zip 结构；仍需把 artifact 带到 clean machine 跑真实 `video-smoke` 矩阵。
- Linux deb/rpm/AppImage 取舍和验证。
- release workflow 缺少签名密钥时必须失败或只产出明确 preview artifact。

验收：

- GitHub Actions 产出可安装包。当前 preview 只产出 Windows portable zip 和 macOS/Linux raw binary，不等同正式安装包。
- SHA256SUMS 和 release notes 完整。`v0.1.0-preview.14` 已完成 preview 级别 SHA256SUMS 和 release notes。
- macOS Gatekeeper 不阻止已公证包。
- Windows clean machine 可启动。

## P2-QUALITY：长期稳定性

任务：

- 长录制磁盘空间持续监控。
- 容器级 media probe。
- `cache/media-info.json`。
- chunked cursor/audio diagnostics。
- 崩溃恢复分级原因。
- 用户可见的诊断包入口。显示录制文件夹已完成：设置窗口 Storage 行会打开当前应用管理的 `data/video`；最近录制包内容入口已完成：Recording package 行会通过后端 `OpenRecordingPackage()` 打开最近一次真实 `.rfrec` 包，且拒绝 `data/video` 外部路径和非 `.rfrec` 目录。

验收：

- 20 分钟以上录制可恢复、可播放、可诊断。
- 失败包不会被误标记 ready。
- 用户可以定位录制包和诊断文件。

## 第一执行顺序

1. A0 音频后端边界。已完成基础代码合同和 `NativeBackendRuntime`。
2. A1 真实音频设备枚举。Windows 已完成；macOS CoreAudio 输入设备枚举代码路径已完成，下一步补 macOS 真机验证和 Linux。
3. A3 麦克风采集。Windows 已完成并 smoke 验证；macOS CoreAudio 麦克风采集代码路径已完成，下一步补 macOS 真机 smoke、长录同步和 Linux。
4. A4 RNNoise native DSP。wrapper 已迁移并恢复 CI/release gate 定向验证；能力矩阵已按 `rnnoise.Available()` 动态展示；preview/release artifact 已改为默认启用 `rnnoise_native` 并通过 `desktop-doctor -require-rnnoise`。下一步是在目标桌面补真实 `audio-smoke -rnnoise`、听感检查和长录诊断。
5. A2 系统声音采集。Windows source 已实现并通过有播放源真实样本 smoke，录屏 runtime 已能启动 WASAPI sidecar 并在停止阶段 mux 到主 `screen.mp4`，Windows 20 分钟音视频长录已通过，Windows portable zip 已能准备 FFmpeg 依赖并在 `v0.1.0-preview.13` release workflow 中通过内容校验；下一步做 release artifact clean-machine 真实录制验收。
6. A5 音频混音、mux 与写盘。Windows 屏幕录制停止阶段 mux 与 audio-only 停止阶段 `audio.m4a` 封装已完成；macOS CoreAudio 麦克风采集源已接入 native audio runtime，下一步补 macOS 真机 mux/sync 验收、Linux 音频源、live PCM pipe/内存水位策略，并做三平台长录同步。
7. A6 预检、UI 和设置联动。
8. A7 三平台手动验证矩阵。

完成三平台真机 video-smoke、目标桌面 RNNoise 实录 smoke、长录同步和 mux 验收前，不对外宣称 RecordingFreedom 的正式录制流程已支持完整真实音视频录制。摄像头 sidecar/PIP 当前暂停，不计入本轮完成声明。
