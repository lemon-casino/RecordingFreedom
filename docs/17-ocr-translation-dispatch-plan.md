# OCR and Translation Dispatch Plan

更新时间：2026-07-06。

本文档是 RecordingFreedom OCR/翻译调度的唯一执行口径。后续开发只按 `G0-G15` 颗粒推进和验收，不再使用另一套 Phase 分层；每一粒必须能单独验证，不能用占位 UI、mock 文本、协议壳或不可用按钮冒充真实能力。

## 本轮对齐结论

用户需求可以压缩成一句话：RecordingFreedom 要拥有一套统一、本地优先、可升级、可验证的 OCR 和识别后翻译调度能力，并服务于区域截图、全屏截图、滚动截图、截图历史、钉图、普通画板和录制中的画板。

2026-07-06 本轮冻结口径：

- `17-ocr-translation-dispatch-plan.md` 是 OCR/翻译唯一执行文档；后续开发只按本文 `G0-G15` 和 `R1-R10` 颗粒推进。
- snow-shot 只作为能力参考：吸收“本地 RapidOCR/PaddleOCR 风格识别 + 识别后翻译 provider 配置”的产品方向，不照搬 Tauri/Rust 插件结构。
- RecordingFreedom 第一版 OCR 的最小闭环是“稳定图片文件 -> 统一 OCR queue -> 独立 worker -> 持久化 result -> floating panel/历史/钉图/画板消费”，任何直接读屏幕、直接读 DOM、假 result、假下载都不算完成。
- “最新 OCR”按 stable/latest/quality 三通道落地：stable 先保证可离线验收，latest/quality 必须经过 RecordingFreedom release catalog、SHA256、manifest、worker smoke、真实截图样例和跨平台 runtime 验证后才能给用户安装。
- 模型下载器现在已经完成可验证 package 下载合同、后端校验、进度事件、设置页 UI、release catalog 刷新入口和 stable catalog 产物生成门禁；真实 PP-OCRv6 latest/quality catalog 与桌面下载回验仍属于 R9 未完成项。
- 翻译只消费 OCR result，默认关闭；没有 provider、密钥和隐私确认时不发送任何识别文字。
- 所有 OCR/翻译结果 UI 必须走 `16-floating-panel-window-plan.md` 的 floating panel，不允许让主胶囊变大或闪烁。

本计划的执行口径如下：

- OCR 不是独立小工具，也不是某个按钮的局部能力；它是截图、钉图、画板共同调用的后端服务。
- 每个入口都必须先保存或导出稳定图片文件，再提交同一个 `OcrRecognizeRequest`；不能让 OCR 直接读屏幕、读窗口句柄或读前端 DOM。
- 第一版以本地 PaddleOCR/RapidOCR 风格 ONNX pipeline 为主，默认稳定模型先保证中英文可离线验收；“最新 OCR”通过 PP-OCRv6 latest/quality 模型通道进入，但必须经过 SHA256、manifest、worker smoke、真实截图样例和全平台 runtime 验证后才能给用户切换。
- 翻译只消费 OCR result，默认关闭；只有用户显式配置 provider、密钥、目标语言并确认隐私后，才允许发送识别文本。
- OCR/翻译结果 UI 必须走独立 floating panel，不允许重新撑大主胶囊，不允许造成胶囊闪烁。
- 录制中的画板允许调度 OCR，但 OCR 只能后台排队，不能阻塞录制、鼠标、写盘或画板绘制。
- 每一粒完成后都必须更新本文档状态，并留下单测、e2e、worker smoke、桌面录屏或 release gate 中至少一种直接证据。

需求到开发颗粒的对应关系：

| 用户能力 | 归属颗粒 | 第一版完成口径 | 不算完成的情况 |
| --- | --- | --- | --- |
| 区域截图识别/翻译 | `G8/R1` + `G13/R8` | 区域 PNG 保存后排队 OCR，ready 后可打开结果、复制、翻译 | 只保存截图但 OCR 状态假 ready |
| 全屏截图识别/翻译 | `G8/R1` + `G13/R8` | 单屏/多屏最终 PNG 识别，坐标按最终图片像素 | 只识别主屏或坐标按物理屏错位 |
| 滚动截图识别/翻译 | `G12/R7` + `G13/R8` | 只识别最终长图，无滚动回退普通区域截图 | 识别中间帧或保存假长图 |
| 截图历史重新识别 | `G7/G8/R1` | 历史项可重新排队、重试、打开 result，缓存命中仍写当前 source result | 结果只存在内存，重启后丢失 |
| 钉图 OCR 高亮 | `G10/R3` | 钉图读取真实 result，缩放/resize 后 block 仍对齐 | 用固定像素 overlay 或修改原图 |
| 画板图片 OCR | `G11/R4/R5` | 背景图、选中图片、整图导出都保存为稳定图片后识别 | 直接读 DOM/dataURL，不落盘 |
| 录制中画板 OCR | `G11/G15/R6` | OCR worker 与录制并行，失败只影响 OCR 状态 | OCR 失败导致录制停止或鼠标卡顿 |
| 图片内容翻译 | `G13/R8` | `deepl`/`openai-compatible` 显式配置后调用，缓存译文 | 默认联网或把 API key 写入 settings/log |
| 最新模型调度 | `G14/R9` | stable/latest/quality 三通道，下载校验 smoke 后主动切换 | 未验证模型直接展示为可用 |
| 全平台发布 | `G15/R10` | Windows/macOS/Linux x64/ARM64 worker/runtime/model smoke 过门禁 | 只在当前机器跑通 |

本轮颗粒度锁定：

| 粒度 | 说明 | 进入下一粒的最低证据 | 不允许用来替代的证据 |
| --- | --- | --- | --- |
| `C` 合同粒 | API、事件、sourceKind、cache key、错误码、持久化路径 | Go 单测或绑定生成验证 | 只有文档描述 |
| `Q` 队列粒 | 入口把稳定图片文件排入同一 OCR queue，状态可追踪 | job queued/running/ready/failed/cancelled 单测或 e2e | 前端自己设置 loading/ready |
| `W` Worker 粒 | worker 用真实 runtime/model 返回 blocks/plainText/box/confidence | worker smoke 或服务层真实 worker smoke | mock 文本、固定 boxes、协议壳 |
| `U` UI 粒 | floating panel/历史/钉图/画板消费真实 result | 前端 e2e 或桌面录屏，证明不扩大主胶囊 | 浏览器里只验证按钮存在 |
| `D` 桌面粒 | Wails 桌面窗口、真实截图、真实 worker、真实文件路径闭环 | Windows/macOS/Linux 对应实机或可复现桌面证据 | 浏览器 fallback、单纯 `npm run build` |
| `R` 发布粒 | release artifact 中 worker/runtime/model/catalog/manifest 完整 | release gate、artifact 下载校验、平台 smoke | 本地开发目录能跑 |

执行时必须按 `C -> Q -> W -> U -> D -> R` 记录证据。某入口只完成 `C/Q/U` 时，只能写“部分落地”，不能对用户说“已完成 OCR”。只有真实 worker 和桌面入口证据都存在，才能进入最终验收。

## 需求对齐

目标不是单独做一个 OCR 小工具，而是让 RecordingFreedom 拥有统一的图片文字识别和识别后翻译调度能力。

必须覆盖的入口：

- 区域截图：用户框选区域后，对保存的截图做 OCR 和翻译。
- 全屏截图：对单屏或多屏合成后的完整截图做 OCR 和翻译。
- 滚动截图：对最终拼接完成的长图做 OCR 和翻译；如果目标区域没有滚动，则按普通区域截图处理。
- 截图历史：历史图片可以随时重新识别、复制文字、翻译、打开 OCR 结果。
- 钉图/固定图片：钉图窗口可以显示 OCR 高亮、复制文字、翻译文字。
- 画板：截图进入画板后，可以对背景图片、选中图片元素、画板导出的图片做 OCR 和翻译。
- 录制中的画板：可以复用同一套 OCR 服务，但 OCR 不能阻塞录制、不能影响鼠标和画板绘制。

不属于第一版 OCR 的范围：

- 不做实时视频帧 OCR。
- 不做摄像头画面 OCR。
- 不把翻译做成默认联网行为。
- 不把 OCR 面板塞回主胶囊导致胶囊变大或闪烁。
- 不为了追最新模型牺牲稳定可用的本地 OCR。

最终用户视角：

1. 用户截图或打开一张截图历史图片。
2. 软件可以本地识别图片文字。
3. 用户可以复制原文、查看识别块、把识别结果导入画板。
4. 用户配置翻译提供方后，可以把识别文字翻译成目标语言。
5. 没有配置翻译时，OCR 仍然完全可用且不联网。

## 入口对齐矩阵

| 入口 | OCR 输入 | 调度 sourceKind | OCR 输出位置 | 翻译入口 | 验收口径 |
| --- | --- | --- | --- | --- | --- |
| 区域截图 | 框选后保存的 PNG | `region-screenshot` | 截图历史、OCR 结果浮层、钉图 | 结果浮层、历史项 | 截图保存不被 OCR 阻塞；同图二次识别走缓存 |
| 全屏截图 | 单屏或多屏合成 PNG | `full-screenshot` | 截图历史、OCR 结果浮层、钉图 | 结果浮层、历史项 | 坐标基于最终 PNG，不基于显示器物理坐标 |
| 窗口截图 | 窗口静态 PNG | `window-screenshot` | 截图历史、OCR 结果浮层、钉图 | 结果浮层、历史项 | 截图后不依赖原窗口仍存在 |
| 焦点窗口截图 | 焦点窗口静态 PNG | `focused-window-screenshot` | 截图历史、OCR 结果浮层、钉图 | 结果浮层、历史项 | 焦点切换后仍可重复识别保存图 |
| 滚动截图 | 最终拼接长图 PNG | `scrolling-screenshot` 或无滚动时 `region-screenshot` | 截图历史、OCR 结果浮层、钉图 | 结果浮层、历史项 | 不识别中间帧；无滚动不保存假长图 |
| 截图历史 | 历史图片原文件 | 沿用历史项 sourceKind | 历史项状态、OCR 结果浮层 | 历史项、结果浮层 | 旧历史缺 OCR 字段也能兼容读取 |
| 钉图/固定图片 | 钉图对应原图 hash | `pinned-screenshot` | 钉图高亮、OCR 结果浮层 | 钉图、结果浮层 | 高亮随缩放映射，不修改原图 |
| 画板背景 | 画板背景截图文件 | `whiteboard` | 画板高亮、OCR 结果浮层 | 画板、结果浮层 | 绑定图片 hash，不只绑定 element id |
| 画板选中图片 | 选中图片元素导出的 PNG | `whiteboard-selection` | 画板高亮、OCR 结果浮层 | 画板、结果浮层 | 选区识别不影响画板保存 |
| 画板整图导出 | 画板快照 PNG | `whiteboard` | 画板/历史/结果浮层 | 画板、结果浮层 | 录制中调度为后台队列，不阻塞录制 |

状态术语：

- `已落地`：代码、文档和测试/验证门禁都已经进入仓库。
- `部分落地`：合同或基础设施已进入仓库，但最终用户能力还不能宣称完成。
- `未完成`：还不能给用户验收。
- `阻塞`：需要先解决明确技术依赖，否则继续开发会制造临时方案。

## 颗粒度对齐

本方案的最小开发颗粒不是“某个按钮”，而是一条可验证的数据链路：

```text
图片来源
  region/full/window/focused-window/scrolling/history/pin/whiteboard
      |
      v
保存或导出为稳定图片文件
      |
      v
OcrRecognizeRequest(sourceKind + sourceId + imagePath + language + modelId)
      |
      v
OCR 队列去重、缓存、worker 推理
      |
      v
OcrResult(blocks + plainText + image pixel coordinates)
      |
      v
截图历史 / 钉图 / 画板 / OCR 结果浮层 / 翻译调度
```

对齐规则：

- 所有入口都必须先产生稳定图片文件，再提交 OCR；OCR 不直接读屏幕、不读窗口句柄、不读画板 DOM。
- `sourceKind` 只描述图片来源，不决定 OCR 算法；同一张图片、同一模型、同一语言必须复用同一份缓存。
- OCR block 坐标永远保存为图片像素坐标，UI 显示时再映射到截图历史、钉图缩放或画板画布坐标。
- 截图保存、画板绘制、录制中的画板不能等待 OCR 完成；OCR 只通过状态事件回填结果。
- 翻译只消费 `OcrResult`，不重新做 OCR，不覆盖原文，不默认联网。
- 任何入口没有真实模型、runtime、worker 或 provider 时，都必须显示明确不可用原因，不能显示假文字、占位译文或伪 ready 状态。

本文件中 `G0-G15` 是唯一开发颗粒。后续每次开发必须做到：

- 先确认对应 `Gx` 的范围、交付、验收。
- 只改对应颗粒需要的模块，避免把 OCR、截图、画板、翻译状态各写一套。
- 完成后更新本文件中对应 `Gx` 状态和“当前已落地颗粒 / 尚未完成”。
- 至少补一条对应颗粒的单测、前端构建验证或 e2e 入口验证；不能只靠手工说“看起来能点”。

颗粒完成标准：

| 层级 | 可以宣称完成 | 不能宣称完成 |
| --- | --- | --- |
| 合同层 | API、事件、数据结构、错误码、缓存 key 已固定，并有单测保护 | 只有文档或空方法，没有调用链 |
| 调度层 | 入口能把稳定图片文件、`sourceKind`、`sourceId`、语言和模型提交到同一 OCR 队列 | 每个入口私自做一套状态，或只在前端假装排队 |
| 推理层 | worker 用真实模型返回 `blocks/plainText/box/confidence`，失败有稳定错误码 | 输出固定文本、mock block、占位 ready |
| UI 层 | floating panel 能展示真实结果、复制、重试、错误原因，且不扩大主胶囊 | 按钮能点但没有真实数据、弹层导致胶囊变大或闪烁 |
| 验收层 | 至少有单测、前端构建、worker smoke、桌面/e2e 或人工证据中的一个直接覆盖该颗粒 | 只说“代码写了”或只跑了无关 build |

剩余开发必须按“先链路、再入口、再体验、最后发布门禁”的顺序推进。任何入口如果没有真实图片文件和真实 worker 结果，必须显示不可用或失败原因，不能显示假 OCR 文本。

## 2026-07-06 颗粒度再对齐后的可落地方案

本节用于对齐“我理解的需求”和“实际开发颗粒”。后续开发仍只落在 `G0-G15` 和 `R1-R10` 中；本节把它们拆成可以逐项验收的最小闭环。

### 最终目标

RecordingFreedom 要拥有统一的 OCR 调度能力，不是给每个入口各做一套识别逻辑。

最终必须同时满足：

- 区域截图、全屏截图、窗口截图、焦点窗口截图、滚动截图、截图历史、钉图、画板、录制中画板都能把“稳定图片文件”提交到同一个 OCR 服务。
- OCR 默认本地离线运行，模型和 runtime 可校验、可诊断、可替换。
- 翻译是 OCR 结果之后的可选动作，默认关闭，不配置 provider 不联网。
- OCR 结果、翻译结果都可缓存、可复用、可重试，失败不会影响截图保存、钉图、画板或录制。
- OCR/翻译 UI 必须走独立 floating panel，不允许让主胶囊扩大、闪烁或被列表撑大。
- 所有可见能力都必须有真实链路，不能有假文字、假结果、假 ready、假下载或不可用按钮。

### 技术选型结论

| 维度 | 第一版采用 | 后续增强 | 不采用为第一版默认 |
| --- | --- | --- | --- |
| OCR 推理 | 本地 PaddleOCR/RapidOCR 风格 ONNX pipeline，通过独立 `rf-ocr-worker` 调度 ONNX Runtime CPU | PP-OCRv6 latest/quality 模型包，必要时引入更强文档模型通道 | 隐藏在线 OCR、浏览器 OCR、主进程直接加载大模型 |
| 默认模型 | `ppocrv5-mobile-zh-en` stable，中英文优先，已经进入模型包和 smoke 流程 | `ppocrv6-mobile-zh-en` latest，`ppocrv6-medium-zh-en` quality，必须先通过 RecordingFreedom smoke | 未验证 SHA256/license/smoke 的“最新模型” |
| 复杂文档能力 | 普通截图文字块 OCR | 后续单独评估 PP-Structure、PaddleOCR-VL、表格/版面/Markdown 输出 | 第一版不把 VLM 文档理解混入截图 OCR |
| 翻译 | `deepl`、`openai-compatible`，用户显式配置 provider/base URL/key/model 后才发送文本 | 私有网关、本地翻译模型作为独立计划 | 默认联网翻译、无隐私确认上传 OCR 文本 |
| 平台 | Windows/macOS/Linux x64/ARM64 worker + runtime 分发 | 分平台 smoke 矩阵和下载器增强 | Android/iOS |

选择原因：

- RapidOCR 已验证“本地 OCR + ONNX Runtime + 多平台部署”的方向可行，适合桌面截图工具。
- PaddleOCR 3.x 的通用 OCR pipeline 已进入 PP-OCRv6 体系，latest/quality 应该以 PP-OCRv6 为升级目标，但必须先由 RecordingFreedom 自己完成模型包、SHA256、smoke 和真实截图样例验证。
- ONNX 模型本身不按平台拆分，平台差异主要在 worker 和 ONNX Runtime 动态库；因此模型下载器和 runtime 分发要分开设计。
- 截图软件的第一目标是快、稳、可离线验收；复杂文档理解、表格还原、VLM 解析属于后续增强，不应该阻塞第一版 OCR。

### 统一链路

每个入口必须落在同一条链路上：

```text
入口动作
  区域/全屏/窗口/焦点窗口/滚动/历史/钉图/画板/录制中画板
      |
      v
保存或导出稳定图片文件
  PNG/JPEG/WebP，路径受应用数据目录保护
      |
      v
提交 OcrRecognizeRequest
  sourceKind + sourceId + imagePath + language + modelId + priority
      |
      v
OCR Service
  去重、排队、缓存、状态事件、失败隔离
      |
      v
rf-ocr-worker
  ONNX Runtime + det/cls/rec + 坐标还原
      |
      v
持久化 OcrResult
  blocks + plainText + image pixel coordinates + source metadata
      |
      v
结果消费
  历史/钉图/画板/结果浮层/翻译/复制/插入画板文本
```

任何入口如果绕过这条链路，就不能算完成。

### 入口颗粒

| 入口 | 最小可验收颗粒 | 必须证明 | 不能接受 |
| --- | --- | --- | --- |
| 区域截图 | 保存 PNG 后提交 `region-screenshot` job | 历史项 queued/running/ready/failed 正确；block 坐标映射到最终 PNG | 截图时同步等待 OCR；只在前端显示假状态 |
| 全屏截图 | 单屏/多屏最终 PNG 提交 `full-screenshot` job | 多屏合成图按最终图片坐标识别，不按物理屏幕坐标错位 | 只识别某个屏幕或丢失第二屏内容 |
| 窗口截图 | 窗口静态图提交 `window-screenshot` job | 截图后原窗口关闭也能重新查看 OCR 结果 | OCR 依赖窗口句柄仍存在 |
| 焦点窗口截图 | 焦点窗口静态图提交 `focused-window-screenshot` job | 焦点切换后仍可从历史识别/复制/翻译 | 只保存焦点窗口 id，不保存图片 |
| 滚动截图 | 最终长图提交 `scrolling-screenshot`；无滚动回退 `region-screenshot` | 只识别最终长图；长图 tile 坐标回填完整图；无滚动不保存假长图 | 识别中间滚动帧或把无滚动目标伪装成长图 |
| 截图历史 | 历史项可重新识别、复制、翻译、打开结果 | 旧历史缺 OCR 字段可兼容；同图缓存命中仍写当前 source result | 历史项只显示一次性内存状态 |
| 钉图/固定图片 | 钉图读取 sidecar 并显示 OCR 高亮 | resize/缩放后 polygon 仍对齐；关闭重开状态恢复 | 修改原图或用固定像素 overlay |
| 画板背景 | 背景图导出稳定 PNG 后识别 | OCR result 绑定图片 hash/scene，不只绑 element id | 直接读画布 DOM 或临时 dataURL 不落盘 |
| 画板选中图片 | 选中图片元素导出独立 PNG，提交 `whiteboard-selection` | 结果浮层读取选中图片自己的 `imagePath`，block 坐标不套整图 | 选区结果 fallback 到整张画板导致坐标错 |
| 画板整图导出 | 当前画板快照保存为图片后识别 | OCR 失败不影响画板保存；可插入原文/译文为文本元素 | OCR 状态覆盖自动保存状态 |
| 录制中画板 | OCR 排队后台执行 | 录制状态、鼠标、绘制、写盘不被 OCR 阻塞 | OCR worker 失败导致录制停止 |

### 数据合同颗粒

| 合同 | 必须固定 | 验收 |
| --- | --- | --- |
| `SourceKind` | `region-screenshot/full-screenshot/window-screenshot/focused-window-screenshot/scrolling-screenshot/pinned-screenshot/whiteboard/whiteboard-selection/image` | Go 单测覆盖截图 mode 到 sourceKind 映射 |
| cache key | `imageSha256 + modelId + language + preprocessingVersion` | 同图、同模型、同语言二次识别不启动 worker；不同 source 仍写独立 result |
| OCR 坐标 | 永远保存图片像素坐标 | 历史、钉图、画板各自映射 UI 坐标，不修改 result |
| result 存储 | `data/ocr/results/`，包含 `imagePath/sourceKind/sourceId/modelId/language/blocks/plainText` | `OpenOcrResult` 只读 result，不触发重新 OCR |
| translation 存储 | `data/ocr/translations/`，不覆盖 OCR 原文 | provider 返回段数或 block id 错误时不写成功缓存 |
| secret 存储 | API key 不进 `settings.json` | Windows DPAPI、macOS Keychain、Linux Secret Service 或明确 fallback |
| UI 窗口 | OCR/翻译结果走 floating panel | 打开结果/翻译/模型面板不改变主胶囊尺寸 |

### 模型通道颗粒

| 通道 | 用户看到的能力 | 开发交付 | 验收标准 |
| --- | --- | --- | --- |
| stable | 默认稳定 OCR，中英文截图可离线识别 | `ppocrv5-mobile-zh-en` 模型包、manifest、SHA256、smoke、release artifact | smoke 识别 `RecordingFreedom` 和 `文字识别`；缺模型时 UI 显示可恢复原因 |
| latest | 新一代 PP-OCRv6 mobile/small | `ppocrv6-mobile-zh-en` catalog、下载、校验、安装、主动切换 | 只有 SHA256 和真实 smoke 通过后才显示可安装；失败不影响 stable |
| quality | 更大 PP-OCRv6 medium/server 类模型 | 独立 quality 包、体积/性能提示、主动安装 | 用户主动安装；可删除；切换失败可回退 stable |
| runtime | ONNX Runtime CPU 分平台分发 | Windows/macOS/Linux x64/ARM64 manifest、下载脚本、release staging | 每个平台 `--capabilities` 可诊断 runtimeAvailable/runtimeVersion/runtimeError |

模型下载器必须做到：

- 只从 RecordingFreedom 验证过的 release catalog 下载，不直接下载未固定 SHA256 的上游文件。
- 下载到 staging 目录，完成 bytes/SHA256/smoke 后再替换安装目录。
- 下载进度、失败原因、重试和取消都来自后端真实状态。
- 安装新模型不自动切换 active；切换 active 必须用户确认。
- latest/quality 识别失败时可以提示回退 stable，但不能静默回退造成用户误判。

### 翻译调度颗粒

翻译不是 OCR 的一部分，只消费已经持久化的 `OcrResult`。

| 颗粒 | 交付 | 验收 |
| --- | --- | --- |
| provider 设置 | provider/base URL/API key/model/source language/target language/隐私确认 | provider 为 `disabled` 或未确认隐私时，不发任何请求 |
| provider 调用 | `deepl` 和 `openai-compatible` 走同一 `TranslateOcr` | provider mock 单测覆盖成功、失败、超时、段数不匹配 |
| 分段合同 | block id、顺序、数量必须一致 | 错位结果失败，不覆盖 OCR 原文 |
| 缓存 | 同一 result/provider/model/language/block 原文命中缓存 | 第二次同请求不访问 provider |
| UI 消费 | 结果浮层、截图历史、钉图、画板复用同一调用 | 译文可复制，可插入画板；未配置时只显示保护提示 |
| 密钥安全 | API key 存 secret store，不写 settings/log | macOS/Linux/Windows 实机写入读取删除回验 |

### 执行顺序

后续开发按这个顺序推进，不能跳过前置真实证据：

1. R1 截图入口真实模型回验：先用 stable worker 跑通区域、全屏、窗口、焦点窗口、滚动最终图，证明保存和 OCR 互不阻塞。
2. R2 OCR 结果浮层桌面回验：真实 result 打开 floating panel，图片、polygon、block 列表、复制、高亮都正确。
3. R3 钉图 OCR 桌面回验：钉图识别、显示/隐藏高亮、resize 对齐、关闭重开恢复。
4. R4 画板选中图片 OCR 收尾：选中图片导出、`whiteboard-selection`、独立 `imagePath`、结果坐标一致。
5. R5 画板 OCR 高亮/文本插入：block 元素、原文/译文文本元素、保存重开不丢。
6. R6 录制中画板安全：OCR 与录制并行，失败不影响录制包、鼠标和绘制。
7. R7 滚动长图真实样例：真实长图 tile OCR、去重阈值、无滚动回退。
8. R8 翻译 provider 实机回验：真实 provider 或可控本地 OpenAI-compatible endpoint、缓存、隐私保护、secret store。
9. R9 模型通道下载器：stable/latest/quality catalog、下载进度、校验、切换确认、失败回退。
10. R10 发布门禁：全平台 worker/runtime/capabilities、至少 stable smoke、发布产物完整性、桌面视觉证据。

### 每粒完成后的固定动作

完成任何一个 `R` 或 `G` 子颗粒后，必须同步做完：

- 更新本文件对应状态，写清“已落地”和“仍未完成”。
- 增加或更新至少一个直接覆盖该颗粒的测试、smoke、e2e 或桌面证据。
- 运行该颗粒需要的最小验证命令；大构建和全量测试留到批量完成后跑，避免每改一行就浪费时间。
- 如果能力未 ready，UI 必须展示明确不可用原因。
- 如果发现旧能力回归，先修回归，再继续下一粒。

## 当前技术基线

参考 `mg-chao/snow-shot` 的结论：

- snow-shot 的 OCR 是本地推理，不是在线 OCR。
- 它使用 RapidOCR/PaddleOCR ONNX 模型，通过 ONNX Runtime 推理。
- 模型文件作为插件资产安装到本地，不直接提交在源码里。
- 它的翻译默认不是本地模型，识别后的文字会交给服务/API，例如有道、DeepL 或 OpenAI-compatible 接口。

外部技术依据，检查日期为 2026-07-06：

- RapidOCR 是开源 OCR 工具，主打多平台、多语言、离线部署，默认支持中英文识别：<https://github.com/RapidAI/RapidOCR>
- PaddleOCR 3.x 文档说明通用 OCR pipeline 已覆盖 PP-OCRv3、PP-OCRv4、PP-OCRv5、PP-OCRv6；RecordingFreedom 的 latest/quality 通道以 PP-OCRv6 系列作为后续升级目标：<https://github.com/PaddlePaddle/PaddleOCR/blob/main/docs/version3.x/pipeline_usage/OCR.en.md>
- ONNX Runtime 官方发布页提供跨平台运行库，模型保持平台无关，运行库按系统和架构分发：<https://github.com/microsoft/onnxruntime/releases>
- ONNX Runtime 发布节奏和版本策略见官方 release/servicing 文档：<https://onnxruntime.ai/docs/reference/releases-servicing.html>

本轮技术校准：

- PaddleOCR 3.x 的通用 OCR 主线已经进入 PP-OCRv6，因此 RecordingFreedom 的“latest/quality”目标不再继续追 PP-OCRv4/PP-OCRv5 新包，而是只把它们作为兼容或 stable fallback。
- 上游模型名称和 RecordingFreedom 内部 `modelId` 可以不同；用户可见名称必须跟随上游模型族，内部 `ppocrv6-mobile-zh-en` 可以作为兼容 ID 保留，但 release catalog 要写清实际 det/cls/rec 来源、版本、license、SHA256 和 smoke 样例。
- ONNX 模型包不按平台拆分，`rf-ocr-worker` 和 ONNX Runtime 动态库按 `windows/darwin/linux` 与 `amd64/arm64` 拆分。模型可用不等于平台可用；平台可用必须以 `--capabilities` 和真实 smoke 为准。
- 如果上游出现更新的 OCR/VL/结构化文档模型，不能直接替换截图 OCR 默认模型；只能作为 `quality` 或后续 `document` 通道，经过 package、manifest、smoke、真实样例、性能和隐私验收后再开放。

本地参考 snow-shot 的实际源码结论：

- `src-tauri/src-crates/tauri-commands/ocr` 使用 `paddle-ocr-rs` 做本地 OCR，不是隐藏云 OCR。
- 前端通过 `ocr_init`、`ocr_detect`、`ocr_detect_with_shared_buffer`、`ocr_release` 调用 OCR，支持 hot start、模型写入内存、角度检测等运行策略。
- 翻译在前端配置中走 Youdao、DeepL、OpenAI-compatible 等 provider，不属于 OCR 模型本身。
- RecordingFreedom 可以吸收“本地 OCR + 可配置翻译”的优点，但不照搬 snow-shot 的 Tauri/Rust 插件结构；本项目继续使用 Go + Wails 主进程、独立 `rf-ocr-worker`、统一缓存和 floating panel UI。

RecordingFreedom 的最新 OCR 采用口径：

- `stable`：第一版稳定默认仍使用已经打通 smoke 的 `ppocrv5-mobile-zh-en`，保证中英文截图 OCR 可离线验收。
- `latest`：优先引入 PP-OCRv6 mobile/small 中英文 ONNX 模型包。只有经过 RecordingFreedom 的 SHA256、manifest、worker smoke、真实截图样例和跨平台 runtime 验证后，才进入可安装列表。
- `quality`：引入 PP-OCRv6 medium/server 类更大模型包，给用户主动安装，不随安装包默认携带，避免体积暴涨。
- `runtime`：当前仓库锁定 ONNX Runtime CPU `1.23.2`，原因写入 `third_party/onnxruntime/manifest.json`：该版本同时覆盖 Windows x64/ARM64、macOS x64/ARM64、Linux x64/ARM64。后续升级 runtime 必须先证明六个平台 archive 齐全，不能为了“最新版本号”牺牲 macOS x64 或 Windows ARM64。
- `worker`：只要 JSON Lines 协议、capabilities、错误码和 smoke 标准不变，worker 内部实现可以从 Go 动态 ONNX Runtime 继续演进，也可以在证据充分时替换为 Rust/其他实现；替换必须跨平台构建和 smoke 通过。

## 硬性决策

- OCR 必须是本地模型优先，不允许隐藏在线 OCR。
- OCR 模型文件不按平台拆分，同一套 `.onnx` 可用于 Windows、macOS、Linux。
- OCR runtime 和 worker 必须按平台/架构拆分。
- OCR 不直接嵌进 Wails/Go 主进程，采用独立 native worker。
- OCR worker 崩溃不能导致 RecordingFreedom 崩溃。
- OCR 结果必须缓存，同一张图片、同一模型、同一语言不重复识别。
- 翻译默认关闭，用户没有配置翻译提供方时不发送任何识别文本。
- 翻译结果是 OCR 结果的附加数据，不覆盖原始 OCR。
- 所有 OCR/翻译 UI 必须走 `16-floating-panel-window-plan.md` 的 floating panel / floating select，不允许扩大主胶囊。
- 第一版只承诺中英文 OCR；更多语言通过模型包扩展。

## 和现有功能的边界

`15-smart-screenshot-region-assist.md` 解决的是截图/录制区域的智能选择，属于“选哪里”。

本方案解决的是图片保存后的文字识别和翻译，属于“图片里有什么文字”。

两者关系：

- 区域选择先完成，保存 PNG。
- OCR 只识别保存后的 PNG 或画板导出的 PNG。
- OCR 不参与区域识别框选，不改变 snow-shot-style 区域辅助逻辑。
- OCR 坐标以图片像素坐标保存，显示时再映射到截图历史、钉图窗口或画板坐标。

## 总体架构

```text
React UI
  screenshot menu / history / pin window / whiteboard
      |
      v
Wails binding
      |
      v
Go OCR Service
  model registry
  worker lifecycle
  job queue
  cache
  screenshot/whiteboard integration
  translation dispatch
      |
      v
rf-ocr-worker
  ONNX Runtime loader
  image preprocessing
  detection/classification/recognition
  postprocess
      |
      v
OCR model pack
  det.onnx
  cls.onnx
  rec.onnx
  keys.txt
```

主进程只做调度。真正的模型加载和推理在 worker 里完成。

## 模型策略

模型包使用 registry 管理，不硬编码到业务逻辑里。

```text
data/
  models/
    ocr/
      registry.json
      ppocrv5-mobile-zh-en/
        manifest.json
        det.onnx
        cls.onnx
        rec.onnx
        keys.txt
        smoke.png
        smoke.expected.json
      ppocrv6-mobile-zh-en/
        manifest.json
        det.onnx
        cls.onnx
        rec.onnx
        keys.txt
        smoke.png
        smoke.expected.json
      ppocrv6-medium-zh-en/
        manifest.json
        det.onnx
        cls.onnx
        rec.onnx
        keys.txt
        smoke.png
        smoke.expected.json
```

