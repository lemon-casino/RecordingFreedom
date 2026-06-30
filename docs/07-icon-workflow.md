# 07. 图标更换与重建流程

RecordingFreedom 当前只面向电脑端发布：Windows、macOS、Linux。Android/iOS 图标模板不纳入首版仓库和发布流水线。

本文档用于记录后续更换正式品牌图标时的固定流程。不要手动单独替换 `.ico`、`.icns` 或某一个 PNG 文件；统一从源图生成所有平台需要的图标，避免不同平台图标不一致。

## 源图要求

- 推荐使用正方形 PNG，至少 `1024x1024`。
- 也支持 JPEG 源图，但正式图标优先使用带透明通道的 PNG。
- 非正方形图片会按 contain 方式缩放到正方形画布中，原图比例不会被拉伸。
- 当前 preview 使用的源图示例：

```text
D:\图库\43574409.png
```

## 更换步骤

1. 准备新的源图，例如：

```text
D:\图库\new-recordingfreedom-icon.png
```

2. 进入 Wails 应用目录：

```bash
cd app
```

3. 执行图标重建命令，把 `-source` 改成新源图路径：

```bash
go run ./cmd/icon-build -source "D:\图库\new-recordingfreedom-icon.png" -sizes "16,24,32,48,64,128,256,512,1024"
```

4. 运行验证命令，确认 Go 服务、前端构建和 Wails 桌面包都能通过。

5. 检查 git 变更，只提交图标产物和必要文档，不提交临时构建目录。

## 当前 preview 图标重建命令

在仓库根目录进入 `app/` 后执行：

```bash
cd app
go run ./cmd/icon-build -source "D:\图库\43574409.png" -sizes "16,24,32,48,64,128,256,512,1024"
```

这个命令会生成或更新：

```text
build/icons/icon-16.png
build/icons/icon-24.png
build/icons/icon-32.png
build/icons/icon-48.png
build/icons/icon-64.png
build/icons/icon-128.png
build/icons/icon-256.png
build/icons/icon-512.png
build/icons/icon-1024.png
build/appicon.png
build/windows/icon.ico
build/darwin/icons.icns
frontend/public/appicon.png
frontend/public/wails.png
```

`frontend/public/wails.png` 目前保留为兼容旧引用；真实 favicon 入口是 `frontend/public/appicon.png`。

## 只生成 PNG

如果当前机器没有安装 Wails CLI，可以先只生成 PNG：

```bash
cd app
go run ./cmd/icon-build -source "D:\图库\43574409.png" -sizes "16,24,32,48,64,128,256,512,1024" -skip-wails
```

之后在安装了 Wails CLI 的机器上再执行不带 `-skip-wails` 的命令，补齐 Windows `.ico` 和 macOS `.icns`。

## 可调整参数

默认情况下不需要调整参数。确实需要时，可以使用：

```bash
go run ./cmd/icon-build \
  -source "D:\图库\43574409.png" \
  -sizes "16,32,64,128,256,512,1024" \
  -appicon-size 1024 \
  -frontend-size 256
```

参数说明：

- `-source`：源图路径，支持 PNG/JPEG。
- `-sizes`：输出到 `build/icons/` 的 PNG 尺寸列表，范围是 `8..4096`。
- `-appicon-size`：生成 `build/appicon.png` 的尺寸，Wails 平台图标从它继续生成。
- `-frontend-size`：生成前端 favicon PNG 的尺寸。
- `-skip-wails`：跳过 Wails `.ico/.icns` 生成，仅输出 PNG。

## 验证命令

更换图标后至少运行：

```bash
cd app
go test ./...
go run ./cmd/preview-smoke
cd frontend
npm run build
cd ..
wails3 build
```

如果要模拟 GitHub Actions 的 Linux GTK3 验证路径：

```bash
go test -tags gtk3 ./...
go run -tags gtk3 ./cmd/preview-smoke
```

## 提交检查

更换图标后需要提交以下类型的文件：

```text
app/build/icons/*.png
app/build/appicon.png
app/build/windows/icon.ico
app/build/darwin/icons.icns
app/frontend/public/appicon.png
app/frontend/public/wails.png
```

不需要提交：

```text
app/bin/
app/frontend/dist/
app/frontend/node_modules/
app/build/android/
app/build/ios/
```

提交前确认：

```bash
git status --short
git diff --check
```

如果需要把图标变更带入 preview 发布，先确认 GitHub Actions 能重新构建 Windows、macOS、Linux 三个平台，再创建新的 preview tag。
