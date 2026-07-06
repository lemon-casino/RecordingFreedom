# RecordingFreedom 重构文档入口

RecordingFreedom 是一个全新的录制软件重构项目，目标技术栈为 Go + React + Wails v3。当前 LikelySnap 项目只作为经验来源，不作为技术栈延续；旧 Electron 主进程、浏览器录制保存路径、旧主题和旧发布链路都不直接迁移。

## 文档阅读顺序

1. [00-project-analysis.md](00-project-analysis.md)  
   识别当前项目架构，列出可继承资产和必须替换的旧债。
2. [01-product-ui-plan.md](01-product-ui-plan.md)  
   定义第一阶段胶囊托盘工具窗口、源选择、音频、摄像头、语言和设置体验。
3. [02-technical-architecture.md](02-technical-architecture.md)  
   定义 Go/Wails/React 新架构、目录、服务接口、事件流和应用内部数据目录。
4. [03-recording-pipeline.md](03-recording-pipeline.md)  
   定义屏幕/窗口/程序录制、系统声音、麦克风降噪、摄像头 sidecar、包结构、暂停恢复和崩溃恢复。
5. [04-ci-release-plan.md](04-ci-release-plan.md)  
   定义全平台 GitHub Actions、签名、公证、artifact 验证和发布门禁。
6. [05-roadmap-acceptance.md](05-roadmap-acceptance.md)  
   定义阶段路线、开发顺序和每个阶段的验收条件。
7. [06-implementation-progress.md](06-implementation-progress.md)  
   记录当前 Wails v3 + React UI Shell 的实际落地状态、验证结果和下一步。
8. [07-icon-workflow.md](07-icon-workflow.md)
   记录电脑端图标替换、多尺寸 PNG 重建、Wails `.ico/.icns` 生成和提交检查流程。
9. [08-unfinished-task-plan-audio-first.md](08-unfinished-task-plan-audio-first.md)
   记录初版 preview 后未完成任务，并把真实音频采集与 RNNoise 降噪列为第一优先级。
10. [09-camera-pip-plan.md](09-camera-pip-plan.md)
   记录摄像头 sidecar 与画中画恢复开发后的需求、架构和验收计划。
11. [10-windows-cursor-stability-and-segmenting.md](10-windows-cursor-stability-and-segmenting.md)
   记录 Windows 鼠标闪烁根因、DDA 捕获、全部屏幕合成、分片写盘和最终合并策略。
12. [11-whiteboard-excalidraw-integration-plan.md](11-whiteboard-excalidraw-integration-plan.md)
   记录 Excalidraw 画板调研、画板胶囊、录制标注 overlay、录制包写入和导出合成计划。
13. [12-annotation-overlay-platform-smoke.md](12-annotation-overlay-platform-smoke.md)
   固定录制标注 overlay 的真实桌面验收矩阵、证据目录和通过标准，区分导出合成 smoke 与透明窗口实机验收。
14. [13-rnnoise-dynamic-module.md](13-rnnoise-dynamic-module.md)
   固定 RNNoise 源码按平台/架构编译动态原生模块、随包发布、运行时加载和 release 门禁标准。
15. [14-screenshot-whiteboard-plan.md](14-screenshot-whiteboard-plan.md)
   记录截图、截图历史、钉图/固定、截图进入画板，以及滚动截图的原生自动化验收路径。
16. [15-smart-screenshot-region-assist.md](15-smart-screenshot-region-assist.md)
   记录 snow-shot 智能窗口识别分析、RecordingFreedom 智能候选合同、边缘吸附、焦点窗口截图和当前限制。
17. [16-floating-panel-window-plan.md](16-floating-panel-window-plan.md)
   记录胶囊列表多窗口浮层方案，目标是让未展开时只占胶囊大小，列表/下拉独立占用实际区域。
18. [17-ocr-translation-dispatch-plan.md](17-ocr-translation-dispatch-plan.md)
   记录 OCR 本地模型、跨平台 worker、截图/滚动截图/画板调度、识别缓存和翻译提供方方案。
19. [18-remaining-work-regroup.md](18-remaining-work-regroup.md)
   重新归拢当前剩余功能、问题、暂停项、批次顺序和完成定义，后续按 A/B/C/D 四个闭环批量推进。
