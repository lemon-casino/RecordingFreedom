# Windows Cursor Stability And Segmenting

## 背景

Windows 旧录制路径使用 FFmpeg `gdigrab`，并通过 `-draw_mouse 1` 把鼠标绘制到录制帧里。这个路径能录制鼠标，但它依赖 GDI 桌面读取和 GDI 光标重绘；当软件同时存在透明置顶窗口、区域录制红框、摄像头 PIP 预览等 overlay 时，用户侧容易观察到鼠标闪烁或录制画面中鼠标不稳定。

本轮确认：问题不应只按“频繁写盘”处理。Windows 和 FFmpeg 本身有文件缓存，持续写 MP4 不等于每一帧同步刷盘；写盘优化能降低长录制风险和恢复成本，但不能替代鼠标捕获链路的修复。

## 本版根因修复

- Windows 屏幕录制和绑定到单显示器的区域录制改为 FFmpeg `ddagrab` filter，也就是 Windows Desktop Duplication API 路径。
- `ddagrab` 仍启用 `draw_mouse=1`，所以继续录制鼠标，但绕过 `gdigrab -draw_mouse 1` 的 GDI 光标重绘路径。
- 多屏“全部屏幕”改为多路 `ddagrab` + `xstack` 合成虚拟桌面，不再因为多显示器直接回落到 GDI 鼠标绘制。
- 锁定窗口录制仍使用 FFmpeg `gdigrab hwnd=`，因为 FFmpeg 的 DDA filter 没有 HWND 锁定输入；该路径被诊断隔离为 `windows-gdi`，后续若窗口锁定仍复现闪烁，应升级为原生 Windows Graphics Capture + Media Foundation writer。

验收时查看包内 `video-diagnostics.json`：

- 正常屏幕、区域、全部屏幕应看到 `FFmpeg input engine: windows-dda.`
- 诊断消息应包含 `GDI cursor drawing is bypassed`
- 如果看到 `FFmpeg input engine: windows-gdi.`，表示该来源仍在 GDI fallback。

## 分片与写盘策略

本版将 Windows FFmpeg writer 从“普通长录一个大 MP4 segment”改为 FFmpeg segment muxer：

- 默认每 60 秒写一个视频分片。
- 分片先写入录制包内 `cache/ffmpeg-video/<output>/segment-xxx-yyy.mp4`。
- 点击结束录制后，使用 FFmpeg concat 合并为最终 `screen.mp4`。
- 如果只有一个分片，则直接移动为 `screen.mp4`。
- 分片时强制 2 秒 keyframe/GOP，保证 segment muxer 有稳定切点。

调试环境变量：

```powershell
$env:RECORDINGFREEDOM_FFMPEG_SEGMENT_SECONDS = "5"
```

允许范围是 5 到 600 秒；默认是 60 秒。该变量只用于调试或特殊机器，不建议普通用户手动设置。

## 为什么不把完整视频放进内存

完整 MP4 放内存不是长期方案：

- 长录制内存不可控，高清和多屏很容易占用数 GB。
- 崩溃或断电会丢失内存里的未落盘内容。
- MP4 容器需要正确收尾，内存里攒一个大 MP4 对恢复不友好。

当前方案使用“有界分片 + 操作系统页缓存 + 停止时合并”。它减少单文件长时间写入风险，保留崩溃恢复空间，也不会把内存压力转嫁给用户机器。

## 已验证

- `video-smoke` 屏幕录制 3 秒：`windows-dda`，鼠标录制开启，生成 ready 包并验证视频轨。
- `video-smoke` 区域录制 3 秒：`windows-dda`，鼠标录制开启，生成 ready 包并验证视频轨。
- `video-smoke` 区域录制 12 秒，调试分片 5 秒：写出 2 个 segment，停止后合并为一个 `screen.mp4`，诊断显示 `Merged 2 FFmpeg segments into screen.mp4.`
