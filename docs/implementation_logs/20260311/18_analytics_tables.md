# 検索ログ・ページビューの記録機能

## 背景

検索されているワードの傾向やよく閲覧されているポケモンを把握するため、分析用のデータをDBに蓄積する。UIはまだ作らず、DB設計とデータ記録のみ。

## DB設計

### `search_log` テーブル
| カラム | 型 | 説明 |
|--------|------|------|
| id | INTEGER PK | 自動採番 |
| query | TEXT NOT NULL | 検索クエリ文字列 |
| result_count | INTEGER NOT NULL | 検索結果件数 |
| created_at | TEXT NOT NULL | 記録日時 (datetime('now')) |

インデックス: `created_at`, `query`

### `page_view` テーブル
| カラム | 型 | 説明 |
|--------|------|------|
| id | INTEGER PK | 自動採番 |
| pokemon_id | INTEGER NOT NULL | 閲覧されたポケモンID (FK → pokemon) |
| created_at | TEXT NOT NULL | 記録日時 (datetime('now')) |

インデックス: `pokemon_id`, `created_at`

## 変更ファイル

### `migrations/002_analytics.sql`
- 上記2テーブルの CREATE TABLE + CREATE INDEX

### `internal/db/db.go`
- `Migrate()` を複数マイグレーションファイル対応に変更
- `filepath.Glob("migrations/*.sql")` でファイル名順に全SQLを実行

### `internal/db/analytics.go`
- `LogSearch(db, query, resultCount)` — 検索ログ記録
- `LogPageView(db, pokemonID)` — ページビュー記録
- エラー時はログ出力のみ（ユーザーレスポンスに影響させない）

### `internal/handler/handler.go`
- `Search`: 検索クエリが空でない場合、`go mydb.LogSearch()` で非同期記録
- `PokemonDetail`: `go mydb.LogPageView()` で非同期記録

## 将来のUI案

- 検索ワードランキング: `SELECT query, COUNT(*) FROM search_log GROUP BY query ORDER BY COUNT(*) DESC`
- 人気ポケモンランキング: `SELECT pokemon_id, COUNT(*) FROM page_view GROUP BY pokemon_id ORDER BY COUNT(*) DESC`
- 期間指定: `WHERE created_at >= datetime('now', '-7 days')`
