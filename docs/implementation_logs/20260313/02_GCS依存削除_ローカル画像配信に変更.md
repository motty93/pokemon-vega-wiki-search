# GCS依存削除・ローカル画像配信に変更

## 背景

画像ストレージをGoogle Cloud Storage（GCS）からローカル（`static/images/pokemon/`）に変更。
Cloud Runから直接画像を配信する構成にすることで、GCSのコスト・設定を不要にする。
386枚程度の画像であればコンテナに含めても問題なく、ブラウザキャッシュ（30日）で通信量も抑えられる。

## 変更内容

### internal/storage/gcs.go
- `UploadPokemonImages()` 関数を削除（GCSへのアップロード処理）
- `uploadImage()` 関数を削除
- `cloud.google.com/go/storage` のimportを削除
- `DownloadPokemonImages()` と `downloadImage()` のみ残存

### cmd/scraper/main.go
- `GCS_BUCKET_NAME` 環境変数による分岐を削除
- 常に `storage.DownloadPokemonImages()` でローカルにダウンロードする形に変更
- `os` パッケージのimportを削除

### internal/model/pokemon.go
- `Validate()` の画像URLバリデーションから `https://` プレフィックスチェックを削除
- ローカルパス（`/static/images/pokemon/001.png`）も許容するように

### .env.example
- `GCS_BUCKET_NAME` と `GOOGLE_APPLICATION_CREDENTIALS` を削除

### go.mod / go.sum
- `go mod tidy` で `cloud.google.com/go/storage` 含むGCP関連依存をすべて削除

### ドキュメント更新
- **CLAUDE.md**: 画像ストレージ、アーキテクチャ説明、環境変数、注意事項を更新
- **README.md**: 技術スタック、プロジェクト構成、環境変数、スクレイパーフローを更新
- **docs/architecture.md**: 技術スタック、構成、スクレイパーフロー、環境変数、実装優先順位、注意事項を更新

## 関連情報

- 静的ファイルには `Cache-Control: public, max-age=2592000`（30日）を設定済み（01で対応）
- Cloud Run の無料枠: ネットワーク送信1GB/月、リクエスト200万回/月