模型通道：

- `stable`：默认稳定通道，第一版建议使用 PP-OCRv5 mobile 中英文 ONNX 或经过充分测试的 PP-OCRv6 mobile。
- `latest`：最新通道，使用 PP-OCRv6 mobile/small/medium，必须通过 RecordingFreedom 的 smoke 和真实截图样例验收后才能展示给用户。
- `quality`：质量优先通道，使用更大的 PP-OCRv6 medium/server 类模型，适合用户愿意接受更大体积和更慢速度的场景。

模型切换规则：

- 安装新模型不自动替换当前模型。
- 切换模型必须由用户在设置中主动选择。
- 新模型安装后必须先校验 SHA256，再运行内置 smoke 图片。
- smoke 失败的模型不能进入可选列表。
- 当前模型识别失败时，可以提示用户切换 stable 模型，但不能静默切换。

模型 manifest 示例：

```json
{
  "schemaVersion": 1,
  "id": "ppocrv6-mobile-zh-en",
  "name": "PP-OCRv6 Mobile Chinese/English",
  "channel": "latest",
  "engine": "onnxruntime",
  "language": ["zh", "en"],
  "version": "2026.07",
  "files": [
    {"name": "det.onnx", "sha256": "...", "bytes": 0},
    {"name": "cls.onnx", "sha256": "...", "bytes": 0},
    {"name": "rec.onnx", "sha256": "...", "bytes": 0},
    {"name": "keys.txt", "sha256": "...", "bytes": 0}
  ],
  "smoke": {
    "image": "smoke.png",
    "mustContain": ["RecordingFreedom", "文字识别"],
    "maxDurationMs": 3000
  }
}
```

## Worker 方案

Worker 名称：

```text
rf-ocr-worker
```

实现口径：

- 产品合同绑定 `rf-ocr-worker` JSON Lines 协议，不绑定某一种 worker 编程语言。
- 当前仓库已落地 Go 版协议外壳，用来固定主进程调度、打包路径、capabilities/preflight、日志和错误合同。
- G6 优先沿用当前 Go worker，在独立 worker 进程内动态加载 ONNX Runtime C API；主 Wails 进程仍不直接 import ONNX Runtime。
- 真正 OCR 推理必须在 `rf-ocr-worker` 内完成。只要可执行文件名、协议、错误码、capabilities 字段保持兼容，内部实现后续可替换；但替换必须有跨平台构建和 smoke 证据。
- 任何 worker 在没有真实 ONNX 推理能力前都必须返回明确的不可用状态，不能输出假 OCR 文本。

推荐实现：

- 第一选择：现有 Go `rf-ocr-worker` + 动态 ONNX Runtime C API 加载，保持 `CGO_ENABLED=0` 发布路径，避免再次引入 Windows ARM64 cgo 风险。
- 动态加载层只放在 worker 内，能力探测必须从“runtime 文件存在”升级到“runtime 可加载、API version 可用、Env/Session 生命周期可创建”。
- Rust worker + `ort` 作为 fallback 方案，仅当 Go 动态加载无法在 Windows x64/ARM64、macOS x64/ARM64、Linux x64/ARM64 上稳定通过 smoke 时启用。

这样选择的原因：

- snow-shot 已验证本地 ONNX Runtime + PaddleOCR/RapidOCR 模型的桌面 OCR 路径可行。
- Worker 是独立可执行文件，不破坏 Go + React + Wails 主技术栈。
- Windows ARM64 的 cgo 路径曾在 RNNoise 场景出现过构建限制，OCR worker 独立后更容易分平台处理。
- 如果 runtime 加载失败，只影响 OCR，不影响录制、截图、画板。

Worker 目录：

```text
tools/
  ocr-worker/
    cmd/rf-ocr-worker/
    internal/protocol/
    internal/runtime/
    internal/pipeline/
    internal/postprocess/
    internal/smoke/
```

平台产物：

```text
tools/
  ocr-worker/
    windows-amd64/rf-ocr-worker.exe
    windows-arm64/rf-ocr-worker.exe
    darwin-amd64/rf-ocr-worker
    darwin-arm64/rf-ocr-worker
    linux-amd64/rf-ocr-worker
    linux-arm64/rf-ocr-worker
  onnxruntime/
    windows-amd64/onnxruntime.dll
    windows-arm64/onnxruntime.dll
    darwin-amd64/libonnxruntime.dylib
    darwin-arm64/libonnxruntime.dylib
    linux-amd64/libonnxruntime.so
    linux-arm64/libonnxruntime.so
```

Worker 协议使用 stdin/stdout 的 JSON Lines。不要用临时 HTTP 服务，避免端口占用、防火墙和退出清理问题。

初始化：

```json
{"id":"job-1","method":"init","params":{"modelDir":"...","runtimeDir":"...","threads":4,"language":"zh-en"}}
```

识别：

```json
{"id":"job-2","method":"recognize","params":{"imagePath":"...","detectAngle":true,"maxSide":2400}}
```

释放：

```json
{"id":"job-3","method":"release"}
```

结果：

```json
{
  "id": "job-2",
  "ok": true,
  "result": {
    "modelId": "ppocrv6-mobile-zh-en",
    "imageSha256": "...",
    "width": 1280,
    "height": 720,
    "blocks": [
      {
        "id": "b1",
        "text": "RecordingFreedom",
        "confidence": 0.97,
        "box": [[10,20],[180,20],[180,52],[10,52]],
        "lineIndex": 0,
        "languageHint": "en"
      }
    ],
    "plainText": "RecordingFreedom",
    "durationMs": 126
  }
}
```

错误：

```json
{
  "id": "job-2",
  "ok": false,
  "error": {
    "code": "model_not_loaded",
    "message": "OCR model is not loaded",
    "recoverable": true
  }
}
```

## Go 服务合同

新增包：

```text
app/internal/ocr/
  service.go
  models.go
  registry.go
  worker.go
  jobs.go
  cache.go
  translation.go
  history_integration.go
  whiteboard_integration.go
```

Wails 方法：

- `ListOcrModels()`
- `InstallOcrModel(modelId string)`
- `RemoveOcrModel(modelId string)`
- `SetActiveOcrModel(modelId string)`
- `GetOcrStatus()`
- `RecognizeImage(request OcrRecognizeRequest)`
- `RecognizeScreenshot(itemId string)`
- `RecognizePinnedScreenshot(itemId string)`
- `RecognizeWhiteboard(request WhiteboardOcrRequest)`
- `TranslateOcr(request OcrTranslateRequest)`
- `CancelOcrJob(jobId string)`
- `OpenOcrResult(resultId string)`

核心类型：

```go
type OcrSourceKind string

const (
    OcrSourceRegionScreenshot OcrSourceKind = "region-screenshot"
    OcrSourceFullScreenshot OcrSourceKind = "full-screenshot"
    OcrSourceWindowScreenshot OcrSourceKind = "window-screenshot"
    OcrSourceFocusedWindowScreenshot OcrSourceKind = "focused-window-screenshot"
    OcrSourceScrollingScreenshot OcrSourceKind = "scrolling-screenshot"
    OcrSourcePinnedScreenshot OcrSourceKind = "pinned-screenshot"
    OcrSourceWhiteboard OcrSourceKind = "whiteboard"
    OcrSourceWhiteboardSelection OcrSourceKind = "whiteboard-selection"
)

type OcrRecognizeRequest struct {
    ImagePath string `json:"imagePath"`
    SourceKind OcrSourceKind `json:"sourceKind"`
    SourceID string `json:"sourceId"`
    Language string `json:"language"`
    ModelID string `json:"modelId,omitempty"`
    Force bool `json:"force"`
    Priority string `json:"priority"` // interactive, normal, background
}

type OcrBlock struct {
    ID string `json:"id"`
    Text string `json:"text"`
    Confidence float64 `json:"confidence"`
    Box [][2]float64 `json:"box"`
    LineIndex int `json:"lineIndex"`
    LanguageHint string `json:"languageHint,omitempty"`
}

type OcrResult struct {
    ID string `json:"id"`
    SourceKind OcrSourceKind `json:"sourceKind"`
    SourceID string `json:"sourceId"`
    ImagePath string `json:"imagePath"`
    ImageSHA256 string `json:"imageSha256"`
    ModelID string `json:"modelId"`
    Language string `json:"language"`
    Width int `json:"width"`
    Height int `json:"height"`
    Blocks []OcrBlock `json:"blocks"`
    PlainText string `json:"plainText"`
    CreatedAt time.Time `json:"createdAt"`
    DurationMS int `json:"durationMs"`
}
```

事件：

- `ocr.status.changed`
- `ocr.model.installed`
- `ocr.model.failed`
- `ocr.job.queued`
- `ocr.job.started`
- `ocr.job.finished`
- `ocr.job.failed`
- `ocr.job.cancelled`
- `ocr.translation.finished`
- `screenshot.history.changed`
- `whiteboard.ocr.changed`

## 调度规则

队列优先级：

- `interactive`：用户主动点击 OCR、翻译、复制前识别，优先执行。
- `normal`：截图保存后自动识别。
- `background`：批量补识别历史图片。

并发规则：

- 第一版只启动一个 OCR worker。
- 同一时间只跑一个 OCR 推理任务。
- 允许多个请求排队。
- 相同 `imageSha256 + modelId + language` 的任务合并，不重复识别。
- 用户删除截图、关闭画板、取消 OCR 面板时，未开始任务可以取消。
- 已进入 ONNX 推理的任务不强杀 worker，标记取消后丢弃结果。

Worker 生命周期：

- App 启动后不立即加载大模型。
- 首次 OCR 请求或用户打开 OCR 设置页时启动 worker。
- 可选设置：空闲预热模型。
- worker 空闲超过配置时间可以释放，默认不频繁释放，避免反复加载造成卡顿。
- worker 崩溃后最多自动重启 1 次；再次失败进入 `failed` 状态并显示诊断。

日志：

- OCR 服务日志写入 `logs/ocr.jsonl`。
- Worker stderr 写入 `logs/ocr-worker.jsonl`。
- 每个任务记录 `jobId`、`sourceKind`、`imageSha256`、`modelId`、`durationMs`、`errorCode`。

## 存储和缓存

OCR 结果是 sidecar 数据，不写进 PNG 本体。

```text
data/
  screenshots/
    history.json
    2026-07/
      screenshot-id.png
      screenshot-id.thumb.png
  ocr/
    cache/
      image-sha256.model-id.language.json
    results/
      ocr-result-id.json
    translations/
      ocr-result-id.provider.target-lang.json
```

截图历史新增字段：

```json
{
  "ocrStatus": "none",
  "ocrResultId": "",
  "ocrModelId": "",
  "ocrLanguage": "zh-en",
  "ocrUpdatedAt": "",
  "ocrError": ""
}
```

状态定义：

- `none`：未识别。
- `queued`：已排队。
- `running`：识别中。
- `ready`：识别完成。
- `failed`：识别失败。

缓存 key：

```text
sha256(image bytes) + modelId + language + preprocessingVersion
```

只要图片内容一致，区域截图、钉图、画板背景可以复用同一份 OCR 结果。

## 图片预处理

第一版 worker 需要内置稳定预处理：

- 读取 PNG/JPEG/WebP。
- 按最长边限制缩放，默认 `maxSide=2400`，保留原图坐标映射。
- 对透明图片先合成到白底，避免透明画板导出导致文字边缘识别差。
- 保留原始宽高和缩放比例，用于结果坐标还原。
- 对滚动长图可切成纵向 tiles 识别，最后合并坐标。

滚动截图长图规则：

- 长边超过 `maxSide` 时不能简单整体缩小到太小。
- 按 1600-2400px 高度切片，重叠 80-120px。
- 合并时去除重复 OCR block。
- block 去重按文本相似度、位置重叠、置信度处理。

## 各入口落地流程

### 区域截图

流程：

1. 用户选择区域。
2. 截图保存 PNG。
3. 写入截图历史。
4. 如果开启自动 OCR，提交 `sourceKind=region-screenshot`。
5. OCR 完成后更新历史项。
6. 历史面板显示复制、翻译、打开结果。

验收：

- 区域截图保存后不会等待 OCR 才返回。
- OCR 失败不影响截图历史保存。
- 同一张图再次点击 OCR 直接读缓存。

### 全屏截图

流程：

1. 捕获当前屏幕或全部屏幕合成图。
2. 保存 PNG。
3. 提交 `sourceKind=full-screenshot`。
4. OCR 坐标使用最终 PNG 坐标，不使用显示器物理坐标。

验收：

- 多屏合成图能识别。
- OCR block 坐标在打开历史图和钉图时位置一致。

### 窗口截图和焦点窗口截图

流程：

1. 使用现有窗口/焦点窗口截图能力保存 PNG。
2. 提交 `sourceKind=window-screenshot` 或 `focused-window-screenshot`。
3. OCR 结果与窗口截图历史绑定。

验收：

- 不依赖当前窗口是否还存在。
- 截图保存后的静态图片可重复识别。

### 滚动截图

流程：

1. 用户选择滚动区域。
2. App 执行滚动捕获和拼接。
3. 最终 PNG 保存后再启动 OCR。
4. 如果未滚动，保存普通区域截图并按 `region-screenshot` OCR。

验收：

- 不对中间帧做 OCR。
- 不保存假长图。
- 长图切片 OCR 后坐标能映射回完整长图。

### 截图历史

功能：

- 每个历史项显示 OCR 状态。
- 未识别：显示“识别文字”。
- 识别中：显示进度或加载态。
- 已识别：显示“复制文字”“翻译”“查看结果”。
- 失败：显示失败原因和“重试”。

验收：

- 历史项不因为 OCR 字段缺失崩溃。
- 旧历史文件能兼容读取。
- 删除截图时同时清理关联 OCR result 索引，但缓存文件可以延迟清理。

### 钉图/固定图片

功能：

- 钉图窗口显示 OCR block 高亮开关。
- 点击 OCR block 可复制原文。
- 配置翻译后可显示译文。

验收：

- 钉图窗口不修改原始图片。
- 高亮坐标跟随钉图缩放。
- 固定图片重开后能恢复 OCR 状态。

### 画板

识别对象：

- 截图作为背景进入画板时，识别背景图。
- 用户选中图片元素时，识别选中图片。
- 用户保存画板快照后，可识别导出的整张画板 PNG。

画板动作：

- 识别文字。
- 复制识别结果。
- 翻译识别结果。
- 插入译文为文本元素。
- 在原文字附近叠加译文文本元素。
- 显示/隐藏 OCR block 边框。

验收：

- OCR 结果绑定图片 hash，不只绑定 Excalidraw element id。
- 画板保存不会覆盖 OCR 结果。
- 录制中打开画板时，OCR 任务只走后台队列，不能阻塞录制。

## 翻译方案

翻译不是 OCR 模型的一部分。

第一版翻译提供方：

- `disabled`：默认。
- `deepl`：用户配置 API URL 和 key。
- `openai-compatible`：用户配置 base URL、API key、model。
- `custom-http`：后续给私有网关使用。
- `local`：保留，不进入第一版。

隐私规则：

- OCR 本地执行。
- 翻译会把识别文字发送给用户选择的提供方。
- 第一次启用翻译时必须明确提示。
- 没有配置提供方时，点击翻译只引导配置，不发请求。

翻译请求：

```go
type OcrTranslateRequest struct {
    OcrResultID string `json:"ocrResultId"`
    BlockIDs []string `json:"blockIds,omitempty"`
    Provider string `json:"provider"`
    SourceLanguage string `json:"sourceLanguage"`
    TargetLanguage string `json:"targetLanguage"`
}
```

翻译结果：

```json
{
  "ocrResultId": "ocr_...",
  "provider": "openai-compatible",
  "sourceLanguage": "auto",
  "targetLanguage": "zh-CN",
  "blocks": [
    {"blockId": "b1", "source": "RecordingFreedom", "translated": "RecordingFreedom"}
  ],
  "createdAt": "..."
}
```

翻译缓存 key：

```text
ocrResultId + provider + sourceLanguage + targetLanguage + promptVersion
```

翻译调度颗粒：

- G13 之前，所有翻译按钮只允许显示“未配置 provider / 翻译不可用”，不能把 OCR 原文发送出去。
- G13 第一版只实现 `deepl` 和 `openai-compatible`，因为两者都可以由用户明确配置 endpoint、key 和目标语言。
- `custom-http` 只保留合同，等用户有私有翻译网关时再做，不提前做不可验证的通用表单。
- `local` 翻译不进入第一版。原因是离线翻译模型体积、语言质量、GPU/CPU 性能和许可证都需要单独评估，不能混进 OCR 第一版。
- 翻译任务必须引用 `ocrResultId` 和 block id；OCR 结果删除后，翻译缓存应进入孤儿清理队列。
- OCR 原文和译文在 UI 中必须分层展示：用户能复制原文、复制译文、插入画板文本；译文不能覆盖 `plainText` 或 block 原始文本。

## UI 方案

设置页增加：

- OCR 总开关。
- 自动识别截图开关。
- OCR 模型选择。
- OCR 模型通道：稳定、最新、质量优先。
- 模型下载、校验、删除。
- OCR 语言：第一版中英文。
- 翻译总开关。
- 翻译提供方选择。
- 翻译目标语言。
- 翻译隐私提示。

截图/画板二级菜单增加：

- 区域截图。
- 全屏截图。
- 窗口截图。
- 焦点窗口截图。
- 滚动截图。
- 画板。
- 截图历史。

截图历史项增加：

- OCR 状态徽标。
- 识别文字按钮。
- 复制文字按钮。
- 翻译按钮。
- 查看 OCR 结果按钮。

OCR 结果浮层：

- 左侧图片预览。
- 右侧文字块列表。
- block 支持悬停高亮图片区域。
- 支持复制全部、复制选中、翻译全部、翻译选中。
- 翻译结果可以复制，也可以插入画板。

布局要求：

- 所有 OCR 相关面板使用 floating panel。
- 二级选择使用 floating select。
- 不改变主胶囊宽高。
- 打开、关闭 OCR 面板不能造成胶囊闪烁。

## 打包和发布

第一版离线可用优先：

- 随安装包带一个 stable 中英文 OCR 模型。
- 随安装包带当前平台 ONNX Runtime。
- 随安装包带当前平台 `rf-ocr-worker`。
- latest/quality 模型通过后续下载获取。

模型下载源：

- 优先从 RecordingFreedom GitHub Releases 下载我们验证过的模型包。
- 不直接在运行时从未校验的第三方地址下载。
- 每个模型包必须有 manifest、SHA256、大小、来源说明、许可证说明。

发布产物检查：

- Windows x64/ARM64 包含对应 worker 和 runtime。
- macOS x64/ARM64 包含对应 worker 和 runtime。
- Linux x64/ARM64 包含对应 worker 和 runtime。
- 默认 stable 模型存在或首启下载器可用。

## CI 验证

新增 CI 任务：

- 构建 `rf-ocr-worker` 全平台产物。
- 校验 ONNX Runtime 文件存在。
- 校验模型 manifest 和 SHA256。
- 使用固定中英文 smoke 图片跑 OCR。
- Go 单测覆盖 registry、cache、queue、history integration。
- 前端测试覆盖截图历史 OCR 状态和面板交互。
- 打包测试确认 worker/runtime/model 进入发布产物。

必跑命令：

```powershell
git diff --check
go test ./...
cd app/frontend
npm run build
npm run test:e2e -- --grep "ocr|screenshot|whiteboard"
```

