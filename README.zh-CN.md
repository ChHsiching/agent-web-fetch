# agent-web-fetch

[English](./README.md) | **简体中文**

一个**免费、不限量的网页阅读器**，面向任何支持 Model Context Protocol 的 agent 客户端，包括 Claude Code 和 ZCode。它接收 agent 已经持有的 URL，返回该页面的正文内容，格式是干净、对模型友好的 markdown。它是众多 agent 客户端内置的付费、限流 fetch 工具的替代品。

## 为什么用它，而不是付费 web-reader

- **免费且不限量。** 在你的本机通过普通 HTTP 运行。无需 API key，没有配额，没有月度限额，不会遇到 429。
- **启动可靠。** 以单个静态链接的二进制分发。不用 `npx`，无需安装运行时，启动时不联网。第一次就能连上，每次都能。
- **几乎零维护。** 静态 HTTP 抓取，加上 Readability 提取算法，即 Firefox 阅读模式同款引擎。没有无头浏览器的维护负担，没有反检测军备竞赛要跟进。

托管式 web-reader 服务把你的 URL 和页面内容送到厂商云端，并按请求计费。这个工具做同样的工作，但在你自己的机器上、由 MCP 客户端管理的进程里完成，每页成本为 0。

## 它做什么

给它一个 URL，返回该页面的主文章，格式为 markdown：

```
fetch({ "url": "https://example.com" })
  → "# Example Domain\n\nThis domain is for use in documentation examples..."
```

它运行 Readability 风格的提取，剥离样板内容（导航、页脚、脚本），只保留主文章。当页面没有明确的主文章时（列表页、仪表盘），它会回退到整个文档，所以永远不会返回空内容。图片 URL 会保留在输出的 markdown 里。

本身就是纯文本的 URL 会完全跳过提取。这覆盖所有 `text/*` 类型（markdown、纯文本、CSV 等）、JSON（含 `+json` 厂商类型）、XML、YAML、JavaScript。正文解码为 UTF-8 后**原样返回**，不做任何包装或重排版。对这类纯文本文件，`return_format` 的两个取值（`markdown` 和 `text`）返回的内容完全相同。文档本身就是对模型友好的形式。二进制及其他非文本类型（PDF、图片、音视频）仍会被明确报错拒绝。

### 工具参数

| 参数            | 必填 | 默认值    | 说明                                                  |
| --------------- | ---- | --------- | ----------------------------------------------------- |
| `url`           | 是   | 无        | 要抓取的 URL。必须是绝对的 `http`/`https` URL。       |
| `timeout`       | 否   | `30s`     | 单次请求超时，Go duration 字符串格式（如 `45s`）。    |
| `return_format` | 否   | `markdown`| `markdown` 或 `text`。                                |
| `no_cache`      | 否   | `false`   | 绕过内存缓存，强制重新抓取。                          |

### 它**不**做什么

- **不做网页搜索 / 发现。** 它抓取你给它的 URL，不会去发现页面。
- **不渲染 JavaScript。** 需要 JS 才能产出内容的页面会返回稀疏结果。这是 v1 为了几乎零维护而做的刻意取舍。
- **不做摘要 / 翻译 / 图片描述。** 它返回内容，不加工内容。没有付费模型调用。
- **不处理需登录的内容。** 匿名抓取；登录墙后的页面不在范围内。

## 快速开始

### 1. 下载对应平台的二进制

从最新 release 下载适合你平台的文件：

| 平台              | 文件                                  |
| ----------------- | ------------------------------------- |
| Windows           | `agent-web-fetch-windows-amd64.exe`   |
| macOS（Apple Silicon） | `agent-web-fetch-darwin-arm64`    |
| macOS（Intel）    | `agent-web-fetch-darwin-amd64`        |
| Linux             | `agent-web-fetch-linux-amd64`         |

无需安装器，也无需安装运行时。不需要 Node、Python 或 Go。

**文件放哪里：** 每个平台都有一个**推荐的用户级位置**。它不需要管理员权限，是用户自己安装的程序约定俗成的存放点：

