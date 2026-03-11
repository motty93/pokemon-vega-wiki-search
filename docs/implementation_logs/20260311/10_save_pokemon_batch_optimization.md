# SavePokemonのバッチ最適化

## 背景

1匹あたり6〜8秒かかっていた。原因はリモートDB（Turso）への個別クエリが1匹あたり数十回発生していたため。

- `GetOrCreateMove`: 技ごとにSELECT+INSERT（1匹あたり30〜50回）
- 各learnsetテーブルへのINSERT: 個別実行

## 変更内容

### `internal/scraper/scraper.go`

- **技名のメモリキャッシュ**: `moveCache map[string]int` を `Run()` で作成し `SavePokemon()` に渡す。一度DBに登録した技は以降キャッシュから取得
- **`cachedGetOrCreateMove()`** → **`collectMoveNames()`** に変更: 未登録の技名だけ先にまとめてDB登録
- **トランザクション化**: `SavePokemon()` 内の全INSERT（pokemon/base_stats/ev_yield/encounter/learnset全種）を1つのトランザクションにまとめて送信
- 1匹あたりのリモートDBリクエストが数十回→実質1〜2回に削減
