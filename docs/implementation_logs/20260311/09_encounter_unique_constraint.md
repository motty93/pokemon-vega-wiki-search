# encounterテーブルのUNIQUE制約追加とINSERT方式変更

## 背景

再実行時にencounterデータが重複挿入されていた。また、重複チェックのために毎回SELECTクエリをリモートDBに発行しており、パフォーマンスが悪化していた。

## 変更内容

### `migrations/001_init.sql`

- `encounter` テーブルに `UNIQUE(pokemon_id, location, method)` 制約を追加

### `internal/db/query.go`

- `InsertEncounter` を `SELECT` による存在チェック方式から `INSERT OR IGNORE` に変更