| 平台 | 推荐位置 |
| ---- | -------- |
| Windows | `%LOCALAPPDATA%\Programs\agent-web-fetch\agent-web-fetch.exe` |
| macOS / Linux | `~/.local/bin/agent-web-fetch` |

不过 MCP 其实不在乎文件放在哪。它是通过你配置里的绝对路径来启动二进制的，所以放在任何你有读/执行权限的地方都行，也**不需要**加到 `PATH`。只要别放进其他用户的目录，或需要管理员权限的系统文件夹。下面的示例用的是推荐位置；如果你放在别处，把路径替换掉即可。

<details>
<summary>偏好 <code>go install</code>（需要 Go 1.26+）</summary>

```bash
go install github.com/ChHsiching/agent-web-fetch/cmd/agent-web-fetch@latest
```

二进制会装到 `$GOBIN`，若未设置则是 `$GOPATH/bin`。Windows 上通常是 `%USERPROFILE%\go\bin`。把 MCP 配置里的路径指向那个绝对路径即可。
</details>

<details>
<summary>从源码构建</summary>

需要 Go 1.26+，与 `go.mod` 一致。

```bash
# 把四个平台的二进制都构建到 dist/
./build.sh          # 任意 bash（Git Bash、WSL、Linux、macOS）
# 或者在 unix 上用 make：
make release
```

每个二进制都是静态链接（`CGO_ENABLED=0`），没有外部运行时依赖。
</details>

### 2. 在你的 MCP 客户端里注册它

