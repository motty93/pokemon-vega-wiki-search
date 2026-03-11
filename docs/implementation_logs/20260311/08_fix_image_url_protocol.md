# 画像URLのプロトコル補完修正

## 背景

画像ダウンロード時に全件404エラーが発生。URLが `https://w.atwiki.jp//img.atwiki.jp/...` となっていた。

## 原因

画像の `src` が `//img.atwiki.jp/...`（プロトコル相対URL）の場合、`baseURL + src` で `https://w.atwiki.jp//img.atwiki.jp/...` になっていた。

## 変更内容

### `internal/scraper/scraper.go`

- `//` 始まりの場合は `"https:" + src` に変更
- `bar.gif`（種族値バーの画像）を除外するフィルタも追加
