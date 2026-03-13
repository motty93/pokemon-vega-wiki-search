# ポケモンベガ 図鑑検索サイト 設計書

## プロジェクト概要

ファンメイドポケモンROM「ポケモンベガ」の図鑑データをWikiからスクレイピングし、高速に検索・閲覧できるWebサイトを構築する。

**データソース:** `https://w.atwiki.jp/altair1/pages/19.html`（図鑑一覧）
**ポケモン数:** No.001〜No.386（386匹）

---

## 技術スタック

| 役割 | 技術 |
|---|---|
| スクレイピング | Go + [colly](https://github.com/gocolly/colly) |
| Webサーバー | Go（net/http or chi） |
| フロントエンド | Go Template + htmx + Alpine.js |
| DB | Turso（LibSQL / SQLite互換） |
| 画像ストレージ | ローカル（static/images/pokemon/） |
| スクレイピング実行 | Cloud Run Jobs（手動実行） |
| Webサーバーデプロイ | Cloud Run |

---

## プロジェクト構成

```
/
├── cmd/
│   ├── server/          # Webサーバーエントリーポイント
│   │   └── main.go
│   └── scraper/         # スクレイパーエントリーポイント
│       └── main.go
├── internal/
│   ├── db/              # Turso接続・クエリ・マイグレーション
│   ├── scraper/         # スクレイピングロジック
│   ├── model/           # 構造体定義（Pokemon, Move等）
│   ├── handler/         # HTTPハンドラー（htmxレスポンス含む）
│   └── storage/         # 画像ダウンロード（Wikiからローカルへ）
├── templates/            # Go Template（base, index, detail, partials）
├── static/               # CSS（Tailwind CDNで可）
├── migrations/           # SQLマイグレーションファイル
├── Dockerfile.server     # Cloud Run用
├── Dockerfile.scraper    # Cloud Run Jobs用
└── docker-compose.yml    # ローカル開発用
```

---

## DBスキーマ（SQLite / LibSQL）

```sql
-- ポケモン基本情報
CREATE TABLE pokemon (
  id              INTEGER PRIMARY KEY,
  name            TEXT NOT NULL,
  type1           TEXT NOT NULL,
  type2           TEXT,
  ability1        TEXT,
  ability2        TEXT,
  image_url       TEXT,
  egg_group1      TEXT,
  egg_group2      TEXT,
  hatch_steps     INTEGER,
  gender_ratio    TEXT,
  catch_rate      INTEGER,
  base_friendship INTEGER,
  base_exp        INTEGER,
  exp_type        TEXT,
  item_50pct      TEXT,
  item_5pct       TEXT
);

-- 種族値
CREATE TABLE base_stats (
  pokemon_id  INTEGER PRIMARY KEY REFERENCES pokemon(id),
  hp          INTEGER,
  attack      INTEGER,
  defense     INTEGER,
  sp_attack   INTEGER,
  sp_defense  INTEGER,
  speed       INTEGER
);

-- 努力値
CREATE TABLE ev_yield (
  pokemon_id  INTEGER PRIMARY KEY REFERENCES pokemon(id),
  hp          INTEGER DEFAULT 0,
  attack      INTEGER DEFAULT 0,
  defense     INTEGER DEFAULT 0,
  sp_attack   INTEGER DEFAULT 0,
  sp_defense  INTEGER DEFAULT 0,
  speed       INTEGER DEFAULT 0
);

-- 進化チェーン
CREATE TABLE evolution (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  from_pokemon_id INTEGER REFERENCES pokemon(id),
  to_pokemon_id   INTEGER REFERENCES pokemon(id),
  condition       TEXT   -- 例: "Lv.16"
);

-- 技マスタ
CREATE TABLE move (
  id    INTEGER PRIMARY KEY AUTOINCREMENT,
  name  TEXT UNIQUE NOT NULL
);

-- レベルアップ習得技
CREATE TABLE learnset_level (
  pokemon_id  INTEGER REFERENCES pokemon(id),
  level       INTEGER,
  move_id     INTEGER REFERENCES move(id),
  PRIMARY KEY (pokemon_id, level, move_id)
);

-- 技マシン習得技
CREATE TABLE learnset_tm (
  pokemon_id  INTEGER REFERENCES pokemon(id),
  tm_number   TEXT,   -- 例: "技01", "秘05"
  move_id     INTEGER REFERENCES move(id),
  PRIMARY KEY (pokemon_id, move_id)
);

-- 教え技
CREATE TABLE learnset_tutor (
  pokemon_id  INTEGER REFERENCES pokemon(id),
  move_id     INTEGER REFERENCES move(id),
  PRIMARY KEY (pokemon_id, move_id)
);

-- タマゴ技
CREATE TABLE learnset_egg (
  pokemon_id    INTEGER REFERENCES pokemon(id),
  move_id       INTEGER REFERENCES move(id),
  parent_chain  TEXT,   -- 例: "ガルーラ→リープン"
  PRIMARY KEY (pokemon_id, move_id)
);

-- 入手方法
CREATE TABLE encounter (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  pokemon_id  INTEGER REFERENCES pokemon(id),
  location    TEXT,   -- 例: "サファリゾーン-東"
  method      TEXT,   -- 例: "草むら", "御三家"
  note        TEXT
);

-- FTS5による名前検索
CREATE VIRTUAL TABLE pokemon_fts USING fts5(
  name,
  type1,
  type2,
  content='pokemon',
  content_rowid='id'
);
```

---

## スクレイパーの実装要件（cmd/scraper）

### 処理フロー

1. 図鑑一覧ページ（`/pages/19.html`）から全386匹の名前とリンクURLを取得
2. 各ポケモンのページを巡回（リクエスト間隔は1〜2秒のランダムスリープ）
3. 各ページから以下を取得・パース：
   - タイプ、特性
   - 種族値（HP/攻撃/防御/特攻/特防/素早さ）
   - 進化前後（ポケモン名 + 条件）
   - 入手方法（場所 + 方法）
   - 努力値
   - タマゴデータ（タマゴグループ・孵化歩数・性別比率・被捕獲率・なつき度・基礎経験値・経験値タイプ）
   - 野生持ち物（50%・5%）
   - 習得技4種（レベルアップ・技マシン・教え技・タマゴ技）
   - 画像URL
4. 画像をローカル（static/images/pokemon/）にダウンロードし、DBのimage_urlを更新
5. 全データをTurso（LibSQL）に挿入

### 環境変数

```
TURSO_URL=
TURSO_AUTH_TOKEN=
```

---

## Webサーバーの実装要件（cmd/server）

### ルーティング

```
GET /                          # トップ・検索ページ
GET /pokemon/{id}              # ポケモン詳細
GET /search?q=&type=&speed_min= # htmx部分レスポンス（検索結果のHTML断片を返す）
```

### 検索機能

- 名前のあいまい検索（FTS5 MATCH または LIKE）
- タイプ絞り込み（type1 / type2）
- 種族値の範囲フィルター（HP・攻撃・防御・特攻・特防・素早さのmin/max）
- 検索結果はhtmxで部分更新（ページリロードなし）

### テンプレート構成

- `base.html`: ヘッダー・フッター共通レイアウト
- `index.html`: 検索フォーム + 結果一覧（htmxターゲット）
- `pokemon_card.html`: 検索結果の1件分（htmxで差し込むpartial）
- `pokemon_detail.html`: 詳細ページ（種族値・技一覧・進化チェーン表示）

---

## Dockerfileの要件

### Dockerfile.server（マルチステージビルド）

```dockerfile
FROM golang:1.23 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -o server ./cmd/server

FROM scratch
COPY --from=builder /app/server /server
COPY --from=builder /app/templates /templates
COPY --from=builder /app/static /static
ENTRYPOINT ["/server"]
```

### Dockerfile.scraper

```dockerfile
FROM golang:1.23 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -o scraper ./cmd/scraper

FROM scratch
COPY --from=builder /app/scraper /scraper
ENTRYPOINT ["/scraper"]
```

---

## 実装の優先順位

1. **DBマイグレーション** `migrations/001_init.sql` を作成しTursoに適用
2. **スクレイパー実装** ローカルでSQLiteに対して動作確認してからTursoに切り替え
3. **Webサーバー + 検索UI** Go Template + htmxで検索・詳細ページを実装
4. **画像ダウンロード** Wikiから画像をローカルにダウンロードする処理を追加
5. **Dockerize + Cloud Runデプロイ** server / scraper それぞれのイメージをビルド・デプロイ
6. **Cloud Run Jobs登録** scraperをCloud Run Jobsとして手動実行できるよう設定

---

## 補足・注意事項

- スクレイピング時はリクエスト間隔を必ず設ける（`time.Sleep(1*time.Second + rand.Duration)`）
- TursoのLibSQLドライバは `github.com/tursodatabase/libsql-client-go` を使用
- collyのパーサーはテーブル構造をベースに実装（各セクションのテーブルヘッダーで判定）
- ローカル開発時はTursoの代わりにファイルベースのSQLite（`file:pokemon.db`）を使うと便利
- 画像URLはスクレイピング時点ではWikiのURLを一時保存し、ローカルダウンロード後にパスを更新する
