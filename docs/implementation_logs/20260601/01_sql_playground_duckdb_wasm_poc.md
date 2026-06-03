# SQL Playground（DuckDB-WASM + parquet）PoC

## 背景

「SQLite をコンテナに内包したまま write を許可するとどうなるか」という議論から派生。
Cloud Run（最大5インスタンス・エフェメラルFS）では内包SQLiteへの書き込みは
インスタンス間で分岐し再起動で消えるため、書き込みは外部DBに分離する方針を確認した。

その上で「マスタデータを parquet 化し、ブラウザでSQLを書いてデータ取得したい」という
要望を実現するPoCを作成。parquet を直接SQLでクエリできるのは SQLite ではなく **DuckDB** のため、
**DuckDB-WASM + GCS想定の parquet** をブラウザ内で実行する構成とした。

既存のSSR図鑑（htmx + Go Template）は一切変更せず、`/playground` を1ページ追加するハイブリッド構成。

## 変更内容

### 新規

- `scripts/export_parquet.sh`
  - DuckDB CLI（`/usr/bin/duckdb` v1.2.0、sqlite_scanner拡張）で
    `data/pokemon.db` の10テーブルを `static/data/*.parquet` に書き出す（zstd圧縮）。
  - 対象: pokemon, base_stats, ev_yield, evolution, move, learnset_level,
    learnset_tm, learnset_tutor, learnset_egg, encounter（FTS5 `pokemon_fts` は除外）。
  - 出力合計 約140KB（元の SQLite 1.8MB / gzip 460KB よりさらに小さい）。
- `static/data/*.parquet`
  - 上記スクリプトの生成物。件数は既存DBと一致（pokemon 386 / move 510 /
    learnset_egg 8419 / encounter 385 を検証）。
- `templates/playground.html`
  - base.html を継承（`{{define "content"}}`）。
  - `@duckdb/duckdb-wasm@1`（jsdelivr ESM）を遅延ロード。blob Worker 経由で初期化。
  - 各 parquet を fetch → `registerFileBuffer` → 既存テーブル名のVIEWを作成し、
    ユーザーが `SELECT * FROM pokemon` のように既存スキーマで書ける。
  - SQLエディタ（textarea, Ctrl+Enter実行）、サンプルクエリ4種、結果テーブル、
    行数・実行時間・エラー表示。スキーマ参照はAlpine.jsで折りたたみ表示。

### 変更

- `internal/handler/handler.go`
  - `New` のページテンプレート一覧に `playground.html` を追加。
  - `Playground` ハンドラーを追加。`playgroundCSP` 定数で
    `script-src`/`connect-src` に `https://cdn.jsdelivr.net`、parquet拡張用に
    `connect-src https://extensions.duckdb.org`、`worker-src 'self' blob:` を許可し、
    SecurityHeadersMiddleware のCSPをこのページだけ上書き（他ページは据え置き）。
- `cmd/server/main.go`
  - `r.Get("/playground", h.Playground)` を追加。

## 検証結果（サーバー側）

- `go build ./...` / `go vet` 通過。
- `/playground` → 200、CSPに jsdelivr / blob worker / connect-src 追加を確認。
- `/static/data/*.parquet` → 200（application/octet-stream）。
- トップ `/` のCSPは元のまま（jsdelivr/worker-src 無し）= 緩和は /playground 限定。
- サーバーログ正常（Turso未設定でアナリティクス無効・起動成功）。

## 動作確認

- 2026-06-01 ブラウザ実機で確認済み（種族値ランキングのサンプルクエリが表に表示）。
- 初回エラー: DuckDB-WASM 1.4.3 は parquet 拡張（`parquet.duckdb_extension.wasm`）を
  `extensions.duckdb.org` から動的ロードするため、`connect-src` に同ホストの許可が必要だった（対応済み）。

## 重いクエリ対策（2026-06-02 追記）

`templates/playground.html` に以下を追加。いずれもブラウザ内で完結する保護で、
サーバー・parquet本体・他ユーザーには影響しない。

- **タイムアウト**: `QUERY_TIMEOUT_MS = 10000`。`Promise.race([conn.query, timeout])` で
  10秒を超えたら中断扱い。
- **中止ボタン**: 実行中のみ表示。クエリは Web Worker 実行のため UI は固まらず押下できる。
- **Worker再生成でリセット**: DuckDB-WASM はクエリ単体のキャンセルが弱いため、
  タイムアウト/中止時は `worker.terminate()` → `setupDuckDB()` で作り直す（`resetDuckDB`）。
- **描画行数の上限**: `MAX_RENDER_ROWS = 2000`。結果が大きくても先頭2000行のみDOM描画し、
  「先頭2000行を表示」と注記。

## ER図表示（2026-06-02 追記）

`templates/playground.html` の左サイド（サンプルクエリの下）にER図を追加。

- **Mermaid.js（`mermaid@11` jsdelivr ESM）の `erDiagram`** で10テーブルの関係を描画。
  CSPは既存の jsdelivr 許可（script-src/connect-src）で充足し、追加設定は不要だった。
- サイドでは縮小表示、クリックで**モーダル拡大**（Alpine.js、`[x-cloak]` で初期チラつき防止）。
- `er-svg-mini` / `er-svg-full` を別IDで2回 render し、同一SVGの id 衝突を回避。

## テーブル一覧に日本語注釈（2026-06-02 追記）

左サイドのテーブル一覧で、各テーブル名の隣に日本語注釈を表示（`tables` 配列に `ja` を追加）。
`learnset_level`/`learnset_tm`/`learnset_tutor`/`learnset_egg` を
「レベルアップ習得技／わざマシン習得技／教え技／タマゴ技」と明示し、4種を区別しやすくした。

## 残課題 / 次のステップ

- parquet を GCS へ移し、CDN配信＋CORS設定。`connect-src` に GCSオリジンを追加。
- 変換（export_parquet.sh）をデプロイ or スクレイパーJobに組み込み自動化。
- エディタ強化（CodeMirror等）、結果のCSV/JSONエクスポート。

## 関連メモ

- Makefile に `server` / `scraper` ターゲットは存在せず、実態は `make build`（→ bin/server）
  と `make dev`（air ホットリロード）。CLAUDE.md の記載と乖離している。
