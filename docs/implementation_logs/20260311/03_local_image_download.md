# GCSなしでのローカル画像ダウンロード機能を追加

## 背景

画像保存はGCS（Google Cloud Storage）のみ対応しており、`GCS_BUCKET_NAME` 未設定時は画像がWikiへのホットリンクのままだった。ローカル開発時に画像を手元に保存する手段がなかった。

## 変更内容

### `internal/storage/gcs.go`

- `DownloadPokemonImages()` を追加
  - DBから `image_url` がHTTP URLのポケモンを取得
  - 画像をダウンロードし `static/images/pokemon/` にローカル保存
  - ファイル名: `001.png` 形式（ポケモンID3桁ゼロ埋め）
  - DBの `image_url` を `/static/images/pokemon/001.png` 形式に更新
- `downloadImage()` を追加: 個別画像のダウンロード処理（HTTPステータスチェック付き）

### `cmd/scraper/main.go`

- `GCS_BUCKET_NAME` 設定時: 従来通りGCSへアップロード
- `GCS_BUCKET_NAME` 未設定時: `DownloadPokemonImages()` でローカルにダウンロード（フォールバック）
