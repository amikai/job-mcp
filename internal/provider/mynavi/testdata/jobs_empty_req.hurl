# Zero-hit search: still HTTP 200 with the result counter reading 0 and no
# job cassettes at all (no recommend/PR cards are injected either).
GET https://tenshoku.mynavi.jp/list/kwzzzqqqxyzabc/
accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8
accept-language: ja,en;q=0.9
user-agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36

HTTP 200

[Asserts]
header "Content-Type" contains "text/html"
xpath "string((//p[@class='result__num'])[1]/em)" == "0"
xpath "//div[contains(@class,'cassetteRecruit__content')]" count == 0
