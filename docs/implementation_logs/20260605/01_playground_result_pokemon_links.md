# SQL Playground の結果テーブルからポケモン詳細へ遷移できるUX

## 背景

`/playground`（DuckDB-WASM）の結果テーブルは単なる文字列の表で、ポケモンを含む行から
詳細ページ（`/pokemon/{id}`）へ辿れなかった。「行ごとにクリックで詳細に飛びたい」という要望。

ただし Playground は任意の SELECT を実行するため、「行＝1匹」とは限らない:

- 集計行（`GROUP BY type` など）にはポケモンが存在しない
- `evolution` の `from_pokemon_id` / `to_pokemon_id` は1行に2匹
- `move` テーブルは `id`+`name` を持つが、これはポケモンではない（誤リンクの罠）

そこで「行クリック」だけでなく、**ポケモンを指す列を自動判定してセル単位でリンク化**し、
1匹だけを指す行は行全体クリックも有効にする、というハイブリッドにした。

## 変更内容

`templates/playground.html` のみ（サーバー側・スキーマ・CSPは変更なし）。

### ポケモン列の判定（`detectPokeCols`）

- `pokemon_id` / `from_pokemon_id` / `to_pokemon_id` … スキーマ上つねにポケモンFK → リンク対象
- `id` 列 … 同じ結果に `name` 列があり、ポケモン名索引と一致する行が6割以上のときだけ
  ポケモンとみなす。これで `move.id`+`name` 等を誤検出しない（先頭30行をサンプリング）。

### ポケモン名索引（`ensurePokeIndex`）

- 既に読み込み済みの parquet に対し `SELECT id, name FROM pokemon` を1回だけ実行し、
  `Map<id, name>` を構築。クエリ実行のたびに描画前へ `await`（ガード付きで実質初回のみ）。
- 失敗しても空 Map にしてクエリ実行自体は妨げない。

### 描画（`renderTable` / `pokeAnchor`）

- ポケモン列のセル … `/static/images/pokemon/NNN.png` のスプライト＋元の値を
  `<a target="_blank" rel="noopener">` でリンク化（別タブ。Playground のクエリを失わない）。
  画像が無い場合は `onerror` で非表示。
- `id`+`name` がポケモンと確定した場合は `name` セルも同じ詳細へリンク。
- 行が一意に1匹だけを指す場合のみ `tr.poke-row` を付与し、行全体をクリック可能に
  （余白クリックは委譲ハンドラで別タブ遷移。明示リンクのクリックはそちらを優先）。
- リンクが1つでもあれば操作ヒント（`#poke-hint`）を表示。

### スタイル

- `.poke-link` / `.poke-mini`（28px スプライト）/ `tr.poke-row`（cursor + emerald のホバー）を
  ページ先頭の `<style>` に追加。`tr.poke-row:hover` は `.data-table tbody tr:hover` より
  詳細度を高くして上書き。

## 検証

- 追加した DuckDB スクリプトを `.mjs` として `node --check` → 構文OK。
- `go build ./...` / `go vet ./...` → OK。
- サーバー起動（DBはコピーを使用）して `GET /playground` → 200、
  テンプレートのパース成功と新要素（`poke-hint` / `detectPokeCols` / `poke-row` 等）の出力を確認。
- ※ ブラウザ実機でのクリック挙動・スプライト表示は未自動検証（要目視確認）。

### サンプルクエリ別の期待動作

- 種族値合計ランキング（`p.id, p.name, ...`）: id+name 一致 → 行クリック可・id/name リンク
- タイプ別平均（`type, n, avg_total`）: ポケモン列なし → リンクなし・ヒント非表示
- `SELECT * FROM evolution`: from/to を各セルでリンク（2匹なので行クリックは無効）
- `SELECT * FROM learnset_level`: `pokemon_id` をリンク・行クリック可
- `SELECT * FROM move`: id+name はあるが名前不一致 → リンクなし（誤検出しない）

## 関連情報

- CSP は通常／`/playground` とも `img-src 'self' data:`。スプライトは同一オリジンなので追加許可不要。
- 画像は `static/images/pokemon/001.png`〜`386.png`（= id のゼロ埋め3桁）で全386匹分そろっている。