这是一个标准的 **stdio MCP server**：没有参数，也没有环境要求。**你不需要自己启动它。** MCP 客户端会根据下面的配置条目拉起并管理进程。手动运行这个二进制对 agent 没有任何作用：它只会停在原地，等待 stdin 上的 MCP 消息。所有 stdio MCP server 都是这样；参见[官方 transports 说明](https://github.com/modelcontextprotocol/servers#transports)。

在每个 MCP 客户端里，配置条目的思路都一样。把 `command` 指向二进制的**绝对路径**，`args` 留空：

```json
"chhsich-web-fetch": {
  "type": "stdio",
  "command": "/absolute/path/to/agent-web-fetch",
  "args": []
}
```

不同客户端之间的区别在于这个条目**放在哪里**、确切的键名，个别客户端（如 opencode）的条目形状也不同。常见客户端的具体示例：

<details>
<summary><strong>ZCode：</strong> 以 server 名为键的扁平对象，没有外层包装</summary>

```json
{
  "chhsich-web-fetch": {
    "type": "stdio",
    "command": "C:/Users/<username>/AppData/Local/Programs/agent-web-fetch/agent-web-fetch.exe",
    "args": []
  }
}
```
</details>

<details>
<summary><strong>Claude Code：</strong> <code>~/.claude.json</code> 的 <code>mcpServers</code> 键下；Windows 上是 <code>%USERPROFILE%\.claude.json</code></summary>

```json
{
  "mcpServers": {
    "chhsich-web-fetch": {
      "type": "stdio",
      "command": "C:/Users/<username>/AppData/Local/Programs/agent-web-fetch/agent-web-fetch.exe",
      "args": []
    }
  }
}
```

或者用 CLI，效果一样：`claude mcp add chhsich-web-fetch "C:/Users/<username>/AppData/Local/Programs/agent-web-fetch/agent-web-fetch.exe"`
</details>

<details>
<summary><strong>Zed：</strong> <code>context_servers</code> 键，没有 <code>type</code> 字段</summary>

最简单的方式是走界面：**Settings → AI → MCP Servers → Add Server → Add Local Server**，把路径指向 exe。也可以编辑设置文件。通过命令面板打开 `zed: open settings file`；Windows 上位于 `C:\Users\<username>\AppData\Roaming\Zed\settings.json`。

```json
{
  "context_servers": {
    "chhsich-web-fetch": {
      "command": "C:/Users/<username>/AppData/Local/Programs/agent-web-fetch/agent-web-fetch.exe",
      "args": []
    }
  }
}
```
</details>

<details>
<summary><strong>opencode：</strong> <code>mcp</code> 键、<code>"type": "local"</code>，且 <code>command</code> 是数组</summary>

配置写在项目根目录的 `opencode.json`，或全局的 `~/.config/opencode/opencode.json`。Windows 上是 `%USERPROFILE%\.config\opencode\opencode.json`。

```json
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "chhsich-web-fetch": {
      "type": "local",
      "command": ["C:/Users/<username>/AppData/Local/Programs/agent-web-fetch/agent-web-fetch.exe"]
    }
  }
}
```
</details>

**其他任何 stdio MCP 客户端。** 找到它存放 MCP server 列表的地方，无论是 JSON/YAML 配置还是设置界面，添加一条：type 为 `stdio`，`command` 为二进制的绝对路径，`args` 为 `[]`。这就是全部契约。没有其他参数要设。

> **替换路径：** 上面的示例用的是推荐安装位置，`<username>` 是占位符。把 `<username>` 换成你的实际用户名；如果你的客户端支持展开环境变量，也可直接用 `%LOCALAPPDATA%`。如果你把二进制放在了别处，相应调整路径。

> **命名说明：** 上面的键名 `chhsich-web-fetch` 是你在客户端侧给 server 起的标签。想叫什么都行。它暴露的工具名为 `fetch`，所以模型调用的是 `fetch(...)`。两个 server 都提供名为 `fetch` 的工具不会冲突，只要 server 的键名不同。server 名就是命名空间。

> **路径提示（Windows）：** 用包含 `.exe` 的完整绝对路径。JSON 里正斜杠也能用，还能避免反斜杠转义。

> **Windows SmartScreen 提示：** release 的二进制未签名，所以 Windows 可能在首次运行时弹出"Windows 已保护你的电脑"提示。点 **更多信息 → 仍要运行**。这对未签名二进制是正常的，且只会出现一次。

编辑配置后重启客户端。`fetch` 工具就会和内置工具一起出现，模型可以像调用其他工具一样调用它。

### 3. 用起来

先确认 server 确实注册上了。每个客户端都有状态视图：

- **Zed**：Settings → AI → MCP Servers，`chhsich-web-fetch` 旁边的指示点应为绿色，即 "Server is active"。
- **opencode**：在终端运行 `opencode mcp list`。
- **Claude Code**：运行 `claude mcp list`。

然后让模型抓取任意公开页面，并在提示词里点名这个 server，让它优先选这个工具而不是内置的：

> 用 chhsich-web-fetch 的 fetch 工具读取 https://example.com

你应该会拿回该页面的内容（markdown 格式）。

**如果 agent 一直用它自带的 fetch：** Zed 和 opencode 都内置了自己的网页抓取工具，注册本 server 并**不会**禁用它。模型可能只是继续选内置的那个，这看起来和"工具没生效"一模一样，而实际上注册是好的。两个解决办法：

- 像上面那样在提示词里点名 server，或在 `AGENTS.md` / 规则文件里加一句：*"需要读网页时，用 `chhsich-web-fetch` 的 fetch 工具。"*
- 或者关掉内置工具。Zed 可以建一个 [agent profile](https://zed.dev/docs/ai/agent-profiles)，关掉内置 `fetch`、只开本 server 的工具。opencode 可以在 `opencode.json` 里用 `tools` 键禁用，例如 `"tools": { "webfetch": false }`。

如果 server 始终不出现或调用失败，见[故障排查](#故障排查)。

## 用法

纯文本文件会逐字节返回。下面是这个工具从 `raw.githubusercontent.com` 读取它自己的 README：

```
fetch({ "url": "https://raw.githubusercontent.com/ChHsiching/agent-web-fetch/main/README.md" })
  → "# agent-web-fetch\n\n**English** | [简体中文](./README.zh-CN.md)\n\nA **free, uncapped web reader*..."
```

返回的是完整的 markdown 源码，不添加代码围栏，不做提取。JSON 的处理方式相同。`package.json`、API 响应、GeoJSON 都原样透传。

普通网页走提取。长文章返回的是正文 markdown，导航、页脚、脚本都被剥离。需要时可用参数：

```
fetch({ "url": "https://example.com", "timeout": "45s", "no_cache": true })
```

提示词里的说法会映射到参数，所以通常不需要把参数念出来：

- *"用 chhsich-web-fetch 读一下……"* → 模型会选这个 server 的工具，而不是内置的。
- *"重新抓一次，别用缓存"* → 模型会设 `no_cache: true`。
- *"这个网站很慢"* → 更大的 `timeout`。

## 对比

| | agent-web-fetch | 托管式 web-reader MCP |
| --- | --- | --- |
| API key / 账号 | 不需要 | 需要，有免费额度 |
| 每页成本 | $0，不限量 | 按额度计费 |
| 内容在哪里处理 | 你的机器 | 厂商云端 |
| 运行时 | 单个静态二进制，无依赖 | `npx` + Node，或远程 URL |
| JavaScript 渲染 | 不支持 | 部分支持 |
| 许可证 | AGPL-3.0，附商业授权选项 | 不一，从 AGPL 到闭源都有 |

以上为 2026 年 10 月的情况。各服务定价请以其当前文档为准。"托管式 web-reader MCP"指 Firecrawl / Jina Reader 这一类工具。

## 故障排查

**server 始终不出现，或调用失败。** 默认情况下 server 是静默的：stdout 是 MCP 传输通道，必须保持干净。在配置条目里给 server 设置 `WEB_FETCH_LOG` 环境变量指向一个文件路径。支持按 server 注入环境变量的客户端可写 `"env": { "WEB_FETCH_LOG": "C:/Users/<username>/awf.log" }`。重启客户端，复现一次调用，然后查看日志。

<details>
<summary>免费？有什么代价？</summary>

定价意义上没有套路：没有任何要付费的服务器，因为二进制就运行在你的机器上。真正的取舍在[它**不**做什么](#它不做什么)里列着：不渲染 JavaScript、不抓需登录的页面。许可证是 AGPL-3.0，使用、修改、分发在其条款下均免费。
</details>

<details>
<summary>可以在公司里用吗？</summary>

可以。AGPL-3.0 下公司内部使用免费。只有当你**分发**二进制、或把它作为网络服务提供时，才触发开源义务：衍生作品须以 AGPL-3.0 开源。闭源商用分发需另行获取商业授权，见[许可证](#许可证)。
</details>

## 工作原理

```
URL → 校验（http/https、绝对地址）→ HTTP GET（真实浏览器 headers）
   → Readability 提取 → markdown → 内存缓存（1 小时 TTL）
                                  ↘ 整篇文档回退（永不返回空）
   ↘ 纯文本（text/*、JSON、XML、YAML、JS）→ 原样透传（解码为 UTF-8）
```

抓取管线是一个单一的深度模块（`internal/fetch`），通过 stdio 暴露为一个 MCP 工具（`internal/mcpserver`）。每一种失败（超时、非 2xx、错误的内容类型，甚至被 recover 的 panic）都会作为模型可读的结构化错误返回。server 进程永远不会崩溃。

项目术语表见 `CONTEXT.md`，架构决策（MCP + 二进制分发、匿名抓取、静态提取、Go、启动可靠性）见 `docs/adr/`。

## 参与贡献

欢迎在 [GitHub](https://github.com/ChHsiching/agent-web-fetch) 提 issue 和 PR。缺陷报告最好带上：你抓取的 URL、期望的结果、实际拿到的内容。

本地开发：

```bash
go test ./...     # 测试套件
go vet ./...      # 静态检查
./build.sh        # 把所有平台二进制构建到 dist/
```

如果你是在这个仓库里工作的 AI agent（或人类）：`CONTEXT.md` 是项目术语表，`docs/adr/` 是编号的架构决策，`AGENTS.md` 是本仓库处理 issue 的工作约定。

提交 PR 即表示你同意下方的[贡献条款](#贡献条款)。

## 许可证

AGPL-3.0-only · Copyright (c) 2026 ChHsiching。见 [LICENSE](LICENSE)。

- 使用（含公司内部使用）、修改、分发均免费；但分发或以本代码提供网络服务时，衍生作品须以 AGPL-3.0 开源。
- 闭源商用须另行获取商业授权：hsichingchang@gmail.com

### 贡献条款

提交 PR 即表示你同意以 AGPL-3.0 授权你的贡献，并授予维护者在 AGPL 之外另行提供商业授权的权利（你的贡献始终以 AGPL 对所有人开放）。
