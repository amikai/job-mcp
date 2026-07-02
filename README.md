# job-mcp

MCP server for searching job listings across job boards and company career sites (104, TSMC, and more).

## Install

With Go:

```
go install github.com/amikai/job-mcp/cmd/jobmcp@latest
```

Upgrade by rerunning the same command; pin a version with `@vX.Y.Z`.

Without Go: download the archive for your platform from
[Releases](https://github.com/amikai/job-mcp/releases) and put `jobmcp` on
your PATH.

## Add the MCP server to your tool

With `jobmcp` on your PATH:

**Claude Code**

```
claude mcp add job-mcp -- jobmcp
```

**Codex**

```
codex mcp add job-mcp -- jobmcp
```

**Gemini CLI**

```
gemini mcp add job-mcp jobmcp
```

## Disclaimer

This is an unofficial tool. It is not affiliated with, endorsed by, or
supported by 104 Corporation, TSMC, or any other job board or company whose
listings it searches.

Some providers rely on undocumented APIs that may change or stop working at
any time without notice. Job listing data belongs to the respective sites and
is fetched on your behalf when you invoke a tool — nothing is stored or
redistributed by this project.

Use this tool for personal job searching at a human pace. You are responsible
for complying with the terms of service of each site you query; do not use it
for bulk scraping or data harvesting.

## License

[MIT](LICENSE)
