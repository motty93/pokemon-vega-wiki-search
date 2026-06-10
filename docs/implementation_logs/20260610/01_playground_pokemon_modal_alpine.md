# SQL Playground の結果からポケモン要約モーダルを表示（Alpine.js）

## 背景

結果テーブルのポケモン行クリックを「別タブで詳細ページへ遷移」にしていたが、
playgroundは DuckDB-WASM / mermaid / parquet を抱えて重く、別タブ遷移が体感で遅かった
（サーバー処理自体は両経路とも ~1ms で差はなく、原因はフロントの別タブ＋ページの重さと実測で確認）。

そこで遷移をやめ、**その場で要約モーダル**を出す方式に変更。データは既にブラウザ内にある
parquet を DuckDB で引くため**サーバー往復ゼロ**で、書きかけのSQLも保持される。

モーダルはユーザー希望により **Alpine.js** で実装（素のDOM/innerHTML構築は避ける）。

## 変更内容

`templates/playground.html` のみ。

### モーダル（Alpine.store + テンプレート）

- `Alpine.store('poke')` … `open / loading / error / data / evo / close()` を保持。
  `alpine:init` で定義（`window.Alpine` 既存時は即定義のフォールバックあり）。
- モーダルDOMは Alpine テンプレートで宣言的に描画。
  - `x-show="$store.poke.open"` ＋ `x-transition.opacity`、`x-cloak` で初期チラつき防止。
  - 背景は `@click.self`、`Esc` は `@keydown.escape.window` で閉じる。
  - `x-data="{ get p(){ return $store.poke.data } }"` のゲッターで常に最新の store.data を参照
    （進化チップで別ポケモンに切り替えても再描画される）。
  - 種族値バー／タイプバッジ／進化チェーンは `x-for`、画像は `:src`。
- 要点サマリ: 画像・No.・名前・タイプ・種族値（バー＋合計）・特性・進化チェーン＋
  「詳細ページを開く ↗」（別タブ）。進化チップのクリックでその場で別ポケモンに切替。

### データ取得（module script 側）

- `openPokeModal(id)` が DuckDB で `pokemon JOIN base_stats`（基本情報＋種族値）と
  `evolution`（進化）を引き、`buildPokeData` / `buildEvo` で整形して store に流し込む。
- `id` は `Number()` 済みの数値のみ埋め込み（インジェクション不可）。
- `window.openPokeModal = openPokeModal` で Alpine テンプレート（進化チップ）から呼べるよう公開。

### クリック導線

- 結果テーブルのリンク（`a.poke-link`）と行（`tr.poke-row`）の**通常クリック → モーダル**。
- `Ctrl/⌘/Shift/Alt＋クリック` は従来どおり**別タブで詳細ページ**（`<a>` はブラウザ標準、
  余白クリックは `window.open`）。`pokeAnchor` は `target="_blank"` を外し `data-poke` を付与、
  通常クリックは `preventDefault` でモーダルへ。
- ヒント文言も「クリックで要約をその場表示（Ctrl/⌘＋クリックで詳細を別タブ）」に更新。

## 検証

- DuckDB スクリプトを `.mjs` 化して `node --check` → 構文OK。重複定義なし。
- サーバー起動（DBコピー）で `GET /playground` → 200、
  Alpine マーカー（`$store.poke.open` / `definePokeStore` / `openPokeModal` /
  `x-data="{ get p` / `evo-chip` / `@keydown.escape.window`）の出力を確認。
- ※ Alpine の実描画・モーダル開閉・進化チップ切替はブラウザ実機未確認（要目視）。

## 関連情報

- CSP は `img-src 'self' data:`、スプライトは同一オリジンで追加許可不要。モーダルは
  サーバー fetch しないので `connect-src` も不要。
- 前提メモ: モーダルは Alpine.js で作る（`memory/modals-use-alpine-js.md`）。

## 不具合修正（同日・初回実装が表示されなかった件）

クリックしてもモーダルが出なかった。原因は2つ:

1. **モーダルのルート要素に `x-data` が無かった（主因）。** Alpine は `x-data` を持つ要素とその
   子孫しか処理しないため、ルートの `x-show`/`x-cloak` が評価されず、`[x-cloak]{display:none}`
   が外れないまま永久に非表示だった。→ `x-data="{ get p(){ return $store.poke.data } }"` を
   ルート要素へ移動（内側 card の x-data は削除）。
2. **`Alpine.store('poke')` を duckdb モジュール側で定義していた（潜在）。** このモジュールは
   Alpine の初期化（DOM評価）より後に実行されるため、Alpine が最初に `$store.poke.open` を
   評価する時点で store が未定義になり得た。→ store 登録を content 先頭の inline `<script>` の
   `alpine:init` に前倒しし、DOM評価前に必ず定義されるようにした（`definePokeStore` は削除）。

`log.ts` は DuckDB-WASM の `ConsoleLogger` の通常ログで、今回の主因ではなかった。

## id 列単体（name 無し）でもモーダル対応（同日・ユーザー選択）

`detectPokeCols` を二分岐に拡張:

- **name 列あり** → 従来どおり name 突き合わせ（`move.id`+`name` 等を誤検出しない）。
- **name 列なし** → `id` 列の全(非null)値が 1〜386（`pokeIndex`）に収まれば pokemon とみなす。
  これで `SELECT p.id, p.type1 FROM pokemon p`（name 無し）でも行クリック/セルでモーダルが出る。

トレードオフ（ユーザー承知）: `evolution.id` / `encounter.id` を `id` 単独で引くと同番号の
別ポケモンを誤表示しうる（`pokemon_id` 列を使えば確実）。`move` は id が 387+ を含めば自動除外。

検証: ロジック単体テスト7ケース（id+name / id単体 / move誤検出回避 / pokemon_id /
範囲外除外 / evolution誤検出 / 集計）すべて期待通り。JS構文OK。
