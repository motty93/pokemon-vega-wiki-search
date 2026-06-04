# ER図のズーム&パン対応（svg-pan-zoom）

## 背景

SQL Playground 左サイドの ER図（Mermaid erDiagram）は、クリックでモーダル拡大していたが
固定サイズで細部が見づらかった。拡大縮小（ズーム&パン）できるようにする要望。

## 変更内容

`templates/playground.html` の3箇所を変更。

- **svg-pan-zoom@3.6.1**（jsdelivr +esm）を追加 import。
  CSPは `cdn.jsdelivr.net` が script-src/connect-src に許可済みのため追加設定は不要だった。
- モーダルの ER図 SVG に svg-pan-zoom を適用し、
  **ホイール=拡大縮小／ドラッグ=移動／組み込みコントロール（＋・−・リセット）** を有効化。
- 初期化タイミング: Alpine の `x-show` は非表示時に幅0になり svg-pan-zoom の fit/center が
  ずれるため、モーダルを開いた後（`setTimeout` 60ms）に `window.__erInit()` で初期化し、
  閉じる時に `window.__erDestroy()` で破棄する（背景クリック・×ボタン双方に連動）。
- スタイル: `#er-full` に固定高さ `70vh` を与え、SVGを `width/height:100%`・`max-width:none`
  にしてビューポートいっぱいに表示。
- モーダルに「ホイールで拡大縮小・ドラッグで移動」のヒントを表示。

## 動作確認

- ローカル（`make dev`）でモーダルのズーム&パン（ホイール・ドラッグ・コントロール）が
  動作することを確認。

## 関連

- ER図の初期実装・playground全体は `docs/implementation_logs/20260601/01_sql_playground_duckdb_wasm_poc.md` 参照。
