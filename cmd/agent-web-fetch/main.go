// Command agent-web-fetch runs the Web Reader as an MCP server over stdio.
//
// Configure it once in your MCP client (e.g. Claude Code's mcp.json) by
// pointing the command at this binary's absolute path. The server exposes a
// single "fetch" tool that retrieves a URL and returns model-friendly
// markdown — a free, uncapped replacement for paid built-in web readers.
//
// The server is silent on stdout by default (stdout is the stdio transport).
// Set WEB_FETCH_LOG=/path/to/file.log to write diagnostics to a file.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ChHsiching/agent-web-fetch/internal/mcpserver"
)

func main() {
	if err := mcpserver.Run(context.Background()); err != nil {
		// A true server failure (not a normal client disconnect). Report on
		// stderr only — never stdout, which is the MCP stdio transport.
		fmt.Fprintln(os.Stderr, "agent-web-fetch:", err)
		os.Exit(1)
	}
}