20. [19-ocr-desktop-evidence-runbook.md](19-ocr-desktop-evidence-runbook.md)
   固定真实 Wails 桌面 OCR evidence 的人工操作 runbook、session start/end、视觉证据命名、导出和 checker 命令。

## 当前硬性决策

- 新代码全部放在 `RecordingFreedom/`，后续连接新仓库 `lemon-casino/RecordingFreedom.git`。
- 第一阶段先完成可运行、可验证、可扩展的胶囊录制工具窗口；录制能力按路线逐步接入。
- 新录制视频和录制包默认存入软件托管数据目录的 `data/video/` 下。
- 生产环境不能把数据写进只读安装包本体；各平台使用应用数据根目录，再在其下保持统一的 `data/video/` 结构。
- 默认录制包目录为 `data/video/recording-YYYY-MM-DD-HH-mm-ss-SSS.rfrec/`。
- 麦克风降噪继承旧项目 RNNoise 原生处理思路，但发布标准改为跨平台动态原生模块：Windows `rnnoise.dll`、macOS `librnnoise.dylib`、Linux `librnnoise.so` 随包进入 `tools/` 并由 `rnnoise_dynamic` 运行时加载。
- 后续画中画摄像头从 v1 数据模型开始预留，先录 webcam sidecar，不把摄像头硬烘焙进屏幕视频。
- 当前已生成 Wails v3 React 工程骨架，并实现第一版胶囊 UI Shell 与 mock `.rfrec` 包创建服务。
- 初版 preview 后的未完成任务以 `08-unfinished-task-plan-audio-first.md` 为执行清单，优先推进真实系统声音、麦克风采集和 RNNoise 降噪。
- 画板能力采用 Excalidraw React 组件作为画布核心，但由 RecordingFreedom 自己负责胶囊 UI、透明 overlay、点击穿透、录制包写入和导出合成。
- 录制标注 overlay 不能只靠 `annotation-export-smoke` 宣称完成；真实平台完成状态必须按 `12-annotation-overlay-platform-smoke.md` 保存 evidence、来源矩阵、元素级标注事件、绘制/穿透 hit-region 诊断、1 分钟/5 分钟 `.rfrec`、`annotations/overlay-diagnostics.jsonl` 和真实导出视频，并通过矩阵验收。
- 胶囊内所有会展开列表或面板的入口，后续应迁移到 `16-floating-panel-window-plan.md` 定义的 floating panel / floating select 合同；不要再通过扩大胶囊主窗口来承载大型 `.popover` 或 `.select-menu-list`。
- OCR 采用本地模型优先的独立 worker 调度方案，截图、滚动截图、钉图和画板统一走 `17-ocr-translation-dispatch-plan.md` 定义的识别/翻译合同；翻译默认关闭，只有用户配置提供方后才会把识别文字发送到外部服务。
- 真实桌面 OCR 验收必须先用 `ocr-desktop-evidence-session` 写入 `session-start/session-end`，再导出 evidence 包并通过检查器；源码树提供 `scripts/export-ocr-desktop-evidence.ps1` 和 `scripts/export-ocr-desktop-evidence.sh`，要求真实 `visual-dir`，导出后会立即运行 `ocr-desktop-evidence-check`，不能用浏览器 fallback、假截图或导出阶段补 marker 替代。

## 外部技术依据

- Wails v3 文档: <https://v3.wails.io/>
- Wails v3 系统托盘: <https://v3.wails.io/features/menus/systray/>
- Wails v3 多窗口: <https://v3.wails.io/features/windows/multiple/>
- Wails v3 构建系统: <https://v3.wails.io/concepts/build-system/>
- Apple ScreenCaptureKit: <https://developer.apple.com/documentation/screencapturekit>
- Microsoft Windows.Graphics.Capture: <https://learn.microsoft.com/en-us/windows/apps/develop/media-authoring-processing/screen-capture>
- XDG Desktop Portal ScreenCast: <https://flatpak.github.io/xdg-desktop-portal/docs/doc-org.freedesktop.portal.ScreenCast.html>
