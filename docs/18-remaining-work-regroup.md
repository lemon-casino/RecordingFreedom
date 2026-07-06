# 剩余功能与问题归拢

更新时间：2026-07-06

本文档用于停止“一个小点接一个小点”的推进方式，把当前剩余功能、问题和验收口径重新归拢成可批量交付的任务。后续只按整批推进：一批内先完成所有代码和证据口径，再集中验证一次，再更新文档；不再改一个点就 build、测一次、发布一次。

## 当前结论

不是只剩测试。

当前已经有大量基础能力接线：OCR worker、模型注册、截图/画板/钉图入口、翻译 provider 壳、evidence plan/export/check、release-config-check、跨平台发布工具链门禁等。但这些还不能等同于用户可验收完成，因为缺少真实 Wails 桌面运行证据、真实发布包下载后回验、真实 provider/secret store 回验和 latest/quality 模型质量闭环。

2026-07-06 本轮重新归拢后的判断：

- 已经完成的是“验收工具链继续变严格”：真实桌面 evidence session 边界、data-root 预检、app-log/job-events 裁剪、checker/release gate、三端发布包工具纳入门禁。
- 还没有完成的是“用户下载软件后能验收的 OCR/翻译能力”：缺真实 Wails 桌面一包到底 evidence，缺真实 provider/secret store 回验，缺 latest/quality 模型质量闭环，缺发布产物下载后完整回验。
- 接下来不能再把 checker、单测、脚本、workflow 的零散补强当成一个个交付点；它们只能服务于下面四个批次的集中验收。

后续剩余工作归拢为四个主批次：

1. A. 真实桌面 OCR 入口闭环。
2. B. 翻译 provider 与隐私闭环。
3. C. 模型通道与质量闭环。
4. D. 发布与长期防回归闭环。

执行顺序固定为 `A -> B -> C -> D`。A 未通过前，不进入 B/C/D；B 未通过前，不宣称翻译可验收；C 未通过前，不开放 latest/quality 给普通用户；D 未通过前，不宣称全平台 OCR/翻译发布完成。

## 当前暂停项

- 暂停继续新增 OCR/翻译入口。
- 暂停逐个入口补零散测试。
- 暂停发布“已完成 OCR/翻译”的结论。
- 暂停把浏览器 e2e、Go 单测、worker smoke 单独当作用户验收结论。
- 暂停把源码目录、本机 `go run` 或开发机数据当作发布包验收。
- 暂停 latest/quality 模型对普通用户 release-ready 展示。

## A. 真实桌面 OCR 入口闭环

目标：把用户能看到、能点击、能验收的 OCR 入口一次性拉齐。

范围：

- 区域截图、全屏截图、窗口截图、焦点窗口截图、滚动截图。
- 截图历史里的手动识别、自动识别、重试、打开结果、复制文字。
- 钉图窗口里的识别、高亮、复制、打开结果、关闭重开恢复。
- 录制前画板、录制中 annotation-overlay、画板整图、画板选中图片 OCR。
- OCR result floating panel 的图片预览、polygon 坐标、block 高亮、复制。

已有基础：

