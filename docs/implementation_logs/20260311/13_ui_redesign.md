# UI全面リニューアル

## 背景

既存UIが素朴なTailwind CSSデフォルトスタイルで見栄えが悪かったため、モダンなデザインに刷新。

## 変更内容

### `internal/handler/handler.go`
- `statColor` テンプレート関数を追加（種族値バーをステータスごとに色分け）
  - HP=赤, 攻撃=橙, 防御=黄, 特攻=青, 特防=緑, 素早さ=ピンク

### `templates/base.html`
- Google Fonts (Noto Sans JP) 導入
- tailwind.config でフォント設定
- ヘッダー: 赤一色 → indigo/purple/pink グラデーション + モンスターボールSVGアイコン
- 背景: 微グラデーション (`from-slate-50 to-gray-100`)
- 共通CSSクラス追加: `.card-hover`, `.pokemon-img`, `.section-card`, `.section-title`, `.data-table`, `.loading-dot`
- ローディングアニメーション (pulse-dot keyframes)

### `templates/index.html`
- 検索inputに虫眼鏡SVGアイコン追加
- 検索ボタンをグラデーション + shadow付きに
- 種族値フィルターの展開をAlpine.jsトランジションで滑らかに
- ローディングをパルスドットアニメーションに変更
- エラー表示をアイコン付きバッジスタイルに

### `templates/search_results.html`
- グリッドを5列対応 (lg:grid-cols-5)
- カードにホバーエフェクト（浮き上がり + 画像拡大）
- 件数表示を追加
- 空結果時にアイコン付きメッセージ

### `templates/pokemon_detail.html`
- ヘッダーカード: 画像を白カード内に配置、基本情報を2カラムgridで整理
- 種族値バー: `statColor`関数でステータス別に色分け
- 努力値: 色付きバッジ表示
- 進化: 矢印SVG付きカード形式
- 各セクション: アイコン付きタイトル、統一 `.section-card` スタイル
- テーブル: `.data-table` クラスでホバーエフェクト付き統一スタイル
- ナビゲーション: 矢印アイコン付き、現在位置表示 (xxx / 386)

## 関連情報

- Tailwind CSS CDN版を使用（`tailwind.config`はscriptタグ内で設定）
- SVGアイコンはインラインで埋め込み（外部ライブラリ不使用）
- アクセシビリティ属性（aria-label, role, sr-only等）は既存のものを維持