本地真实 OCR smoke：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\run-local-ocr-smoke.ps1
```

该脚本会在 Windows 本机按当前 `GOARCH` 构建 `rf-ocr-worker.exe`，准备 ONNX Runtime，按 `third_party/ocr-models/manifest.json` 下载并校验 stable 模型包，解包后执行 `rf-ocr-worker --smoke`。通过标准是 `ok=true`，且 `plainText` 同时包含 `RecordingFreedom` 和 `文字识别`。

脚本还会设置 `RF_OCR_SCREENSHOT_SMOKE=1` 运行 `TestScreenshotOCRRealWorkerSmoke`。该测试把模型包内的真实 smoke 图片保存为多种截图历史模式，再走 `QueueRecognizeScreenshot -> OCR queue -> real worker -> screenshot sidecar -> OpenOcrResult -> ReadOcrResultImage`，并在禁用 worker 后验证同图缓存命中仍会为当前截图来源写入独立 result。

OCR smoke 标准：

- smoke 图片包含英文 `RecordingFreedom`。
- smoke 图片包含中文 `文字识别`。
- OCR 结果必须包含这两段文本。
- 首次冷启动不超过 5 秒。
- 模型已加载后的普通区域图识别不超过 1 秒作为目标，不作为第一版硬失败门槛。

## 开发阶段和执行颗粒度

下面的颗粒度是后续开发的唯一拆分口径。每一粒必须能独立验证，完成后更新本文件的“当前已落地颗粒”和“尚未完成”，不能用“UI 有按钮”“协议能跑通”替代真实能力。

### G0 计划和合同冻结

范围：

- 固定 OCR/翻译调度目标、入口、隐私边界和完成定义。
- 固定 `rf-ocr-worker` 协议、模型 manifest、缓存 key、Wails API、事件名。
- 固定不扩大主胶囊、不引入隐藏联网 OCR、不输出假 OCR。

交付：

- `docs/17-ocr-translation-dispatch-plan.md`。
- `docs/README.md` 文档入口。

验收：

- 文档明确覆盖区域截图、全屏截图、滚动截图、截图历史、钉图/固定图片、画板、录制中画板。
- 文档明确翻译默认关闭，只有用户配置 provider 后才联网。
- 文档明确真实 OCR 未接入前不能宣称 OCR 已完成。

状态：已落地。

### G1 Go 主服务和模型注册表

范围：

- 新增 `app/internal/ocr`。
- 建立模型 registry、active model、模型安装状态、manifest 校验。
- 建立 Wails API 和前端 backend wrapper。

交付：

- `ListOcrModels`、`GetOcrStatus`、`SetActiveOcrModel`、`InstallOcrModelPackage`、`RemoveOcrModel`。
- `ModelInfo` 能展示安装状态、来源、许可证、smoke 资产状态。

验收：

- 未安装模型时状态是 `no-model`。
- 模型文件缺失或 SHA256 不匹配时不能进入可用状态。
- 导入 zip/目录不会绕过 registry。
- 导入新模型不会自动切换当前 active model。

状态：已落地。

### G2 模型包安全导入和 smoke 资产校验

范围：

- 支持本地目录和 `.zip` 模型包。
- 使用 staging 目录导入，验证成功后原子替换。
- 校验 `smoke.image` 和 `smoke.expected.json`。

交付：

- 拒绝 zip-slip、绝对路径、`..` 路径、symlink。
- 失败替换不破坏原已安装模型。

验收：

- 危险 zip 会被拒绝。
- 缺失 manifest/模型文件/smoke expected 会返回明确错误。
- 安装包 manifest 元数据会回显到 UI 可用的 `ModelInfo`。

状态：已落地。

### G3 Worker 协议壳和发布资产路径

范围：

- 新增 `rf-ocr-worker` 可执行文件。
- 实现 `--capabilities`、`init`、`recognize`、`release` 协议壳。
- 主程序可以按当前平台查找 `tools/ocr-worker/<goos-goarch>/rf-ocr-worker`。
- CI/release 能构建并把 worker 放进发布产物。

交付：

- worker capabilities 能报告 `runtimeDir`、`runtimeLibrary`、`runtimeAvailable`、`runtimeError`。
- worker capabilities 能报告 `runtimeVersion` 和 `runtimeApiVersion`；`runtimeAvailable=true` 必须代表动态库可加载且 Env/SessionOptions 生命周期探测通过，不再只代表文件存在。
- Windows/macOS/Linux 发布验证脚本检查 worker 文件和 `--capabilities`。

验收：

- worker 缺失时状态是 `worker-absent`。
- worker 存在但不支持识别时状态是 `worker-unavailable`。
- 未接真实 ONNX 推理前 `supportsRecognize` 必须为 `false`。
- 本阶段不允许生成任何假 OCR 文本。

状态：已落地。

### G4 ONNX Runtime 获取、校验和随包发布

范围：

- 固定 ONNX Runtime 版本。
- 为 Windows x64/ARM64、macOS x64/ARM64、Linux x64/ARM64 准备 CPU runtime。
- 下载、解压、SHA256 校验、许可证/NOTICE 记录。
- 发布产物包含 `tools/onnxruntime/<goos-goarch>/`。

交付：

- `scripts/fetch-onnxruntime.*` 或等价平台脚本。
- `third_party/onnxruntime/manifest.json` 记录版本、下载 URL、SHA256、license。
- release workflow 调用 runtime 准备步骤。
- 验证脚本检查 runtime 动态库存在。
- 验证脚本通过 `rf-ocr-worker --capabilities --runtime-dir` 证明 worker 能看到随包 runtime。

验收：

- 断网安装包仍能启动 OCR preflight。
- 缺少 runtime 时显示 `onnx_runtime_missing`。
- runtime 架构不匹配时显示明确诊断，不让主程序崩溃。

状态：已落地 runtime 获取、校验、随包分发和 capabilities preflight；真实 ONNX 推理仍属于 G6。

### G5 Stable OCR 模型包和模型发布通道

范围：

- 固定第一版 stable 中英文模型包。
- 准备 PP-OCRv5/PP-OCRv6 mobile 中英文 ONNX 模型、keys、manifest、smoke 图片、expected JSON。
- 建立模型包发布和校验流程。

交付：

- `ppocrv5-mobile-zh-en` 或经验证的 `ppocrv6-mobile-zh-en` stable 包。
- `latest`、`quality` 通道只进入可安装列表，不替换 stable。

验收：

- 模型包可以从 release 下载或本地导入。
- SHA256、bytes、license/source 全部可验证。
- smoke 资产存在，但真实 smoke pass 要等 G6 worker 推理完成。

状态：已落地 stable 模型包 catalog、下载校验打包器和 release artifact 通道；app 内模型下载/更新 UI 仍属于后续 G14。

### G6 Native OCR 推理 worker

范围：

- 在 worker 内接入 ONNX Runtime 动态加载，不把 ONNX Runtime 绑定进 Wails 主进程。
- 实现图片读取、透明图白底合成、缩放、det/cls/rec 推理、postprocess、坐标还原。
- 支持 PNG/JPEG/WebP 输入。

施工颗粒：

- G6.1 Runtime loader：worker 能加载随包 `onnxruntime` 动态库，调用 `OrtGetApiBase`，读取 runtime version，创建/释放 Env 和 SessionOptions。
- G6.2 Model session：加载 `det.onnx`、`cls.onnx`、`rec.onnx` 三个 session，记录 input/output name、shape、dtype，不成功时返回 `model_session_failed`。
- G6.3 Image preprocess：统一 PNG/JPEG/WebP 解码、透明图白底合成、最长边限制、归一化、CHW tensor、原图坐标映射。
- G6.4 Det postprocess：输出文本候选框，完成阈值过滤、膨胀/裁剪、按阅读顺序排序。
- G6.5 Cls angle：对候选框做方向分类，按需旋转裁剪图。
- G6.6 Rec decode：按 `keys.txt` 做 CTC decode，生成每个 block 的 `text` 和 `confidence`。
- G6.7 Result mapping：把 block 坐标还原为原图像素坐标，生成稳定 `plainText`。
- G6.8 Smoke CLI：固定 smoke 图片必须识别 `RecordingFreedom` 和 `文字识别`，CI/release 都能调用。
- G6.9 Failure contract：任何失败都返回稳定错误码，不写缓存、不更新历史为 ready。

交付：

- `recognize` 返回真实 `OcrResult`，包含 `blocks`、`plainText`、`confidence`、图片宽高和耗时。
- `--smoke` 或等价 smoke 命令可以用固定图片验证中英文。
- `--capabilities` 在 runtime 可加载且 session 生命周期通过后报告可诊断字段；只有 G6.8 真实 smoke 通过后才允许 `supportsRecognize: true`。

验收：

- smoke 图片必须识别出 `RecordingFreedom` 和 `文字识别`。
- worker 崩溃只影响 OCR，不影响主程序。
- OCR 失败返回稳定错误码，不写入假结果。
- Windows/macOS/Linux 目标架构至少各通过一次 worker smoke。
- G6 完成前，前端只能显示 OCR 不可用或待安装/待模型状态，不能给用户展示伪造识别结果。

状态：已落地第一版。G6.1 Runtime loader 已完成：worker 已能动态加载随包 ONNX Runtime，调用 `OrtGetApiBase`，读取 runtime version，并创建/释放 Env 与 SessionOptions。G6.2 Model session 已完成：`init` 会加载 `det.onnx`、`cls.onnx`、`rec.onnx` 三个 ONNX session，并返回 input/output name、dtype 和 shape 摘要；加载失败返回 `model_session_failed`。G6.3 Image preprocess 已完成：`recognize` 会先读取 PNG/JPEG/WebP，透明像素合成白底，按最长边和 32 倍数缩放，并生成归一化 CHW tensor 与坐标映射摘要。G6.4 Det postprocess 已完成第一版：worker 会执行真实 det ONNX 推理，读取 score map，按阈值生成连通域候选框，外扩、按阅读顺序排序，并映射回原图坐标。G6.5-G6.7 已完成第一版：候选框会进入 cls 方向分类、rec 文字识别、CTC decode，并组装为 `ocr.Result` 的 `blocks` 和 `plainText`。G6.8 Smoke CLI 和默认 smoke 资产已完成：`rf-ocr-worker --smoke` 可以使用模型包内 `smoke.png` 识别 `RecordingFreedom` 和 `文字识别`，CI/release 都会调用。G6.9 Failure contract 已完成第一版：runtime、模型、图片、推理错误都返回稳定可恢复错误，不写假结果。runtime 可加载时 `supportsRecognize=true`。

### G7 OCR 队列、缓存和截图历史 sidecar

范围：

- 识别请求进入队列。
- 按 `interactive`、`normal`、`background` 调度。
- 按 `imageSha256 + modelId + language + preprocessingVersion` 缓存。
- 更新截图历史 OCR sidecar。

交付：

- `QueueRecognizeImage`、`QueueRecognizeScreenshot`、`QueueRecognizePinnedScreenshot`、`QueueRecognizeWhiteboard`。
- `data/ocr/results/`、`data/ocr/cache/`。
- `screenshot.history.changed` 同步 OCR 状态。

验收：

- 相同图片、模型、语言只识别一次。
- 未开始任务可以取消。
- 已取消任务完成后不能把 UI 改成 ready。
- 旧截图历史兼容读取。

状态：已落地基础队列和 sidecar，并已补真实 worker smoke 的缓存回验和服务层取消防护。G6 已有真实 worker 第一版后，`TestScreenshotOCRRealWorkerSmoke` 现在不仅覆盖截图历史项写入 ready sidecar，还会在禁用 worker 后对同图执行同步 `RecognizeScreenshot` cache hit，并再次创建相同截图走 `QueueRecognizeScreenshot` 入队 cache hit；两条路径都必须为当前 `sourceKind/sourceId/imagePath` 写入独立 result。截图历史事件层新增取消守卫：同一 `jobId/sourceKind/sourceId` 收到 cancelled 后，即使后续迟到 ready/failed/running 事件到达，也不会把 sidecar 改成 ready；新的 OCR job 仍可正常覆盖旧取消状态。仍需在 G15 中补真实 worker 运行中取消的桌面事件流证据，不能只依赖服务层事件单测。

### G8 截图入口接入

范围：

- 区域截图、全屏截图、窗口截图、焦点窗口截图、滚动截图在保存 PNG 后提交 OCR。
- 截图保存不能等待 OCR。
- 未滚动的滚动截图按普通区域截图处理。

施工颗粒：

- G8.1 sourceKind mapping：把截图 `mode` 明确映射为 `region-screenshot`、`full-screenshot`、`window-screenshot`、`focused-window-screenshot`、`scrolling-screenshot`。
- G8.2 manual queue：历史项点击“识别文字”提交 `interactive` job，失败可重试。
- G8.3 auto queue：设置开启自动 OCR 后，截图保存完成再提交 `normal` job，不能阻塞保存和历史写入。
- G8.4 coordinate review：识别完成后，历史图预览必须能显示 block 叠层，坐标以最终 PNG 像素为准。
- G8.5 failure isolation：OCR 失败只更新 OCR sidecar，不影响截图历史、钉图、保存路径和剪贴板。

交付：

- 每种截图入口都能带 `sourceKind` 写入 OCR job。
- 截图历史项能显示识别状态。

验收：

- 区域截图和全屏截图 OCR 坐标能映射回历史图片。
- 多屏合成截图按最终 PNG 坐标识别。
- 滚动长图不对中间帧做 OCR，只对最终图做 OCR。

状态：部分落地。截图历史 UI 已能显示 OCR sidecar 状态，并提供“识别文字 / 查看结果 / 复制文字 / 重试”操作；操作会走 `QueueRecognizeScreenshot` 和 `OpenOcrResult`，不会阻塞截图保存。设置中已新增默认关闭的“截图后自动识别文字”开关；开启后，`saveScreenshotImage` 保存 PNG、写入历史、发出截图事件之后，会在后台按 `sourceKind` 异步提交 normal 优先级 OCR job。自动排队覆盖经统一保存路径进入历史的区域截图、全屏截图、窗口截图、焦点窗口截图、滚动截图和画板快照。R1 子颗粒已补强：手动截图 OCR、自动截图 OCR 都会在提交前写入 queued sidecar，排队失败时只写 failed sidecar 和错误原因，不影响截图历史、缩略图或保存文件；已新增模式到 sourceKind 的矩阵测试，覆盖 region/full/screen/window/focused-window/scrolling/whiteboard。真实模型端到端 UI/e2e、坐标可视化映射验收仍未完成。

### G9 OCR 结果浮层

范围：

- 新增 OCR 结果 floating panel，不回塞主胶囊。
- 图片预览、文字块列表、悬停高亮、复制全部/复制选中。
- 与截图历史、钉图、画板共用同一个结果查看器。

施工颗粒：

- G9.1 result reader：`OpenOcrResult` 只返回持久化 `OcrResult`，不触发重新识别。
- G9.2 image reader：按 `sourceKind/sourceId/imagePath` 读取截图、钉图、画板 source 图片，优先使用结果内 `imagePath`，缺失时走 source-specific reader。
- G9.3 preview overlay：浮层预览按结果图片宽高渲染，block polygon 与图片缩放同步。
- G9.4 copy actions：复制全文、复制单块、键盘聚焦复制都不依赖 OCR worker。
- G9.5 translation placeholder：G13 前翻译入口只显示未配置/不可用，不发网络请求。

交付：

- `OpenOcrResult(resultId)` 能打开结果浮层。
- 浮层内只读展示 OCR 原文和块坐标。

验收：

- 打开/关闭 OCR 浮层不改变主胶囊宽高。
- 点击空白区域关闭符合 floating panel 规则。
- 复制文字不依赖重新识别。

状态：部分落地。已新增 `ocr-result` floating panel 类型，截图历史里的 ready 结果可通过 `contextId=resultId` 打开独立浮层读取持久化 OCR result；浮层支持图片预览、OCR block 坐标叠层、block hover 高亮、原文展示、block 列表、置信度、复制全文和点击 block 复制单块文本，不扩大主胶囊。已新增 `ReadOcrResultImage(resultId)`，结果浮层优先通过 `OcrResult.imagePath` 读取受应用数据根目录保护的 PNG/JPEG/WebP；旧结果缺少 `imagePath` 时，会按截图类、钉图和整张画板 `sourceKind/sourceId` 回退读取截图历史里的受管理 PNG。`whiteboard-selection` 等选区类结果不允许用整张截图 fallback，必须保留自己的 `imagePath`，避免坐标映射错误。G13 前的翻译入口已接入结果浮层，但只显示“翻译提供方未配置”的本地提示，不调用 provider、不发送识别文字。R2 前端 e2e 已补充 `floating OCR result panel shows screenshot image blocks and translation guard`，验证独立 floating panel 可以展示图片、两个 OCR block、预览 polygon、高亮联动和翻译未配置保护。G13 后截图历史 ready 项新增紧凑翻译入口，会复用同一 `TranslateOcr` 配置检查与缓存，成功后复制整段译文；浏览器 e2e 已覆盖显式 provider 配置后的历史项翻译链路。剩余：真实桌面捕获入口视觉回验，以及 G13 的真实 provider 桌面回验。

### G10 钉图/固定图片 OCR

范围：

- 钉图窗口显示 OCR 高亮开关。
- 点击 block 可复制原文。
- OCR 坐标随钉图缩放映射。

施工颗粒：

- G10.1 pin OCR state：钉图加载时读取截图历史 OCR sidecar；没有 ready 时显示识别入口。
- G10.2 queue pinned：从钉图触发 `QueueRecognizePinnedScreenshot`，priority 使用 `interactive`。
- G10.3 highlight overlay：钉图图片上显示可开关的 OCR polygon 高亮，缩放、窗口 resize 后仍对齐。
- G10.4 block actions：点击或键盘选择 block 可以复制单块文本；工具栏可以复制全文。
- G10.5 result handoff：钉图可打开同一个 `ocr-result` floating panel，不再创建私有结果面板。
- G10.6 persistence：关闭再打开同一张钉图，OCR 状态、result id 和高亮可恢复。

交付：

- 钉图/固定图片使用图片 hash 查询已有 OCR 结果。
- 支持从钉图触发识别和查看结果。

验收：

- 不修改原始图片。
- 关闭再打开钉图后 OCR 状态可恢复。
- 高亮和图片缩放始终对齐。

状态：部分落地。钉图窗口已接入 OCR sidecar 状态读取和 `screenshot.history.changed` 刷新；工具栏可以从钉图触发 `QueueRecognizePinnedScreenshot`，后端按 `pinned-screenshot` sourceKind 和 `interactive` priority 入队，并写入截图历史 queued 状态。ready 后钉图会读取持久化 OCR result，支持打开同一个 `ocr-result` floating panel、显示/隐藏 OCR polygon 高亮、复制全文、点击或键盘选择 block 复制单块文本；钉图预览不会修改原始图片。已新增测试固定钉图识别入队 sourceKind 和历史 sidecar 状态。R3 前端 e2e 已补充 `pinned screenshot window restores OCR highlight and result floating panel after resize`，覆盖从截图历史打开钉图、恢复 ready OCR、显示 OCR 高亮、resize 后保持原图 `viewBox` 坐标、block hover/copy、翻译并复制译文，以及从钉图打开 `ocr-result` floating panel。R3 真实 worker evidence e2e 已补充 `pinned screenshot window renders real worker smoke evidence after resize`，把 `ppocrv5-mobile-zh-en` smoke 的 `900x280` 图片尺寸和两个真实 block polygon 坐标注入钉图窗口，验证钉图 resize 后仍按 OCR 图片像素坐标映射，不使用固定假 block。剩余：需要补真实桌面视觉/e2e，验证 Wails 钉图独立窗口在真实缩放/resize 下高亮始终对齐，并回验真实 provider 翻译入口。

### G11 画板 OCR

范围：

- 背景截图 OCR。
- 选中图片元素 OCR。
- 整张画板导出图片 OCR。
- 支持插入识别文字或译文为画板文本元素。

施工颗粒：

- G11.1 whiteboard source contract：定义 `whiteboard-background`、`whiteboard-selection`、`whiteboard-export` 三种内部来源，但对外仍归入 `whiteboard` / `whiteboard-selection`。
- G11.2 image export：背景、选中图片元素、整张画板都必须导出为稳定 PNG 后再识别。
- G11.3 hash binding：OCR 结果绑定图片 hash 和画板 scene id，不只绑定 Excalidraw element id。
- G11.4 overlay mapping：OCR block 可以显示在画板坐标系上，画板缩放/平移后仍对齐。
- G11.5 insert text：支持把识别原文或 G13 后的译文插入为画板文本元素。
- G11.6 recording safety：录制中打开画板时，OCR 只能走 background/interactive 队列，不阻塞录制、鼠标和绘制。

交付：

- `RecognizeWhiteboard` 支持背景、选区、导出快照三类 source。
- OCR block 可以在画板上显示/隐藏边框。

验收：

- OCR 结果绑定图片 hash，不只绑定 Excalidraw element id。
- 录制中打开画板时 OCR 走后台队列，不阻塞录制和绘制。
- 画板保存不会覆盖 OCR 结果。

状态：部分落地。后端 `whiteboardOCRRequest` 已区分整张画板 `whiteboard` 和选中图片元素 `whiteboard-selection` sourceKind；`QueueRecognizeWhiteboard` / `RecognizeWhiteboard` 会在 sourceId 对应截图历史项时回写 OCR sidecar，`handleOCRJobEvent` 已把 `whiteboard` 和 `whiteboard-selection` 都纳入历史状态同步，但 `whiteboard-selection` 仍不允许缺少自己的 `imagePath` 后 fallback 到整张画板图片。白板窗口已新增 OCR 操作：先导出当前画板为稳定 PNG 快照并保存为截图历史项，再按 `whiteboard` sourceKind 调用 `QueueRecognizeWhiteboard`；OCR job ready 后可打开同一个 `ocr-result` floating panel 查看结果，过程中不阻塞画板绘制。选中图片元素 OCR 基础 UI 已接入：用户选中 Excalidraw 图片元素后可导出该元素图片为 PNG，并按 `whiteboard-selection` 提交 `QueueRecognizeWhiteboard`；前端生产构建已通过。已新增测试覆盖白板快照 OCR 入队、sourceKind、history queued 状态、选中元素请求映射为 `whiteboard-selection`、`whiteboard-selection` ready 事件回写历史，以及缺少 `imagePath` 时拒绝整图 fallback。R4 前端 e2e 已补充 `whiteboard selected image OCR queues its own exported image as whiteboard-selection`：预置选中图片元素后点击“Recognize selected image”，验证请求携带 `elementId`、导出的 PNG `imagePath`、`whiteboard-selection` sourceKind、`zh-en` language 和 interactive priority；同时修复 OCR busy 期间自动保存状态覆盖 OCR queued 文案的问题。R4 真实 worker 服务层 smoke 已补充 `TestWhiteboardSelectionOCRRealWorkerSmoke`：本地 Windows smoke 脚本会用 stable `ppocrv5-mobile-zh-en` 模型通过 `QueueRecognizeWhiteboard` 跑 `whiteboard-selection`，验证 ready sidecar、`OpenOcrResult`、`ReadOcrResultImage`、真实 blocks/plainText 和选中图片 `imagePath`，并生成 `release-out/ocr-smoke-evidence/whiteboard-ocr-real-worker-smoke.json`。R5 已补充画板坐标系 OCR 高亮和文本插入：ready 结果会保留 `OcrResult`，可把 OCR block 映射为带 `customData.recordingFreedomOcr.kind=block` 的 Excalidraw rectangle 元素，支持显示/隐藏；可把识别原文插入为带 `customData.recordingFreedomOcr.kind=text` 的 Excalidraw text 元素；G13 后同一路径已支持把译文插入为 Excalidraw text 元素。OCR 操作后的状态文案短时间内不会被自动保存覆盖。R5 前端 e2e `whiteboard OCR blocks map to the selected image and recognized text inserts as scene text` 已验证选中图片坐标映射、block 元素写入、原文插入和隐藏 block 后文本保留；G13 e2e 已验证显式 provider 配置后的译文插入链路。R6 已补浏览器/e2e 安全回归 `whiteboard OCR failure during an audio recording does not stop the recorder shell`：主胶囊处于音频录制紧凑态时，独立白板的 OCR 失败只更新白板状态，不会停止录制、不改变主胶囊 recording 状态，且主胶囊仍可正常结束录制。R6 还已把视频录制中的 `annotation-overlay` 接入 OCR：annotation 胶囊可把当前 Excalidraw snapshot 强制保存为录制包内 `annotations/snapshots/*.png`，再以 `sourceKind=whiteboard`、`sourceId=packageDir`、`priority=background` 调用 `QueueRecognizeWhiteboard`；ready 后打开同一个 `ocr-result` floating panel，不把结果塞回 overlay 或主胶囊。已新增 Go 单测 `TestRecordingAnnotationSnapshotQueuesBackgroundWhiteboardOCR` 和 `TestWhiteboardOCRRequestAllowsBackgroundPriorityForRecordingAnnotation`，并新增 Playwright e2e `recording annotation overlay queues background OCR and opens result panel`；release-config-check 已把 annotation overlay OCR 安全 e2e 纳入防回归。剩余：选中图片元素 OCR 仍需要补真实 Wails 桌面/e2e 验收，证明真实桌面白板选中图片导出、真实 worker ready 结果、结果图片和 block 坐标一致；视频录制中 `annotation-overlay` OCR 已有后端合同和浏览器/e2e 保护，但仍缺真实 Wails 桌面录制期间 worker 并行日志/录屏；真实桌面录制中 OCR 与录制日志并行安全仍需回验。

### G12 滚动长图切片识别

范围：

- 长边超过阈值时按 tile 识别。
- 合并坐标。
- 去重重叠 block。

施工颗粒：

- G12.1 long image detector：根据最终 PNG 的宽高判断是否进入 tile 模式，不读取滚动过程中的中间帧。
- G12.2 tile split：按 1600-2400px 高度切片，重叠 80-120px，记录 tile origin。
- G12.3 per-tile recognize：每个 tile 走同一 worker pipeline，但结果坐标要加回 tile origin。
- G12.4 dedupe：按文本相似度、box IoU、纵向距离和置信度去除重叠区域重复 block。
- G12.5 fallback：目标区域没有滚动时，不保存假长图，直接沿用普通区域截图和 `region-screenshot`。

交付：

- tile 高度、重叠高度、去重阈值可配置在 worker 内部常量。
- 结果坐标映射回完整长图。

验收：

- 长滚动图文字不会因整体缩小变得不可识别。
- 重叠区域不会重复输出大量相同文本。
- 无滚动目标时不会保存假长图，直接按区域截图识别。

状态：部分落地。R7 已在 OCR 服务层接入滚动长图 tile 聚合：当 `RecognizeImage` 收到 `sourceKind=scrolling-screenshot` 且最终 PNG 高度超过单次 worker `maxSide` 时，不再把整张长图缩小后一次性识别，而是按内部常量切成最高 `2200px`、重叠 `120px` 的 PNG tile；每个 tile 仍走同一个 `rf-ocr-worker` `recognize` 协议，worker 返回的局部 block 坐标会加回 tile origin，最终 result 的 `width/height/imageSha256/sourceKind/sourceId` 仍绑定原始长图。服务层已实现重叠 block 去重：同文本、box IoU 足够高或纵向/横向重叠足够接近时，只保留置信度更高的 block，并重新按完整长图坐标排序、重建 `lineIndex` 和 `plainText`。已新增 Go 单测 `TestRecognizeScrollingScreenshotUsesTilesAndDedupesOverlap`，验证一张 `320x4520` scrolling PNG 被拆成 3 次 worker 识别、坐标回填到完整长图、重叠文本去重、同图二次识别命中原图 cache 且不会再次要求 worker。`TestScreenshotOCRRealWorkerSmoke` 已扩展 `scrolling-long` 真实 stable 样例：使用模型包 smoke 图片竖向生成 `900x2800` 长图，真实 worker 返回 `scrolling-screenshot` result，当前 evidence 记录 `blockCount=20`，禁用 worker 后同图 cache hit 仍可返回当前来源 result。仍未完成：长图 tile 参数在更多真实滚动截图质量下的阈值调优、前端/桌面滚动截图入口的视觉证据。

### G13 翻译 provider 和翻译缓存

范围：

- 翻译默认关闭。
- 实现 `deepl`、`openai-compatible` provider。
- 翻译结果作为 OCR 附加数据保存。

施工颗粒：

- G13.1 settings：设置 provider、base URL、API key、model、source language、target language、隐私提示确认时间。
- G13.2 secrets：API key 不写入 `settings.json`；Windows 使用当前用户 DPAPI 加密落盘，macOS 使用系统 Keychain，Linux 优先使用 freedesktop Secret Service，缺少 session bus 或 Secret Service 时明确回退到 `0600` 权限本地 secret 文件后端；日志不得输出 key、完整请求体或 OCR 全文。
- G13.3 provider interface：统一 `TranslateBlocks(ctx, request)` 接口，provider 失败返回稳定错误码。
- G13.4 segment contract：翻译按 block id 保持顺序，provider 返回段数不匹配时进入失败或降级解析，不能错位覆盖。
- G13.5 cache：翻译结果写入 `data/ocr/translations/`，缓存 key 包含 provider、base URL、source language、target language、model、promptVersion、block id 和 block 原文。
- G13.6 UI actions：结果浮层、截图历史、钉图、画板都只调用同一 `TranslateOcr`，不私自直连 provider。

交付：

- 设置页 provider、base URL、API key、target language。
- `TranslateOcr` 支持全部 block 或选中 block。
- `data/ocr/translations/` 持久化翻译结果。

验收：

- 未配置 provider 时不会发起任何网络请求。
- 首次启用翻译时有隐私提示。
- 翻译失败不覆盖 OCR 原文。

状态：部分落地。R8 已完成后端 provider/cache 基础：`TranslateOcr` 现在只在显式传入 provider 配置时工作，provider 为空或 `disabled` 时会直接返回“翻译 provider 已禁用”错误，且不会读取 base URL、不会发起网络请求。后端已实现 `deepl` 和 `openai-compatible` 两个 provider；DeepL 通过表单提交 OCR block 文本，OpenAI-compatible 通过 `/chat/completions` 提交严格 JSON block prompt。两条 provider 路径都会按 OCR block id 和顺序校验返回结果，段数不匹配、block id 错位或译文为空都会失败，不会错位覆盖 OCR 原文。翻译结果会写入 `data/ocr/translations/`，缓存 key 已覆盖 result id、provider、base URL、source language、target language、model、promptVersion、block id 和 block 原文；重复调用同一请求会直接读缓存，不再访问 provider。设置页和 floating settings 已新增 `OCR 翻译` 配置，支持 provider、base URL、API key、model、source language、target language、隐私确认和确认时间；API key 已迁移到 `internal/secrets` 本地 secret store，`settings.json` 只保留 `apiKeySet`，Windows 使用 DPAPI，macOS 使用 Security.framework Keychain API，Linux 优先使用 freedesktop Secret Service，并在 session bus/Secret Service 不可用时明确回退到 `0600` 本地 secret 文件，前端不会回显已保存 key，日志只记录是否设置 key，不输出 key 本身。`TranslateOcr` 在桌面端可以不接收明文 key，并只在请求 provider/base URL/model 与当前 settings 匹配时从 secret store 补齐 key。OCR 结果浮层已用同一份 settings 调用 `TranslateOcr`，未配置、未确认、缺 endpoint/key/model 时只显示保护提示；成功后按 block 展示译文，不覆盖 OCR 原文，并可复制整段译文。截图历史和钉图 ready 项已新增紧凑翻译入口，均复用同一 `TranslateOcr` 调用并在成功后复制整段译文，不直连 provider、不保存私有翻译状态。画板窗口已复用同一 `TranslateOcr` 调用，把译文通过现有 Excalidraw text 元素路径插入画板，不新增私有翻译真状态。已同步 Wails TypeScript bindings，使 `TranslateRequest` 包含 `baseUrl/apiKey/model/force`，`TranslationResult` 包含 `model/promptVersion`，settings binding 包含 `apiKeySet`。已新增 Go 单测覆盖禁用 provider 不联网、OpenAI-compatible 成功并命中缓存、DeepL 分段提交、provider 返回段数不匹配时失败且不写缓存、settings 不持久化 OCR 翻译明文 key、`TranslateOcr` 可从 secret store 使用已保存 key；本轮新增服务层 smoke `TestTranslateOcrServiceUsesStoredKeyAndCachesOpenAICompatibleEndpoint`，使用可控本地 OpenAI-compatible `httptest` endpoint 证明桌面服务入口会从 secret store 读取 key、请求 `/chat/completions`、请求体包含 OCR block 和目标语言，且关闭 provider 后第二次调用仍能从 `data/ocr/translations/` 缓存返回，不再访问网络。已新增 release gate 防止 macOS Keychain 和 Linux Secret Service 后端被删除，并固定本地 OpenAI-compatible endpoint/cache smoke；已新增前端 e2e 覆盖 OCR 翻译 provider 配置持久化、截图历史/钉图翻译复制，以及白板译文插入链路。仍未完成：真实 DeepL 或外部 OpenAI-compatible provider 桌面回验、Linux Secret Service 实机写入/读取/删除回验、macOS Keychain 实机写入/读取/删除回验，以及截图历史/钉图/画板在真实桌面窗口中打开结果浮层后的译文视觉证据。

### G14 latest/quality 模型通道

范围：

- latest 使用经 RecordingFreedom smoke 验证的较新模型包。
- quality 使用更大模型，用户主动安装。
- stable/latest/quality 可切换、可回滚。

施工颗粒：

- G14.1 catalog：扩展 `third_party/ocr-models/manifest.json`，登记 stable/latest/quality、来源、license、bytes、SHA256、smoke。
- G14.2 ppocrv6 package：生成 PP-OCRv6 latest 模型包，至少通过 `RecordingFreedom`、`文字识别` smoke 和真实截图样例。
- G14.3 quality package：生成 PP-OCRv6 medium/server 类 quality 包，标注体积、预计性能和适用场景。
- G14.4 download UI：设置页显示模型通道、安装状态、下载进度、校验失败原因、删除按钮。
- G14.5 rollback：切换 active model 必须由用户确认；latest/quality 失败时可回退 stable。
- G14.6 runtime audit：如果升级 ONNX Runtime，必须先在六个平台 manifest 中证明 archive 齐全，再改默认 runtime。

交付：

- 模型下载 UI。
- 模型安装进度、校验失败原因、删除模型。
- active model 持久化。

验收：

- latest 失败不影响 stable。
- 用户可以回退 stable。
- 新模型必须通过真实 smoke 后才能进入可选列表。

状态：部分落地。R9 已完成本地模型管理、可验证下载器、release catalog 刷新链路、PP-OCRv6 candidate 本地模型包 smoke 和命令行生命周期 smoke：设置页和 floating settings 中新增 `OCR 模型` 区块，会通过同一个 `GetOcrStatus` 读取后端真实状态，展示 stable/latest/quality 三个 registry 模型、安装/校验/active 状态、缺失文件或校验失败原因、worker 路径和 runtime 路径。用户可以输入本地模型包 `.zip` 或解压目录路径并调用 `InstallOcrModelPackage` 导入；导入后重新读取后端状态，不在前端伪造安装成功。已安装且 verified 的非 active 模型必须先进入内联确认态，用户点击 `Confirm switch` 后才调用 `SetActiveOcrModel`；后端仍先验证 installed/verified，成功后才保存 active state，失败时前端重新读取真实状态并保留原 active model。已安装的非 active 模型可以调用 `RemoveOcrModel` 删除。应用内下载器现在支持两类可验证来源：优先下载 registry 中带 `package.url/package.bytes/package.sha256` 的 RecordingFreedom verified package；如果没有 release package，但默认 registry 中每个必需文件都有固定官方 `downloadUrl/bytes/sha256`，则允许手动按官方源逐文件下载、逐文件校验，并对 PaddleOCR `inference.yml` 生成 `keys.txt` 后再次校验生成文件 bytes/SHA256。PP-OCRv6 latest/quality 因 `textlineOrientation.mode=none` 不再误报缺少 `cls.onnx`，但仍保持 `releaseStatus: candidate`，这代表它们可按固定官方源手动下载验收，不代表已进入 RecordingFreedom release-ready catalog。已新增 `RefreshOcrModelCatalog`：默认从 `https://github.com/lemon-casino/RecordingFreedom/releases/latest/download/ocr-model-catalog.json` 拉取目录，只接受 HTTPS 或测试用 loopback HTTP，只允许已知 model id，必须包含完整 package URL、bytes、SHA256 和必需文件清单；通过校验后写入 `data/models/ocr/registry.json` 并合并到运行时 registry。release workflow 已通过 `-catalog-output` 和 `-release-base-url` 从真实生成的 stable zip 派生 `ocr-model-catalog.json`，release-config-check 已固定该门禁。PP-OCRv6 latest/quality 仍保持 `releaseStatus: candidate`，虽然本机 Windows x64 已完成包哈希、worker smoke、截图/滚动/白板 smoke 和“安装不自动激活 -> 确认切换 -> 真实识别 -> 删除 active -> 回退 stable”命令行生命周期 evidence，但还不能进入 release-ready catalog。浏览器/e2e 环境仍只用于 UI 链路验证，不会把模型导入、下载或目录刷新模拟成桌面成功。已新增前端 e2e `settings exposes local OCR model package management`、`settings downloads a verified OCR model package without auto-switching active model`、`settings refreshes the verified OCR model catalog before exposing downloads`、`settings confirms before switching the active OCR model` 和 `settings keeps the current OCR model when a confirmed switch fails`；已新增后端单测固定 default registry 与 `third_party/ocr-models/manifest.json` 同步、PP-OCRv6 no-orientation 不要求 `cls.onnx`、官方逐文件下载生成 `keys.txt`、生成结果校验失败不安装。仍未完成：PP-OCRv6 latest/quality 的 release-ready catalog、跨平台 worker smoke matrix、真实用户截图样例质量回验；下一次 release 发布后用真实桌面刷新 latest catalog、下载/取消/重试/安装/删除/切换 active 的回验；真实 Wails 设置页模型包导入、切换 active、删除和回退 stable 的桌面回验；latest/quality 真实识别失败后的用户可见回退 stable 流程。

### G15 发布和防回归

范围：

- CI 构建 worker、检查 runtime、检查模型 manifest、跑 smoke。
- 发布产物检查 worker/runtime/model 是否齐全。
- 前端 e2e 覆盖截图历史 OCR 状态和浮层交互。

施工颗粒：

- G15.1 unit tests：覆盖 registry、zip 安全导入、cache key、queue merge/cancel、history sidecar、translation cache。
- G15.2 worker smoke：每个平台 worker 都能 `--capabilities`，至少发布前有一个平台执行真实 stable 模型 smoke；全平台 smoke 作为 release gate 逐步补齐。
- G15.3 frontend build：`cd app/frontend && npm run build` 必须通过，OCR 类型不能直接依赖不稳定 generated binding。
- G15.4 e2e：覆盖截图历史 OCR、OCR 结果浮层、钉图高亮、画板 OCR 入口、翻译未配置保护、floating panel 不扩大胶囊。
- G15.5 artifact check：Windows/macOS/Linux 发布包检查 worker、runtime、模型包或模型下载入口、license/notice。
- G15.6 regression guard：`release-config-check` 必须防止 worker/runtime/model/smoke 路径被删、平台架构缺失、模型未校验即进入 ready。

交付：

- release-config-check 覆盖 OCR worker/runtime/model/smoke 规则。
- Windows/macOS/Linux 包全部带当前架构 worker/runtime。

验收：

- `git diff --check`
- `go test ./...`
- `cd app/frontend && npm run build`
- worker smoke。
- 相关 e2e 覆盖 `ocr|screenshot|whiteboard`。

状态：部分落地。已落地 worker/runtime 构建、stable 模型包 artifact、发布 staging、release-config-check 基础、stable 模型包 smoke 基础和跨平台 worker stable smoke workflow gate。R10 已进一步加固 `release-config-check`：实际仓库检查现在为 `ok=true`，并会防止 OCR 结果浮层/钉图 e2e、画板 OCR e2e、OCR 模型管理 e2e、翻译 provider/cache 单测、滚动长图 tile 单测、平台 worker capabilities/smoke 命令和桌面矩阵 smoke workflow 被删除或失效；同时修正 release notes 中 OCR release-gated 文案检查的大小写匹配。已新增 `app/internal/ocrevidence` 共享真实桌面 source kind 与视觉证据合同，`ocr-desktop-evidence-check`、`ocr-desktop-evidence-export` 和 `ocr-desktop-evidence-plan` 均复用同一合同，防止白板整图/白板选区等验收口径漂移；窗口/焦点窗口 sourceKind 只做兼容保留，不进入当前桌面 evidence 必验合同。已新增 `app/cmd/ocr-desktop-evidence-check`，把真实 Wails 桌面 OCR evidence 包格式固定下来：必须包含 README、平台信息、app log、OCR job 事件、用户可见截图/钉图/画板 source kind 的真实 result、视觉截图和可选翻译结果。已新增 `app/cmd/ocr-desktop-evidence-export`，用于从真实 data root、真实视觉截图目录和平台信息导出 checker-ready evidence 包；该工具要求显式 `-data-root` 和 `-visual-dir`，不会猜默认数据目录，也不会合成假视觉证据。已新增 `app/cmd/ocr-desktop-evidence-plan`，可输出 `visual-capture-checklist.md/json` 并通过 `-check` 预检真实视觉截图目录，缺少任何必需桌面场景时提前失败；该工具只检查和列清单，不生成假图片。已新增 `scripts/export-ocr-desktop-evidence.ps1` 和 `scripts/export-ocr-desktop-evidence.sh`，用于 Windows/macOS/Linux 实机运行后先执行 checklist/precheck，再导出 evidence 并立即调用 checker 验收。checker/exporter/plan 已纳入 CI/Release desktop smoke tools 构建、Windows/macOS/Linux 发布包 staging 和三端 verifier；源码树导出脚本和发布后下载回验脚本已纳入 `release-config-check`，防止真实桌面验收流程只停留在源码里或退回到只复制文件不验收。checker/exporter/plan/scripts 只证明“真实桌面 evidence 的导出、预检与验收口径、发布工具链已固定”，不能替代真实 Wails 桌面运行证据。仍未完成：下一次 Actions 产出 Windows/macOS/Linux 各架构真实 worker smoke evidence、真实桌面 OCR 入口录屏/e2e、真实 provider 桌面回验和发布产物下载后的完整性回验。

## 当前已落地颗粒

更新时间：2026-07-06。

已完成：

- 新增 `app/internal/ocr` 基础服务包。
- 已有 OCR 模型 registry，默认包含：
  - `ppocrv5-mobile-zh-en`
  - `ppocrv6-mobile-zh-en`
  - `ppocrv6-medium-zh-en`
- 已有模型 manifest 校验：
  - `manifest.json`
  - `det.onnx`
  - `cls.onnx`
  - `rec.onnx`
  - `keys.txt`
  - 可校验文件大小和 SHA256。
  - manifest 声明 `smoke` 时，会校验 `smoke.image`。
  - `smoke.image` 必须是安全相对路径，文件必须存在且能被解码为图片。
  - manifest 声明 `smoke` 时，会校验 smoke expected JSON。
  - `smoke.expected` 未显式填写时默认使用 `smoke.expected.json`。
  - smoke expected JSON 必须存在且是合法 JSON。
- 已有本地模型包导入：
  - 支持从本地目录导入。
  - 支持从 `.zip` 模型包导入。
  - 支持 `manifest.json` 位于包根目录或单层顶级目录内。
  - 导入前先解压/复制到 `data/models/ocr/.install-*` staging 目录。
  - 验证通过后再原子替换 `data/models/ocr/<modelId>/`。
  - 替换失败时保留原有已安装模型。
  - 拒绝 unknown model id，避免未登记模型绕过 registry。
  - 拒绝 zip-slip、绝对路径、`..` 路径和 symlink。
  - 导入不会自动切换 active model，必须由用户显式选择。
  - 已安装包的 `source.url`、`source.license`、版本等展示元数据来自包内 manifest。
  - 已安装包的 smoke 资产状态会回显到 `ModelInfo`：
    - `smokeImage`
    - `smokeExpected`
    - `smokeAssetReady`
    - `smokeError`
- 已有 OCR state：
  - 默认 active model 是 `ppocrv5-mobile-zh-en`。
  - 未安装模型时状态是 `no-model`。
  - 模型已校验但 worker 未接入时状态是 `worker-absent`。
  - worker 文件存在但能力探测失败或不支持识别时状态是 `worker-unavailable`。
  - 只有模型已校验且 worker capabilities 明确 `supportsRecognize: true` 时才进入 `ready`。
- 已有 OCR result/cache 存储合同：
  - `data/ocr/results/`
  - `data/ocr/cache/`
  - cache key 使用 `imageSha256 + modelId + language`。
- 已有主服务 Wails API：
  - `ListOcrModels`
  - `InstallOcrModel`
  - `InstallOcrModelPackage`
  - `RemoveOcrModel`
  - `SetActiveOcrModel`
  - `GetOcrStatus`
  - `RecognizeImage`
  - `RecognizeScreenshot`
  - `RecognizePinnedScreenshot`
  - `RecognizeWhiteboard`
  - `TranslateOcr`
  - `CancelOcrJob`
  - `OpenOcrResult`
- 已有异步 OCR 队列 Wails API：
  - `QueueRecognizeImage`
  - `QueueRecognizeScreenshot`
  - `QueueRecognizePinnedScreenshot`
  - `QueueRecognizeWhiteboard`
- 已有 OCR 事件注册：
  - `ocr.status.changed`
  - `ocr.job.queued`
  - `ocr.job.started`
  - `ocr.job.finished`
  - `ocr.job.failed`
  - `ocr.job.cancelled`
- 已有 OCR 模型安装事件注册：
  - `ocr.model.installed`
  - `ocr.model.failed`
- 已有截图历史变更事件：
  - `screenshot.history.changed`
- 截图历史 `ScreenshotItem` 已增加 OCR sidecar 状态字段：
  - `ocrStatus`
  - `ocrResultId`
  - `ocrModelId`
  - `ocrLanguage`
  - `ocrUpdatedAt`
  - `ocrError`
- 旧截图历史兼容：
  - 缺失 OCR 字段的历史项会归一为 `ocrStatus: none`。
  - 非法 OCR 状态会归一为 `none` 并清理 stale result/error 字段。
- 前端 `recorderBackend` 已增加 OCR 稳定业务类型和 API 包装，UI 后续不需要直接引用 generated binding。
  - 已包装模型列表、状态、安装、模型包导入、切换、移除。
  - 已同步 `sourceUrl` 和 `license` 元数据字段。
  - 已同步 `smokeImage`、`smokeExpected`、`smokeAssetReady`、`smokeError` 元数据字段。
- 截图历史 UI 已接入 OCR sidecar 状态：
  - 历史项显示 `none/queued/running/ready/failed` 对应文案。
  - 历史项支持点击排队识别、识别完成后打开结果、复制全文、失败后重试。
  - 操作走 `QueueRecognizeScreenshot`、`OpenOcrResult` 和 OCR result/cache，不生成假识别文本。
- 截图入口自动 OCR 第一版已接入：
  - `settings.Settings` 新增 `ocr.autoRecognizeScreenshots`，默认关闭。
  - `PatchSettingsPreferences` 新增 `autoOcr`，完整设置窗口和胶囊 floating settings 都可切换。
  - 截图保存后通过后台 goroutine 读取当前设置，开启时调用 OCR queue，不阻塞截图保存。
  - 自动 OCR 走 `screenshotOCRRequest`，按截图 `mode` 分发 `region-screenshot/full-screenshot/window-screenshot/focused-window-screenshot/scrolling-screenshot/whiteboard`。
  - 已有单测覆盖默认关闭、设置持久化、偏好 patch、`SaveSettings` 不覆盖 OCR 偏好、开启后截图保存会进入 OCR sidecar 状态。
- OCR 结果浮层已接入第一版：
  - 新增 `FloatingPanelKind=ocr-result`。
  - `FloatingPanelRequest/State` 新增 `contextId`，用于传递 OCR result id。
  - `FloatingPanelWindow` 根据 `contextId` 调用 `OpenOcrResult` 读取持久化结果。
  - `ReadOcrResultImage(resultId)` 已作为截图历史、钉图、画板和后续白板选区共用的结果图片读取接口，优先读取 `OcrResult.imagePath`，并限制图片必须位于应用数据根目录内；旧结果缺少 `imagePath` 时，可按截图类、钉图和整张画板 source 回退读取截图历史 PNG，选区类结果不允许回退到整张图。
  - 浮层展示截图预览、OCR block 坐标叠层、原文、block 数量、block 文本和置信度。
  - 鼠标移入/键盘聚焦 block 列表项时会高亮预览图中的对应 block。
  - 支持复制全文，也支持点击单个 block 复制单块文本。
  - 翻译按钮已接入 G13 前保护状态：点击只显示 provider 未配置提示，不调用 `TranslateOcr`，不发网络请求。
  - 已新增前端 e2e：`floating OCR result panel shows screenshot image blocks and translation guard`，直接打开 `/#/floating-panel` 验证 OCR 结果浮层窗口本体，覆盖预览图、block polygon、hover 高亮、文字展示和翻译未配置保护。
- 钉图/固定图片 OCR 第一版已接入：
  - 钉图窗口会读取当前截图历史 OCR sidecar 状态，并订阅 `screenshot.history.changed` 刷新。
  - 钉图工具栏可触发 `QueueRecognizePinnedScreenshot`，后端按 `pinned-screenshot` sourceKind 和 `interactive` priority 入队。
  - 钉图 ready 后会读取持久化 OCR result，支持打开同一个 `ocr-result` floating panel。
  - 钉图可显示/隐藏 OCR polygon 高亮，支持复制全文、点击 block 复制单块文本、键盘选择 block 后复制。
  - 已有后端测试覆盖钉图 OCR 入队 sourceKind、priority、language 和截图历史 queued sidecar 状态。
  - 已新增前端 e2e：`pinned screenshot window restores OCR highlight and result floating panel after resize`，覆盖截图历史打开钉图、钉图窗口恢复 ready OCR、高亮 overlay、窗口 resize 后仍保持原图坐标 `viewBox`、block hover/copy、以及从钉图按钮打开 `ocr-result` floating panel。
- 画板 OCR 第一版已接入：
  - `whiteboardOCRRequest` 已区分 `whiteboard` 和 `whiteboard-selection` sourceKind。
  - `QueueRecognizeWhiteboard` / `RecognizeWhiteboard` 在 sourceId 对应截图历史项时会回写 OCR sidecar。
  - `handleOCRJobEvent` 已把 `whiteboard` 纳入截图历史 OCR 状态同步。
  - 白板窗口新增 OCR 操作，会导出当前画板为稳定 PNG 快照、保存为截图历史项，并按 `whiteboard` sourceKind 提交 `QueueRecognizeWhiteboard`。
  - 白板窗口选中图片 OCR 基础 UI 已接入，会把选中 Excalidraw 图片元素导出为 PNG，并按 `whiteboard-selection` sourceKind 提交 `QueueRecognizeWhiteboard`；当前已通过前端生产构建，仍需真实桌面/e2e。
  - 白板 OCR ready 后可打开同一个 `ocr-result` floating panel 查看结果。
  - 已有测试覆盖白板快照 OCR 入队、history queued 状态，以及选中元素请求映射为 `whiteboard-selection`。
  - 已新增前端 e2e：`whiteboard selected image OCR queues its own exported image as whiteboard-selection`，覆盖选中图片元素、导出独立 PNG、携带 `elementId`、按 `whiteboard-selection` 入队、语言和优先级。
  - 已修复白板 OCR busy 状态保护：自动保存仍然执行，但 OCR 排队/运行期间不会把状态栏文案覆盖回 `Unsaved/Saved`。
- 浏览器预览 fallback 不会生成假 OCR 结果，只返回 `no-model` 状态或明确错误。
- 已实现 Go 侧 `rf-ocr-worker` JSON Lines 协议客户端：
  - 每次识别启动独立 worker 进程。
  - 发送 `init`、`recognize`、`release` 请求。
  - 校验 worker response id。
  - worker stderr 会附加到错误信息中。
  - worker 超时会返回明确错误。
- 已实现 Go 侧 OCR job queue：
  - `interactive`、`normal`、`background` 三档优先级。
  - 同一时间只执行一个 OCR job。
  - 相同 `imageSha256 + modelId + language` 的队列任务会合并。
  - 合并任务只执行一次 worker 识别，但会按每个 source 分发结果事件。
  - 未开始任务可以取消，并发出 `ocr.job.cancelled`。
  - 已经进入 worker 的任务可以标记取消，完成后丢弃结果，不写入前端 ready 状态。
  - 队列事件会同步截图历史 OCR sidecar 状态。
  - 截图历史 UI 已订阅 `screenshot.history.changed`，后续 OCR 状态变化不需要手动刷新历史。
- 已实现 worker capabilities 探测：
  - 主程序调用 `rf-ocr-worker --capabilities`。
  - 主程序会把当前平台 ONNX Runtime 目录传给 worker。
  - capabilities 必须声明 `schemaVersion` 和 `protocolVersion`。
  - capabilities 已包含：
    - `runtimeDir`
    - `runtimeLibrary`
    - `runtimeAvailable`
    - `runtimeVersion`
    - `runtimeApiVersion`
    - `runtimeError`
  - `runtimeAvailable=true` 已升级为真实动态库可加载，并且 ONNX Runtime C API version、Env 和 SessionOptions 生命周期探测通过。
  - 缺少 capabilities、capabilities JSON 非法、探测超时、runtime 不可加载、worker 自报不支持识别时，都不会进入 `ready`。
  - worker 可以区分“runtime 缺失”“runtime 文件存在但不可加载”和“runtime 可加载但 OCR 推理管线尚未接入”。
  - 前端 `OcrStatus` 已同步 `workerCapabilities` 字段，UI 后续可以展示真实 worker 能力。
- `RecognizeImage` 已接入 worker 通道：
  - 先检查图片和 SHA256。
  - 命中缓存时不启动 worker。
  - 校验 active model 已安装且 verified。
  - 校验 worker 文件存在并且 capabilities 支持识别。
  - worker 返回结果后补齐 `sourceKind`、`sourceId`、`imagePath`、`imageSha256`、`modelId`、`language`、图片宽高和 `plainText`。
  - 写入 `data/ocr/results/` 和 `data/ocr/cache/`。
- 已有测试用 worker helper 验证协议和缓存，但它只在 Go 测试进程中启用，不作为产品 OCR 能力。
- 已新增可编译的 `app/cmd/ocr-worker` 协议外壳：
  - 支持 `--capabilities`。
  - `--capabilities` 支持 `--runtime-dir` 参数。
  - 支持 JSON Lines 协议的 `init`、`recognize`、`release`。
  - `init` 已校验 `modelDir`、`runtimeDir`、`manifest.json`、`det.onnx`、`cls.onnx`、`rec.onnx`、`keys.txt`。
  - `init` 对缺失 runtime 返回 `onnx_runtime_missing`。
  - `init` 对缺失模型文件返回 `model_file_missing`。
  - 未 `init` 就 `recognize` 会返回 `model_not_loaded`。
  - runtime 可加载时会报告 `supportsRecognize: true`。
  - runtime 缺失或不可加载时 `supportsRecognize: false`，并返回明确诊断。
  - 不产生假 OCR 文本；识别失败时返回错误，不写 ready 结果。
- 已实现 `rf-ocr-worker` native ONNX Runtime session preflight：
  - `init` 会在 worker 进程内加载 `det.onnx`、`cls.onnx`、`rec.onnx`。
  - 模型加载使用 `CreateSessionFromArray`，避免 Windows `ORTCHAR_T` 路径编码差异。
  - `init` 成功响应会返回每个模型的 input/output name、`elementTypeCode`、dtype 名称和 dimensions。
  - 三个模型任意一个加载失败时返回 `model_session_failed`，不会进入 initialized 状态。
  - `release` 或 stdin EOF 会释放 session、SessionOptions、Env 和 ONNX Runtime 动态库。
- 已实现 `rf-ocr-worker` 图片预处理：
  - `recognize` 会读取 PNG/JPEG/WebP 输入。
  - 透明图片会先合成到白底，避免画板透明背景影响文字边缘。
  - 按 `maxSide` 限制最长边，默认 `2400`，并把输入宽高约束到 32 的倍数。
  - 生成 `float32`、NCHW、RGB 三通道归一化 tensor，使用 mean `[0.485, 0.456, 0.406]` 和 std `[0.229, 0.224, 0.225]`。
  - `recognize` 会返回 `preprocess` 摘要；在最终 rec decode 完成前不会输出假文字。
  - 图片打开、解码或预处理失败统一返回 `image_preprocess_failed`。
- 已实现 `rf-ocr-worker` det 推理和候选框后处理：
  - worker 通过 ONNX Runtime C API 创建 CPU tensor，调用 `Run` 执行 det session。
  - 读取 output tensor 的 shape 和 float32 data，返回 det 输出摘要。
  - det score map 使用阈值 `0.30` 生成连通域，过滤过小区域，候选框向外扩展并映射回原图像素坐标。
  - 候选框按阅读顺序排序，并限制最大候选数量，避免异常图片产生过量结果。
  - `recognize` 会返回 `inference.candidates`，方便 smoke 和诊断确认候选框来源。
- 已实现 `rf-ocr-worker` cls/rec 第一版真实识别链路：
  - det 候选框会从白底原图裁剪文字条。
  - cls 输入按 `3x48x192` 归一化和 padding，识别 0/180 方向，置信度超过阈值才旋转。
  - rec 输入按 `3x48x动态宽度` 归一化和 padding，宽度限制在第一版安全范围内。
  - `keys.txt` 会加载为 CTC 字符表，并追加 `blank` 和普通空格以匹配 stable 模型输出类别数。
  - rec 输出通过 CTC 去重、跳过 blank、平均置信度生成文本。
  - `recognize` 已可返回真实 `ocr.Result.blocks`、`plainText`、图片宽高和 block 坐标。
  - 本地协议实测：宽图样例可识别 `RecordingFreedom` 和 `OCR`。
- 已实现 `rf-ocr-worker --smoke` 第一版：
  - 支持 `--runtime-dir`、`--model-dir`、`--image`、`--max-side` 和可重复/逗号分隔的 `--must-contain`。
  - 未传 `--image` 时默认使用模型目录下的 `smoke.png`。
  - 未传 `--must-contain` 时默认检查 `RecordingFreedom` 和 `文字识别`。
  - smoke 会真实加载 runtime、模型 session、图片预处理和完整识别链路。
  - smoke 输出 JSON，包含 `ok`、`plainText`、`blocks`、`candidateCount` 和失败原因。
  - 本地实测命令已通过：`rf-ocr-worker --smoke --runtime-dir app/tools/onnxruntime/windows-amd64 --model-dir <stable-model>`，使用模型包内默认 `smoke.png` 识别 `RecordingFreedom` 和 `文字识别`。
- 已接入 OCR worker 构建和发布资产路径：
  - CI build matrix 会为每个桌面架构构建 `rf-ocr-worker`。
  - Release build matrix 会为每个桌面架构构建 `rf-ocr-worker`。
  - worker 使用 `CGO_ENABLED=0` 独立构建，不绑主程序/RNNoise 的 cgo 路径。
  - Windows portable 包含 `tools/ocr-worker/windows-<amd64|arm64>/rf-ocr-worker.exe`。
  - Windows installer 递归安装 `tools/ocr-worker/`。
  - macOS app bundle 包含 `Contents/MacOS/tools/ocr-worker/darwin-<amd64|arm64>/rf-ocr-worker`。
  - Linux portable 包含 `tools/ocr-worker/linux-<amd64|arm64>/rf-ocr-worker`。
  - Windows/macOS/Linux 发布验证脚本都会检查 worker 文件、架构和 `--capabilities`。
  - `release-config-check` 已加入 OCR worker 构建/发布/验证防回归规则。
- 已接入 ONNX Runtime 获取、校验和发布资产路径：
  - 固定 ONNX Runtime CPU `v1.23.2`，因为该版本同时覆盖 Windows x64/ARM64、macOS x64/ARM64、Linux x64/ARM64。
  - `third_party/onnxruntime/manifest.json` 记录每个平台 archive name、download URL、bytes、SHA256、license 和 required library。
  - `scripts/ensure-windows-onnxruntime.ps1` 支持 Windows x64/ARM64 runtime 下载、大小校验、SHA256 校验、解压和最小 DLL/notice 安装。
  - `scripts/ensure-unix-onnxruntime.sh` 支持 macOS/Linux x64/ARM64 runtime 下载、大小校验、SHA256 校验、安全解包和 symlink 库文件补齐。
  - CI build matrix 会为每个桌面架构准备 `app/tools/onnxruntime/<goos-goarch>/`。
  - Release build matrix 会为每个桌面架构准备 `app/tools/onnxruntime/<goos-goarch>/`。
  - Windows portable 包含 `tools/onnxruntime/windows-<amd64|arm64>/`。
  - Windows installer 递归安装 `tools/onnxruntime/`。
  - macOS app bundle 包含 `Contents/MacOS/tools/onnxruntime/darwin-<amd64|arm64>/`。
  - Linux portable 包含 `tools/onnxruntime/linux-<amd64|arm64>/`。
  - Windows/macOS/Linux 发布验证脚本都会检查 runtime 动态库、架构和 `rf-ocr-worker --capabilities --runtime-dir` 的 `runtimeAvailable: true`。
  - `release-config-check` 已加入 ONNX Runtime manifest、下载脚本、发布 staging 和验证脚本防回归规则。
- 已接入 stable OCR 模型包发布通道：
  - `third_party/ocr-models/manifest.json` 固定 `ppocrv5-mobile-zh-en` stable 模型包来源、版本、license、文件下载 URL、bytes 和 SHA256。
  - stable 模型文件来自公开 `jingsongliu/onnxocr_model` Hugging Face 仓库的 PP-OCRv5 ONNX 文件，记录 license 为 Apache-2.0。
  - `app/cmd/ocr-model-package` 可以下载 `det.onnx`、`cls.onnx`、`rec.onnx`、`keys.txt`，逐个校验大小和 SHA256。
  - `app/cmd/ocr-model-package` 会生成 RecordingFreedom 可导入的 `ppocrv5-mobile-zh-en-<version>.zip`，包内包含 `manifest.json`、四个模型文件、`smoke.png` 和 `smoke.expected.json`。
  - 内置 `smoke.png` 已修正为 Go PNG decoder 可解码、stable 模型可识别的中英文图。
  - 生成的包不会自动切换 active model，仍沿用现有 `InstallOcrModelPackage` 本地导入合同。
  - CI workflow 新增 `OCR Model Smoke` job，生成 stable 模型包后使用 Linux x64 worker/runtime 解包执行 `rf-ocr-worker --smoke`。
  - Release workflow 新增 `model-package` job，只构建一次 stable OCR 模型包，并作为 `RecordingFreedom-ocr-models` artifact 上传。
  - Release workflow 上传模型包前会解包执行 `rf-ocr-worker --smoke`，验证 `RecordingFreedom` 和 `文字识别`。
  - Release workflow 会生成 `SHA256SUMS-ocr-models.txt`，并检查 zip 内 required 文件存在。
  - `release-config-check` 已加入 stable OCR 模型 catalog、打包器和 release artifact 防回归规则。
- 已有单测覆盖：
  - 默认无模型状态。
  - 模型 manifest 和文件校验。
  - worker capabilities 从 `worker-unavailable` 到 `ready` 的状态判断。
  - worker 协议识别和缓存写入。
  - worker runtime capabilities 诊断。
  - worker `init` 的 runtime/model 文件 preflight。
  - worker PNG/JPEG/WebP 预处理、透明图白底合成、CHW tensor 尺寸和缺失图片错误。
  - worker det score map 连通域、候选框坐标映射和阅读顺序排序。
  - worker fake protocol 测试覆盖 `recognize` 返回 `ocr.Result`。
  - worker smoke CLI 本地实测覆盖 stable 模型真实识别。
  - 模型包内置 smoke PNG 通过 Go `png.Decode`。
  - 生成后的 stable 模型包本地解包 smoke 已通过。
  - CI/release OCR model smoke workflow 配置防回归检查。
  - OCR worker 发布流水线配置防回归检查。
  - ONNX Runtime 发布流水线配置防回归检查。
  - OCR stable 模型包 catalog 合同、zip 生成和 `InstallModelPackage` 导入合同。
  - OCR job queue 的合并、取消和优先级调度。
  - OCR job event 对截图历史 sidecar 状态的更新。
  - `rf-ocr-worker` 协议外壳 capabilities 和未接 ONNX 的错误合同。
  - 模型包 `.zip` 导入、manifest 元数据回显、不自动切换 active model。
  - 模型包 smoke 图片和 expected JSON 资产校验。
  - 危险 zip 路径拒绝。
  - 无效替换不破坏原有已安装模型。

尚未完成：

- G8 截图入口真实模型端到端 UI/e2e 尚未完成：
  - 区域截图、全屏截图、滚动截图都要用真实 stable 模型跑通一次；窗口截图/焦点窗口截图只保留后端兼容 sourceKind，不再作为用户工具入口验收项。
  - 历史图上的 block 坐标需要用截图预览叠层验证，不能只看 `plainText`。
  - 无滚动目标必须按普通区域截图识别，不保存假长图。
- G9 OCR 结果浮层仍需真实桌面视觉回验：
  - 现在截图历史 ready 结果已可打开浮层。
  - 结果浮层已通过 `ReadOcrResultImage(resultId)` 共用 `OcrResult.imagePath` 图片读取；旧结果缺 `imagePath` 时已支持截图历史 source fallback，选区类结果仍必须有自己的 `imagePath`。
  - 前端 e2e 已覆盖独立浮层内的图片、block、hover 高亮和翻译未配置保护。
  - 真实桌面截图入口打开结果浮层还需要录屏或桌面 e2e 证据；真实 provider、翻译缓存和译文展示仍归 G13。
- G10 钉图/固定图片 OCR 第一版已部分落地，仍需桌面回验：
  - 前端 e2e 已覆盖钉图窗口恢复 ready OCR、高亮 overlay、resize 后原图坐标 viewBox 保持和从钉图打开 OCR 结果浮层。
  - 仍需要真实 Wails 桌面视觉/e2e 验证钉图独立窗口缩放、resize、关闭再打开后 OCR 高亮始终对齐。
  - 钉图翻译入口已接入同一 `TranslateOcr` 调用并有浏览器 e2e；仍必须补真实桌面 provider 与独立窗口视觉回验。
- G11 画板 OCR 第一版已部分落地，仍需收尾：
  - 选中图片元素导出 OCR 基础 UI 已接入，前端 e2e 已覆盖 `whiteboard-selection` 请求、独立导出图片路径和 `elementId`；服务层真实 worker smoke 已覆盖 `QueueRecognizeWhiteboard` 到 ready result、选中图片 `imagePath` 和 `ReadOcrResultImage`，仍需真实 Wails 桌面/e2e 验收 ready 结果。
  - OCR block 在画板坐标系上的显示/隐藏高亮已通过前端 e2e 验证，使用 Excalidraw rectangle 元素持久化，缩放/平移随画板自身处理；仍需真实桌面 ready 结果回验。
  - 识别原文插入为画板文本元素已通过前端 e2e 验证；译文也已通过同一 Excalidraw text 元素路径插入，浏览器 e2e 覆盖 provider 显式配置后的译文插入链路。
  - 音频录制中独立白板 OCR 失败不影响主胶囊录制状态已通过前端 e2e 验证；视频录制中的 `annotation-overlay` 已可把包内 snapshot 作为 `whiteboard` OCR 后台任务排队并打开同一 OCR 结果浮层，但真实桌面录制并行日志/录屏仍未完成。
- G12 滚动长图切片识别第一版已部分落地：
  - OCR 服务层已对 `scrolling-screenshot` 最终长图启用 tile 识别、坐标还原和重叠去重。
  - 已有单测证明长图不会只走一次 worker，也不会把重叠区域重复输出为多条相同文本。
  - 仍需真实 stable 模型长滚动图样例和桌面滚动截图入口视觉/e2e 证据。
- G13 翻译 provider 第一版已部分落地，仍需真实 provider 和桌面回验：
  - `deepl`、`openai-compatible`、翻译缓存和段落校验已在后端落地并有 provider mock 单测。
  - 设置页和 floating settings 已支持 provider/base URL/API key/model/source language/target language/隐私确认；API key 已移出 `settings.json`，settings 只保存 `apiKeySet`，Windows 走 DPAPI，macOS 走系统 Keychain，Linux 优先走 freedesktop Secret Service，缺少 session bus 或 Secret Service 时明确走 `0600` 本地 secret 文件 fallback，日志只记录 key 是否存在。
  - OCR 结果浮层已按同一 settings 调用 `TranslateOcr` 并展示 block 译文；默认关闭、未确认或配置缺失时不会发送任何识别文字；翻译成功后可复制整段译文。
  - 截图历史和钉图 ready 项已新增紧凑翻译入口，显式配置 provider 后调用同一 `TranslateOcr` 并复制译文；未配置时只返回本地保护提示。
  - 画板窗口已复用同一 settings 和 `TranslateOcr` 调用，译文可插入为 Excalidraw text 元素；浏览器 e2e 只验证 UI 链路，桌面真实 provider 仍需回验。
  - 仍缺真实 DeepL/OpenAI-compatible 桌面回验、Linux Secret Service 实机写入/读取/删除回验、macOS Keychain 实机写入/读取/删除回验，以及截图历史/钉图/画板真实桌面窗口里的译文视觉证据。
- G14 模型通道和下载 UI 第一版本地管理已部分落地，仍需下载器和真实模型回验：
  - PP-OCR stable 模型包已可由 release 生成并作为独立 artifact 发布，但尚未内置进桌面安装包，也尚未接入 app 内联网下载/更新 UI。
  - 设置页已能展示 stable/latest/quality 状态、导入本地模型包、显示校验失败原因、切换 active model、删除非 active 已安装模型。
  - latest/quality PP-OCRv6 模型包尚未进入经过 RecordingFreedom smoke 的 catalog；真实模型包导入/切换/回退还需要桌面回验。
- G15 发布和防回归第一版已部分落地，仍需真实桌面和发布产物回验：
  - `release-config-check` 已覆盖 OCR worker/runtime/model/smoke、OCR result/pin/whiteboard/model-management e2e、translation cache 单测和 scrolling tile 单测；当前真实仓库执行结果为 `ok=true`。
  - 仍需全平台真实 worker smoke 矩阵、真实桌面 OCR 入口录屏/e2e、真实 provider 桌面回验和发布产物下载后的完整性回验。
  - G7 队列合并/取消/缓存命中仍需要用真实 worker 再回验。

## 后续开发顺序

后续只按当前状态从 `R1` 到 `R10` 推进。`R` 是剩余执行颗粒，归属仍落在 `G8-G15` 中；每个 `R` 完成后必须同步更新对应 `Gx` 状态、本节状态和“尚未完成”列表。

| 顺序 | 归属 | 目标 | 必须交付 | 验收证据 |
| --- | --- | --- | --- | --- |
| R1 | G7/G8 | 截图入口真实模型回验 | 区域、全屏、滚动最终图都能提交 stable OCR；窗口/焦点窗口 sourceKind 只做兼容回归；失败只写 OCR sidecar，不影响截图保存 | 真实 stable worker smoke；至少一组桌面截图历史结果截图；同图二次识别命中缓存日志 |
| R2 | G9 | OCR 结果浮层桌面回验 | `ocr-result` floating panel 展示真实图片、真实 block、复制全文/单块、hover 高亮；关闭不影响主胶囊尺寸 | 前端 e2e 或桌面录屏；`ReadOcrResultImage` 单测已覆盖 imagePath/fallback/越界拒绝 |
| R3 | G10 | 钉图/固定图片 OCR 回验 | 钉图窗口触发识别、显示/隐藏 block、高亮随 resize/缩放保持对齐，关闭重开后状态恢复 | 桌面视觉/e2e；钉图识别入队和 sidecar 单测 |
| R4 | G11 | 画板选中图片 OCR 收尾 | 选中图片导出 OCR 在桌面真实可用；sourceKind 为 `whiteboard-selection`；结果打开时使用选中图片自己的 `imagePath` | 前端构建已通过；仍需桌面/e2e 证明选中图片、结果图片、block 坐标一致 |
| R5 | G11 | 画板 OCR 高亮和文本插入 | 画板内显示/隐藏 OCR block；缩放/平移保持对齐；可把原文插入为 Excalidraw 文本元素 | 前端 e2e 或桌面录屏；保存/重新打开后不丢 OCR result |
| R6 | G11/G15 | 录制中画板安全回验 | 录制中使用画板 OCR 不阻塞录制、鼠标、绘制；OCR 失败不影响录制文件 | 桌面录屏或 e2e；日志中录制状态和 OCR job 并行但互不阻塞 |
| R7 | G12 | 滚动长图 tile OCR | 最终长图进入 tile 模式，坐标回填完整长图，重叠区域去重；无滚动时按普通区域图识别 | worker 单测覆盖 tile split/dedupe；真实长图 OCR 样例 |
| R8 | G13 | 翻译 provider 和缓存 | `deepl`、`openai-compatible` 通过同一 `TranslateOcr`；默认关闭；未配置时不发请求；译文缓存到 `data/ocr/translations/` | provider mock 单测；未配置保护 e2e；日志不泄露 API key |
| R9 | G14 | 模型通道和下载 UI | stable/latest/quality catalog、下载、校验、删除、切换、回滚；PP-OCRv6 latest/quality 只有 smoke 通过才展示 | manifest 单测；模型包 smoke；设置页 e2e |
| R10 | G15 | 发布门禁和全平台回归 | release 产物检查 worker/runtime/model；跨平台 capabilities；OCR 入口 e2e；文档状态同步 | `git diff --check`、`go test ./...`、`npm run build`、worker smoke、release-config-check |

当前 R1 进度：

- 已落地截图模式到 OCR sourceKind 的单测矩阵，覆盖 `region`、`full`、`screen`、`window`、`focused-window`、`scrolling`、`whiteboard`。
- 已落地手动 `QueueRecognizeScreenshot` 的失败隔离：图片缺失或排队失败时，截图历史仍保留，只把 OCR sidecar 写成 `failed` 并记录错误。
- 已落地自动截图 OCR 的失败隔离：自动排队前先写 queued，排队失败后写 failed；截图保存、缩略图和历史项不受影响。
- 已修复缓存命中的 result 归属：同一图片、同一模型、同一语言命中缓存时，会为当前 `sourceKind/sourceId/imagePath` 写入独立 result，避免截图历史项引用到其他来源的 OCR result。
- 已落地 `scripts/run-local-ocr-smoke.ps1`，可在 Windows 本机复现真实 stable OCR smoke：构建 `rf-ocr-worker.exe`、生成并校验 `ppocrv5-mobile-zh-en` 模型包、加载本地 ONNX Runtime、执行 `rf-ocr-worker --smoke`。
- 本机已通过 `powershell -NoProfile -ExecutionPolicy Bypass -File scripts\run-local-ocr-smoke.ps1`，输出 `ok=true`，`plainText` 包含 `RecordingFreedom` 和 `文字识别`，`blocks=2`，`candidateCount=2`。
- 同一脚本已通过 `TestScreenshotOCRRealWorkerSmoke`：真实 worker/model 识别 `region`、`full`、`window`、`focused-window`、`scrolling` 五类截图历史项，全部写入 ready sidecar；`OpenOcrResult` 和 `ReadOcrResultImage` 都指向当前截图图片；禁用 worker 后再次识别同图可从缓存返回当前来源 result。
- `TestScreenshotOCRRealWorkerSmoke` 现在支持 `RF_OCR_EVIDENCE_DIR`，`scripts/run-local-ocr-smoke.ps1` 会自动写出 `release-out/ocr-smoke-evidence/screenshot-ocr-real-worker-smoke.json`，并复制每个截图 mode 的原图和 `*-ocr-overlay.png` 红框叠层图。该证据记录每个截图 mode 的 `sourceKind/sourceId/resultId/imagePath/modelId/language/plainText/block box/confidence`，并标记禁用 worker 后的 cache 命中；本机已生成并核对五类截图项都包含 `RecordingFreedom` 和 `文字识别` 两个真实 block，且 overlay 文件可直接打开检查坐标。
- 已新增独立 evidence checker `app/cmd/ocr-smoke-evidence-check`：
  - 校验 `screenshot-ocr-real-worker-smoke.json` 的 `schemaVersion`、必需场景、sourceKind、modelId、language、plainText、block 数量、block 坐标范围、evidence PNG 和 overlay PNG 尺寸。
  - 必需场景包括 `region/full/window/focused-window/scrolling/scrolling-long/region-queued-cache-hit`。
  - `scrolling-long` 必须大于 `2400px` 高，防止长图 tile OCR 被普通短图冒充。
  - 至少一个场景必须证明禁用 worker 后 `cacheHitWithoutWorker=true` 且带有 `cachedResultId`；`region-queued-cache-hit` 必须证明 queued job 在 worker 不存在时仍通过 cache 写成 ready。
  - checker 拒绝 evidence 图片路径逃逸 evidence 目录，避免引用外部无关图片伪装证据。
  - `scripts/run-local-ocr-smoke.ps1` 已在生成真实 worker evidence 后自动执行 `go run .\cmd\ocr-smoke-evidence-check -evidence-dir $evidenceDir`。
- `TestScreenshotOCRRealWorkerSmoke` 已补强真实缓存入队回验：在禁用 `rf-ocr-worker` 后，先用同步 `RecognizeScreenshot` 验证同图 cache hit 会返回当前截图来源 result，再创建相同图片的新截图历史项并调用 `QueueRecognizeScreenshot`；该 queued job 必须在 worker 不存在时通过 cache 写成 ready sidecar，且 `OpenOcrResult` 的 `sourceId/imagePath/plainText` 必须指向新截图项。evidence JSON 新增 `region-queued-cache-hit`，记录 `queuedCacheHit=true` 和 `queuedCacheResultId`。
- 已新增截图历史取消事件防护：
  - `handleOCRJobEvent` 会记录已取消的 `jobId/sourceKind/sourceId`。
  - 同一 job 取消后的迟到 `running/ready/failed` 不再回写截图历史 sidecar，避免 UI 从取消/none 被旧结果污染成 ready。
  - 新 job 的 queued/ready 仍可正常更新同一截图项，避免取消标记阻塞后续重试。
  - 新增 `TestCancelledOCRJobEventDoesNotAllowLateReadyHistoryState` 覆盖 cancelled -> late ready 不污染 sidecar，以及新 job 可重新写 ready。
  - release-config-check 已把 `TestCancelledOCRJobEventDoesNotAllowLateReadyHistoryState` 和 “late ready after cancel” 文案加入截图 OCR 门禁，防止取消防护被后续改动删除。
- 本轮已通过 `cd app && go test . -run "TestQueueRecognizeScreenshot|TestCaptureScrollingScreenshotStaticTargetQueuesRegionOCR" -count=1`；真实 smoke 在普通 `go test` 中仍按环境变量跳过，完整脚本用于生成 worker/model/evidence。
- 本轮已通过 `cd app && go test . -run "TestOCRJobEventUpdatesScreenshotHistoryState|TestCancelledOCRJobEventDoesNotAllowLateReadyHistoryState|TestQueueRecognizeScreenshot|TestCaptureScrollingScreenshotStaticTargetQueuesRegionOCR" -count=1`。
- 本轮已通过 `cd app && go test ./internal/ocr -run "TestCancelQueuedJobEmitsCancelledAndRemovesJob|TestQueuedJobsPreferInteractivePriority" -count=1`。
- 本轮已通过 `cd app && go run ./cmd/release-config-check`，真实仓库报告 `ok=true`。
- 本轮已通过 `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\run-local-ocr-smoke.ps1`，真实 worker/model smoke 通过，并确认 `release-out/ocr-smoke-evidence/screenshot-ocr-real-worker-smoke.json` 中存在 `region-queued-cache-hit`、`queuedCacheHit=true` 和 `queuedCacheResultId`。
- 本轮已通过 `cd app && go test ./cmd/ocr-smoke-evidence-check`，覆盖完整 evidence 可通过、缺 queued cache proof 会失败、overlay 路径逃逸会失败、`scrolling-long` 短图会失败。
- 本轮已通过 `cd app && go test ./...`。
- 本轮已通过 `git diff --check`。
- 已新增前端 e2e `floating OCR result panel renders real worker smoke evidence coordinates`：浏览器 fallback 支持注入 `__RF_OCR_RESULTS__` / `__RF_OCR_IMAGES__`，测试把真实 worker smoke evidence 的 `900x280`、`ppocrv5-mobile-zh-en`、两个 block 及其浮点坐标注入 `ocr-result` floating panel，并断言 SVG `viewBox=0 0 900 280`、两个 polygon 的 `points` 精确等于 worker 输出坐标。这样可以证明结果浮层 UI 消费真实 `OcrResult` 坐标，而不是只使用浏览器固定 mock block。
- 已新增前端 e2e `screenshot history ready item opens OCR result floating panel with real worker evidence`：在胶囊主界面预置 ready 截图历史项、真实 worker smoke result 和图片，使用 `__RF_FORCE_FLOATING_PANEL_WINDOWS__` 只在 e2e 中模拟桌面多窗口浮层分支；测试从“Screenshot / board”打开历史列表，点击 ready 项的 `View OCR result`，断言 `__RF_FLOATING_PANEL__` 被设置为 `kind=ocr-result/contextId=<resultId>/380x420`，再打开 `/#/floating-panel` 验证同一真实坐标 result 被渲染。该测试覆盖“从胶囊历史项进入 OCR 浮层”的调用链，但仍不能替代真实 Wails 桌面录屏证据。
- 已通过 `cd app && go test ./...`。
- 仍未完成：真实桌面捕获入口还需要在 UI/e2e 中选择区域、全屏和滚动目标并回验；历史图 block 坐标叠层仍需视觉/e2e 证据。

当前 R2 进度：

- 已补浏览器/e2e 专用 OCR 结果 fallback：只在非 Wails 桌面环境中，从截图历史 ready 项构造可验证的 `OcrResult` 和预览图片；桌面环境仍然严格读取真实持久化 result，失败即抛错。
- 已新增 Playwright e2e `floating OCR result panel shows screenshot image blocks and translation guard`：
  - 直接进入 `/#/floating-panel`，验证独立 `ocr-result` floating panel，不经过主胶囊内联 fallback。
  - 验证结果标题、模型摘要、`RecordingFreedom` 和 `文字识别` 原文、预览图片、两个 polygon、两个 block row。
  - 验证鼠标移入 block row 后对应 polygon 进入 active 高亮。
  - 验证翻译按钮只显示 provider 未配置提示，不调用真实翻译。
- 已通过 `cd app/frontend && npm run build`。
- 已通过 `cd app/frontend && npm run test:e2e -- --grep "floating OCR result panel"`。
- 已通过 `cd app/frontend && npm run test:e2e -- --grep "real worker smoke evidence"`，验证 floating OCR result panel 可以渲染真实 worker smoke evidence 的图片尺寸和 block 坐标。
- 已通过 `cd app/frontend && npm run test:e2e -- --grep "screenshot history ready item opens OCR result floating panel"`，验证主胶囊截图历史 ready 项可以走桌面浮层调用路径打开 `ocr-result`。
- 仍未完成：真实桌面截图历史 ready 项从胶囊点击打开 `ocr-result` 浮层的录屏/e2e 证据；这一项需要在桌面 Wails 窗口中回验，不能用浏览器 fallback 代替最终桌面验收。

当前 R3 进度：

- 已新增 Playwright e2e `pinned screenshot window restores OCR highlight and result floating panel after resize`：
  - 从截图历史 ready 项点击钉图，验证 pinned state 保留当前 item 的 OCR sidecar 状态。
  - 打开 `/#/screenshot-pin` 后，钉图窗口自动读取持久化 OCR result，并显示 `2 text blocks`。
  - 点击 OCR 高亮按钮后显示两个 polygon，SVG `viewBox` 保持原图 `640 x 360` 坐标。
  - hover/click block polygon 会进入 active/copy 状态，证明 polygon 可交互且不只是静态装饰。
  - 改变窗口 viewport 后，画布尺寸变大但 SVG `viewBox` 和 polygon 数量保持不变，证明高亮仍按原图坐标映射。
  - 点击钉图内 OCR 结果按钮会打开 `ocr-result` floating panel，并携带同一个 `ocrResultId`。
- 已新增 Playwright e2e `pinned screenshot window renders real worker smoke evidence after resize`：
  - 使用真实 worker smoke evidence 的 `900x280`、`ppocrv5-mobile-zh-en`、`RecordingFreedom`、`文字识别` 和两个浮点 polygon 坐标。
  - 从截图历史钉图后打开 `/#/screenshot-pin`，验证钉图窗口展示的 OCR overlay `viewBox=0 0 900 280`。
  - 断言第一个 polygon points 精确等于真实 worker 输出坐标，并在窗口 resize 后保持 polygon 数量和 viewBox 不变。
- 已通过 `cd app/frontend && npm run test:e2e -- --grep "pinned screenshot window restores OCR"`。
- 已通过 `cd app/frontend && npm run test:e2e -- --grep "pinned screenshot window renders real worker smoke evidence"`。
- 已通过 `cd app && go test ./cmd/release-config-check`。
- 已通过 `cd app && go run ./cmd/release-config-check`，release gate 已固定 R3 真实 worker evidence 钉图 e2e 测试名。
- 仍未完成：真实 Wails 钉图窗口的桌面视觉/e2e 证据；浏览器 e2e 只能证明 React/CSS/状态链路，不能替代真实窗口透明/resize 行为。

当前 R4 进度：

- 已修复白板选中图片状态链路：`onSceneChange` 会缓存当前选中图片的 `elementId/fileId/dataURL`，点击选中图片 OCR 时优先使用这份已验证数据，避免按钮启用但 Excalidraw API 实时选择态丢失导致无法排队。
- 已新增浏览器/e2e 专用白板 OCR queued fallback：只记录请求并返回 queued snapshot，不生成 OCR ready 结果、不输出假文字；Wails 桌面环境仍然必须走真实后端。
- 已新增 Playwright e2e `whiteboard selected image OCR queues its own exported image as whiteboard-selection`：
  - 预置一个已选中的 Excalidraw 图片元素。
  - 点击 `Recognize selected image` 后，验证白板导出独立 PNG：`browser-preview/data/whiteboards/exports/whiteboard.png`。
  - 验证排队请求为 `sourceKind=whiteboard-selection`、`elementId=selected-image-element`、`language=zh-en`、`priority=interactive`。
  - 验证 OCR queued 文案不会被自动保存的 `Unsaved` 覆盖。
- 已新增 Playwright e2e `whiteboard selected image OCR opens result panel with its exported image`：
  - `whiteboard-selection` ready 后点击 `View board OCR result`，验证打开独立 `ocr-result` floating panel。
  - 结果浮层读取选中图片自己的导出图，SVG `viewBox=0 0 320 180`，不是整张画板或截图历史 fallback。
  - 第一个 polygon points 精确等于选中图片 result 坐标 `32,18 160,18 160,54 32,54`，证明结果面板消费的是选中图坐标。
- 已新增 Playwright e2e `whiteboard selected image OCR result panel renders real worker smoke evidence coordinates`：
  - 使用真实 worker smoke evidence 的 `region.png` 尺寸 `900x280` 和真实 OCR block 坐标，但按 `sourceKind=whiteboard-selection` 注入结果。
  - 点击 `View board OCR result` 后验证结果浮层 `viewBox=0 0 900 280`，并精确匹配两个真实 worker block polygon points。
  - 该测试证明白板选中图片结果面板消费的是选中图片自己的 `imagePath/imageWidth/imageHeight/blocks`，不是整张画板 fallback，也不是前端伪造坐标。
- 已修复后端事件回写遗漏：`handleOCRJobEvent` 现在会处理 `whiteboard-selection` ready/failed/cancelled，历史 sidecar 不会停在 queued；同时 `ReadOcrResultImage` 仍禁止 `whiteboard-selection` 缺少 `imagePath` 时 fallback 到整张画板。
- 已新增 Go 回归测试 `TestWhiteboardSelectionOCRJobEventUpdatesHistory`，直接模拟 `whiteboard-selection` ready 事件并验证截图历史 OCR sidecar 进入 ready。
- 已新增真实 worker 服务层 smoke `TestWhiteboardSelectionOCRRealWorkerSmoke`：
  - 使用 stable `ppocrv5-mobile-zh-en`、真实 `rf-ocr-worker`、真实 ONNX Runtime 和模型包 smoke 图。
  - 通过 `QueueRecognizeWhiteboard` 提交 `sourceKind=whiteboard-selection`、`elementId=whiteboard-selection-real-worker-image`、`language=zh-en`、`priority=interactive`。
  - 验证历史 ready、`OpenOcrResult` source/imagePath/dimensions、`ReadOcrResultImage`、`RecordingFreedom`、`文字识别` 和非空 blocks。
  - 生成 `release-out/ocr-smoke-evidence/whiteboard-ocr-real-worker-smoke.json`，并由 `ocr-smoke-evidence-check` 校验 `whiteboard-selection-real-worker` 场景。
- 已通过 `cd app/frontend && npm run test:e2e -- --grep "whiteboard selected image OCR"`。
- 已通过 `cd app/frontend && npm run test:e2e -- --grep "whiteboard selected image OCR opens result panel"`。
- 已通过 `cd app/frontend && npm run test:e2e -- --grep "whiteboard selected image OCR result panel renders real worker smoke evidence coordinates"`。
- 已通过 `cd app && go test . -run "Test(WhiteboardOCRRequestUsesSelectionSourceForElement|QueueRecognizeWhiteboardSnapshotUsesWhiteboardSourceAndUpdatesHistory|WhiteboardSelectionOCRJobEventUpdatesHistory|ReadOcrResultImageRejectsSelectionFallbackWithoutImagePath|WhiteboardSelectionOCRRealWorkerSmoke)$" -count=1`。
- 已通过 `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\run-local-ocr-smoke.ps1`，真实 worker/model smoke 生成并校验 `screenshot-ocr-real-worker-smoke.json` 和 `whiteboard-ocr-real-worker-smoke.json`。
- 仍未完成：真实 Wails 桌面白板选中图片 OCR ready 回验，包括真实桌面导出的图片、真实 worker result 的 `imagePath`、结果浮层预览图和 block 坐标一致性。当前新增服务层 smoke 证明真实 worker/队列/sidecar/result/image 链路，仍不能替代桌面窗口、真实 Excalidraw 导出和 Wails 多窗口验收。

当前 R5 进度：

- 已在白板窗口保存 OCR ready result，并增加画板工具栏操作：
  - 显示/隐藏 OCR block。
  - 插入识别原文为 Excalidraw 文本元素。
- OCR block 不使用 DOM 悬浮层，而是写入 Excalidraw scene：
  - block 使用 rectangle 元素。
  - 坐标从 OCR 图片像素坐标映射到选中图片元素坐标。
  - 元素带 `customData.recordingFreedomOcr.kind=block`，便于后续隐藏和持久化识别。
  - 隐藏 block 只移除 OCR block 元素，不删除用户绘制内容和已插入文本。
- OCR 原文插入使用 Excalidraw text 元素：
  - 元素带 `customData.recordingFreedomOcr.kind=text`。
  - 默认插入在识别图片元素下方。
  - 后续 G13 可复用同一插入路径插入译文。
- 已新增浏览器/e2e OCR job 事件 fallback：
  - 只用于 Playwright 注入 ready 事件验证 UI 链路。
  - 不生成假 OCR 结果，不改变 Wails 桌面路径；桌面仍必须由后端 worker 产生真实 `ocr.job.*` 事件。
- 已修复 OCR 操作状态文案保护：
  - 自动保存仍然执行。
  - OCR ready、显示/隐藏 block、插入文本后的状态短时间内不会被 `Unsaved/Saved` 抢走。
- 已新增 Playwright e2e `whiteboard OCR blocks map to the selected image and recognized text inserts as scene text`：
  - 预置一个选中图片元素。
  - 注入 `whiteboard-selection` ready 结果。
  - 验证 `RecordingFreedom` block 从 `320x180` 图片坐标正确映射到画板图片元素坐标。
  - 验证插入文本元素包含 `RecordingFreedom\n文字识别`。
  - 验证隐藏 OCR block 后，block 元素被移除，但 OCR text 元素仍保留。
- 已新增 Playwright e2e `whiteboard OCR blocks and recognized text persist after reopen`：
  - 注入 `whiteboard-selection` ready 结果后显示 OCR block，并插入识别原文。
  - 验证 scene 持久化内容包含 `recordingFreedomOcr` 自定义元数据。
  - 重新打开白板后再次读取同一 scene，验证 2 个 block 和 1 个 OCR text 元素仍存在，坐标和文本不丢。
- 已通过 `cd app/frontend && npm run build`。
- 已通过 `cd app/frontend && npm run test:e2e -- --grep "whiteboard OCR blocks"`。
- 已通过 `cd app/frontend && npm run test:e2e -- --grep "whiteboard OCR blocks and recognized text persist"`。
- 仍未完成：真实桌面 ready 结果回验、保存/关闭/重新打开后的桌面视觉回验、真实桌面录制中白板/annotation-overlay OCR 并行安全回验；译文插入同一文本元素路径已接入但仍需真实桌面回验。

当前 R6 进度：

- 已新增 Playwright e2e `whiteboard OCR failure during an audio recording does not stop the recorder shell`：
  - 主胶囊进入音频录制状态。
  - 音频录制中打开独立白板。
  - 白板选中图片 OCR 进入 queued。
  - 注入 `whiteboard-selection` failed 事件。
  - 验证失败原因只显示在白板状态栏。
  - 验证主胶囊仍保持 recording compact 状态，并且仍可点击结束录制恢复到 start 状态。
- 已把视频录制中的 `annotation-overlay` 接入同一 OCR 队列：
  - annotation 胶囊新增 OCR 识别和打开结果入口。
  - 点击识别时先强制保存当前 Excalidraw scene 和透明 PNG snapshot，输入图片为录制包内 `annotations/snapshots/*.png`。
  - 录制中 annotation OCR 请求使用 `sourceKind=whiteboard`、`sourceId=packageDir`、`priority=background`，不伪装成截图历史项。
  - ready 后通过 `ocr-result` floating panel 打开真实 result，仍不扩大主胶囊或 annotation overlay。
- 已新增 Go 单测：
  - `TestWhiteboardOCRRequestAllowsBackgroundPriorityForRecordingAnnotation`
  - `TestRecordingAnnotationSnapshotQueuesBackgroundWhiteboardOCR`
- 已新增 Playwright e2e `recording annotation overlay queues background OCR and opens result panel`：
  - 验证 annotation overlay 保存的包内 snapshot 进入 `whiteboard` OCR 队列。
  - 验证优先级为 `background`。
  - 注入 ready 事件后，annotation overlay 可打开同一个 `ocr-result` floating panel。
- 已把 `Frontend e2e covers recording annotation OCR safety` 加入 `release-config-check` 防回归。
- 已通过 `cd app/frontend && npm run test:e2e -- --grep "whiteboard OCR failure during an audio recording"`。
- 已通过 `cd app && go test . -run "TestRecordingAnnotationSnapshotQueuesBackgroundWhiteboardOCR|TestWhiteboardOCRRequestAllowsBackgroundPriorityForRecordingAnnotation|TestQueueRecognizeWhiteboardSnapshotUsesWhiteboardSourceAndUpdatesHistory" -count=1`。
- 已通过 `cd app/frontend && npm run test:e2e -- --grep "recording annotation overlay queues background OCR"`。
- 已通过 `cd app/frontend && npm run build`。
- 已通过 `cd app && go test ./cmd/release-config-check`。
- 已通过 `cd app && go run ./cmd/release-config-check`。
- 仍未完成：
  - 真实 Wails 桌面录制中运行 OCR worker 的并行日志证据。
  - 视频录制中的 `annotation-overlay` OCR 已有后端合同和浏览器/e2e 证据，但还没有真实 Wails 桌面录制期间的 OCR ready/failed 事件流证据。
  - OCR 失败不影响真实录制包写盘、鼠标、绘制事件的桌面录屏或桌面 e2e 证据。

当前 R7 进度：

- 已在 `app/internal/ocr` 增加服务层滚动长图 tile OCR：
  - 只对 `sourceKind=scrolling-screenshot` 生效。
  - 只读取最终保存的 PNG/JPEG/WebP 图片，不读取滚动过程中的中间帧。
  - 图片高度超过单次 worker `maxSide` 时进入 tile 模式。
  - tile 高度固定为 `2200px`，重叠高度固定为 `120px`。
  - tile 临时 PNG 写入系统临时目录，识别完成后清理。
  - 每个 tile 独立调用同一个 `rf-ocr-worker` `recognize` 协议。
  - tile block 的 `x/y` 坐标会加回 tile origin，最终 result 坐标仍是完整长图像素坐标。
- 已实现重叠区域去重：
  - 同文本 block 先按完整长图坐标排序。
  - box IoU 达到阈值时去重。
  - 或者纵向边界接近且横向重叠足够高时去重。
  - 去重后重新生成 `lineIndex` 和 `plainText`。
- 已保持 cache 合同：
  - cache key 仍使用完整长图 `imageSha256 + modelId + language`。
  - 同一长图二次识别命中 cache 时会克隆为当前 `sourceKind/sourceId/imagePath`，不要求 worker 存在。
- 已新增 Go 单测 `TestRecognizeScrollingScreenshotUsesTilesAndDedupesOverlap`：
  - 使用 `320x4520` 长图。
  - 验证 worker 被调用 3 次。
  - 验证 tile-2 的 block y 坐标从局部 `40` 回填为完整长图 `2120`。
  - 验证重叠区域只保留 `overlap-2080` 和 `overlap-4160` 各一条。
  - 验证二次识别走完整长图 cache。
- 已新增 Go 单测 `TestCaptureScrollingScreenshotStaticTargetQueuesRegionOCR`：
  - 用注入式 capture/scroll 模拟目标区域无法滚动但滚动自动化已尝试。
  - 验证保存的截图 item `mode=region`、尺寸保持直接区域截图 `80x120`，不保存假长图。
  - 验证后续 `QueueRecognizeScreenshot` 生成 `region-screenshot` OCR 请求，并把历史项状态更新为 queued。
- 已扩展真实 stable worker smoke `TestScreenshotOCRRealWorkerSmoke`：
  - 新增 `scrolling-long` 场景，使用模型包内 `smoke.png` 竖向重复生成 `900x2800` 长图，确保超过 worker `maxSide=2400` 并触发 R7 tile 聚合路径。
  - 验证真实 OCR result 仍为 `sourceKind=scrolling-screenshot`、`mode=scrolling`、图片尺寸保持完整长图，`plainText` 包含 `RecordingFreedom` 和 `文字识别`。
  - `scripts/run-local-ocr-smoke.ps1` 已重新通过并写出 `release-out/ocr-smoke-evidence/scrolling-long.png`、`scrolling-long-ocr-overlay.png` 和 evidence JSON；当前 `scrolling-long` evidence 为 `900x2800`、`blockCount=20`、`cacheHitWithoutWorker=true`。
  - `ocr-smoke-evidence-check` 已把 `scrolling-long imageHeight > 2400` 固定为 evidence 验收条件，防止后续长图 smoke 退化成普通短图。
- 已通过 `cd app && go test ./internal/ocr`。
- 已通过 `cd app && go test . -run "TestCaptureScrollingScreenshotStaticTargetQueuesRegionOCR|TestCaptureScrollingScreenshotImageFallsBackToDirectShotForStaticTarget" -count=1`。
- 已通过 `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\run-local-ocr-smoke.ps1`，真实 worker/model 长图 smoke 通过。
- 已通过 `cd app && go test ./...`。
- 仍未完成：
  - 使用更多真实网页/应用长滚动截图样例观察 tile 高度/overlap 对文本质量和重复率的影响。
  - 桌面滚动截图 UI 入口保存最终长图后，自动/手动 OCR 的视觉/e2e 证据。
  - 真实桌面无滚动目标的区域选择、保存、OCR ready 视觉/e2e 证据；当前 Go 单测已覆盖保存和 OCR sourceKind 合同，但不能替代真实窗口/滚动自动化回验。

当前 R8 进度：

- 已在后端 `TranslateOcr` 接入明确 provider 配置：
  - provider 为空或 `disabled` 时直接失败，不访问网络。
  - `deepl` 必须显式提供 base URL 和 API key。
  - `openai-compatible` 必须显式提供 base URL、API key 和 model。
- 已实现翻译分段合同：
  - 翻译只消费已经持久化的 `OcrResult`。
  - 可翻译全部 block 或指定 block id。
  - provider 返回必须和 OCR block id、顺序、段数一致。
  - 段数不一致、block id 错位或译文为空时整体失败，不写入成功缓存，不覆盖 OCR 原文。
- 已实现翻译缓存：
  - 缓存目录为 `data/ocr/translations/`。
  - cache key 覆盖 OCR result id、provider、base URL、source language、target language、model、promptVersion、block id 和原文。
  - `Force=false` 时相同请求命中缓存，不再调用 provider。
- 已同步 Wails TypeScript bindings：
  - `TranslateRequest` 已包含 `baseUrl`、`apiKey`、`model`、`force`。
  - `TranslationResult` 已包含 `model`、`promptVersion`。
- 已在 settings 合同中新增 `ocr.translation`：
  - 支持 provider、base URL、API key、model、source language、target language、隐私确认和确认时间。
  - 默认 provider 为 `disabled`，未确认隐私或配置缺失时不会发送任何 OCR 文本。
  - API key 不再写入 `settings.json`；settings 只保存 `apiKeySet`。
  - Windows secret 后端为 DPAPI 当前用户加密文件；macOS 后端为 Security.framework Keychain；Linux 后端优先使用 `org.freedesktop.secrets` Secret Service，并在 Secret Service 不存在时回退到 `0600` 权限本地 secret 文件。
  - 旧版 `settings.json` 中的明文 `apiKey` 会在服务读取/保存前迁移到 secret store，并在下一次保存时从 settings 中清除。
  - 日志只记录是否设置 key，不输出完整 key。
- 已在完整设置页和 floating settings 中新增 `OCR 翻译` 区块：
  - provider 只开放 `disabled`、`deepl`、`openai-compatible`。
  - endpoint、model 使用 settings 持久化；API key 通过 `PatchSettingsPreferences` 写入/删除 secret store，前端只看到 `apiKeySet`，不会回显已保存 key。
  - provider 切换、隐私确认和语言选择都走同一 settings 真状态，不在前端保存私有真状态。
- 已在 OCR 结果浮层接入真实 `TranslateOcr` 调用：
  - 翻译按钮先检查 provider、隐私确认、endpoint、API key 和 OpenAI-compatible model。
  - 成功后按 OCR block 展示译文，不覆盖原文。
  - 翻译成功后可复制整段译文。
  - 配置不完整时只显示本地保护提示，不调用 provider。
- 已在截图历史和钉图接入紧凑翻译动作：
  - ready OCR 历史项和钉图工具条都调用同一个 `translateAndCopyOcrResultText` helper。
  - helper 会先执行 provider/隐私/endpoint/API key/model 检查，未配置时只返回本地提示，不发送 OCR 文本。
  - 成功后复制整段译文，列表和钉图只显示结果消息；译文详情仍可通过 `ocr-result` floating panel 查看。
- 已在画板窗口接入译文复用：
  - 翻译按钮复用 settings 中的 provider/base URL/API key/model/source language/target language/隐私确认，不在白板窗口保存私有真状态。
  - 成功后通过 `buildOcrTextElement` 的同一 Excalidraw text 元素路径插入译文，原文插入和译文插入不会覆盖 OCR 原始 result。
  - 浏览器/e2e translation fallback 只在非 Wails 桌面环境中启用，用于验证 UI 链路；桌面环境仍必须走后端真实 provider 或返回明确错误。
- 已新增 Go 单测：
  - `TestTranslateDisabledProviderDoesNotCallNetwork`
  - `TestTranslateOpenAICompatibleWritesAndReadsCache`
  - `TestTranslateDeepLSegments`
  - `TestTranslateRejectsSegmentMismatchWithoutCache`
  - `TestTranslateOcrUsesStoredOCRTranslationAPIKey`
  - `TestTranslateOcrServiceUsesStoredKeyAndCachesOpenAICompatibleEndpoint`
- 本轮新增服务层可控 provider smoke：
  - 使用本地 `httptest` 模拟 OpenAI-compatible `/chat/completions`，不依赖外网真实 API。
  - 通过 `PatchSettingsPreferences` 保存 provider/base URL/model/privacy/API key，确认 `settings.json` 不包含明文 API key。
  - `TranslateOcr` 请求不传 `apiKey`，由桌面服务从 secret store 补齐，并验证 Authorization header。
  - 第一次调用写入翻译缓存；关闭 provider 后第二次调用仍成功，证明命中 `data/ocr/translations/`，没有再次联网。
- 已新增运行时翻译 smoke 命令 `app/cmd/ocr-translation-smoke`：
  - 使用真实 `appdata.Service`、`settings.Service`、`internal/secrets.Store` 和 `internal/ocr.Service`，先写入持久化 `OcrResult`，再从 secret store 读取 API key 后调用同一个 `ocr.Translate` provider/cache 实现。
  - 默认 `provider-mode=local` 使用本地 `httptest` OpenAI-compatible provider，会校验 `/chat/completions`、`Authorization: Bearer ...`、model、target language、`RecordingFreedom`、`文字识别` 和 block id token。
  - 新增 `provider-mode=external-openai-compatible`，可用 `-base-url` 或 `RF_OCR_TRANSLATION_BASE_URL` 指向真实 OpenAI-compatible provider，只从 `-api-key-env` 指定的环境变量读取 key，默认变量为 `RF_OCR_TRANSLATION_API_KEY`；命令行参数不接收明文 key。
  - 第一次翻译默认 `Force=true`，避免复用旧 data root 时只靠缓存通过；第二次会清空请求中的 API key 并命中 `data/ocr/translations/` 缓存，证明缓存命中发生在 provider key 校验和网络请求之前。
  - smoke 会写出 `release-out/ocr-translation-smoke/translation-smoke.json`，证据包含 providerMode、provider、脱敏 providerBaseUrl、model、apiKeyEnv、apiKeyProvided、source/target language、blockCount、providerRequestCountKnown、providerRequestCount、providerAuthHeaderOk、providerRequestVerified、cacheHitAfterProviderDown、externalProviderCacheHitWithoutApiKey、translationFiles 和 translatedBlocks 的长度/状态信息，但不写入 API key 明文、完整 OCR 原文或完整译文。
  - CI 与 Release Gate 已新增 `Run OCR translation smoke`，执行默认离线 local provider，不依赖外网；真实外部 provider 诊断必须由人工或受控环境显式传 `-provider-mode external-openai-compatible`。
  - `ocr-translation-smoke` 已纳入 CI/Release desktop smoke tools 构建、Windows portable、macOS app bundle 和 Linux portable 包 staging；三端 verifier 会检查工具存在和架构，Windows portable smoke runner 会运行离线 `translation-smoke` step。
  - 当前 Windows 工作区已实跑 `cd app && go run ./cmd/ocr-translation-smoke`，输出 `ok=true`、`schemaVersion=2`、`providerMode=local`、`secretBackend=windows-dpapi`、`settingsJsonContainsRawKey=false`、`providerRequestCount=1`、`providerRequestPath=/chat/completions`、`cacheHitAfterProviderDown=true`，并生成 `translation-smoke.json`。
  - `release-config-check` 已固定命令文件、外部 provider 参数、CI/Release 构建、三端 staging、三端 verifier 和 Windows portable smoke runner，防止翻译 smoke 退回源码-only 或本地 mock-only 证据链。
- 已新增平台 secret store smoke 命令 `app/cmd/ocr-secret-store-smoke`：
  - 使用真实 `internal/secrets.Store` 执行 `Save`、`Load`、`Delete` 和删除后再次 `Load`，不会走 mock secret store。
  - 生成 `release-out/ocr-secret-store-smoke/secret-store-smoke.json`，记录 `goos/goarch`、`secretBackend`、`secretStatus`、saved/loaded/deleted、`loadAfterDeleteFound=false`、`rawSecretInDataRoot=false` 和扫描文件数；证据不包含 secret 明文。
  - 保存后会扫描当前 data root，发现 raw secret 明文即失败；删除后仍能读到 secret 也会失败。
  - 当前 Windows 工作区已实跑 `cd app && go run ./cmd/ocr-secret-store-smoke`，输出 `ok=true`、`goos=windows`、`goarch=amd64`、`secretBackend=windows-dpapi`、`saved=true`、`loaded=true`、`deleted=true`、`loadAfterDeleteFound=false`、`rawSecretInDataRoot=false`。
  - CI 与 Release Gate 的 validate 阶段已新增 `Run OCR secret store smoke`；Windows/macOS/Linux x64/arm64 desktop build matrix 已新增 `Run platform OCR secret store smoke`，下一次 Actions 会在每个目标宿主执行同一 save/load/delete smoke。
  - CI desktop build matrix 会上传 `recordingfreedom-ocr-secret-store-smoke-${{ matrix.name }}` artifact，保留对应平台的 `release-out/ocr-secret-store-smoke/${{ matrix.name }}/secret-store-smoke.json`，用于后续确认 Windows DPAPI、macOS Keychain 或 Linux fallback/Secret Service 的真实运行证据。
  - Release desktop artifact 会随包携带 `ocr-secret-store-smoke` 诊断工具：Windows portable 为 `tools/ocr-secret-store-smoke.exe`，macOS app bundle 为 `Contents/MacOS/tools/ocr-secret-store-smoke`，Linux portable 为 `tools/ocr-secret-store-smoke`；对应平台 verifier 会检查工具存在、可执行权限和架构。
  - Windows portable clean-machine runner 已增加 `secret-store-smoke` step，会在解压后的目标目录运行 `tools/ocr-secret-store-smoke.exe -data-dir <portable>/data-smoke -evidence-dir <portable>/data-smoke/secret-store-smoke`，把 secret store 证据与其他录制 smoke 诊断一起留在 portable 目录下。
  - 该 smoke 在 Linux 无 Secret Service/DBus 的 runner 上会记录 `linux-secret-service+local-file-0600-fallback`，只能证明 fallback 可用；不能把它冒充为 GNOME/KDE Secret Service 实机回验。
- 已新增前端 e2e：
  - `settings persists explicit OCR translation provider configuration`
  - `screenshot history translates ready OCR text through the configured provider`
  - `pinned screenshot window restores OCR highlight and result floating panel after resize`
  - `whiteboard translated OCR text inserts as scene text after explicit provider configuration`
- 已通过 `cd app && go test ./internal/ocr`。
- 已通过 `cd app && go test ./internal/secrets ./internal/settings`。
- 已通过 `cd app && go test . -run "TestTranslateOcr(UsesStoredOCRTranslationAPIKey|ServiceUsesStoredKeyAndCachesOpenAICompatibleEndpoint)$" -count=1`。
- 已通过 `cd app && go test ./...`。
- 已通过 `cd app/frontend && npm run build`。
- 已通过 `cd app/frontend && npm run test:e2e -- --grep "OCR translation provider"`。
- 已通过 `cd app && go run ./cmd/release-config-check`，release gate 已固定 service-level secret storage 测试名。
- 已通过 `cd app && go test ./cmd/ocr-translation-smoke -count=1`。
- 已通过 `cd app && go run ./cmd/ocr-translation-smoke`，生成 `release-out/ocr-translation-smoke/translation-smoke.json`。
- 已通过 `cd app && go test ./cmd/ocr-translation-smoke ./cmd/release-config-check -count=1`，覆盖 local provider、安全 external OpenAI-compatible provider 参数、缺 key 失败、证据不泄露 key/完整译文和 release gate fixture。
- 已通过 `cd app && go run ./cmd/release-config-check`，真实仓库报告 `ok=true`，翻译 smoke 发布包工具链门禁为 `ready`。
- 已通过 PowerShell Parser 静态解析 `scripts/run-windows-portable-smoke.ps1` 和 `scripts/verify-windows-portable.ps1`。
- 已通过 `bash -n scripts/verify-macos-app-zip.sh && bash -n scripts/verify-linux-portable.sh`。
- 已通过 `cd app && go test ./cmd/ocr-secret-store-smoke -count=1`。
- 已通过 `cd app && go run ./cmd/ocr-secret-store-smoke`，生成 `release-out/ocr-secret-store-smoke/secret-store-smoke.json`。
- 仍未完成：
  - 使用真实 DeepL 或 OpenAI-compatible provider 的桌面回验。
  - macOS Keychain 实机写入/读取/删除回验仍需下一次 macOS Actions 或目标 Mac 生成 `secret-store-smoke.json` 后确认；当前 Windows 工作区只能证明 Windows DPAPI 和 workflow gate，不替代 macOS 钥匙串运行证据。
  - Linux Secret Service 实机写入/读取/删除回验仍需真实 GNOME/KDE Secret Service 环境；CI runner 若走 local-file fallback，只能证明 fallback 可用，不替代 Secret Service 实机证据。
  - 截图历史、钉图、画板在真实桌面窗口中打开结果浮层后的译文视觉证据。

当前 R9 进度：

- 已在设置页和 floating settings 中接入本地 OCR 模型管理：
  - 调用 `GetOcrStatus` 读取后端真实状态。
  - 展示 stable/latest/quality 三个 registry 模型。
  - 展示 installed、verified、active、missing files、verification error、smoke asset、worker path 和 runtime path。
- 已接入真实后端动作：
  - 本地模型包 `.zip` 或解压目录通过 `InstallOcrModelPackage` 导入。
  - 已安装且 verified 的非 active 模型通过 `SetActiveOcrModel` 切换。
  - 已安装的非 active 模型通过 `RemoveOcrModel` 删除。
  - 每次动作完成后重新读取后端状态，不在前端保存私有真状态。
- 已接入 app 内模型下载器合同和前端进度：
  - 后端新增 `StartOcrModelDownload`、`CancelOcrModelDownload`、`GetOcrModelDownloads`。
  - 下载器只接受 registry 中带 `package.url/package.bytes/package.sha256` 的 RecordingFreedom verified package；没有完整 package metadata 时，UI 明确显示“暂无已校验下载包”，不会展示假下载。
  - 下载写入 `.download-*` staging 文件，完成 bytes/SHA256 校验后复用 `InstallModelPackage` 做 manifest/smoke/atomic install。
  - 下载事件通过 `ocr.model.download.changed` 回填 queued/running/installed/failed/cancelled、进度、错误原因和安装后的模型状态。
  - 前端设置页和 floating settings 显示下载、取消、百分比、大小和失败原因；安装完成后不自动切换 active，仍必须用户点击 `Use` 并确认。
  - 浏览器 fallback 只用于 e2e 验证 UI 状态，不替代 Wails 桌面真实下载能力。
- 已补下载取消后端回归：
  - `TestCancelModelDownloadStopsTransferWithoutInstall` 使用可控 HTTP 下载流，在传输中调用 `CancelModelDownload`。
  - 验证最终状态为 `ModelDownloadCancelled`，不会安装目标模型目录。
  - 验证 `.download-*` staging 文件被清理，取消不会污染后续重试。
- 已补取消后重试后端回归：
  - `TestRetryModelDownloadAfterCancelInstallsWithoutAutoSwitch` 使用同一个可控 HTTP package endpoint，第一次请求只传输部分 zip 后取消，第二次重新发起完整下载。
  - 验证 retry 会生成新的 download id，重新下载后安装 verified 模型，并且 `.download-*` staging 仍被清理。
  - 验证 retry 安装后不会自动切换 active model，仍保持默认 stable active，防止用户取消后重新下载导致模型被静默切换。
- 已接入 release catalog 刷新链路：
  - 后端新增 `RefreshModelCatalog` 和 Wails `RefreshOcrModelCatalog`。
  - 默认目录为 `https://github.com/lemon-casino/RecordingFreedom/releases/latest/download/ocr-model-catalog.json`。
  - catalog 下载只接受 HTTPS；测试环境只允许 loopback HTTP。
  - catalog 只能声明 `defaultModelRegistry()` 中已知模型 ID，不允许第三方未知模型注入。
  - 每个 catalog model 必须带完整 `package.url/package.sha256/package.bytes` 和 `det.onnx/cls.onnx/rec.onnx/keys.txt` 清单。
  - 校验通过后保存到 `data/models/ocr/registry.json`，后续 `GetOcrStatus` 会合并该 registry 并暴露真实下载按钮。
  - 设置页新增 `刷新目录 / Refresh catalog`，刷新成功只更新可下载元数据，不会自动下载、安装或切换 active model。
- 已把 release catalog 生成纳入发布门禁：
  - `app/cmd/ocr-model-package` 支持 `-model all`、`-catalog-output` 和 `-release-base-url`。
  - `-model all` 会按 manifest 中完整可验证的模型规格逐个生成 RecordingFreedom 模型包，并把每个生成包的真实 package URL、bytes、SHA256 写入同一个 `ocr-model-catalog.json`；当前 manifest 只有 stable 具备完整文件 SHA/smoke，因此 catalog 仍只发布 stable，不会把 latest/quality placeholder 暴露成可安装包。
  - 下载上游模型文件时已加入不完整下载重试；CDN 中途 EOF、bytes 不足或 hash 不匹配会清理 partial 文件并重试，最终仍以 bytes/SHA256 校验为准。
  - release workflow 和 CI model smoke 都改为 `go run ./cmd/ocr-model-package -model all ...`；stable smoke 仍必须解压 `ppocrv5-mobile-zh-en` 并通过 worker `--smoke`。
  - release artifact `RecordingFreedom-ocr-models` 会携带所有已生成的 verified 模型包、`ocr-model-catalog.json` 和 `SHA256SUMS-ocr-models.txt`。
  - release-config-check 会检查 workflow 使用 `-model all`、模型包生成器支持 `selectedModelIDs/packageModel/writeDownloadCatalog`、catalog e2e 和 Go catalog 单测，防止回退到单模型或假下载元数据。
- 已新增 PP-OCRv6 latest/quality 源审计工具和防回归门禁：
  - `app/cmd/ocr-model-source-audit` 会联网审计官方 Hugging Face PP-OCRv6 det/rec ONNX 源，并输出 `release-out/ocr-model-source-audit/ppocrv6-source-audit.json`。
  - 默认审计只读取模型 API 和 `inference.yml` 的轻量信息；需要准备 manifest 级证据时可显式传 `-hash-files`，审计工具会流式下载已存在的 `inference.onnx`、`inference.yml`、`README.md` 并记录每个文件的 bytes/SHA256 到 `fileHashes`，不会把大 ONNX 文件一次性读入内存。
  - latest 候选 `ppocrv6-mobile-zh-en` 对应 `PaddlePaddle/PP-OCRv6_small_det_onnx` commit `28fe5895c24fd108c19eb3e8479f4ab385fbfc62` 与 `PaddlePaddle/PP-OCRv6_small_rec_onnx` commit `b8f84f0b80c529de40b4fbb3544b84fa7233a513`，license 均为 `apache-2.0`，rec `character_dict` 为 18708 项，预期 CTC class 数为 18709；审计工具已从 `inference.yml` 生成包内 `keys.txt` 证据，bytes 为 `74947`，SHA256 为 `b5f2bfe2bdd9448429e3e82b51c789775d9b42f2403d082b00662eb77e401c5d`。
  - quality 候选 `ppocrv6-medium-zh-en` 对应 `PaddlePaddle/PP-OCRv6_medium_det_onnx` commit `61323801669c338b7891481ec7bac61ce31b576a` 与 `PaddlePaddle/PP-OCRv6_medium_rec_onnx` commit `50c7eacafc52fa7bcf4194e8cd08e46f8558504b`，license 均为 `apache-2.0`，rec `character_dict` 为 18708 项，预期 CTC class 数为 18709；审计工具已从 `inference.yml` 生成包内 `keys.txt` 证据，bytes 为 `74947`，SHA256 为 `b5f2bfe2bdd9448429e3e82b51c789775d9b42f2403d082b00662eb77e401c5d`。
  - `-hash-files` 已生成官方文件级 evidence，写入 `ppocrv6-source-audit.json` 的 `fileHashes`；README 哈希也在 JSON 中保留，但 manifest 草案只应使用以下 ONNX/YAML/生成 keys 证据：

    | 候选 | 源 | 文件 | bytes | SHA256 |
    | --- | --- | --- | ---: | --- |
    | latest | small det | `inference.onnx` | 9880512 | `d73e0058b7a8086bbd57f3d10b8bcd4ff95363f67e06e2762b5e814fe9c9410e` |
    | latest | small det | `inference.yml` | 885 | `193f435274bf9f0b5f71a929bbfbcf148282df7e633b34e7c373e8f44741b516` |
    | latest | small rec | `inference.onnx` | 21159378 | `5435fd747c9e0efe15a96d0b378d5bd157e9492ed8fd80edf08f30d02fa24634` |
    | latest | small rec | `inference.yml` | 150579 | `ab078671bb49f06228eadccd34f1bb501e157f7a047095ffb943ba81512c77d1` |
    | latest | generated rec keys | `keys.txt` | 74947 | `b5f2bfe2bdd9448429e3e82b51c789775d9b42f2403d082b00662eb77e401c5d` |
    | quality | medium det | `inference.onnx` | 62032837 | `eb13b44b25bb36f89528b68720af8a61d9cf381176107f465db1757b65d086e1` |
    | quality | medium det | `inference.yml` | 886 | `7298d5ead546584af2504d03355f881ac7a7bc0eb1e282d3e159277c1d0af871` |
    | quality | medium rec | `inference.onnx` | 76554979 | `9c09abf0957f7968c7586464b7397b84ad2387a0497a351af40e9acc71b673ba` |
    | quality | medium rec | `inference.yml` | 150580 | `991b700facf5b50a7de193468207d5f4255b538dde0d312ae3b7c7a9b6873129` |
    | quality | generated rec keys | `keys.txt` | 74947 | `b5f2bfe2bdd9448429e3e82b51c789775d9b42f2403d082b00662eb77e401c5d` |

  - 当前两个 PP-OCRv6 候选均为 `readyForManifest=false`：官方 det/rec ONNX 仓库没有提供 `cls.onnx`；RecordingFreedom 已支持显式 `textlineOrientation.mode=none` 的 no-cls 合同，但 PP-OCRv6 manifest 还必须固定 no-orientation mode、真实文件哈希和 worker smoke；rec 的字符表嵌在 `inference.yml` 的 `character_dict` 中，需要生成与 class count 匹配的 `keys.txt` 并固定 bytes/SHA256；PP-OCRv6 det/rec 的预处理、后处理和阈值还必须通过 RecordingFreedom worker smoke。
  - release-config-check 已固定 `ocr-model-source-audit` 中的候选 ID、官方源仓库、`ppocrv6-source-audit.json`、`-hash-files`、`hashRemoteFile`、`fileHashes`、`RecCharacterCount`、`ExpectedRecClasses`、`cls.onnx`、`keys.txt`、`worker smoke` 和 `ReadyForManifest`，防止 latest/quality 在未完成兼容性证据前被误加入可安装 catalog。
- 已新增 PP-OCRv6 `keys.txt` 生成合同：
  - `app/cmd/ocr-model-package` 的 `sourceFile.generate` 支持 `paddleocr-character-dict-keys`，可从已固定 bytes/SHA256 的 PaddleOCR `inference.yml` 中解析 `PostProcess.character_dict`，生成 RecordingFreedom worker 需要的包内 `keys.txt`；`app/cmd/ocr-model-source-audit` 也会在 `ppocrv6-source-audit.json` 的 rec source 中写入同口径 `generatedKeys` evidence，防止后续重新使用旧的缩进计数或手写字符表。
  - 生成链路会先校验源 `inference.yml` 的 `sourceBytes/sourceSha256`，再校验生成后 `keys.txt` 的 bytes/SHA256；不允许未固定哈希的上游 YAML 或生成结果进入模型包。
  - 当前 stable 包不受影响，仍直接下载和校验已有 `keys.txt`；PP-OCRv6 latest/quality 后续进入 manifest 时，可以使用该生成合同补齐 `keys.txt`，但仍必须继续固定生成后 bytes/SHA256 并通过 worker smoke。
  - release-config-check 已固定 `paddleocr-character-dict-keys`、`generatePaddleOCRCharacterDictKeys`、`SourceBytes`、`SourceSHA256` 和 `character_dict`，防止模型包生成器退回到只能下载现成 `keys.txt`。
- 已新增 PP-OCRv6 candidate manifest gate：
  - `third_party/ocr-models/manifest.json` 已写入 `ppocrv6-mobile-zh-en` 和 `ppocrv6-medium-zh-en` 候选 manifest，包含官方 det/rec ONNX bytes/SHA256、rec `inference.yml` source bytes/SHA256、生成后 `keys.txt` bytes/SHA256，并显式设置 `textlineOrientation.mode=none`。
  - 两个 PP-OCRv6 条目都设置 `releaseStatus: candidate`；`app/cmd/ocr-model-package` 的 `-model all` 只选择 `releaseStatus` 为空或 `ready` 的模型，显式选择 candidate 也会返回 `cannot be packaged for release`，防止未经 worker smoke 的模型进入发布 catalog。
  - 已通过实际 `go run ./cmd/ocr-model-package -model all ...` 验证输出仍只有 `ppocrv5-mobile-zh-en-2025.10-hf-jingsongliu.zip` 和只包含 stable 的 `ocr-model-catalog.json`，stable zip SHA256 仍为 `4ac5749aad47c1ed7d5092758ac0eee4c4f08693a1a6c11f2584ab32508713d2`。
  - release-config-check 已固定 `releaseStatus: candidate`、`textlineOrientation.mode=none`、PP-OCRv6 官方 URL、`paddleocr-character-dict-keys` 和打包器 `modelIsReleaseReady` 过滤逻辑，防止候选模型被误发布。
- 已新增 PP-OCRv6 candidate 本地 smoke 入口：
  - `app/cmd/ocr-model-package` 新增 `-include-candidates`，只允许本地打包 `releaseStatus: candidate` 模型；该开关不能和 `-catalog-output` 同时使用，避免 candidate 进入 release catalog。
  - `scripts/run-local-ocr-smoke.ps1` 新增 `-IncludeCandidates`，会把当前 `$Model` 传给 `ocr-model-package -include-candidates`，并把同一个 `$Model` 传给 `ocr-smoke-evidence-check -expected-model`。
  - `app/cmd/ocr-smoke-evidence-check` 新增 `-expected-model`，默认仍为 `ppocrv5-mobile-zh-en`；只有脚本显式传入 candidate model id 时才接受 PP-OCRv6 evidence，防止 checker 被放宽成“不管什么模型都通过”。
  - `app/screenshot_test.go` 的真实 worker smoke fixture 已从安装包返回值读取实际 `modelId` 并激活，不再写死 `ppocrv5-mobile-zh-en`，同一 smoke 能验证 stable、latest candidate 和 quality candidate。
  - 已通过 `.\scripts\run-local-ocr-smoke.ps1 -Model ppocrv6-mobile-zh-en -IncludeCandidates -OutputRoot .\release-out\ocr-candidate-local-smoke`，生成 `ppocrv6-mobile-zh-en-2026.07-hf-paddle-small-candidate.zip`，zip SHA256 为 `677b62ba08c13cec72995c904976055badfb7a50f64898c0ecdfec0e1c726085`；worker `--smoke`、截图 region/full/window/focused-window/scrolling/scrolling-long、queued cache、白板选中图片和 evidence checker 全部 `ready`，识别文本为 `RecordingFreedom\n文字识别`。
  - 已通过 `.\scripts\run-local-ocr-smoke.ps1 -Model ppocrv6-medium-zh-en -IncludeCandidates -OutputRoot .\release-out\ocr-candidate-medium-local-smoke`，生成 `ppocrv6-medium-zh-en-2026.07-hf-paddle-medium-candidate.zip`，zip SHA256 为 `b69a566caa0900349b8d3b123f289aa7feba06823ec7d3650c63c9f100768a10`；worker `--smoke`、截图 region/full/window/focused-window/scrolling/scrolling-long、queued cache、白板选中图片和 evidence checker 全部 `ready`，识别文本为 `RecordingFreedom\n文字识别`。
- 已新增模型生命周期 smoke：
  - `internal/ocr.Service` 新增正式 `ServiceOptions` / `NewServiceWithOptions` / `ApplyOptions`，允许 smoke 工具配置真实 worker、ONNX Runtime、worker args/env/timeout 和模型 registry；命令行工具不再依赖测试私有字段或包内绕路。
  - 新增 `app/cmd/ocr-model-lifecycle-smoke`，参数包含 `-stable-package`、可重复的 `-candidate-package`、`-worker-path`、`-runtime-dir`、`-data-dir` 和 `-evidence-dir`。
  - smoke 会安装 stable 包并强制真实 worker 识别 stable `smoke.png`；安装每个 candidate 后验证 active model 仍为 stable，不允许安装后静默切换；确认切换 candidate 后强制真实 worker 识别 candidate `smoke.png`；删除当前 active candidate 后验证回退 stable，并再次强制真实 worker 识别 stable。
  - evidence 写入 `release-out/ocr-model-lifecycle-smoke/ocr-model-lifecycle-smoke.json`，记录每一步的 `activeModelId/status/installed/verified/smokeImage/resultId/plainText/blockCount/forceWorker`，并要求 `plainText` 同时包含 `RecordingFreedom` 和 `文字识别`。
  - 已通过 `cd app && go run ./cmd/ocr-model-lifecycle-smoke -worker-path ".\tools\ocr-worker\windows-amd64\rf-ocr-worker.exe" -runtime-dir ".\tools\onnxruntime\windows-amd64" -stable-package "..\release-out\ocr-models\ppocrv5-mobile-zh-en-2025.10-hf-jingsongliu.zip" -candidate-package "..\release-out\ocr-candidate-local-smoke\ocr-models\ppocrv6-mobile-zh-en-2026.07-hf-paddle-small-candidate.zip" -candidate-package "..\release-out\ocr-candidate-medium-local-smoke\ocr-models\ppocrv6-medium-zh-en-2026.07-hf-paddle-medium-candidate.zip" -evidence-dir "..\release-out\ocr-model-lifecycle-smoke"`，本机 Windows x64 输出 `ok=true`，最终 active model 为 `ppocrv5-mobile-zh-en`，两个 PP-OCRv6 candidate 均完成“安装不自动激活 -> 确认切换 -> 真实识别 -> 删除 active -> 回退 stable -> 再识别 stable”。
  - release-config-check 已固定 `ServiceOptions` 和 `ocr-model-lifecycle-smoke` 的关键步骤名，防止模型生命周期证据退回 UI 私有状态或只验证安装目录。
- 已新增文字行方向分类 manifest 合同：
  - `ModelManifest.textlineOrientation.mode` 支持 `cls` 和 `none`，空值默认按旧 stable 合同处理为 `cls`，保持 PP-OCRv5 stable 兼容。
  - `mode=cls` 时 catalog、安装校验、worker 预检和 ONNX session 加载仍要求 `cls.onnx`，识别时继续执行角度分类。
  - `mode=none` 时 catalog、安装校验和 worker 允许模型包不包含 `cls.onnx`，worker 只加载 `det.onnx` 与 `rec.onnx`，识别时 angle 固定为 0；该模式只适合 PP-OCRv6 这类准备按官方 `use_textline_orientation=false` 路线验证的模型，仍必须通过 RecordingFreedom smoke 后才能进入 release catalog。
  - release-config-check 已固定服务层 no-cls 安装/catalog 测试、worker no-cls 预检测试和模型包生成器 `RequiredModelFileNames/TextlineOrientation` 合同，防止回退到硬编码 `cls.onnx`。
- 已补 active model 切换确认和失败回退保护：
  - 前端点击非 active 模型的 `Use` 时只进入内联确认态，不会立刻调用 `SetActiveOcrModel`。
  - 用户点击 `Confirm switch` 后才调用后端切换。
  - 后端 `SetActiveModel` 仍先校验模型已安装且 verified，成功后才保存 state；失败不会修改当前 active model。
  - 前端失败后会重新读取 `GetOcrStatus`，显示真实 active model 和失败原因，不做乐观切换。
- 已补模型删除 active 状态回退保护：
  - `TestRemoveInactiveModelKeepsActiveModel` 验证删除非 active 模型时，当前 active model、持久化 state 和已安装 verified stable 模型都保持不变。
  - `TestRemoveActiveModelFallsBackToDefaultModel` 验证删除当前 active 模型时，后端会回退到默认 stable active model，并移除目标模型目录。
  - release-config-check 已固定这两个测试名，防止删除模型逻辑退回到 UI 私有状态或误清 active state。
- 已新增 i18n：
  - 简体中文和英文都覆盖模型状态、通道名、导入/下载/取消下载/切换/删除动作和错误提示。
- 已新增样式：
  - 模型管理区块在完整设置页和 floating settings 都保持紧凑，不扩大主胶囊窗口。
- 已新增前端 e2e：
  - `settings exposes local OCR model package management`
  - 覆盖设置入口、stable/latest/quality 展示、模型包路径输入和浏览器环境导入保护。
  - `settings downloads a verified OCR model package without auto-switching active model`
  - 覆盖 verified package 下载按钮、下载事件、安装完成不自动切换 active、仍需确认切换。
  - `settings refreshes the verified OCR model catalog before exposing downloads`
  - 覆盖未刷新前不展示下载按钮，刷新 release catalog 后才展示下载大小和下载按钮。
  - `settings confirms before switching the active OCR model`
  - 覆盖首次点击 `Use` 不调用切换、确认后才调用 `SetActiveOcrModel`、active model 更新为目标模型。
  - `settings keeps the current OCR model when a confirmed switch fails`
  - 覆盖确认切换失败时展示失败原因，当前 active model 保持原模型。
- 已通过 `cd app/frontend && npm run build`。
- 已通过 `cd app/frontend && npm run test:e2e -- --grep "OCR model"`。
- 已通过 `cd app && go test ./internal/ocr -run "TestStartModelDownload|TestInstallModelPackage" -count=1`。
- 已通过 `cd app && go test ./internal/ocr -run "TestRefreshModelCatalog|TestStartModelDownload|TestInstallModelPackage" -count=1`。
- 已通过 `cd app && go test ./internal/ocr -run "Test(StartModelDownload|CancelModelDownload|InstallModelPackage|RefreshModelCatalog)" -count=1`。
- 已通过 `cd app && go test ./internal/ocr -run "Test(StartModelDownload|CancelModelDownload|RetryModelDownload|InstallModelPackage|RefreshModelCatalog)" -count=1`。
- 已通过 `cd app && go test ./internal/ocr -run "Test(StartModelDownload|CancelModelDownload|RetryModelDownload|InstallModelPackage|RefreshModelCatalog|Remove.*Model)" -count=1`。
- 已通过 `cd app && go test ./cmd/ocr-model-package ./cmd/release-config-check`。
- 已通过 `cd app && go test ./cmd/ocr-model-package ./cmd/release-config-check -count=1`，覆盖 `-model all` 选择、多模型 catalog 输出和模型文件下载重试。
- 已通过 `cd app && go run ./cmd/ocr-model-package -model all -output ../release-out/ocr-models-multi-smoke -catalog-output ../release-out/ocr-models-multi-smoke/ocr-model-catalog.json -release-base-url https://github.com/lemon-casino/RecordingFreedom/releases/download/local-smoke -force`，生成 `ppocrv5-mobile-zh-en-2025.10-hf-jingsongliu.zip` 和 `ocr-model-catalog.json`；当前 catalog 只包含 stable，因为 latest/quality 仍是 `releaseStatus: candidate`，尚未完成跨平台 worker smoke、真实用户截图样例质量回验和桌面下载/切换/回退验收。
- 已通过 `cd app && go run ./cmd/ocr-model-source-audit -evidence-dir ../release-out/ocr-model-source-audit -timeout 12s`，生成 PP-OCRv6 small/medium det/rec 官方源审计 evidence；两个候选均保留为 `readyForManifest=false`。
- 已通过 `cd app && go test ./cmd/ocr-model-source-audit ./cmd/release-config-check -count=1`，覆盖 PP-OCRv6 source audit evidence 输出、`character_dict` 计数、manifest 阻塞项和 release-config-check 防回归门禁。
- 已通过 `cd app && go run ./cmd/ocr-model-source-audit -evidence-dir ../release-out/ocr-model-source-audit -timeout 60s -hash-files`，生成 PP-OCRv6 small/medium det/rec 官方 `inference.onnx`、`inference.yml`、`README.md` 文件级 bytes/SHA256 evidence；两个候选仍保留为 `readyForManifest=false`。
- 已通过 `cd app && go test ./cmd/ocr-model-source-audit ./cmd/release-config-check -count=1`，覆盖 `-hash-files` 文件级 bytes/SHA256 evidence 输出和 release-config-check 防回归门禁。
- 已通过 `cd app && go run ./cmd/ocr-model-package -model all -output ../release-out/ocr-models-candidate-gate-smoke -catalog-output ../release-out/ocr-models-candidate-gate-smoke/ocr-model-catalog.json -release-base-url https://github.com/lemon-casino/RecordingFreedom/releases/download/local-smoke -force`，验证 candidate manifest 不进入 release catalog，输出仍只有 stable 模型包。
- 已通过 `cd app && go test ./cmd/ocr-model-package ./cmd/ocr-model-source-audit ./cmd/release-config-check -count=1`，覆盖 candidate 过滤、PP-OCRv6 source audit 和 release-config-check 防回归门禁。
- 已通过 `.\scripts\run-local-ocr-smoke.ps1 -Model ppocrv6-mobile-zh-en -IncludeCandidates -OutputRoot .\release-out\ocr-candidate-local-smoke`，覆盖 PP-OCRv6 small/latest candidate no-orientation worker smoke、截图/滚动长图/白板真实 worker smoke 和 evidence checker。
- 已通过 `.\scripts\run-local-ocr-smoke.ps1 -Model ppocrv6-medium-zh-en -IncludeCandidates -OutputRoot .\release-out\ocr-candidate-medium-local-smoke`，覆盖 PP-OCRv6 medium/quality candidate no-orientation worker smoke、截图/滚动长图/白板真实 worker smoke 和 evidence checker。
- 已通过 `cd app && go test ./cmd/ocr-smoke-evidence-check ./cmd/ocr-model-package ./cmd/release-config-check -count=1`，覆盖 `-expected-model`、`-include-candidates` 和 release-config-check 防回归门禁。
- 已通过 `cd app && go test ./cmd/ocr-model-package -count=1`，覆盖从 PaddleOCR `inference.yml` 的 `character_dict` 生成 `keys.txt`、校验源 YAML 和生成结果 hash，并把生成的 `keys.txt` 写入模型 zip。
- 已通过 `cd app && go test ./internal/ocr ./cmd/ocr-worker ./cmd/ocr-model-package ./cmd/release-config-check -count=1`，覆盖 `textlineOrientation.mode=none` 时 catalog/安装包/worker 允许缺少 `cls.onnx`，以及默认模式仍拒绝缺少 `cls.onnx`。
- 已通过 `cd app && go test ./cmd/ocr-model-lifecycle-smoke ./internal/ocr -run TestNewServiceWithOptionsAppliesRuntimeOverrides -count=1`，覆盖新 smoke 命令可编译和 `ServiceOptions` 正式配置入口。
- 已通过 `cd app && go run ./cmd/ocr-model-lifecycle-smoke ...`，使用 stable、PP-OCRv6 mobile candidate、PP-OCRv6 medium candidate 三个真实模型包完成生命周期 evidence。
- 已通过 `cd app && go test ./cmd/release-config-check -count=1`，覆盖新增生命周期 smoke gate 的 fixture。
- 已通过 `cd app && go run ./cmd/release-config-check`，真实仓库报告 `ok=true`。
- 仍未完成：
  - PP-OCRv6 latest/quality 真实模型包 candidate zip、SHA256、no-orientation worker smoke、截图/滚动长图/白板真实 worker smoke 已完成；但还没有进入 release-ready manifest/catalog，不能把 candidate 包当作用户可安装发布包。
  - PP-OCRv6 latest/quality 已完成本机 Windows x64 命令行生命周期 smoke，但还没有完成真实 Wails 桌面设置页中的模型下载/导入/切换/删除/回退 stable 回验，也没有跨平台 worker smoke matrix 和真实用户截图样例集质量回验；这些完成前 `releaseStatus` 必须继续保持 `candidate`。
  - 真实桌面模型包导入、切换 active、删除和回退 stable 的 UI/窗口回验。
  - 下一次 release 发布后，用真实桌面从 `latest/download/ocr-model-catalog.json` 刷新 stable catalog，并验证下载、取消、重试、安装、删除和切换 active。
  - latest/quality 模型识别失败后的用户可见回退 stable 流程仍需要真实模型包和桌面回验。

当前 R10 进度：

- 已修复真实仓库 `release-config-check` 的 OCR release note gate：
  - release notes 已包含 `Native ONNX OCR inference, stable model smoke, and release-catalog package metadata are release-gated`。
  - 检查器改为匹配真实 release-catalog 门禁文案，避免已存在的 OCR 发布说明被误判为缺失，也避免 catalog 元数据缺失时继续发布。
- 已新增 OCR 前端 e2e 防回归 gate：
  - `Frontend e2e covers OCR result and pin flows`
  - 覆盖 OCR result floating panel、预览 polygon、翻译未配置保护、钉图 OCR 高亮和从钉图打开结果浮层。
  - `Frontend e2e covers whiteboard OCR and recording safety`
  - 覆盖白板选中图片 OCR、OCR block 映射/文本插入、音频录制中 OCR 失败不影响录制壳状态。
  - `Frontend e2e covers recording annotation OCR safety`
  - 覆盖视频录制 annotation overlay 的包内 snapshot OCR、background 优先级和打开同一 OCR 结果浮层。
  - `Frontend e2e covers OCR model management settings`
  - 覆盖 stable/latest/quality 模型状态、模型包路径输入、release catalog 刷新、verified package 下载、不自动切换 active、切换确认和浏览器环境不伪造导入成功。
- 已新增 Go OCR 单测防回归 gate：
  - `Go OCR tests protect translation cache and scrolling tiles`
  - 覆盖禁用 provider 不联网、OpenAI-compatible 缓存、DeepL 分段、段数不匹配失败不写缓存、滚动长图 tile split/dedupe、release catalog 校验保存、模型下载校验安装且不自动激活、checksum 错误不污染安装目录、取消下载清理 staging、取消后重试安装且不自动切换 active model、删除非 active 模型不改变当前 active、删除 active 模型回退默认 stable。
- 已新增模型包发布防回归 gate：
  - `app/cmd/ocr-model-package` 会把真实 stable zip 的 URL、bytes、SHA256 写入 `ocr-model-catalog.json`。
  - release workflow 会验证 `ocr-model-catalog.json` 存在、包含 stable model id、真实 zip 文件名和 `sha256`。
  - release-config-check 会检查 `-catalog-output`、`-release-base-url`、`ocr-model-catalog.json`、catalog 刷新 e2e 和 catalog Go 单测。
- 已新增 OCR smoke evidence 防回归 gate：
  - `release-config-check` 会检查 `app/cmd/ocr-smoke-evidence-check/main.go` 必须保留必需截图场景、`region-queued-cache-hit`、`scrolling-long imageHeight`、`cacheHitWithoutWorker`、`queuedCacheHit`、路径不逃逸、`RecordingFreedom` 和 `文字识别` 校验。
  - `release-config-check` 会检查 `whiteboard-ocr-real-worker-smoke.json`、`requiredWhiteboardScenarios`、`whiteboard-selection-real-worker` 和 `elementId` 校验，防止白板选中图片真实 worker smoke 退化。
  - `release-config-check` 会检查 `scripts/run-local-ocr-smoke.ps1` 必须设置 `RF_OCR_WHITEBOARD_SMOKE`，并同时运行 `TestScreenshotOCRRealWorkerSmoke` 与 `TestWhiteboardSelectionOCRRealWorkerSmoke` 后调用 `ocr-smoke-evidence-check`。
  - 已通过 `cd app && go test ./cmd/ocr-smoke-evidence-check`、`cd app && go test ./cmd/release-config-check` 和 `cd app && go run ./cmd/release-config-check`。
- 已新增白板选中图片真实 evidence 坐标防回归 gate：
  - `release-config-check` 会检查 `whiteboard selected image OCR result panel renders real worker smoke evidence coordinates` 测试名、`sourceKind: 'whiteboard-selection'`、`View board OCR result`、`viewBox', '0 0 900 280'` 和两个真实 worker evidence polygon points。
  - 该 gate 防止白板选中图片 OCR 结果面板退回整张画板 fallback、假图尺寸或非真实 worker evidence 坐标。
- 已新增 release artifact 下载完整性验收工具：
  - 新增 `scripts/verify-release-artifacts.ps1`，可按 tag 或最近发布版本从 GitHub Release 下载 Windows/macOS/Linux x64/arm64 桌面产物、OCR 模型包、`ocr-model-catalog.json` 和 `SHA256SUMS-*`。
  - 脚本会校验每个下载资产的 SHA256SUMS；Windows portable 会复用 `verify-windows-portable.ps1`，Windows installer 可通过 `-RunWindowsInstallers` 复用 `verify-windows-installer.ps1`，macOS/Linux 会在对应宿主或有 bash 的环境中复用 `verify-macos-app-zip.sh` / `verify-linux-portable.sh`。
  - OCR 模型资产会校验 `ppocrv5-mobile-zh-en-*.zip`、`SHA256SUMS-ocr-models.txt`、`ocr-model-catalog.json`，并检查模型 zip 内必须包含 `manifest.json`、`det.onnx`、`cls.onnx`、`rec.onnx`、`keys.txt`、`smoke.png` 和 `smoke.expected.json`。
  - Windows portable 验证已新增可选 `-OcrModelPackagePath`：在目标 Windows 架构可执行时，解压下载或构建出的 stable 模型包，并用发布包内的 `tools/ocr-worker/windows-*/rf-ocr-worker.exe` 与 `tools/onnxruntime/windows-*` 执行 `--smoke`，必须识别出 `RecordingFreedom` 和 `文字识别`；非兼容宿主仍只做 ZIP/PE/Runtime 文件结构校验，不能伪装成已运行识别。
  - Release Windows build 会把同一矩阵生成的 stable `ppocrv5-mobile-zh-en-*.zip` 传给 `verify-windows-portable.ps1 -OcrModelPackagePath`，防止 Windows 便携包只验证 worker/runtime 文件存在却没有验证二者能和模型包真实推理。
  - macOS app zip 与 Linux portable verifier 已新增第三个可选 stable 模型包参数：macOS 会用 app bundle 内 `Contents/MacOS/tools/ocr-worker/darwin-*` 与 `Contents/MacOS/tools/onnxruntime/darwin-*` 跑 `--smoke`；Linux 会在宿主架构兼容时用 portable 内 `tools/ocr-worker/linux-*` 与 `tools/onnxruntime/linux-*` 跑同一 stable smoke，不兼容时只明确跳过执行层，不冒充已识别。
  - Release macOS/Linux build 会把同一矩阵生成的 stable `ppocrv5-mobile-zh-en-*.zip` 传给 `verify-macos-app-zip.sh` / `verify-linux-portable.sh`，防止 Unix 包只验证 worker/runtime 文件存在却没有验证二者能和模型包真实推理。
  - `verify-release-artifacts.ps1 -Targets all` 会先下载并校验 OCR model artifact，再把 stable 模型包路径传给 Windows/macOS/Linux verifier；在目标兼容宿主运行时可复验“下载的发布包 + 下载的模型包”的组合识别闭环。
  - 验收结果会写入 `release-artifact-verification.json`，作为发布后下载级回验证据；`-RequireContentVerification` 可强制目标平台内容校验不可跳过。
  - `release-config-check` 已固定该脚本必须覆盖 Windows portable/setup、macOS app zip、Linux portable、OCR 模型包、SHA256 校验、平台验证器、Windows portable stable OCR smoke 参数和 JSON evidence 报告。
- 已把 OCR 翻译运行时 smoke 纳入发布门禁：
  - `app/cmd/ocr-translation-smoke` 会生成 `translation-smoke.json`，覆盖 settings 不保存明文 API key、secret store 读取、OpenAI-compatible `/chat/completions` provider 请求校验、OCR block 翻译和 provider 关闭后的缓存命中。
  - CI 与 Release Gate 都执行 `go run -tags gtk3 ./cmd/ocr-translation-smoke`；`release-config-check` 已固定命令文件、CI 执行和 release 执行。
  - 当前 Windows 工作区已通过 `cd app && go test ./cmd/ocr-translation-smoke -count=1` 与 `cd app && go run ./cmd/ocr-translation-smoke`，生成 `release-out/ocr-translation-smoke/translation-smoke.json`。
- 已把 OCR secret store 平台 smoke 纳入发布门禁：
  - `app/cmd/ocr-secret-store-smoke` 会生成 `secret-store-smoke.json`，覆盖真实 secret store `Save/Load/Delete`、删除后不可读、data root 无明文 secret。
  - CI/Release validate 阶段执行 `go run ./cmd/ocr-secret-store-smoke`；Windows/macOS/Linux x64/arm64 desktop build matrix 也执行 `go run ./cmd/ocr-secret-store-smoke -evidence-dir "../release-out/ocr-secret-store-smoke/${{ matrix.name }}"`。
  - CI build matrix 会把每个平台的 `secret-store-smoke.json` 作为 `recordingfreedom-ocr-secret-store-smoke-${{ matrix.name }}` artifact 上传；Release artifact 会带上对应平台的 `ocr-secret-store-smoke` 诊断工具，Windows portable runner 会执行 `secret-store-smoke` 并保存 evidence。
  - `release-config-check` 已固定命令文件、CI validate、release validate 和三端 build matrix 执行入口，防止平台 secret store smoke 被删除。
  - 当前 Windows 工作区已通过 `cd app && go test ./cmd/ocr-secret-store-smoke -count=1` 与 `cd app && go run ./cmd/ocr-secret-store-smoke`，生成 `release-out/ocr-secret-store-smoke/secret-store-smoke.json`。
- 已把 OCR worker 平台 capabilities + stable 模型 smoke 纳入桌面矩阵：
  - 新增 `app/cmd/ocr-worker-platform-smoke`，会解压 stable 模型包，执行 `rf-ocr-worker --capabilities --runtime-dir`，要求 `runtimeAvailable=true` 和 `supportsRecognize=true`，再执行 `rf-ocr-worker --smoke`，要求 `plainText` 同时包含 `RecordingFreedom` 和 `文字识别`，并写出 `ocr-worker-platform-smoke.json`。
  - CI `desktop-build` matrix 在每个平台准备 ONNX Runtime、构建对应架构 worker 后，会为该平台生成 stable `ppocrv5-mobile-zh-en` 模型包并运行 `go run ./cmd/ocr-worker-platform-smoke`；CI 会上传 `recordingfreedom-ocr-worker-platform-smoke-${{ matrix.name }}` artifact。
  - Release `build` matrix 也在每个平台构建 worker 后运行同一 smoke，并用 `test -s ../release-out/ocr-worker-platform-smoke/${{ matrix.name }}/ocr-worker-platform-smoke.json` 固定 evidence 必须存在；Release artifact 的 `release-out/**` 会包含该 evidence。
  - 当前 Windows 工作区已通过 `cd app && go run ./cmd/ocr-worker-platform-smoke -worker-path ".\tools\ocr-worker\windows-amd64\rf-ocr-worker.exe" -runtime-dir ".\tools\onnxruntime\windows-amd64" -model-package "..\release-out\ocr-models\ppocrv5-mobile-zh-en-2025.10-hf-jingsongliu.zip" -evidence-dir "..\release-out\ocr-worker-platform-smoke\windows-amd64"`，输出 `ok=true`，capabilities 显示 ONNX Runtime `1.23.2`、`supportsRecognize=true`，smoke 识别 `RecordingFreedom\n文字识别`，evidence 写入 `release-out/ocr-worker-platform-smoke/windows-amd64/ocr-worker-platform-smoke.json`。
  - `release-config-check` 已固定 `ocr-worker-platform-smoke` 命令、CI 矩阵 smoke、Release 矩阵 smoke 和 evidence 文件名，防止全平台 worker smoke 退回单一 Linux smoke。
- 已把 release 发布后下载回验接入 GitHub Actions：
  - `publish` job 在 `gh release upload` 后执行 `Verify published release downloads`，调用 `scripts/verify-release-artifacts.ps1 -Targets all -Architectures x64,arm64` 从刚发布的 GitHub Release 重新下载所有桌面和 OCR 模型资产。
  - 下载脚本在 GitHub API 尚未返回刚上传 asset 时会刷新 release 并等待重试，避免发布后立即查询造成偶发失败。
  - 回验完成后上传 `RecordingFreedom-release-download-verification` artifact，包含 `release-artifact-verification.json`。
  - `verify-windows-portable.ps1` 已补 `Can-ExecuteWindowsTarget`，在 ubuntu 等非 Windows/非兼容宿主只执行 ZIP 结构和 PE 元数据校验，不会错误执行 Windows OCR worker；真正的 worker `--capabilities` 仍保留给目标 Windows 宿主。
  - `verify-windows-portable.ps1`、`verify-macos-app-zip.sh` 和 `verify-linux-portable.sh` 现在都会在兼容宿主上继续执行 stable OCR 模型 smoke；`release-config-check` 已固定 `Verify published release downloads`、`verify-release-artifacts.ps1`、`release-download-verification`、`RecordingFreedom-release-download-verification` 和三端 stable OCR smoke 参数，防止发布流程退回“只上传不下载复验”或“只看 OCR 文件存在”。
  - `verify-release-artifacts.ps1` 现在支持可选真实 OCR desktop evidence 回验参数：`-OcrDesktopEvidenceVisualDir`、`-OcrDesktopEvidenceToolsDir`、`-OcrDesktopEvidenceDataRoot`、`-OcrDesktopEvidenceOutputDir`、`-OcrDesktopEvidenceMustContain` 和 `-OcrDesktopEvidenceRequireTranslation`。
  - 只有显式提供真实 `OcrDesktopEvidenceVisualDir` 时才执行 OCR desktop evidence 回验；此时也必须显式提供 `OcrDesktopEvidenceDataRoot`，脚本会使用发布包/安装包 tools 目录中的 `ocr-desktop-evidence-export/check/plan` 调用 `export-ocr-desktop-evidence.ps1` 或 `.sh`，先生成 `visual-capture-checklist.md/json` 并预检真实视觉截图目录和同一 data root 的运行链路，再生成 evidence 包和 `check-report.json`，并把结果写入 `release-artifact-verification.json` 的 `ocr-desktop-evidence` 条目。
  - 在 Windows 兼容宿主上，如果没有显式传 `OcrDesktopEvidenceToolsDir`，脚本会从已下载并校验 SHA256 的 Windows portable zip 自动解出 `tools/ocr-desktop-evidence-export.exe`、`tools/ocr-desktop-evidence-check.exe` 和 `tools/ocr-desktop-evidence-plan.exe` 到 `release-out/.../ocr-desktop-evidence-tools/windows-<arch>`，再用这些发布包工具执行 OCR desktop evidence checklist/precheck、导出和验收。
  - 在 macOS 兼容宿主上，如果没有显式传 `OcrDesktopEvidenceToolsDir`，脚本会从已下载并校验 SHA256 的 macOS app zip 自动解出 `RecordingFreedom.app/Contents/MacOS/tools/ocr-desktop-evidence-export`、`ocr-desktop-evidence-check` 和 `ocr-desktop-evidence-plan` 到 `release-out/.../ocr-desktop-evidence-tools/macos-<arch>`，再用 app bundle 内工具执行同一 evidence checklist/precheck、导出和验收。
  - 在 Linux 兼容宿主上，如果没有显式传 `OcrDesktopEvidenceToolsDir`，脚本会从已下载并校验 SHA256 的 Linux portable tar.gz 自动解出 `tools/ocr-desktop-evidence-export`、`ocr-desktop-evidence-check` 和 `ocr-desktop-evidence-plan` 到 `release-out/.../ocr-desktop-evidence-tools/linux-<arch>`，再用 portable 包内工具执行同一 evidence checklist/precheck、导出和验收。
  - Windows/macOS/Linux 自动解出的工具只在当前宿主和目标架构兼容时使用；不兼容宿主仍要求显式传 `OcrDesktopEvidenceToolsDir`，不能伪装成已经用发布包工具完成真实桌面验收。
  - 这条接线用于“下载后的发布包 + 真实 Wails 桌面运行数据 + 真实视觉截图”回验，不会在缺少真实桌面数据时伪装完成。
- 已新增真实 Wails 桌面 OCR evidence 验收检查器：
  - `app/cmd/ocr-desktop-evidence-check` 固定真实桌面验收包格式，要求 `README.md`、`platform.txt`、`app-log.jsonl`、`ocr-job-events.jsonl`、`results/*.json`、`visual/*`，并可通过 `-require-translation` 要求 `translations/*.json`。
  - 检查器要求当前用户可见入口 `region-screenshot`、`full-screenshot`、`scrolling-screenshot`、`pinned-screenshot`、`whiteboard`、`whiteboard-selection` 全部有 queued/ready 事件和真实 OCR result；`window-screenshot` / `focused-window-screenshot` 仅作为后端兼容 sourceKind 和旧历史读取保留，不再作为 A 批桌面 evidence 必验项。
  - result 校验会检查 `imagePath` 必须留在 evidence 目录内、图片尺寸与 result 一致、OCR block 坐标不能越界，并支持 `-must-contain RecordingFreedom`、`-must-contain 文字识别` 这类关键文本要求。
  - `release-config-check` 已固定该命令和 `TestRunAcceptsCompleteDesktopOCREvidence`、`TestRunRejectsMissingDesktopOCRSourceKind`、`TestRunRejectsOutOfBoundsOCRBlock`，防止真实桌面 evidence 验收退回浏览器 fallback 或假结果。
  - 注意：该颗粒只是把真实桌面 OCR evidence 的验收标准和检查器落地；仍需要下一次真实 Wails 桌面运行产出 evidence 包并由该 checker 验收。
- 已把真实 Wails 桌面 OCR evidence checker/exporter/plan 纳入发布工具链：
  - CI `Build desktop smoke tools` 会为 Windows/macOS/Linux 矩阵构建 `ocr-desktop-evidence-export`、`ocr-desktop-evidence-check` 和 `ocr-desktop-evidence-plan`。
  - Release `Build desktop smoke tools` 会构建同一组工具；Windows portable zip 会包含 `tools/ocr-desktop-evidence-export.exe`、`tools/ocr-desktop-evidence-check.exe` 和 `tools/ocr-desktop-evidence-plan.exe`，macOS app bundle 会包含 `Contents/MacOS/tools/ocr-desktop-evidence-export`、`Contents/MacOS/tools/ocr-desktop-evidence-check` 和 `Contents/MacOS/tools/ocr-desktop-evidence-plan`，Linux portable 会包含 `tools/ocr-desktop-evidence-export`、`tools/ocr-desktop-evidence-check` 和 `tools/ocr-desktop-evidence-plan`。
  - `verify-windows-portable.ps1`、`verify-macos-app-zip.sh`、`verify-linux-portable.sh` 会检查这三个工具存在并校验目标架构；`run-windows-portable-smoke.ps1` 也会要求 portable tools 目录里存在这三个工具。
  - `release-config-check` 已固定 CI/Release 构建、三端 staging、三端 verifier、发布后下载回验自动解包和 Windows portable smoke runner 里的 `ocr-desktop-evidence-export` / `ocr-desktop-evidence-check` / `ocr-desktop-evidence-plan`，防止发布包缺少真实桌面 evidence checklist、导出和验收工具。
- 已新增真实 Wails 桌面 OCR evidence 导出工具：
  - `app/cmd/ocr-desktop-evidence-export` 会从显式传入的真实 RecordingFreedom data root 导出 checker-ready evidence 包；缺少 `-data-root` 会直接失败，不再回退到猜测的默认 app data root。它会复制 app log、`data/ocr/evidence/ocr-job-events.jsonl`、每个必需 `sourceKind` 的最新 OCR result、对应原图到 evidence-relative `images/`、真实桌面视觉截图到 `visual/`、平台信息到 `platform.txt`，并写出 `README.md` 和 `export-report.json`。
  - 导出工具要求 `-visual-dir` 指向真实桌面截图目录；缺少视觉目录会失败，不会合成假图。OCR result 的 `imagePath` 必须留在 data root 内，导出时才会复制到 evidence 包，防止任意路径逃逸。
  - `-include-translations` 默认会复制已有 `data/ocr/translations/*.json`；没有翻译结果时不会冒充翻译验收完成。
  - 已新增 `TestRunExportsDesktopOCREvidenceFromDataRoot`、`TestRunRejectsMissingVisualEvidenceDirectory` 和 `TestRunRejectsMissingDataRoot`，并由 `release-config-check` 固定 exporter 源码、测试、CI/Release 构建和三端打包/验证门禁。
- 已新增真实 Wails 桌面 OCR evidence checklist/precheck 工具：
  - `app/internal/ocrevidence` 统一声明必需 `sourceKind` 与必需视觉场景，checker、exporter 和 plan 都复用该合同；白板整图不能被 `whiteboard-selection` 冒充，窗口/焦点窗口 sourceKind 只做兼容保留，不进入当前必验合同。
  - `app/cmd/ocr-desktop-evidence-plan` 可输出 JSON 或 Markdown checklist；传入 `-visual-dir` 时会扫描真实桌面截图目录，传入 `-out-dir` 时会写出 `visual-capture-checklist.md` 和 `visual-capture-checklist.json`，传入 `-check` 时缺任一必需视觉场景会非零退出。
  - plan 工具只读取文件名并生成清单/预检结果，不生成、不复制、不合成任何假视觉图片。
  - 已新增 `TestRunWritesChecklistAndAcceptsCompleteVisualDir`、`TestRunReportsMissingVisualRequirements`、`TestMarkdownChecklistIncludesRecommendedFilenames` 和 `app/internal/ocrevidence` 合同测试，防止视觉证据清单和 checker/exporter 验收口径漂移。
- 已新增真实 Wails 桌面 OCR evidence 一键导出/验收脚本：
  - Windows 使用 `scripts/export-ocr-desktop-evidence.ps1 -VisualDir <真实桌面截图目录>`；macOS/Linux 使用 `scripts/export-ocr-desktop-evidence.sh --visual-dir <真实桌面截图目录>`。
  - 两个脚本都会先调用 `ocr-desktop-evidence-plan -out-dir <evidenceDir> -data-root <DataRoot> -check` 生成 `visual-capture-checklist.md/json` 并预检真实视觉截图目录和同一 data root 运行链路，再调用 `ocr-desktop-evidence-export -data-root <DataRoot>`，随后立即调用 `ocr-desktop-evidence-check`；如果缺少真实 `visual-dir`、`data-root`、任一必需视觉场景、平台信息、app log、OCR job events、必需 source kind result 或图片坐标不合法，会直接失败。
  - 脚本支持 `-RequireTranslation` / `--require-translation`、`-MustContain` / `--must-contain`、显式 `DataRoot`、`EvidenceDir`、`PlatformFile`、版本、commit、artifact 和 known failures。
  - Windows 脚本可用 `System.Windows.Forms.Screen` 自动写 `platform.txt` 的 display count/resolution；macOS/Linux 脚本优先用 `system_profiler SPDisplaysDataType` 或 `xrandr --current` 自动写平台显示信息，无法推断分辨率时要求显式传入平台信息。
  - 两个平台脚本现在都会把 `ocr-desktop-evidence-check` 的 JSON 输出保存到 evidence 包内的 `check-report.json`，同时继续打印到终端；checker 失败时也会保留报告路径，方便 Actions artifact 或人工验收直接复核失败原因。
  - macOS/Linux 脚本已把 `visual-dir`、`evidence-dir`、`data-root` 和 `platform-file` 归一成绝对路径，避免脚本进入 `app/` 目录后相对路径指向错误。
  - 两个平台脚本新增发布包 tools 目录模式：Windows 可传 `-ToolsDir <portable-or-installed-tools-dir>` 调用 `ocr-desktop-evidence-plan.exe` / `ocr-desktop-evidence-export.exe` / `ocr-desktop-evidence-check.exe`；macOS/Linux 可传 `--tools-dir <tools-dir>` 调用 app bundle 或 portable 包内的 `ocr-desktop-evidence-plan` / `ocr-desktop-evidence-export` / `ocr-desktop-evidence-check`。未传 tools 目录时仍保留源码 `go run` 模式。
  - 因此真实用户或 Actions 下载发布产物后，不需要 Go 环境和源码目录，也能用发布包内工具导出并验收 OCR desktop evidence。
  - `release-config-check` 已固定这两个脚本必须保留 plan、exporter、checker、真实 `visual-dir`、`visual-capture-checklist`、platform file、translation gate 和 must-contain 参数，防止导出流程退回到“只复制文件不验收”或“凭记忆采集视觉证据”。
- 已把真实桌面 OCR job 事件落成持久化 JSONL 原始证据：
  - `emitOCRJobEvent` 现在会先把事件追加到 `data/ocr/evidence/ocr-job-events.jsonl`，再广播给 Wails 前端；因此前端没有订阅、窗口关闭或后续导出 evidence 时，仍能从数据目录恢复 queued/ready/failure/cancelled 事件。
  - 持久化事件包含 `event`、`jobId`、`sourceKind`、`sourceId`、`status`、`cacheKey`、`merged`、`error` 和 ready 时的真实 `ocr.Result`，可直接作为 `ocr-desktop-evidence-check` 要求的 `ocr-job-events.jsonl` 来源。
  - 已新增 `TestOCRJobEventsPersistEvidenceJSONL`，验证 queued 与 ready 事件都会写入 JSONL，且 ready 行保留真实 result 文本。
  - `release-config-check` 已固定 `app/ocr.go` 的 `writeOCRJobEvidenceEvent`、`ocrEvidenceDir`、`ocr-job-events.jsonl` 和服务测试，防止 OCR job evidence 又退回只广播不落盘。
  - 注意：这只是 evidence 包的事件源材料；完整 evidence 包仍需要导出 README、platform、app-log、results/images、visual 截图和可选 translations。
- 已把真实桌面 OCR 操作日志补进 evidence 验收链：
  - `QueueRecognizeImage` 会写入安全的 `ocr/queue-request` 与 `ocr/queue-accepted` 日志；`queueScreenshotOCRAfterSave` 已改为复用同一路径，避免自动截图 OCR 绕过操作日志。
  - `OpenOcrResult` 会写入 `ocr/open-result`，`ReadOcrResultImage` 会写入 `ocr/read-result-image`，用于证明真实结果浮层不仅生成了 result 文件，还实际读取并展示了对应图片。
  - `TranslateOcr` 会写入 `ocr/translate-request` 与 `ocr/translate-ready`；日志只保留 provider、语言、model、result id、block 数等安全字段，不记录 API key、base URL 或 OCR 原文。
  - `ocr-desktop-evidence-check` 现在要求 `app-log.jsonl` 同时具备 app startup、OCR floating panel、每个必需 `sourceKind` 的 `queue-request`、`open-result` 和 `read-result-image`；开启 `-require-translation` 时还要求 `translate-request` 与 `translate-ready`。
  - 已新增 `TestRunRejectsMissingOCROperationAppLog` 与 `TestOCROperationLogsPersistSafeDesktopEvidence`，并由 `release-config-check` 固定 `logOCRRecognizeRequest`、`logOCRTranslateRequest`、OCR 操作日志事件和敏感字段不落日志的测试样例。
- 已把 OCR 结果浮层的前端渲染证据补进真实桌面 evidence 链：
  - `OcrResultPanel` 读取预览图片后会写入 `client.ocr-result/preview-loaded`，只记录 `resultId/sourceKind/sourceId/width/height/blockCount/available/bytes`，不记录 OCR 原文或 data URL。
  - OCR 预览图和 polygon overlay 进入渲染帧后会写入 `client.ocr-result/rendered`，记录 `hasPreview=true`、图片宽高、block 数和 polygon 数，用于证明前端浮层真实渲染了图片与 OCR block 叠层。
  - block 复制和翻译结果渲染也会写入 `client.ocr-result/copy-block`、`client.ocr-result/translation-rendered` 作为附加诊断证据；同样不记录 block 文本或译文内容。
  - `ocr-desktop-evidence-check` 现在要求 `app-log.jsonl` 包含 `client.ocr-result/preview-loaded available=true` 和 `client.ocr-result/rendered hasPreview=true`，防止验收只停留在后端读图而没有前端浮层渲染证据。
  - `release-config-check` 已固定 `app/frontend/src/App.tsx` 的 OCR result client evidence 日志、checker 中的 `client.ocr-result` 要求和测试夹具中的缺失日志失败断言。
- 已把真实桌面 OCR 视觉证据包加固为可校验 manifest：
  - `ocr-desktop-evidence-export` 复制真实桌面视觉截图时会写入 `visual/visual-manifest.json`，每个视觉文件都记录 evidence-relative path、bytes、SHA256、width 和 height。
  - 导出时会忽略 `.DS_Store`、`Thumbs.db` 和旧 `visual-manifest.json` 这类元数据文件，但其他非图片文件会被判定为失败，不能混入 evidence 包。
  - `ocr-desktop-evidence-check` 现在要求 `visual-manifest.json` 存在，并重新读取文件大小、反算 SHA256、解码图片尺寸、检查重复 path 和必需视觉场景文件名。
- `ocr-desktop-evidence-export` 现在也会在导出阶段校验必需视觉场景：区域截图、全屏截图、滚动截图、OCR result floating panel、截图历史 ready、钉图 OCR 高亮、白板 OCR、白板选区 OCR、录制中 annotation OCR 安全；缺任一项时不会写出 `ok=true` 的 evidence 包。
- `export-report.json` 现在写入 `visualRequirements`，记录每个必需视觉场景匹配到的真实图片路径、关键 terms 和排除 terms；`ocr-desktop-evidence-check` 会根据 `visual-manifest.json` 重新匹配并反校验该字段，防止只靠图片数量或宽泛文件名通过验收。
- 已新增 `TestRunRejectsTamperedVisualManifest` 与 `TestRunRejectsNonImageVisualEvidence`，防止视觉证据退回到“只看文件名存在”或导出不可解码图片。
- 已把视觉证据从“文件名 + 可解码”继续收紧为“文件名 + 可解码 + 最小桌面尺寸”：`app/internal/ocrevidence` 的每个 `VisualRequirement` 都包含 `minWidth/minHeight`，`ocr-desktop-evidence-plan` 预检会读取真实 PNG/JPEG 尺寸，`ocr-desktop-evidence-export` 会把尺寸写入最终 `visual-capture-checklist.json`，`ocr-desktop-evidence-check` 会按同一合同反校验 `visual-manifest.json`、checklist 和 `export-report.json`，防止 80x40 这类小占位图或随手裁剪图冒充真实 Wails 桌面窗口。
- 已新增 `TestVisualDimensionFailuresRejectsTinyPlaceholderImages`、`TestRunReportsTooSmallVisualRequirement`、`TestRunRejectsNonImageVisualPrecheck`、`TestRunRejectsTooSmallVisualEvidence` 和 `TestRunRejectsTooSmallVisualRequirement`，并由 `release-config-check` 固定 `VisualFileDimension`、`minWidth/minHeight`、`existingVisualDimensions` 和 `visualDimensionFailures` 不可回退。
- 已把真实桌面 evidence 的预检从“只看视觉截图目录”扩展为“视觉截图 + data-root 运行链路”：`app/internal/ocrevidence` 新增 `AuditDataRoot` / `DataRootPrecheckReport`，`ocr-desktop-evidence-plan` 新增 `-data-root` 参数，可在导出前检查 `logs/recordingfreedom-*.log`、`data/ocr/evidence/ocr-job-events.jsonl` 和 `data/ocr/results/*.json` 是否覆盖每个必需 `sourceKind` 的 queue/open/read/client render/job ready/result image，并单独检查录制中 annotation 的 `save-capture packageDir` 与后台 OCR `sourceId` 是否匹配。
- `scripts/export-ocr-desktop-evidence.ps1` 与 `.sh` 已强制要求用户传入 `DataRoot` / `--data-root`，并把该目录同时传给 plan 预检和 exporter；底层 `ocr-desktop-evidence-export` 也会拒绝缺少 `-data-root`，发布后下载回验在提供 `OcrDesktopEvidenceVisualDir` 时同样要求 `OcrDesktopEvidenceDataRoot`。如果真实桌面运行缺少某个入口的前端 render、job ready、result image 或 annotation 关联，第一步 `visual-capture-checklist.md/json` 就会显示缺口并在 `-check` 下失败，不必等最终 checker 才发现。
- data-root 预检已新增 run window 合同：每个必需 `sourceKind` 的 `OcrResult.createdAt` 必须落在同一真实验收时间窗口内，默认最大跨度为 6 小时；匹配到的 app-log 事件必须带 `timestamp`，并落在该结果窗口前后 2 小时容差内，防止把不同历史运行的日志、结果和前端渲染记录拼成一包假完整 evidence。
- `ocr-desktop-evidence-export` 现在会把 data-root 预检结果作为 `data-root-precheck.json` 写进最终 evidence 包，并把同一份 `DataRootPrecheckReport` 嵌入 `visual-capture-checklist.json/md`；`ocr-desktop-evidence-check` 已把 `data-root-precheck.json` 作为必需文件校验，要求 `checkComplete=true`、必需 source 数量完整、run window 存在、无 missing requirements，并要求 `export-report.json` 显式指向 `data-root-precheck.json`。最终 evidence 包不再只保存视觉 checklist，运行链路预检结果也会随包保留。
- `ocr-desktop-evidence-export` 已把最终 evidence 包里的 `app-log.jsonl` 收敛到 `data-root-precheck.json` 的 `appEventStart -> appEventEnd` 窗口内，只导出本次真实验收 run window 相关日志；历史 app log 即使含有同 sourceKind/sourceId/resultId 的旧事件也不会进入最终包。`ocr-desktop-evidence-check` 新增 `app-log run window` 检查，要求导出的每一条 app-log 都带 `timestamp` 且处在同一窗口内，防止最终包被历史日志污染或误匹配。
- `ocr-desktop-evidence-export` 已把最终 evidence 包里的 `ocr-job-events.jsonl` 收敛到 `data-root-precheck.json` 选定的 source/result 链路：每个必需 `sourceKind` 只保留匹配 `sourceId` 的 queued 事件和匹配 `sourceId/resultId` 的 ready 事件，历史 ready、错误 resultId、无关 status 都不会导出。`ocr-desktop-evidence-check` 已在 `evidence chain` 中拒绝任何不属于当前结果链路的 job event，防止手工拼包或历史 job event 污染验收。
- 已新增真实桌面 evidence session 边界合同：`app/cmd/ocr-desktop-evidence-session` 可在真实桌面操作前写入 `ocr-desktop-evidence/session-start`，操作结束后用同一 `sessionId` 写入 `session-end`；`AuditDataRoot` 会优先选择包住当前 OCR result window 的 start/end 对，并把 `RunWindow.AppEventStart/AppEventEnd` 收敛到该 session 窗口。缺少 session、sessionId 不匹配、start/end 无 timestamp、session 未包住当前 result window、annotation 事件不在 session 内，都会让 data-root precheck 失败。
- `ocr-desktop-evidence-check` 已新增 `desktop evidence session` 校验，要求最终 evidence 包的 `app-log.jsonl` 里真实存在和 `data-root-precheck.json.session` 完全一致的 start/end 标记；`visual-capture-checklist.json/md` 也会显示 session id、session window 和 duration。CI/Release、Windows portable、macOS app、Linux portable、发布后下载回验和三端 verifier 都已纳入 `ocr-desktop-evidence-session` 工具，防止真实桌面验收流程退回源码-only 或无边界日志拼包。
- 已新增 `TestAuditDataRootAcceptsCompleteDesktopOCRChain`、`TestAuditDataRootReportsMissingPerSourceClientRender`、`TestAuditDataRootReportsRecordingAnnotationMismatch`、`TestAuditDataRootRejectsMixedHistoricalResultRuns`、`TestAuditDataRootRejectsAppLogEventsWithoutTimestamps`、`TestRunWritesDataRootPrecheck` 和 `TestRunReportsIncompleteDataRootPrecheck`，并由 `release-config-check` 固定 `data_root_precheck.go`、`dataRootPrecheck`、`DataRootPrecheckRunWindow`、`-data-root` 脚本接线不可回退。
- 已新增 `TestRunRejectsMissingDataRootPrecheck`、`TestRunRejectsAppLogOutsideDataRootPrecheckWindow`、`TestRunRejectsUnexpectedScopedJobEvent`、exporter -> checker 组合回归和 `release-config-check` 针脚，固定 `data-root-precheck.json`、`validateDataRootPrecheck`、`validateAppLogRunWindow`、`validateScopedJobEvents`、`copyJSONLLinesInWindow`、`copyScopedJobEvents`、`AuditDataRoot`、`DataRootPrecheck` 和 `export-report.json.dataRootPrecheck` 不可回退。
- 已新增 `TestRunWritesStartAndEndMarkers`、`TestRunRequiresSessionIDForEnd`、`TestAuditDataRootRejectsMissingEvidenceSession`、`TestAuditDataRootRejectsEventsOutsideEvidenceSession` 和 `TestRunRejectsMissingEvidenceSessionMarkers`，并由 `release-config-check` 固定 `EvidenceSessionComponent`、`WriteSessionMarker`、`DataRootPrecheckSession`、`dataRootEvidenceSession`、`validateSessionMarkers`、`ocr-desktop-evidence-session` 的 CI/Release/三端打包/验证门禁不可回退。
- 注意：该颗粒只加固真实桌面 evidence 包合同；仍需要真实 Wails 桌面运行后导出包含这些视觉截图的 evidence 包并由 checker 验收。
- 已把真实桌面 OCR `export-report.json` 从导出摘要升级为 checker 反校验合同：
  - `ocr-desktop-evidence-check` 现在要求 `export-report.json` 存在且 `ok=true`，并校验 `generatedAt`、`dataRoot`、`evidenceDir`、`visualDir`、`visualManifest`、`resultCount` 和每个 `sourceKind` 的 `resultId/imagePath/resultPath`。
  - checker 会重新统计 `app-log.jsonl`、`ocr-job-events.jsonl`、`visual/visual-manifest.json`、`translations/*.json` 与实际 result/image 文件，要求它们和 `export-report.json` 一致。
  - 每个 `sourceKind` 在 export report 中不能缺失、重复或未知；报告指向的 result JSON 必须能读取，并且 result 内的 `sourceKind/resultId/imagePath` 必须与报告一致。
  - 已新增 `TestRunRejectsMismatchedExportReport`，防止真实桌面 evidence 包退回到“报告写 ok 但实际包内容不一致”的弱验收。
- 已把真实桌面 OCR README 中的 `known failures` 改为强验收项：
  - `ocr-desktop-evidence-check` 现在要求 `known failures` 明确为 `none/no/n/a/无/没有` 这类无阻塞值；如果 evidence 包记录了具体已知失败，checker 默认判定 blocked。
  - `-known-failures` 仍可用于导出诊断包记录问题，但这种包不能作为“真实桌面 OCR 已验收完成”的证据。
  - 已新增 `TestRunRejectsKnownFailures`，并由 `release-config-check` 固定 `validateKnownFailuresAreClear` 与 `known failures must be none`，防止带阻塞项的 evidence 包被误判为 ready。
- 已通过 `cd app && go test ./cmd/ocr-desktop-evidence-export ./cmd/ocr-desktop-evidence-check ./cmd/release-config-check -count=1`，覆盖真实桌面 OCR evidence exporter、checker 和 release gate fixture。
- 已通过 `cd app && go test ./cmd/ocr-desktop-evidence-export ./cmd/ocr-desktop-evidence-check ./cmd/release-config-check -count=1`，覆盖 `visual/visual-manifest.json` 导出、篡改拒绝、非图片视觉证据拒绝和 release gate fixture。
- 已通过 `cd app && go test ./cmd/ocr-desktop-evidence-export ./cmd/ocr-desktop-evidence-check ./cmd/release-config-check -count=1`，覆盖 `visualRequirements` 导出与 checker 反校验、必需视觉场景缺失拒绝、用户隐藏的窗口/焦点窗口入口不进入必验合同、白板整图与白板选区不互相冒充、录制中 annotation OCR 视觉证据不可缺失。
- 已通过 `cd app && go run ./cmd/release-config-check`，真实仓库报告 `ok=true`，新增视觉 manifest 加固门禁均为 `ready`。
- 已通过 `cd app && go test ./cmd/ocr-desktop-evidence-check ./cmd/release-config-check -count=1`，覆盖 `export-report.json` 与实际 evidence 包内容不一致时的失败路径。
- 已通过 `cd app && go test ./cmd/ocr-desktop-evidence-export ./cmd/ocr-desktop-evidence-check ./cmd/release-config-check -count=1`，覆盖 exporter 生成的 report 与 checker 新增反校验合同兼容。
- 已通过 `cd app && go test ./cmd/ocr-desktop-evidence-check ./cmd/release-config-check -count=1`，覆盖 `known failures` 非空时 checker 默认拒绝。
- 已通过 `cd app && go test ./cmd/release-config-check -count=1`，覆盖 `export-ocr-desktop-evidence.ps1` / `.sh` 脚本门禁夹具。
- 已通过 PowerShell Parser 静态解析 `scripts/export-ocr-desktop-evidence.ps1`，覆盖新增 `check-report.json` 保存逻辑的语法有效性。
- 已通过 `bash -n scripts/export-ocr-desktop-evidence.sh` 和 `bash scripts/export-ocr-desktop-evidence.sh --help`，覆盖 Unix 脚本新增绝对路径归一化与 `check-report.json` 保存逻辑的语法有效性。
- 已通过 `cd app && go test ./cmd/release-config-check -count=1` 与 `cd app && go run ./cmd/release-config-check`，固定两个 evidence 导出脚本必须保留 `check-report.json` 产物。
- 已通过 PowerShell Parser 静态解析 `scripts/export-ocr-desktop-evidence.ps1`，覆盖新增 `-ToolsDir` 发布包工具模式的语法有效性。
- 已通过 `bash -n scripts/export-ocr-desktop-evidence.sh` 和 `bash scripts/export-ocr-desktop-evidence.sh --help`，覆盖新增 `--tools-dir` 发布包工具模式的语法有效性。
- 已通过 `cd app && go test ./internal/ocrevidence ./cmd/ocr-desktop-evidence-session ./cmd/ocr-desktop-evidence-plan ./cmd/ocr-desktop-evidence-export ./cmd/ocr-desktop-evidence-check ./cmd/release-config-check -count=1`，覆盖 session start/end 标记、data-root session 审计、export/check 反校验和 release gate fixture。
- 已通过 `cd app && go run ./cmd/release-config-check`，真实仓库报告 `ok=true`，`ocr-desktop-evidence-session` 已固定进入 CI/Release desktop smoke tools、Windows/macOS/Linux 发布包 staging、三端 verifier、发布后下载回验和脚本工具目录模式。
- 已通过 PowerShell Parser 静态解析 `scripts/export-ocr-desktop-evidence.ps1`、`scripts/verify-release-artifacts.ps1`、`scripts/verify-windows-portable.ps1`，以及 `bash -n scripts/export-ocr-desktop-evidence.sh && bash -n scripts/verify-macos-app-zip.sh && bash -n scripts/verify-linux-portable.sh`，覆盖 session 工具接入后的脚本语法有效性。
- 已通过 `cd app && go test ./cmd/release-config-check -count=1` 与 `cd app && go run ./cmd/release-config-check`，固定 evidence 导出脚本必须保留发布包 tools 目录模式，防止验收流程重新依赖本机 Go 或源码目录。
- 已通过 PowerShell Parser 静态解析 `scripts/verify-release-artifacts.ps1`，覆盖新增可选 OCR desktop evidence 回验参数和 `Invoke-OcrDesktopEvidenceExport` 接线的语法有效性。
- 已通过 `cd app && go test ./cmd/release-config-check -count=1` 与 `cd app && go run ./cmd/release-config-check`，固定发布后下载回验脚本必须保留 `OcrDesktopEvidenceVisualDir/OcrDesktopEvidenceToolsDir`、`check-report.json` 和 `release-artifact-verification.json` 中的 OCR desktop evidence 条目。
- 已通过 PowerShell Parser 静态解析 `scripts/verify-release-artifacts.ps1`，覆盖 Windows portable OCR desktop evidence tools 自动解包逻辑的语法有效性。
- 已通过 `cd app && go test ./cmd/release-config-check -count=1` 与 `cd app && go run ./cmd/release-config-check`，固定发布后下载回验脚本必须保留 `Expand-WindowsPortableToolsDir`、`Can-ExecuteWindowsTarget`、`ocr-desktop-evidence-tools` 和自动解出的 Windows portable tools 接线。
- 已通过 PowerShell Parser 静态解析 `scripts/verify-release-artifacts.ps1`，覆盖 macOS app zip 与 Linux portable OCR desktop evidence tools 自动解包逻辑的语法有效性。
- 已通过 `cd app && go test ./cmd/release-config-check -count=1` 与 `cd app && go run ./cmd/release-config-check`，固定发布后下载回验脚本必须保留 `Can-ExecuteMacOSTarget`、`Can-ExecuteLinuxTarget`、`Expand-MacOSAppToolsDir`、`Expand-LinuxPortableToolsDir`、`macos-$arch`、`linux-$arch` 和“compatible release package tools were auto-extracted”错误口径，防止 OCR desktop evidence 回验再次退回 Windows-only。
- 已通过 `cd app && go test ./internal/ocrevidence ./cmd/ocr-desktop-evidence-plan ./cmd/ocr-desktop-evidence-export ./cmd/ocr-desktop-evidence-check -count=1`，覆盖共享 evidence 合同、checklist/precheck、exporter 和 checker 的基础闭环。
- 已通过 `cd app && go test ./cmd/release-config-check -count=1`，覆盖 `ocr-desktop-evidence-plan` 纳入 CI/Release 构建、三端发布包 staging、三端 verifier、Windows portable smoke runner、导出脚本和发布后下载回验脚本的防回归 fixture。
- 已通过 `cd app && go run ./cmd/release-config-check`，真实仓库报告 `ok=true`，新增的共享 evidence 合同、plan CLI、发布工具链和自动解包门禁均为 `ready`。
- 已通过 `cd app && go test . -run TestOCRJobEventsPersistEvidenceJSONL -count=1`，覆盖 OCR job queued/ready 事件持久化到 `data/ocr/evidence/ocr-job-events.jsonl`。
- 已通过 `cd app && go run ./cmd/release-config-check`，真实仓库新增的 `OCR desktop evidence exporter builds checker-ready packages`、`OCR desktop evidence exporter tests protect real-data package assembly`、`OCR desktop evidence checker fixes real Wails acceptance contract` 与 `OCR desktop evidence checker tests protect required sources and geometry` 门禁均为 `ready`。
- 已通过 `cd app && go test ./cmd/ocr-desktop-evidence-check ./cmd/release-config-check -count=1`，覆盖 OCR 操作日志纳入 desktop evidence checker 与 release gate fixture。
- 已通过 `cd app && go test . -run "TestOCRJobEventsPersistEvidenceJSONL|TestOCROperationLogsPersistSafeDesktopEvidence" -count=1`，覆盖 OCR job evidence JSONL 和安全 app-log 操作日志。
- 已通过 `cd app && go run ./cmd/release-config-check`，真实仓库报告 `ok=true`，新增 OCR 操作日志门禁均为 `ready`。
- 已通过 `cd app/frontend && npm run build`，覆盖 `OcrResultPanel` 新增 client render evidence 日志的 TypeScript 和生产构建。
- 已通过 `cd app && go test ./cmd/ocr-desktop-evidence-check ./cmd/release-config-check -count=1`，覆盖 checker 新增 `client.ocr-result/preview-loaded` 与 `client.ocr-result/rendered` 要求。
- 已通过 `cd app && go run ./cmd/release-config-check`，真实仓库报告 `ok=true`，新增前端 OCR result render evidence 门禁为 `ready`。
- 已通过 PowerShell Parser 静态解析 `scripts/export-ocr-desktop-evidence.ps1`。
- 已通过 `bash -n scripts/export-ocr-desktop-evidence.sh` 和 `bash scripts/export-ocr-desktop-evidence.sh --help`。
- 已通过 PowerShell Parser 静态解析 `scripts/verify-windows-portable.ps1` 和 `scripts/run-windows-portable-smoke.ps1`，覆盖 Windows portable 新增工具检查语法。
- 已通过 `bash -n scripts/verify-macos-app-zip.sh && bash -n scripts/verify-linux-portable.sh`，覆盖 macOS/Linux verifier 新增工具检查语法。
- 已通过 `cd app && go test ./cmd/release-config-check`。
- 已通过 `cd app && go run ./cmd/release-config-check`，真实仓库报告 `ok=true`。
- 已通过 `cd app && go test ./cmd/ocr-worker-platform-smoke ./cmd/release-config-check -count=1`，覆盖新增平台 worker smoke 命令编译和 release gate fixture。
- 已通过本机 Windows x64 `cd app && go run ./cmd/ocr-worker-platform-smoke ...`，生成 `release-out/ocr-worker-platform-smoke/windows-amd64/ocr-worker-platform-smoke.json`。
- 已通过 PowerShell Parser 静态解析 `scripts/verify-windows-portable.ps1` 和 `scripts/verify-release-artifacts.ps1`，覆盖新增 `-OcrModelPackagePath`、stable OCR smoke 和 model artifact 预下载接线的语法有效性。
- 已通过 `bash -n scripts/verify-macos-app-zip.sh && bash -n scripts/verify-linux-portable.sh`，覆盖 macOS/Linux verifier 新增 stable OCR 模型包参数和 smoke 逻辑的 shell 语法有效性。
- 已通过 `cd app && go test ./cmd/release-config-check -count=1` 和 `cd app && go run ./cmd/release-config-check`，release gate 已固定 Windows portable stable OCR smoke 不可回退。
- 已通过 `cd app && go test ./internal/ocr -run "Test(StartModelDownload|CancelModelDownload|RetryModelDownload|InstallModelPackage|RefreshModelCatalog|Remove.*Model)" -count=1`。
- 已通过 `cd app/frontend && npm run test:e2e -- --grep "recording annotation overlay queues background OCR"`。
- 已通过 `cd app && go test ./cmd/ocr-model-package ./cmd/release-config-check`。
- 已通过 `cd app/frontend && npm run test:e2e -- --grep "OCR model"`。
- 已通过 `cd app/frontend && npm run build`。
- 已通过 `cd app && go test ./...`。
- 已通过 PowerShell Parser 静态解析 `scripts/verify-release-artifacts.ps1` 和 `scripts/verify-windows-portable.ps1`。
- 已通过 `git diff --check`。
- 仍未完成：
  - v0.1.32 已完成全平台 release artifact 下载后的 workflow 级完整性回验，并上传 `RecordingFreedom-release-download-verification` artifact；当前本地未认证环境不能直接下载 Actions artifact zip，后续如需离线归档，需要在有 GitHub token 的环境下载并保存其中的 `release-artifact-verification.json`。
  - Windows/macOS/Linux 发布包 stable OCR smoke 已接入 build/download verifier，v0.1.32 Release workflow 已证明该门禁可通过；仍未形成真实用户桌面操作的 OCR evidence 包。
  - Windows/macOS/Linux 各架构真实 worker `--capabilities` 和 stable 模型 smoke 矩阵 gate 已进入 CI/Release，并随 v0.1.32 build matrix 通过；如需长期审计，需要下载 Actions build artifacts 中的各平台 `ocr-worker-platform-smoke.json` 做离线归档。
  - 真实桌面截图/钉图/画板 OCR 的视觉/e2e 证据。

执行纪律：

- 每个 `R` 只能在上一个 `R` 的核心链路可验证后推进；如果发现前置链路失败，先修前置链路。
- 每个 `R` 完成前都不能发布“已完成 OCR/翻译”结论，只能说明“某入口已部分可验收”。
- 对用户可见的按钮如果后端能力未 ready，必须显示明确不可用原因；不能留下点击后无响应的控件。
- 真实翻译进入前，所有翻译入口只允许走未配置保护提示，不允许发送 OCR 原文。

## 剩余功能与问题重新归拢（2026-07-06）

本节用于替代继续“单颗粒零散推进”的临时执行方式。后续开发按下面四条主线成批闭环，每一批必须同时包含功能、证据、发布包和文档状态；没有真实证据的代码只能算“已接线”，不能算“完成”。

当前结论：不是“只剩测试”。剩余工作分成四个可验收大批次，必须按 `A -> B -> C -> D` 推进；每批结束时只跑该批一次集中验证，避免继续出现“改一个点、跑一次、再改一个点”的低效循环。

### 剩余事项总控看板

从现在开始，OCR/翻译线不再按单个按钮、单个测试、单个脚本零散推进。下面这张表是剩余工作的执行总表；每一行必须作为一个完整交付包处理，包含功能闭环、真实证据、验证命令和文档状态。

| 顺序 | 批次 | 包含的用户能力 | 已有基础 | 剩余功能/问题 | 下一次集中动作 | 通过后才能宣称 |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | A. 真实桌面 OCR 入口闭环 | 区域/全屏/滚动截图、截图历史、钉图、画板、录制中画板 OCR | OCR 队列、worker、result、evidence contract、export/check 工具已接线；窗口/焦点窗口 sourceKind 仅兼容保留 | 缺真实 Wails 桌面 evidence；OCR result 浮层、polygon、高亮、画板选区、录制中 annotation 还未形成一包验收证据 | 一次性跑完所有用户可见入口的真实桌面采集，导出 evidence 包并用 checker 通过 | “OCR 入口进入桌面验收” |
| 2 | B. 翻译 provider 与隐私闭环 | OCR 结果翻译、复制、缓存、provider 配置、隐私保护 | local/external provider smoke 和 release gate 基础已接线 | 真实 DeepL/OpenAI-compatible、三端 secret store、日志脱敏和 UI 调用证据不足 | 一次性验证 provider、secret store、缓存、强制请求、失败提示、日志无 key/原文 | “翻译进入桌面验收” |
| 3 | C. 模型通道与质量闭环 | stable/latest/quality 模型、下载、切换、删除、回退 | stable 默认方向、模型包/下载/worker smoke 门禁已具备 | PP-OCRv6 latest/quality 仍是 candidate；真实截图样例质量、跨平台 smoke 和失败回退证据不足 | 一次性完成 latest/quality 模型包 manifest、SHA256、smoke、真实样例、回退和设置页桌面证据 | “模型通道可给用户切换” |
| 4 | D. 发布与长期防回归闭环 | Windows/macOS/Linux 发布包、worker/runtime/model/tool 回验 | release-config-check、verifier、evidence scripts 已加固；v0.1.32 Release workflow 已通过发布后下载复验并上传 `RecordingFreedom-release-download-verification` artifact | 真实桌面 OCR evidence 包尚未作为 release 后验收资产固定；Actions artifact 仍需在有 token 的环境下载归档 | 下载并归档 v0.1.32 `release-artifact-verification.json` 和各平台 smoke evidence；后续把真实桌面 OCR evidence 包纳入 release 后验收 | “全平台 OCR/翻译发布可验收” |

### 问题重新归类

真正未闭环的功能问题：

- A 批次：所有 OCR 入口虽然有接口、队列、worker 和 evidence 工具，但还缺真实桌面一包到底的用户验收证据；这是当前第一优先级。
- A 批次：画板选中图片必须证明使用自己的 `imagePath` 和自己的 block 坐标，不能被整张画板结果冒充。
- A 批次：录制中画板 OCR 必须证明在录制、鼠标、写盘和绘制并行时不造成阻塞。
- B 批次：翻译必须证明 provider、secret store、缓存和脱敏日志真实可用；不能只停留在 smoke 命令或未配置提示。
- C 批次：latest/quality 模型不能因为 catalog 存在就开放给用户；必须先过质量样例、跨平台 runtime 和回退验证。
- D 批次：v0.1.32 已完成发布包下载产物反向验证；后续仍要把真实桌面 OCR evidence 包纳入 release 后验收，不能用源码目录或本机开发环境代替。

已经有基础但不能宣称完成的事项：

- OCR evidence plan/export/check、visual checklist、app-log/job-events/result chain、known failures gate 已经加固；这些只证明验收工具更严格，不等于真实桌面 OCR 已完成。
- `ocr-translation-smoke`、secret store smoke、worker platform smoke、release verifier 的接线已进入门禁；这些只证明发布工具链准备好，不等于真实 provider、真实模型和真实包已验收。
- Go 单测、浏览器 e2e、worker smoke、release-config-check 都只能作为批次证据的一部分，不能单独当作用户可验收结论。

暂停事项：

- 暂停新增 OCR/翻译入口。
- 暂停继续按单个入口补零散测试。
- 暂停发布“已完成 OCR/翻译”的结论。
- 暂停把 latest/quality 模型展示为普通用户可用。
- 暂停把源码 `go run` 或本机开发目录当作发布包验收。

下一次只允许进入 A 批次：

1. 准备一套真实 Wails 桌面操作 runbook。
2. 一次性覆盖全部 OCR 入口。
3. 采集视觉截图/录屏和安全 app-log。
4. 导出 evidence 包。
5. 用发布包工具模式跑 `ocr-desktop-evidence-check`。
6. 通过后更新本文档，并只声明“A 批次进入桌面验收”，不越级宣称 B/C/D 完成。

后续开发口径：

- 不再新增新的 OCR/翻译入口，先把已列入口做成真实桌面闭环。
- 不再把 Go 单测、浏览器 e2e、worker smoke 当作“用户可验收完成”，它们只能算批次内证据的一部分。
- 不再逐个小点反复 build；每个批次内集中改完，再集中执行该批最小验证。
- 每个批次结束必须更新本文档，写清“已完成、未完成、证据位置、验证命令和是否可发布验收版”。

### A. 真实桌面 OCR 入口闭环

目标：先把用户能看到、能点击、能验收的 OCR 入口一次性拉齐，而不是分散修单个入口。

范围：

- 区域截图、全屏截图、滚动截图；窗口截图/焦点窗口截图不再是用户工具入口，只做 sourceKind 兼容保留。
- 截图历史里的手动识别、自动识别、重试、打开结果、复制文字。
- 录制前画板、录制中 annotation-overlay、画板整图、画板选中图片 OCR。
- 钉图窗口的识别、高亮、复制、打开结果、关闭重开恢复。

必须补齐的问题：

- 真实 Wails 桌面视觉/e2e 证据不足，当前大量证据仍是 Go 单测、浏览器 e2e 或 worker smoke。
- OCR result floating panel 在真实桌面窗口中打开、预览图片、polygon 坐标、高亮联动还需要统一录屏或 evidence 包。
- 画板选中图片必须证明使用自己的 `imagePath`，不能 fallback 到整张画板。
- 录制中 OCR 必须证明与录制并行且不影响录制、鼠标、写盘和绘制。

验收输出：

- 一份真实桌面 OCR evidence 包，至少覆盖用户可见的 `region-screenshot`、`full-screenshot`、`scrolling-screenshot`、`pinned-screenshot`、`whiteboard`、`whiteboard-selection` 和录制中 annotation OCR。
- evidence 包必须通过 `ocr-desktop-evidence-plan`、`ocr-desktop-evidence-export`、`ocr-desktop-evidence-check`。
- 桌面录屏或截图必须能看到真实图片、真实 OCR block、真实坐标叠层，而不是浏览器 fallback。

本轮 A 批次基础设施收口：

- A1 真实桌面 runbook 已落地：`docs/19-ocr-desktop-evidence-runbook.md` 固定了 session start/end、逐入口操作顺序、视觉证据命名、导出和 checker 命令；该文档只代表 A1 完成，不代表 A2 真实桌面 evidence 已采集。
- `app/internal/ocrevidence` 新增 `RequiredCaptureSteps`，把区域、全屏、滚动截图、截图历史 ready、OCR result 浮层、钉图、白板、白板选中图片和录制中 annotation OCR 拆成机器可读 capture runbook。
- `ocr-desktop-evidence-plan` 的 JSON 和 Markdown 输出现在包含 `evidenceSessionRunbook` / `Session boundary runbook` 和 `captureSteps` / `Capture runbook`，每一步都有 action、recommended visual file、required log events 和 acceptance criteria；真实桌面采集不再只靠视觉文件名清单，也不能漏掉 session 边界。
- `ocr-desktop-evidence-check` 已从“只要求出现一次 open-result/read-result-image/client render”收紧为“每个必需 sourceKind 都必须有 `queue-request`、`open-result`、`read-result-image`、`client.ocr-result/preview-loaded` 和 `client.ocr-result/rendered`”，防止只打开一个 OCR 结果浮层却让整包通过。
- `recording-annotation` 已从普通 `whiteboard` 验收里单独拆出来：checker 现在要求 `annotation-overlay/show packageDir`、`annotation-overlay/save-capture packageDir bytes`，并要求 `ocr/queue-request sourceKind=whiteboard priority=background` 的 `sourceId` 与保存的 `packageDir` 匹配，防止普通白板 OCR 冒充录制中 annotation OCR 安全验收。
- 已新增 `evidence chain` 一致性验收：`app-log.jsonl`、`ocr-job-events.jsonl` 和 `results/*.json` 不能只按 `sourceKind` 松散匹配，必须按每个真实 `sourceId/resultId` 对齐；同一个 `sourceKind` 出现重复 result、app-log 打开错结果或 job-events ready 指向错结果都会被拒绝。
- `ocr-desktop-evidence-plan` 的 JSON/Markdown checklist 已输出 `Evidence chain requirements`，真实桌面采集前就能看到 `sourceId/resultId` 对齐、无重复 sourceKind、annotation `packageDir` 匹配等硬要求。
- `ocr-desktop-evidence-export` 现在会在最终 evidence 包根目录重新写入 `visual-capture-checklist.md/json`，避免脚本先生成 checklist 后又被 exporter 清空目录删掉；`ocr-desktop-evidence-check` 已要求这两个 checklist 文件存在、`checkComplete=true`，并包含 capture runbook 与 evidence chain 要求。
- `ocr-desktop-evidence-export` 已新增 exporter -> checker 组合回归：测试夹具会生成完整 app-log/job-events/result/visual/checklist 链路，导出 evidence 包后立即用 `ocr-desktop-evidence-check` 和 `-must-contain RecordingFreedom/文字识别` 验收，防止 exporter 与 checker 各自通过但真实 evidence 包合起来失败。
- 已新增/更新单测锁定 runbook 与逐 sourceKind 日志验收；这仍然只是 A 批次 evidence 合同收紧，不能替代真实 Wails 桌面 evidence 包。
- A2 真实桌面采集口径已按用户最新要求调整：截图/画板工具面板只保留区域、全屏、滚动截图和画板入口，`窗口截图` 与 `焦点窗口截图` 已从工具菜单删除；对应 e2e 与 `release-config-check` 固定这两个按钮不得再出现在工具菜单中。后端 `window-screenshot` / `focused-window-screenshot` sourceKind 和旧历史显示仍兼容保留，但已从 `app/internal/ocrevidence` 的必需 source、capture step 和 visual requirement 中移除，不能再通过胶囊工具菜单或 A 批桌面 evidence 合同暴露为必验入口。
- Windows OCR 命令窗口闪烁修复已落地：`queryWorkerCapabilities` 和 `runWorkerRecognize` 两条 OCR worker 启动路径都调用平台级 `configureBackgroundCommand`；Windows 下设置 `HideWindow` 与 `CREATE_NO_WINDOW`，macOS/Linux 为空实现。`release-config-check` 已固定该防回归门禁，避免打开设置或执行识别时再次弹出一闪而过的命令窗口。
- A2 evidence 导出链路已强制要求显式 `DataRoot`：Windows/macOS/Linux 脚本、底层 `ocr-desktop-evidence-export` 和发布后下载回验都会拒绝缺少或不存在的 data root，并把同一个解析后的目录同时传给 plan/export；`release-config-check` 已固定该门禁，避免只用视觉截图目录或猜测默认数据目录导出一个缺少真实 app-log/job-events/results/session 链路的假 evidence 包。
- `ocr-desktop-evidence-plan -check` 已同步收紧：只要传入 `-visual-dir` 进入验收检查，就必须同时传入同一次真实桌面 session 使用的 `-data-root`；直接运行 plan 工具也不能再只凭完整截图目录返回可验收结果。新增 `TestRunCheckRejectsVisualDirWithoutDataRoot` 与 release gate 针脚固定该行为。

### B. 翻译 provider 与隐私闭环

目标：翻译必须从“本地可控 provider smoke”升级到“真实 provider 可诊断、可发布包验收、不会泄露 key/原文”的闭环。

范围：

- `deepl` 和 `openai-compatible` provider。
- 结果浮层、截图历史、钉图、画板的同一 `TranslateOcr` 调用。
- Windows DPAPI、macOS Keychain、Linux Secret Service 或 0600 fallback secret store。
- 翻译缓存、强制请求、缓存命中、provider 不可用错误。

必须补齐的问题：

- 真实 DeepL 或外部 OpenAI-compatible provider 桌面回验缺失。
- macOS Keychain 与 Linux Secret Service 实机 Save/Load/Delete 证据缺失。
- 发布包内翻译诊断工具需要成为固定工具，不能只在源码 `go run` 中存在。
- evidence/log/settings 必须证明 API key 不落盘、不进日志、不进入 evidence JSON。

验收输出：

- `ocr-translation-smoke` 支持离线 local provider 和外部 OpenAI-compatible provider 两种模式。
- 外部 provider 模式只从环境变量读取 key，证据只记录 provider mode、model、语言、是否提供 key、缓存命中和脱敏 base URL。
- Windows/macOS/Linux 发布包都包含并校验 `ocr-translation-smoke`。
- Windows portable smoke runner 至少运行离线 translation smoke。

### C. 模型通道与质量闭环

目标：stable 保持默认可用；latest/quality 只有通过真实质量、跨平台 smoke 和回退验证后才允许进入用户可安装目录。

范围：

- stable/latest/quality catalog。
- PP-OCRv6 latest candidate、quality candidate。
- 模型下载、取消、重试、安装、删除、切换 active、失败回退 stable。
- ONNX Runtime 与 worker capabilities 矩阵。

必须补齐的问题：

- PP-OCRv6 latest/quality 仍是 candidate，不能对用户显示为 release-ready。
- 真实用户截图样例质量回验不足。
- 真实 Wails 设置页的模型导入、下载、切换、删除、回退流程还缺桌面证据。
- 下一次 release 后需要用真实发布 catalog 测下载、校验、安装和回退。

验收输出：

- 每个平台生成 `ocr-worker-platform-smoke.json`。
- latest/quality 模型包必须有 manifest、SHA256、smoke、真实截图样例质量记录和失败回退记录。
- 设置页真实桌面回验包含下载进度、取消、重试、校验失败、确认切换和删除。

### D. 发布与长期防回归闭环

目标：不再用“本地能跑”替代“发布包可验收”。发布包必须自带 worker、runtime、诊断工具和证据导出工具。

范围：

- Windows/macOS/Linux x64/ARM64 发布包。
- worker、ONNX Runtime、stable 模型 smoke。
- OCR desktop evidence tools、translation smoke、secret store smoke。
- 下载发布产物后的完整性回验。

必须补齐的问题：

- 下一次实际 Actions 发布 run 还未产出完整 `release-artifact-verification.json`。
- 当前工作区没有新的真实 portable/app zip 可完成本机下载后回验。
- 全平台 `ocr-worker-platform-smoke.json` 仍需要下一次 Actions run 产出。
- 真实桌面 OCR evidence 包还没有作为 release 后验收资产固定下来。

验收输出：

- `RecordingFreedom-release-download-verification/release-artifact-verification.json`。
- Windows/macOS/Linux 每个平台的 worker capabilities 和 stable OCR smoke evidence。
- 发布包内工具清单必须包括 `ocr-desktop-evidence-plan`、`ocr-desktop-evidence-export`、`ocr-desktop-evidence-check`、`ocr-translation-smoke`、`ocr-secret-store-smoke`。
- release-config-check 固定所有工具、脚本、workflow 和文档证据，不允许回退成源码-only 验收。

### 重新排序后的执行顺序

1. 已收口当前未验证/半接线的翻译 smoke 工具改动：`ocr-translation-smoke` 已支持 local/external-openai-compatible 两种模式，CI/Release/三端发布包/Windows portable smoke runner/三端 verifier/release-config-check 已固定；这一步只算基础设施修正，不算功能验收完成。
2. 下一批批量完成 A：真实桌面 OCR 入口 evidence 包。只有这一批通过后，才能说 OCR 入口进入桌面验收。
3. 再批量完成 B：真实 provider/secret store/翻译缓存 evidence。只有这一批通过后，才能说翻译进入桌面验收。
4. 再批量完成 C：latest/quality 模型通道和真实质量样例。没有跨平台 smoke 前不开放 latest/quality 给普通用户。
5. 最后批量完成 D：发布包下载后完整回验。v0.1.32 已证明基础发布下载回验闭环可通过；没有真实桌面 OCR evidence 包进入 release 后验收前，仍不能发布“全平台 OCR/翻译已完成”的结论。

当前暂停项：

- 暂停继续新增 OCR/翻译功能点。
- 暂停逐个入口继续补小测试。
- 暂停发布“已完成 OCR/翻译”结论。
- 只允许先做归拢后的基础设施收口、真实桌面 evidence 批量采集和发布包证据闭环。

### 2026-07-06 收口记录

- OCR 原位可复制文本已落地：`ocr-result` 浮层在预览图的 OCR block 坐标上渲染可点击文本按钮，点击即可复制对应识别文本；钉图 OCR 高亮也使用同一套原位文本按钮，并按 `object-fit: contain` 的真实图片显示区域重新计算坐标，窗口 resize 后不漂移。对应前端 e2e 已固定 `.ocr-position-text-button`，防止后续退回“只显示框线和列表”。
- macOS OCR 翻译 secret store 已从废弃 `SecKeychain*` API 迁移到 `SecItemCopyMatching` / `SecItemAdd` / `SecItemUpdate` / `SecItemDelete`，保留 keychain 不可用时的本地 0600 fallback。`release-config-check` 已加入 forbidden gate，禁止 `SecKeychain` 旧符号重新进入 `keychain_darwin.go`。
- v0.1.30 发布失败根因已确认并修复：`Verify published release downloads` 中 `-Architectures x64,arm64` 被 PowerShell 当作单个参数，触发 ValidateSet 失败。`verify-release-artifacts.ps1` 已新增 `Normalize-Architectures`，保留 workflow 的单参数逗号格式并在脚本内拆分、去重、校验 x64/arm64。
- v0.1.32 Release workflow 已通过：Release Gate、OCR Model Package、Windows x64/ARM64、macOS x64/ARM64、Linux x64/ARM64 和 Publish GitHub Release 全部 success；Actions run 为 <https://github.com/lemon-casino/RecordingFreedom/actions/runs/28810435505>。GitHub Release 包含 17 个资产，包括 Windows portable/setup、macOS app.zip、Linux portable.tar.gz、stable OCR 模型包、catalog 和 SHA256SUMS。Actions 已上传 `RecordingFreedom-release-download-verification` artifact，说明发布后下载复验已在 workflow 中通过。
- 已验证项：`go test ./...`、`go test -tags gtk3 ./...`、`cd app/frontend && npm run build`、`cd app/frontend && npm run test:e2e`、`cd app && go run ./cmd/release-config-check`、`git diff --check`、以及 `verify-release-artifacts.ps1 -Targets ocr-models -Architectures x64,arm64` 均通过。
- 未扩大结论：上述内容只代表 OCR UI 原位复制、macOS Keychain warning 修复、发布复验参数修复已经收口；真实 Wails 桌面 OCR evidence、真实 provider/secret store 桌面回验、PP-OCRv6 release-ready catalog 和全平台发布产物下载后完整回验仍按本计划继续验收。

## 验收清单

功能验收：

- 区域截图 OCR 可用。
- 全屏截图 OCR 可用。
- 滚动截图 OCR 可用。
- 截图历史 OCR 可用。
- 钉图 OCR 高亮可用。
- 画板背景 OCR 可用。
- 画板选中图片 OCR 可用。
- OCR 原文复制可用。
- OCR 翻译可用且默认不联网。
- 模型切换可用。

稳定性验收：

- worker 崩溃主程序不崩。
- 模型缺失时显示可恢复错误。
- 模型校验失败不会进入可用列表。
- OCR 失败不影响截图保存。
- OCR 面板不扩大胶囊。
- OCR 面板不造成胶囊闪烁。

跨平台验收：

- Windows x64。
- Windows ARM64。
- macOS x64。
- macOS ARM64。
- Linux x64。
- Linux ARM64。

性能验收：

- 小区域截图 OCR 目标 1 秒内返回。
- 1080p 全屏截图 OCR 目标 2 秒内返回。
- 长滚动图允许切片后台处理，但 UI 不能卡死。
- worker 冷启动目标 5 秒内完成。

## 风险和控制

- 模型过大：只内置一个 stable 模型，latest/quality 下载。
- PP-OCRv6 兼容性：保留 PP-OCRv5 stable fallback。
- ONNX Runtime 加载失败：隔离在 worker，主程序显示诊断。
- Windows ARM64 构建风险：worker 单独构建和测试，不绑死主程序。
- 翻译隐私风险：默认关闭，首次启用明确提示。
- 长图 OCR 重复文本：滚动长图切片后做 block 去重。
- UI 回归：所有面板走 floating panel，不允许主胶囊 expanded。

## 完成定义

该功能只有同时满足以下条件才算完成：

- OCR 在 Windows、macOS、Linux 的目标架构上可离线运行。
- 区域截图、全屏截图、滚动截图、截图历史、钉图、画板都走同一套 OCR 服务。
- OCR 结果可缓存、可持久化、可复用。
- 翻译是显式配置后的可选能力，不是默认联网。
- 最新模型有可安装、可验证、可回滚通道。
- CI 验证 worker、runtime、model manifest、smoke 图片和前端交互。
- 主胶囊不因为 OCR 或翻译面板扩大、闪烁或抖动。
