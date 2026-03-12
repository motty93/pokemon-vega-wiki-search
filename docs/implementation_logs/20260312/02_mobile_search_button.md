# モバイル検索ボタン改善 & 将来的な自動検索の検討

## 背景

モバイルでは検索バー右端の虫眼鏡アイコンがボタンだと認識しづらく、検索アクションが取りにくい問題があった。

## 今回の対応

- 検索ボタンをアイコンのみ → 「検索」テキスト付きの緑色ボタンに変更
- `bg-emerald-600` で視認性を確保、`active:bg-emerald-800` でタップフィードバック追加
- `shrink-0` + `min-w-0` で入力欄が潰れないよう調整

### 対象ファイル

- `templates/index.html`: 検索バーのボタン部分

## 将来的な実装: 自動検索（インクリメンタルサーチ）

入力中に自動で検索結果を表示する機能。ボタンを押す手間が省けてUXが向上する。

### 実装方針

1. htmxの`hx-trigger`に `input changed delay:500ms` を追加
2. JS側で最低文字数チェック（2文字未満では発火させない）
3. 日本語IMEの変換中は発火しないよう `compositionstart`/`compositionend` イベントで制御

### DB負荷への対策

- **delayを500〜800msに設定**: タイピング中の連続リクエストを抑制
- **最低文字数制限**: 1文字検索は結果が多すぎるため2文字以上に制限
- **IME対応**: `compositionend` 後にのみリクエスト発火（変換確定待ち）
- **リクエストキャンセル**: htmxの`hx-sync="abort"` で前のリクエストをキャンセル
- **検討**: クライアントサイドキャッシュで同じクエリの再リクエストを防止

### 実装イメージ

```html
<input type="text" name="q"
  hx-get="/search" hx-target="#results" hx-indicator="#loading"
  hx-trigger="input changed delay:500ms, submit"
  hx-sync="closest form:abort"
  autocomplete="off">
```

```javascript
// IME変換中はhtmxリクエストを抑止
var composing = false;
searchInput.addEventListener('compositionstart', function() { composing = true; });
searchInput.addEventListener('compositionend', function() {
  composing = false;
  // 変換確定後に手動でリクエスト発火
  if (searchInput.value.length >= 2) {
    htmx.trigger(searchInput, 'input');
  }
});

// 文字数チェック
document.body.addEventListener('htmx:configRequest', function(e) {
  if (composing) { e.preventDefault(); return; }
  var q = e.detail.parameters.q;
  if (e.detail.triggeringEvent?.type === 'input' && (!q || q.length < 2)) {
    e.preventDefault();
  }
});
```
