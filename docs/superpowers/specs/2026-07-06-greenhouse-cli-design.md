# Greenhouse CLI 設計

日期:2026-07-06
狀態:已與使用者確認

## 目標

為 Greenhouse provider 補上獨立 CLI(`cmd/greenhouse`),介面與 `cmd/lever`、`cmd/ashby` 同構:`companies` / `search` / `get` 三個子指令,遵循專案一貫的「search 摘要 → 拿 id → get 全文」兩段式模式。

## 背景與現況

- `internal/provider/greenhouse/` 已有 ogen 生成的 client(`listJobs`、`getJob`)、mocksrv 與 client 測試(PR #77)。
- `companies.yaml` 已完成:62 家經 live 驗證(回 200 且 jobs 非空)的公司,格式同 ashby(`company` + `board`)。
- 尚缺:`companies.go`(embed roster)與 `cmd/greenhouse`。

## API 特性(決策依據)

Greenhouse Job Board API(`boards-api.greenhouse.io/v1`,公開免認證):

| Endpoint | 參數 | 特性 |
|---|---|---|
| `GET /boards/{board_token}/jobs` | `content=true`(選用) | 一次回全部職缺,無分頁、無伺服器端過濾。不帶 content 時是輕量摘要;帶了會附上 JD 全文 + departments/offices,回應肥數倍 |
| `GET /boards/{board_token}/jobs/{job_id}` | `questions`、`pay_transparency`(選用) | 單筆 detail,含 JD。`pay_transparency=true` 回 `pay_input_ranges`(薪資範圍)— **只有這個 endpoint 拿得到薪資** |

與 Ashby 的關鍵差異:Ashby 沒有單筆 endpoint,其 CLI 的 `get` 只能重抓整包再挑;Greenhouse 有,`get` 應直接打單筆。

## 決策記錄

1. **search 過濾**:`--keyword`(title)+ `--location`(location.name),皆為 client-side 不分大小寫子字串比對。不做 `--department`(需要 `content=true` 的肥請求才有 departments 欄位,不值得)。
2. **get 內容**:固定帶 `pay_transparency=true` 印薪資範圍;不帶 `questions`(應徵表單結構對「看職缺」是噪音)。
3. **請求策略**:search 打輕量 list(不帶 `content=true`);get 打單筆 `getJob`。曾考慮「search/get 都抓整包」(與 ashby 一比一)但被否決:整包拿不到薪資、get 變肥請求、放著單筆 endpoint 不用。

## 指令介面

```
greenhouse companies [--format text|json]
greenhouse --board BOARD search [--keyword TEXT] [--location TEXT] [--format text|json]
greenhouse --board BOARD get --id JOB-ID [--format text|json]
```

- 根 flags:`--board`(search/get 必填)、`--timeout`(預設 60s)、`--format`(`text`|`json`,預設 `text`)。
- flag 框架:`peterbourgon/ff/v4`,命令樹結構照抄 `cmd/ashby/main.go`。
- `--id` 是 search 結果裡的職缺 id(整數,即 posting `id`,非 `internal_job_id`)。

### companies

列出 embed 的 62 家 roster(公司名 + board token),依公司名排序,不打網路。text 版每行 `Name (board)`;json 版直接 encode roster 陣列。

### search

1. 驗證 `--board`:小寫正規化後查 `CompaniesByBoard`;查無 → 錯誤訊息提示跑 `greenhouse companies`(樂觀呼叫 + 教學錯誤訊息,同 lever/ashby)。
2. 打 `listJobs`(不帶 `content`)。
3. Client-side 過濾:`--keyword` 對 title、`--location` 對 `location.name`,皆不分大小寫子字串;兩者同時給時取 AND。
4. 輸出摘要。

### get

1. `--board` 驗證同上;`--id` 必填。
2. 打 `getJob`,`pay_transparency=true`。
3. 輸出全文。

## 前置工作:companies.go

在 `internal/provider/greenhouse/` 加 `companies.go`,照 lever/ashby 模式:

- `//go:embed companies.yaml`
- `Company` struct:`Name`(yaml `company`)、`Board`(yaml `board`)
- export `Companies []Company`(載入時依公司名排序)與 `CompaniesByBoard map[string]Company`
- 直接 export 變數,不做 wrapper getter(使用者既定偏好)
- yaml 解析失敗 panic(build-time bug,非 runtime 情況)

## 輸出格式

### search 摘要

text:編號 + title,之後每行一個欄位:Location、Posted(`first_published` 取 `2006-01-02`)、URL(`absolute_url`)、ID。
json:`{"total": <過濾前總數>, "jobs": [...]}`,shape 同 ashby 的 `searchResultJSON`。

摘要 JSON 投影(`jobSummaryJSON`)欄位:`id`、`title`、`location`、`postedAt`、`updatedAt`、`url`。空欄位 `omitempty`。

### get 全文

text:title、Company(`company_name`)、Location、Posted、URL,接著:

- **薪資範圍**(`pay_input_ranges` 非空時):每筆印 `title: min – max CURRENCY`(cents 轉元,幣別用 `currency_type` 原文,不硬編 `$` 符號)+ blurb(有的話);空陣列或欄位缺席時整段不印。
- **Description**:`content` 是 HTML entity-encoded 的 HTML — 先 entity decode(`html.UnescapeString`),再走 `jaytaylor/html2text` 轉純文字;轉換失敗時印 decode 後的原文而非吞掉。

json:直接 encode 完整 `JobDetail`(不投影 — detail 是給人或下游程式看全貌的)。

## 錯誤處理

- `--board` 不在 roster:本地擋下,提示 `greenhouse companies`。
- roster 內的 board 上游回 404(理論上不會發生):回報 "board not found upstream",不吞掉。
- `get` 的 job id 上游回 404:回報 "job not found on board"。
- flag 解析錯誤 / 缺子指令:印 ffhelp usage 到 stderr,exit 1(同 lever/ashby 的 main 骨架)。

## 測試

- `cmd/greenhouse/main_test.go`:測純函式 helper(投影、過濾、薪資格式化),對齊 `cmd/ashby/main_test.go` 的量級。
- provider 層的 mocksrv / client 測試已存在,不重複。
- 手動驗收:對 roster 內公司跑 `companies` / `search` / `get` 各一次(含 `--format json`)。

## 不做的事(YAGNI)

- 不做 `--department` 過濾、不做 `questions` 輸出。
- 不做分頁(API 本身無分頁)。
- 不接 MCP server(另案,與 lever/ashby/workday 一起)。
- 不做動態 board 探測(roster 外的公司直接報錯)。
