# OCR 桌面验收 Runbook

更新时间：2026-07-06

本文档对应 `17-ocr-translation-dispatch-plan.md` 的 A1 颗粒：真实 Wails 桌面 OCR evidence runbook。它只定义如何采集和验收 A 批次 evidence，不宣称 OCR/翻译已经完成。

## 目标

用一次真实 RecordingFreedom 桌面运行，覆盖 OCR 的所有用户入口，并导出可被 `ocr-desktop-evidence-check` 验收的 evidence 包。

必须证明：

- OCR 来源是真实桌面图片，不是浏览器 fallback、小图、假图或 mock 文本。
- `region-screenshot`、`full-screenshot`、`window-screenshot`、`focused-window-screenshot`、`scrolling-screenshot`、`pinned-screenshot`、`whiteboard`、`whiteboard-selection` 全部有真实 result。
- 录制中 annotation OCR 与录制、绘制、鼠标、写盘并行，不阻塞录制。
- OCR result 浮层显示真实预览图、真实 block polygon、高亮、复制入口。
- session-start/session-end 包住同一次真实桌面操作，不能用历史日志拼包。

## 硬性规则

- 只用真实 Wails 桌面应用采集视觉证据。
- session start 必须在打开 OCR 入口或排队 OCR 前写入。
- session end 必须在最后一张视觉证据截图和最后一个 OCR result render 之后写入。
- plan/export/check 必须使用同一个 `DataRoot`。
- 导出阶段不能补写 session marker；导出和检查只消费真实运行留下的 marker。
- `KnownFailures` 必须是 `none`，带已知失败的包只能算诊断包。
- 优先使用发布包内 tools；源码 `go run` 只允许开发自测。

## 目录变量

Windows PowerShell:

```powershell
$ToolsDir = "D:\RecordingFreedom\tools"
$DataRoot = "$env:APPDATA\RecordingFreedom"
$VisualDir = "D:\RecordingFreedom-ocr-evidence\visual"
$EvidenceDir = "D:\RecordingFreedom-ocr-evidence\package"
New-Item -ItemType Directory -Force -Path $VisualDir, $EvidenceDir | Out-Null
```

macOS/Linux Bash:

```bash
TOOLS_DIR="/Applications/RecordingFreedom.app/Contents/MacOS/tools"
DATA_ROOT="${HOME}/Library/Application Support/RecordingFreedom"
VISUAL_DIR="${HOME}/RecordingFreedom-ocr-evidence/visual"
EVIDENCE_DIR="${HOME}/RecordingFreedom-ocr-evidence/package"
mkdir -p "${VISUAL_DIR}" "${EVIDENCE_DIR}"
```

Linux 的 `DATA_ROOT` 按实际发行包数据目录调整；关键是 app、session、plan、export、check 全部使用同一个目录。

## 采集前清单

1. 打开 RecordingFreedom 发布包或待测本地桌面包。
2. 确认 OCR stable 模型可用。
3. 准备一个可识别目标窗口，窗口内必须能看到 `RecordingFreedom` 和 `文字识别` 这类稳定文本。
4. 清空或新建本次 `VisualDir`，避免混入旧截图。
5. 打开截图/画板工具面板，确认能看到 `区域截图`、`全屏截图`、`窗口截图`、`焦点窗口截图`、`滚动截图` 和 `画板`。
6. 先生成 checklist，确认本次需要采集哪些文件。

Windows:

```powershell
& "$ToolsDir\ocr-desktop-evidence-plan.exe" -out-dir $EvidenceDir
```

macOS/Linux:

```bash
"${TOOLS_DIR}/ocr-desktop-evidence-plan" -out-dir "${EVIDENCE_DIR}"
```

## 开始 Session

Windows:

```powershell
$StartReport = & "$ToolsDir\ocr-desktop-evidence-session.exe" -event start -data-root $DataRoot | ConvertFrom-Json
$SessionId = $StartReport.sessionId
$SessionId
```

macOS/Linux:

```bash
SESSION_ID="$("${TOOLS_DIR}/ocr-desktop-evidence-session" -event start -data-root "${DATA_ROOT}" | python3 -c 'import json,sys; print(json.load(sys.stdin)["sessionId"])')"
printf '%s\n' "${SESSION_ID}"
```

保存 `SessionId`。后续结束命令必须使用同一个值。

## 桌面操作顺序

按下面顺序一次性完成，不要跨天、跨版本或跨 data-root 拼接 evidence。

1. 区域截图 OCR
   框选包含稳定文本的区域，保存截图，从截图历史触发 OCR，打开 OCR result 浮层。保存视觉证据为 `region-screenshot-capture.png`。

