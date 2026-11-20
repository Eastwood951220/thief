# thief

用 Go 与 macOS 原生 AppKit 编写的本地小说阅读器。支持 TXT、EPUB 纯文字导入、章节识别、书架与进度保存，在菜单栏或悬浮区域阅读。

## 运行与构建

需要 macOS、Xcode Command Line Tools 和 Go 1.26 及以上。开发验证使用 Go 1.27.4 / Apple Silicon。最低系统目标为 macOS 13，旧版本系统及 Intel 实机兼容性尚未验证。

```sh
brew install go
bash scripts/build-app.sh
```

构建产物在 `dist/arm64/thief.app`（Intel 构建为 `dist/amd64/thief.app`），双击打开。构建脚本使用本地 ad-hoc 签名，不包含开发者证书或 Apple 公证。

```sh
# 本机运行开发版本
go run ./cmd/thief

# 指定架构构建；需本机 Xcode SDK 支持
ARCH=amd64 bash scripts/build-app.sh

# 核心、原生排版测试与静态检查
CGO_CFLAGS=-mmacosx-version-min=13.0 CGO_LDFLAGS=-mmacosx-version-min=13.0 go test ./...
go test -race ./internal/importer ./internal/library ./internal/reader ./internal/app
go vet ./...
```

## 使用

1. 在“我的书架”导入本地 TXT 或 EPUB；导入面板展开“选项”可指定 TXT 编码、章节/卷识别正则。
2. 首次默认使用悬浮阅读；书架上显示当前模式，可直接切换到菜单栏阅读。双击书籍或选择“继续阅读”。右键书籍选择“章节列表”，可搜索章节标题或序号并跳转。
3. “设置”集中调整阅读模式、区域尺寸与位置、背景和文字的独立颜色/透明度、字号、行距、收起延迟、菜单栏宽度和快捷键。
4. 背景和文字颜色都支持颜色选择器及 `#RRGGBB` 输入。透明度 0% 完全可见，100% 完全透明。
5. 悬浮区可拖动，右下角可调整大小。启用自动收起后移出区域收成图标，移入图标恢复；右键可打开章节列表和书架。

菜单栏项目显示当前一段正文；点击菜单栏文字/图标可打开操作菜单。菜单栏显示空间受刘海、其他图标与当前应用菜单限制，空间不足可减小菜单栏宽度或切换悬浮模式。

| 操作 | 默认全局快捷键 |
| --- | --- |
| 上一页 / 下一页 | Control + Option + ← / → |
| 上一章 / 下一章 | Control + Option + ↑ / ↓ |
| 隐藏 / 恢复正文 | Control + Option + H |
| 切换阅读模式 | Control + Option + M |

快捷键使用系统注册机制，注册冲突会在设置窗口下方提示。管理窗口关闭后仍继续运行，通过菜单选择“退出 thief”退出。再次打开应用可恢复书架管理窗口。重启保留阅读位置，正文默认隐藏，需要主动继续阅读。

## 书籍与数据

TXT 支持 UTF-8、带 BOM 的 UTF-16、GB18030/GBK，识别中文章节、卷、英文 Chapter、序章及数字顿号标题，支持高级正则。识别逻辑参考 EPUB Studio 的 TXT 解析思路。

EPUB 按 spine 顺序提取正文，读取 EPUB 2 NCX / EPUB 3 nav 目录与章节锚点，丢弃图片、样式和脚本。无文字的图片书及加密正文不支持；不读取在线资源。

数据保存在 `~/Library/Application Support/thief/`。导入生成应用内部纯文字副本，移动或删除来源文件不影响阅读；从书架移除只删除内部副本及进度。重复导入相同文件保留已有进度。书架索引和设置采用原子替换写入。

开发时可通过 `THIEF_DATA_DIR=/absolute/path go run ./cmd/thief` 使用独立数据目录。`examples/示例小说.txt` 是可导入的原创演示文字。

## 源码

- `internal/importer`：TXT / EPUB 解析与编码。
- `internal/library`：书架、配置及阅读位置持久化。
- `internal/reader`：分页锚点、跨章导航及设置校验。
- `internal/app`：应用命令、异步导入与隐藏/收起状态。
- `internal/native`：AppKit 窗口、菜单栏、文本排版与系统快捷键。

设计和实现计划位于 `docs/superpowers/`；原生界面验证范围见 `docs/verification/2026-11-20-macos.md`。
