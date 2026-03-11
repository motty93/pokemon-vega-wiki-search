# Makefile作成とマイグレーションSQL分割の修正

## 背景

- `.env` の環境変数が `os.Getenv()` で読み込めず、`go run` 直接実行でTurso接続に失敗していた
- 本番環境はシェル環境変数を直接設定する方針のため、コード側での `.env` 読み込みは行わない
- マイグレーションのSQL分割が `strings.Split(";")` だったため、`CREATE TRIGGER ... BEGIN ... END;` 内のセミコロンで誤分割されていた

## 変更内容

### `Makefile`（新規作成）

- `include .env` + `export` で `.env` を自動読み込み
- ターゲット: `server`, `scraper`, `build`, `build-server`, `build-scraper`, `test`, `clean`

### `internal/db/db.go`

- `splitSQL()` を追加: `BEGIN...END` ブロック内のセミコロンを考慮したSQL分割関数
- `Migrate()` を `splitSQL()` を使うように変更
- コメント行（`--`）と空行をスキップ

### `CLAUDE.md`

- ビルド・実行コマンドのセクションをMakefile経由に更新
