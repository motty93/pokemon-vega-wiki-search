-- 検索ログ
CREATE TABLE IF NOT EXISTS search_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  query TEXT NOT NULL,
  result_count INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_search_log_created_at ON search_log(created_at);
CREATE INDEX IF NOT EXISTS idx_search_log_query ON search_log(query);

-- ページビュー
CREATE TABLE IF NOT EXISTS page_view (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  pokemon_id INTEGER NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  FOREIGN KEY (pokemon_id) REFERENCES pokemon(id)
);

CREATE INDEX IF NOT EXISTS idx_page_view_pokemon_id ON page_view(pokemon_id);
CREATE INDEX IF NOT EXISTS idx_page_view_created_at ON page_view(created_at);