- OCR 队列、缓存、sidecar、真实 worker 第一版已经接线。
- OCR result floating panel、钉图 OCR、画板 OCR、滚动长图切片已经部分落地。
- `ocr-desktop-evidence-plan/export/check` 已经固定 evidence 合同。
- visual checklist、app log、job events、results、sourceId/resultId 链路检查已经开始加固。
- 2026-07-06 已收口 data-root 预检：`ocr-desktop-evidence-plan -data-root` 会检查 app-log、job-events、results、result image、client render、录制中 annotation packageDir 匹配，并新增 run window 合同，拒绝缺 `timestamp` 的 app-log 或跨历史运行拼接出的 evidence。
- 2026-07-06 已把 `data-root-precheck.json` 固定为最终 evidence 包必需文件：exporter 会写入该文件并嵌入 checklist，checker 会验证它存在、通过、source 完整且 run window 存在，release gate 防止回退成“只导出视觉 checklist”。
- 2026-07-06 已把最终 evidence 包的 `app-log.jsonl` 裁剪到 `data-root-precheck.json` 的 app event window，历史日志不会随包导出；checker 同时拒绝窗口外 app-log 事件，避免历史日志污染验收或参与误匹配。
- 2026-07-06 已把最终 evidence 包的 `ocr-job-events.jsonl` 收敛到当前 source/result 链路，只保留每个必需 source 的 queued 和 ready 事件；checker 会拒绝无关 source、错误 resultId 或非 queued/ready 的 job event，避免历史 OCR job 污染验收包。
- 2026-07-06 已新增真实桌面 evidence session 边界：`ocr-desktop-evidence-session` 可写入 `session-start/session-end`，`data-root-precheck.json`、`visual-capture-checklist` 和 `ocr-desktop-evidence-check` 都必须验证同一个 session id 与同一时间窗口，避免把历史 OCR 结果、历史日志和新截图拼成假 evidence。
- 2026-07-06 已把 session 工具纳入 CI/Release、Windows portable、macOS app、Linux portable、三端 verifier、发布后下载回验和导出脚本工具目录模式；这只是 A 批次验收边界完成，不代表真实桌面 OCR 入口已经验收完成。

剩余问题：

- 缺真实 Wails 桌面一包到底 evidence。
- 真实桌面窗口中 OCR result panel、polygon、高亮联动还没有统一录屏或 evidence 包。
- 画板选中图片必须证明使用自己的 `imagePath`，不能 fallback 到整张画板。
- 录制中 annotation OCR 必须证明与录制、鼠标、写盘、绘制并行时不阻塞录制。
- evidence 不能由假图、小图、浏览器 fallback 或单个 sourceKind 冒充整包验收。

本批重新拆分：

| 子批 | 状态 | 内容 | 完成标准 |
| --- | --- | --- | --- |
| A0. Evidence 边界基础设施 | 已接线，需一次 `git diff --check` 收口 | session-start/session-end、data-root precheck、checker、exporter、release gate、三端工具 staging | 相关 Go 单测、脚本静态解析、release-config-check 已通过，且文档记录清楚“不等于桌面验收完成” |
| A1. 真实桌面 runbook | 已落地，等待 A2 真实执行 | 规定用户/测试者如何启动 session、逐项操作 OCR 入口、结束 session、导出 evidence | `19-ocr-desktop-evidence-runbook.md` 已落地，`ocr-desktop-evidence-plan` 生成的 checklist 已包含 session boundary runbook |
| A2. 一包到底桌面采集 | 未完成，采集入口缺口已补 | 用真实 Wails 桌面覆盖所有截图、历史、钉图、画板、录制中 annotation OCR 入口 | 同一 session 内产生 app-log、job-events、results、visual、checklist；截图/画板面板已提供窗口截图和焦点窗口截图入口 |
| A3. Checker 通过 | 未完成 | 用发布包内 `ocr-desktop-evidence-check` 验收 A2 包 | checker 输出 ok，known failures 必须为 none |
| A4. A 批文档冻结 | 未完成 | 更新 `17-ocr-translation-dispatch-plan.md` 与本文档 | 只声明“OCR 入口进入桌面验收”，不越级声明翻译、模型和全平台发布完成 |

本批验收输出：

- 一份真实桌面 OCR evidence 包。
- 必须覆盖 `region-screenshot`、`full-screenshot`、`window-screenshot`、`focused-window-screenshot`、`scrolling-screenshot`、`pinned-screenshot`、`whiteboard`、`whiteboard-selection` 和录制中 annotation OCR。
- evidence 包必须通过 `ocr-desktop-evidence-plan`、`ocr-desktop-evidence-export`、`ocr-desktop-evidence-check`。
- 视觉证据必须能看到真实图片、真实 OCR block、真实坐标叠层。

通过后只能宣称：

- “OCR 入口进入桌面验收。”

不能宣称：

- “OCR 全部完成。”
- “翻译完成。”
- “全平台发布完成。”

## B. 翻译 provider 与隐私闭环

目标：翻译从本地 smoke 升级为真实 provider 可诊断、可缓存、可脱敏、可发布包验收。

