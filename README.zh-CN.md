# agent-web-fetch

[English](./README.md) | **简体中文**

一个面向 MCP-capable agent（Claude Code、ZCode，以及任何支持 Model Context Protocol 的客户端）的**免费、不限量的网页阅读器**。它接收 agent 已经持有的 URL，返回该页面的正文内容，格式是干净、对模型友好的 markdown —— 是众多 agent 客户端内置的付费、限流 fetch 工具的替代品。

- **免费且不限量** —— 在你的本机通过普通 HTTP 运行。无需 API key，没有配额，没有月度限额，不会遇到 429。
- **启动可靠** —— 以单个静态链接的二进制分发。不用 `npx`，无需安装运行时，启动时不联网。第一次就能连上，每次都能。
- **几乎零维护** —— 静态 HTTP 抓取 + Readability 提取算法（Firefox 阅读模式同款引擎）。没有无头浏览器的维护负担，没有反检测军备竞赛要跟进。

## 它做什么

给它一个 URL，返回该页面的主文章，格式为 markdown：

```
fetch({ "url": "https://example.com" })
  → "# Example Domain\n\nThis domain is for use in documentation examples..."
```

它运行 Readability 风格的提取，剥离样板内容（导航、页脚、脚本），只保留主文章。当页面没有明确的主文章时（列表页、仪表盘），它会回退到整个文档，所以永远不会返回空内容。图片 URL 会保留在输出的 markdown 里。

### 工具参数

| 参数            | 必填 | 默认值    | 说明                                                  |
| --------------- | ---- | --------- | ----------------------------------------------------- |
| `url`           | 是   | —         | 要抓取的 URL。必须是绝对的 `http`/`https` URL。       |
| `timeout`       | 否   | `30s`     | 单次请求超时，Go duration 字符串格式（如 `45s`）。    |
| `return_format` | 否   | `markdown`| `markdown` 或 `text`。                                |
| `no_cache`      | 否   | `false`   | 绕过内存缓存，强制重新抓取。                          |

### 它**不**做什么

- **不做网页搜索 / 发现** —— 它抓取你给它的 URL，不会去发现页面。
- **不渲染 JavaScript** —— 需要 JS 才能产出内容的页面会返回稀疏结果。（这是 v1 为了几乎零维护而做的刻意取舍。）
- **不做摘要 / 翻译 / 图片描述** —— 它返回内容，不加工内容。没有付费模型调用。
- **不处理需登录的内容** —— 匿名抓取；登录墙后的页面不在范围内。

## 安装

### 1. 下载对应平台的二进制

从最新 release 下载适合你平台的文件，放在机器上任意位置（比如 `~/bin/` 或 `C:\Users\you\bin\`）：

| 平台              | 文件                                  |
| ----------------- | ------------------------------------- |
| Windows           | `agent-web-fetch-windows-amd64.exe`   |
| macOS（Apple Silicon） | `agent-web-fetch-darwin-arm64`    |
| macOS（Intel）    | `agent-web-fetch-darwin-amd64`        |
| Linux             | `agent-web-fetch-linux-amd64`         |

无需安装器，无需安装运行时（不需要 Node、Python 或 Go）。

### 2. 在你的 MCP 客户端里注册它

这是一个标准的 **stdio MCP server**：没有参数，也没有环境要求。在每个 MCP 客户端里，配置条目的思路都一样 —— 把 `command` 指向二进制的**绝对路径**，`args` 留空：

```json
"chhsich-web-fetch": {
  "type": "stdio",
  "command": "/absolute/path/to/agent-web-fetch",
  "args": []
}
```

不同客户端之间唯一的区别是这个条目**放在哪里**以及确切的键名。常见客户端的具体示例：

**ZCode** —— 把条目加到它的 MCP servers 配置里（以 server 名为键的扁平对象，没有外层包装）：

```json
{
  "chhsich-web-fetch": {
    "type": "stdio",
    "command": "C:/Users/you/bin/agent-web-fetch.exe",
    "args": []
  }
}
```

**Claude Code** —— `~/.claude.json`（Windows 上是 `%USERPROFILE%\.claude.json`），server 放在 `mcpServers` 键下：

```json
{
  "mcpServers": {
    "chhsich-web-fetch": {
      "type": "stdio",
      "command": "C:/Users/you/bin/agent-web-fetch.exe",
      "args": []
    }
  }
}
```

或者用 CLI（效果一样）：`claude mcp add chhsich-web-fetch "C:/Users/you/bin/agent-web-fetch.exe"`

**其他任何 stdio MCP 客户端** —— 找到它存放 MCP server 列表的地方（JSON/YAML 配置、设置界面等），添加一条：type 为 `stdio`，`command` 为二进制的绝对路径，`args` 为 `[]`。这就是全部契约 —— 没有其他参数要设。

> **命名说明：** 上面的键名（`chhsich-web-fetch`）是你在客户端侧给 server 起的标签 —— 想叫什么都行。它暴露的工具名为 `fetch`，所以模型调用的是 `fetch(...)`。两个 server 都提供名为 `fetch` 的工具不会冲突，只要 server 的键名不同（server 名就是命名空间）。

> **路径提示（Windows）：** 用包含 `.exe` 的完整绝对路径。在 JSON 里正斜杠也能用，还能避免反斜杠转义（`"C:/Users/you/bin/agent-web-fetch.exe"`）。

> **Windows SmartScreen 提示：** release 的二进制未签名，所以 Windows 可能在首次运行时弹出"Windows 已保护你的电脑"提示。点 **更多信息 → 仍要运行**。这对未签名二进制是正常的，且只会出现一次。

编辑配置后重启客户端。`fetch` 工具就会和内置工具一起出现，模型可以像调用其他工具一样调用它。

### 3. 验证它能工作

重启客户端后，让模型抓取任意公开页面，例如：

> 用 fetch 工具读取 https://example.com

你应该会拿回该页面的内容（markdown 格式）。如果什么都没返回或工具缺失，设置 `WEB_FETCH_LOG`（见下文）并检查日志文件。

### 4.（可选）启用文件日志

默认情况下 server 是静默的（stdout 是 MCP 传输通道，必须保持干净）。如需将诊断信息写入文件以排查问题：

把 `WEB_FETCH_LOG` 环境变量设为一个文件路径。日志会追加到那里，绝不触碰 stdout。

## 从源码构建

需要 Go 1.22+。

```bash
# 把四个平台的二进制都构建到 dist/
./build.sh          # 任意 bash（Git Bash、WSL、Linux、macOS）
# 或者在 unix 上用 make：
make release
```

每个二进制都是静态链接（`CGO_ENABLED=0`），没有外部运行时依赖。用 `go test ./...` 运行测试套件。

## 工作原理

```
URL → 校验（http/https、绝对地址）→ HTTP GET（真实浏览器 headers）
   → Readability 提取 → markdown → 内存缓存（1 小时 TTL）
                                  ↘ 整篇文档回退（永不返回空）
```

抓取管线是一个单一的深度模块（`internal/fetch`），通过 stdio 暴露为一个 MCP 工具（`internal/mcpserver`）。每一种失败（超时、非 2xx、错误的内容类型，甚至被 recover 的 panic）都会作为模型可读的结构化错误返回 —— server 进程永远不会崩溃。

项目术语表见 `CONTEXT.md`，架构决策（MCP + 二进制分发、匿名抓取、静态提取、Go、启动可靠性）见 `docs/adr/`。
