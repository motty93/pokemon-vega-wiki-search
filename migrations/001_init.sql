-- ポケモン基本情報
CREATE TABLE IF NOT EXISTS pokemon (
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
CREATE TABLE IF NOT EXISTS base_stats (
  pokemon_id  INTEGER PRIMARY KEY REFERENCES pokemon(id),
  hp          INTEGER,
  attack      INTEGER,
  defense     INTEGER,
  sp_attack   INTEGER,
  sp_defense  INTEGER,
  speed       INTEGER
);

-- 努力値
CREATE TABLE IF NOT EXISTS ev_yield (
  pokemon_id  INTEGER PRIMARY KEY REFERENCES pokemon(id),
  hp          INTEGER DEFAULT 0,
  attack      INTEGER DEFAULT 0,
  defense     INTEGER DEFAULT 0,
  sp_attack   INTEGER DEFAULT 0,
  sp_defense  INTEGER DEFAULT 0,
  speed       INTEGER DEFAULT 0
);

-- 進化チェーン
CREATE TABLE IF NOT EXISTS evolution (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  from_pokemon_id INTEGER REFERENCES pokemon(id),
  to_pokemon_id   INTEGER REFERENCES pokemon(id),
  condition       TEXT,
  UNIQUE(from_pokemon_id, to_pokemon_id)
);

-- 技マスタ
CREATE TABLE IF NOT EXISTS move (
  id    INTEGER PRIMARY KEY AUTOINCREMENT,
  name  TEXT UNIQUE NOT NULL
);

-- レベルアップ習得技
CREATE TABLE IF NOT EXISTS learnset_level (
  pokemon_id  INTEGER REFERENCES pokemon(id),
  level       INTEGER,
  move_id     INTEGER REFERENCES move(id),
  PRIMARY KEY (pokemon_id, level, move_id)
);

-- 技マシン習得技
CREATE TABLE IF NOT EXISTS learnset_tm (
  pokemon_id  INTEGER REFERENCES pokemon(id),
  tm_number   TEXT,
  move_id     INTEGER REFERENCES move(id),
  PRIMARY KEY (pokemon_id, move_id)
);

-- 教え技
CREATE TABLE IF NOT EXISTS learnset_tutor (
  pokemon_id  INTEGER REFERENCES pokemon(id),
  move_id     INTEGER REFERENCES move(id),
  PRIMARY KEY (pokemon_id, move_id)
);

-- タマゴ技
CREATE TABLE IF NOT EXISTS learnset_egg (
  pokemon_id    INTEGER REFERENCES pokemon(id),
  move_id       INTEGER REFERENCES move(id),
  parent_chain  TEXT,
  PRIMARY KEY (pokemon_id, move_id)
);

-- 入手方法
CREATE TABLE IF NOT EXISTS encounter (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  pokemon_id  INTEGER REFERENCES pokemon(id),
  location    TEXT,
  method      TEXT,
  note        TEXT,
  UNIQUE(pokemon_id, location, method)
);

-- FTS5による名前検索
CREATE VIRTUAL TABLE IF NOT EXISTS pokemon_fts USING fts5(
  name,
  type1,
  type2,
  content='pokemon',
  content_rowid='id'
);

-- FTSトリガー
CREATE TRIGGER IF NOT EXISTS pokemon_ai AFTER INSERT ON pokemon BEGIN
  INSERT INTO pokemon_fts(rowid, name, type1, type2) VALUES (new.id, new.name, new.type1, new.type2);
END;

CREATE TRIGGER IF NOT EXISTS pokemon_ad AFTER DELETE ON pokemon BEGIN
  INSERT INTO pokemon_fts(pokemon_fts, rowid, name, type1, type2) VALUES('delete', old.id, old.name, old.type1, old.type2);
END;

CREATE TRIGGER IF NOT EXISTS pokemon_au AFTER UPDATE ON pokemon BEGIN
  INSERT INTO pokemon_fts(pokemon_fts, rowid, name, type1, type2) VALUES('delete', old.id, old.name, old.type1, old.type2);
  INSERT INTO pokemon_fts(rowid, name, type1, type2) VALUES (new.id, new.name, new.type1, new.type2);
END;
