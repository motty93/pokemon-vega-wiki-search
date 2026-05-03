# Cloud Run max-instances 設定

## 背景

AI crawler などによる大量アクセス時に Cloud Run が自動でスケールアウトし、想定外の課金が発生するリスクがある。
個人サイト規模では同時インスタンス数を絞っても実用上問題ないため、上限を物理的に低く設定して保険をかける。
コード変更を伴わないため即効性が高く、最初の保険として実施。

## 変更内容

### 対象サービス

- サービス名: `vega-pokedex`
- リージョン: `asia-northeast1`

### 変更前

```
autoscaling.knative.dev/maxScale=3
```

（`minScale` は未設定＝デフォルト 0）

### 変更後

```
autoscaling.knative.dev/maxScale=5
```

`minScale` は 0 のまま（コールドスタート許容、アイドル時の課金なし）。

### 実行コマンド

```sh
gcloud run services update vega-pokedex \
  --max-instances=5 \
  --region=asia-northeast1
```

新リビジョン `vega-pokedex-00003-xmz` が作成され、100% のトラフィックがルーティングされた。
ダウンタイムなし。

### 確認コマンド

```sh
gcloud run services describe vega-pokedex \
  --region=asia-northeast1 \
  --format="value(spec.template.metadata.annotations.'autoscaling.knative.dev/maxScale')"
# => 5
```

### cloudbuild.yml の同期

`cloudbuild.yml` の `gcloud run deploy` ステップに `--max-instances 3` がハードコードされていたため、
次回 Cloud Build 経由のデプロイで 3 に戻ってしまう問題があった。
`cloudbuild.yml` の値も `5` に更新して同期。

## 関連情報

- 1 インスタンスあたりデフォルトで 80 並列リクエスト処理可能 → 5 インスタンスで同時 400 リクエスト処理可能
- 上限超過分のリクエストは 429 Too Many Requests が返る
- トラフィック増加や正常クローラーで 429 が頻発する場合は `10`〜`20` への引き上げを検討（`cloudbuild.yml` も合わせて更新すること）
- `min-instances=0` のままなのでアイドル時のコストは発生しない（コールドスタート許容）