2. 全屏截图 OCR
   执行全屏或全部屏幕截图，触发 OCR 并打开结果。保存为 `full-screen-capture.png`。

3. 指定窗口截图 OCR
   使用明确的窗口选择流程，不要复用焦点窗口截图。保存为 `window-screenshot-capture.png`。

4. 焦点窗口截图 OCR
   先切换焦点到目标窗口，再执行焦点窗口截图。保存为 `focused-window-screenshot-capture.png`。

5. 滚动截图 OCR
   对可滚动目标执行滚动截图；目标无法滚动时必须走明确 no-scroll fallback，不能保存假长图。保存为 `scrolling-screenshot-capture.png`。

6. 截图历史 ready 状态
   打开截图历史，等待 OCR 项变为 ready，确认结果、复制、钉图、翻译入口可见。保存为 `screenshot-history-ready.png`。

7. OCR result 浮层
   打开任意真实 OCR result 浮层，确认预览图和 polygon 已渲染。保存为 `ocr-result-floating-panel.png`。

8. 钉图 OCR
   将真实截图钉图，触发钉图 OCR，打开高亮，缩放或调整钉图窗口后确认 polygon 仍对齐。保存为 `pinned-screenshot-ocr-highlight.png`。

9. 画板整图 OCR
   打开画板，放入带文字的真实图片或绘制内容，导出画板快照并触发 OCR，打开结果。保存为 `whiteboard-ocr.png`。

10. 画板选中图片 OCR
    选中画板内的图片元素，仅对该图片元素导出并识别。结果预览必须是选中图片，不是整张画板。保存为 `whiteboard-selection-ocr.png`。

11. 录制中 annotation OCR 安全性
    开始真实录制，打开录制中 annotation overlay，保存 annotation 快照到录制包，排队后台 OCR，在录制仍然运行时打开/观察 OCR 状态和结果。保存为 `recording-annotation-ocr-safety.png`。

## 结束 Session

Windows:

```powershell
& "$ToolsDir\ocr-desktop-evidence-session.exe" -event end -data-root $DataRoot -session-id $SessionId
```

macOS/Linux:

```bash
"${TOOLS_DIR}/ocr-desktop-evidence-session" -event end -data-root "${DATA_ROOT}" -session-id "${SESSION_ID}"
```

## 导出并检查

Windows:

```powershell
.\scripts\export-ocr-desktop-evidence.ps1 `
  -VisualDir $VisualDir `
  -EvidenceDir $EvidenceDir `
  -DataRoot $DataRoot `
  -ToolsDir $ToolsDir `
  -KnownFailures none `
  -MustContain RecordingFreedom `
  -MustContain 文字识别
```

macOS/Linux:

```bash
./scripts/export-ocr-desktop-evidence.sh \
  --visual-dir "${VISUAL_DIR}" \
  --evidence-dir "${EVIDENCE_DIR}" \
  --data-root "${DATA_ROOT}" \
  --tools-dir "${TOOLS_DIR}" \
  --known-failures none \
  --must-contain RecordingFreedom \
  --must-contain 文字识别
```

通过条件：

- `visual-capture-checklist.json` 的 `checkComplete` 为 `true`。
- `data-root-precheck.json` 的 `checkComplete` 为 `true`。
- `check-report.json` 整体为通过。
- `app-log.jsonl` 只包含本次 session 窗口内的日志。
- `ocr-job-events.jsonl` 只包含本次 source/result 链路。
- `known failures` 为 `none`。

## 失败归类

- 缺视觉截图：回到对应桌面操作补采，不改 checker。
- 缺 session：重新从 session start 开始整包采集，不能导出阶段补 marker。
- 缺 sourceKind/resultId：修对应 OCR 入口或排队链路。
- polygon 不对齐：修 result image、坐标映射或预览缩放。
- annotation 录制中失败：归入 A 批录制中 OCR 安全问题，不进入 B/C/D。
- provider/key/翻译失败：不在 A 批修，归入 B 批。
- latest/quality 模型质量问题：不在 A 批修，归入 C 批。

## A1 完成定义

A1 只在以下条件成立时算完成：

- 本文档存在，并被 `docs/README.md` 索引。
- `ocr-desktop-evidence-plan` 生成的 `visual-capture-checklist.md/json` 包含 session boundary runbook。
- release-config-check 固定 session boundary runbook 关键字，防止回退。

A1 完成后，只能进入 A2 真实桌面一包到底采集；不能宣称 OCR/翻译完成。
