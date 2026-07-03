<h1 align="center">job-mcp</h1>

<p align="center">
  <strong>Search job listings from any MCP client — job boards and company career sites, one server.</strong>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go 1.26+">
  <img src="https://img.shields.io/badge/MCP-stdio-7c6cff?style=flat-square" alt="MCP stdio">
  <img src="https://img.shields.io/badge/Docker-amd64%20%7C%20arm64-2496ED?style=flat-square&logo=docker&logoColor=white" alt="Docker multi-arch">
  <img src="https://img.shields.io/badge/License-MIT-yellow?style=flat-square" alt="MIT License">
</p>

**job-mcp** searches job listings across job boards and company career sites —
currently **[104](https://www.104.com.tw)**, **[Cake](https://www.cake.me)**, and
**[NVIDIA careers](https://nvidia.wd5.myworkdayjobs.com/NVIDIAExternalCareerSite)** —
from any MCP client: Claude Code, Codex, Gemini CLI, and others.

## Install

With Go:

```
go install github.com/amikai/job-mcp/cmd/jobmcp@latest
```

Upgrade by rerunning the same command; pin a version with `@vX.Y.Z`.

Without Go: download the archive for your platform from
[Releases](https://github.com/amikai/job-mcp/releases) and put `jobmcp` on
your PATH.

With Docker (multi-arch: linux/amd64, linux/arm64):

```
docker pull ghcr.io/amikai/job-mcp
```

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

With Docker instead, replace `jobmcp` with `docker run -i --rm ghcr.io/amikai/job-mcp`, e.g.:

```
claude mcp add job-mcp -- docker run -i --rm ghcr.io/amikai/job-mcp
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
