# トップページ検索バーのクリック不能バグ修正

## 症状

トップページの検索inputをクリックしても入力できない（フォームの一部分のみ反応する）。

## 原因

2つの要素が重なって検索バーのクリックイベントを奪っていた。

### 原因1: `::after` 疑似要素のオーバーレイ

```css
.hero-bg::after {
  content: '';
  position: absolute;
  inset: 0;       /* ← 親要素全体を覆う */
  background: radial-gradient(...);
}
```

`.hero-bg` の装飾用に `::after` 疑似要素を `position: absolute; inset: 0` で配置していた。
これにより疑似要素が画面全体を覆い、その下にある検索バーへのクリックが届かなかった。

**修正**: `::after` 疑似要素を廃止し、radial gradientを `.hero-bg` の `background` プロパティに統合。

```css
/* Before: 2つのレイヤー */
.hero-bg { background: linear-gradient(...); }
.hero-bg::after { background: radial-gradient(...); }  /* ← クリックをブロック */

/* After: 1つのbackgroundに統合 */
.hero-bg {
  background:
    radial-gradient(ellipse at center, rgba(99,102,241,0.15) 0%, transparent 70%),
    linear-gradient(135deg, #1e293b 25%, #334155 25%, ...);
  background-size: 100% 100%, 40px 40px;
}
```

### 原因2: htmx-indicator のローディングオーバーレイ

```html
<div id="loading" class="htmx-indicator fixed inset-0 z-50 ...">
```

htmxのローディングインジケーターが `fixed inset-0 z-50`（画面全体を覆う + 最前面）で配置されていた。
htmxの仕組みでは `.htmx-indicator` に `opacity: 0` を設定して非表示にするが、**opacityはポインターイベントに影響しない**。
つまり見えないが常にクリックを奪う透明な壁が画面全体に存在していた。

**修正**: `pointer-events` で制御を追加。

```css
/* Before */
.htmx-indicator { opacity: 0; transition: opacity 0.2s ease; }
.htmx-request .htmx-indicator { opacity: 1; }

/* After */
.htmx-indicator { opacity: 0; pointer-events: none; transition: opacity 0.2s ease; }
.htmx-request .htmx-indicator { opacity: 1; pointer-events: auto; }
```

- 通常時: `pointer-events: none` でクリックを透過（下の要素に届く）
- リクエスト中: `pointer-events: auto` でオーバーレイとして機能

## 変更ファイル

- `templates/index.html`: `::after` 疑似要素を削除、backgroundに統合
- `templates/base.html`: `.htmx-indicator` に `pointer-events: none/auto` を追加

## 教訓

- `position: absolute/fixed` + `inset: 0` の要素は `opacity: 0` でも**クリックイベントをブロックする**
- 見た目の非表示（opacity, visibility）とインタラクションの無効化（pointer-events, display）は別の概念
- オーバーレイ系UIは必ず `pointer-events: none` をデフォルトにし、表示時のみ `auto` にすべき
