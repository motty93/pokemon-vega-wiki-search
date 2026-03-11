# スクレイパーのパーサー全面書き換え

## 背景

DBに保存されたデータを確認したところ、type1が全てノーマル、type2/ability空、種族値全て0など、パースが全滅していた。原因はWikiの実際のHTML構造がパーサーの想定と完全に異なっていたため。

### 想定していた構造
- 各セクション（タイプ、種族値、進化など）が別々のテーブル
- th/td のヘッダー行でテーブル種別を判定

### 実際の構造
- Table#1: 基本情報が全て1つのテーブルに入っている（th なし、td×2列）
- セクションは1列のみのtd行（「種族値」「前後の進化」等）で区切られる
- タイプは「・」区切り（「くさ・ひこう」）
- 特性は「・」区切り（「しんりょく・あついしぼう」）
- Table#2以降: 習得技テーブル（TH ヘッダーあり）

## 変更内容

### `internal/scraper/scraper.go`

- `ScrapePokemonPage()`: テーブルインデックスで判定する方式に変更
  - Table#1 → `parseMainTable()` で基本情報を一括パース
  - Table#2以降 → THヘッダーで技テーブルを判定
- `parseMainTable()` を新規追加: セクションヘッダー（1列td）でモード切替し、2列tdからデータ取得
  - info / stats / evolution / encounter / ev / egg / hidden / item のセクション管理
- 各セクションの個別パース関数を追加:
  - `parseMainInfoRow()`, `parseStatRow()`, `parseEVRow()`, `parseEvolutionRow()`
  - `parseEncounterRow()`, `parseEggRow()`, `parseHiddenRow()`, `parseItemRow()`
- タイプ区切り: `・` と空白の両方に対応
- 特性区切り: `・` と `/` の両方に対応
- 画像: `bar.gif` を除外するフィルタ追加
- 旧パーサー関数（`parseBasicInfoTable`, `parseBaseStatsTable`, `parseEVYieldTable`, `parseEvolutionTable`, `parseEncounterTable`, `parseEggDataTable`）を削除