范围：

- `deepl` 和 `openai-compatible` provider。
- OCR 结果浮层、截图历史、钉图、画板共用同一个 `TranslateOcr`。
- Windows DPAPI、macOS Keychain、Linux Secret Service 或 0600 fallback secret store。
- 翻译缓存、强制请求、缓存命中、provider 错误提示。

已有基础：

- Provider 调用、分段校验、缓存 key、未配置保护已经接线。
- API key 已迁移到 secret store，settings 不应保存明文 key。
- 本地 OpenAI-compatible smoke 和 release gate 已有基础。

剩余问题：

- 缺真实 DeepL 或外部 OpenAI-compatible 桌面回验。
- 缺 macOS Keychain、Linux Secret Service、Windows DPAPI 的实机 Save/Load/Delete evidence。
- 缺发布包内 `ocr-translation-smoke` 的下载后回验。
- 缺 evidence/log/settings 证明 key 不落盘、不进日志、不进入 evidence JSON。

本批验收输出：

- `ocr-translation-smoke` 同时支持离线 local provider 和外部 OpenAI-compatible provider。
- 外部 provider 只从环境变量读取 key，证据只记录 provider mode、model、语言、是否提供 key、缓存命中和脱敏 base URL。
- Windows/macOS/Linux 发布包都包含并校验 translation smoke 和 secret store smoke。

通过后只能宣称：

- “翻译进入桌面验收。”

## C. 模型通道与质量闭环

目标：stable 保持默认可用；latest/quality 只有通过真实质量、跨平台 smoke 和回退验证后才允许给用户切换。

范围：

- stable/latest/quality catalog。
- PP-OCRv6 latest candidate、quality candidate。
- 模型下载、取消、重试、安装、删除、切换 active、失败回退 stable。
- ONNX Runtime 与 worker capabilities 矩阵。

已有基础：

- stable 模型包、下载校验、模型注册和 worker smoke 已接线。
- 设置页已有模型状态、导入、下载、切换、删除的基础 UI。
- release catalog 生成和校验已有门禁。

剩余问题：

- PP-OCRv6 latest/quality 仍是 candidate，不能直接展示为 release-ready。
- 缺真实用户截图样例质量回验。
- 缺真实 Wails 设置页下载、取消、重试、校验失败、确认切换、删除、回退 stable 的桌面证据。
- 缺 release 后真实 catalog 下载、校验、安装和回退回验。

本批验收输出：

- 每个平台生成 `ocr-worker-platform-smoke.json`。
- latest/quality 模型包必须有 manifest、SHA256、smoke、真实截图样例质量记录和失败回退记录。
- 设置页真实桌面 evidence 覆盖下载、取消、重试、确认切换和删除。

通过后才能宣称：

- “模型通道可给用户切换。”

## D. 发布与长期防回归闭环

目标：不再用“本地能跑”代替“发布包可验收”。发布包必须自带 worker、runtime、模型、诊断工具和证据导出工具，并且能从下载产物反向验证。

范围：

- Windows/macOS/Linux x64/ARM64 发布包。
- worker、ONNX Runtime、stable 模型 smoke。
- OCR desktop evidence tools、translation smoke、secret store smoke。
- 下载发布产物后的完整性回验。

已有基础：

- release-config-check、发布 verifier、evidence scripts 已经加固。
- CI/release workflow 已经开始固定 worker/runtime/model/tools staging。
- 发布包 verifier 已接入 OCR/translation 相关工具检查。

剩余问题：

- 下一次实际 Actions 发布 run 还未产出完整 `release-artifact-verification.json`。
- 全平台 `ocr-worker-platform-smoke.json` 仍需要下一次 Actions run 产出。
- 真实桌面 OCR evidence 包还没有作为 release 后验收资产固定。
- 还需要下载发布产物后运行 verifier，而不是只在源码目录验证。

本批验收输出：

