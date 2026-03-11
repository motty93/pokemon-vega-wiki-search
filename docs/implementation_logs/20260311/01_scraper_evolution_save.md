# 進化データの保存機能を実装

## 背景

スクレイパーで進化テーブルをパースしていたが、`SavePokemon()` 内に進化データの保存処理がなく、DBに保存されていなかった。

## 変更内容

### `internal/scraper/scraper.go`

- `Run()` に進化データの2パス保存処理を追加
  - 1パス目: 各ポケモン処理時に名前→ID解決を試み、解決できたらINSERT、できなければ `pendingEvolutions` に退避
  - 2パス目: 全ポケモン挿入後、未解決分を再処理（進化先が後のIDの場合に対応）

### `internal/db/query.go`

- `GetPokemonIDByName()` を追加: 名前からポケモンIDを取得するクエリ関数

## 関連

- `evolution` テーブル: `from_pokemon_id`, `to_pokemon_id`, `condition`
- `InsertEvolution()` は既存だったが呼び出されていなかった
