# トップページを検索特化UIに変更

## 背景

未来屋書店の在庫検索UIのように、トップページはフルスクリーンの検索バーのみにして初期表示を爆速にしたい。
従来はIndexハンドラーで `GetAllTypes()` をDB問い合わせしていたが、タイプリスト（18種）は固定なのでテンプレートにハードコードすることでDB不要に。

## 変更内容

### `internal/handler/handler.go`
- `Index` ハンドラーから `mydb.GetAllTypes()` 呼び出しを削除
- テンプレートに `nil` を渡すだけのシンプルなハンドラーに

### `templates/base.html`
- header/footerを削除し、各ページのcontentテンプレートに移譲
- bodyに共通スタイルのみ保持（トップと詳細でレイアウトが大きく異なるため）

### `templates/index.html`
- フルスクリーンのダーク背景（幾何学パターン + radial gradient）
- 中央に大きな検索バー（Google検索風の白い丸角input）
- タイプselectは「詳細フィルター」トグル内に格納（ハードコードの18タイプ）
- 種族値フィルターも同じトグル内
- 検索結果が来たらヒーロー外にスクロール表示
- DB問い合わせゼロで初期表示

### `templates/pokemon_detail.html`
- base.htmlからheader/footerが消えたため、ページ内にheader/footerを配置

## 効果

- トップページの初期表示がDB接続不要で高速化
- TursoなどリモートDBのレイテンシが初期表示に影響しなくなった