- `RecordingFreedom-release-download-verification/release-artifact-verification.json`。
- Windows/macOS/Linux 每个平台的 worker capabilities 和 stable OCR smoke evidence。
- 发布包内工具清单包含 `ocr-desktop-evidence-plan`、`ocr-desktop-evidence-export`、`ocr-desktop-evidence-check`、`ocr-translation-smoke`、`ocr-secret-store-smoke`。
- release-config-check 固定所有工具、脚本、workflow 和文档证据。

通过后才能宣称：

- “全平台 OCR/翻译发布可验收。”

## 横向问题池

这些问题不再单独打散处理，必须归入上面的批次内解决：

- OCR 入口问题归入 A：截图、滚动截图、钉图、画板、录制中 annotation、结果浮层、坐标、高亮、复制。
- 翻译、API key、日志脱敏、缓存问题归入 B。
- 模型下载、切换、质量、回退、runtime 问题归入 C。
- Actions、发布包、跨平台 smoke、下载后验证问题归入 D。
- 胶囊闪烁、浮层占用、下拉窗口问题如果再次出现，只作为 UI 回归单独修复；不能混入 OCR/翻译主线造成批次失控。
- 录制、摄像头、声音、画中画如果用户再次发现回归，必须先复现和归档为独立缺陷；不能在 OCR 批次里顺手改。

## 下一步只允许做的事

下一步只进入 A 批次，且按下面顺序一次性推进：

1. A0 收尾：只跑与 evidence/session 边界直接相关的检查，包含 `git diff --check`、相关 Go 单测、脚本语法解析和 `release-config-check`；不再扩展新功能。
2. A1 已落地：`19-ocr-desktop-evidence-runbook.md` 和 `ocr-desktop-evidence-plan` 生成的 checklist 已明确 session start、逐入口点击顺序、视觉证据命名、session end、导出和 checker 命令。
3. A2 用真实 Wails 桌面一次性覆盖全部 OCR 入口，采集视觉截图/录屏、安全 app log、job events、results，不允许浏览器 fallback 或假图替代。
4. A3 用发布包工具模式跑 `ocr-desktop-evidence-plan/export/check`，checker 不通过就按 A 批问题集中修，不切去 B/C/D。
5. A4 通过后更新 `17-ocr-translation-dispatch-plan.md` 和本文档，只声明 A 批次状态。

A2 当前进展：

- 已核对 OCR queue、open-result、read-result-image、client render 和 annotation save-capture 证据链。
- 已补齐截图/画板面板中的 `窗口截图` 与 `焦点窗口截图` 可点击入口，避免真实采集无法覆盖 `window-screenshot` / `focused-window-screenshot`。
- 已用前端 e2e 和 release-config-check 固定上述入口不再回退。
- 尚未完成真实 Wails 桌面一包到底采集、导出和 checker 通过。

下一步不做：

- 不发布“已完成 OCR/翻译”的版本。
- 不继续新增 OCR 入口。
- 不处理 provider、模型切换或 release catalog，除非 A 批 checker 明确依赖。
- 不混入胶囊、录制、摄像头、声音、画中画 UI 回归；这些如果复现，单独建缺陷批次处理。

## 批次验证原则

- 每批只在收口后集中验证一次。
- 验证命令必须和该批直接相关，避免全量 build/test 反复空跑。
- 失败时先归类到该批的功能、证据、发布工具或文档合同，不再散落成多个临时 TODO。
- 文档必须记录：已完成、未完成、证据位置、验证命令、是否可发布验收版。

## 完成定义

全部四批通过后，才允许宣称 OCR/翻译主线完成：

- OCR 在 Windows、macOS、Linux 目标架构上离线可用。
- 区域截图、全屏截图、滚动截图、截图历史、钉图、画板、录制中 annotation 都走同一套 OCR 服务。
- OCR 结果可缓存、可持久化、可复用、可复制。
- 翻译默认关闭，用户显式配置 provider 后才联网。
- API key 不落盘、不进日志、不进入 evidence。
- latest/quality 模型有可安装、可验证、可回滚通道。
- 发布包下载后能通过 worker/runtime/model/tools/evidence 的完整回验。
- 胶囊不因为 OCR、翻译、列表或浮层扩大、闪烁或抖动。
