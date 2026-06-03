#!/usr/bin/env bash
set -euo pipefail

# pokemon.db の各テーブルを static/data/ 配下に parquet として書き出す。
# DuckDB CLI の sqlite_scanner 拡張を使用する（初回は INSTALL でネットアクセスが発生）。
# FTS5 仮想テーブル（pokemon_fts）とトリガーは対象外。

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DB="${ROOT}/data/pokemon.db"
OUT="${ROOT}/static/data"

mkdir -p "${OUT}"

TABLES=(
  pokemon base_stats ev_yield evolution move
  learnset_level learnset_tm learnset_tutor learnset_egg encounter
)

SQL="INSTALL sqlite; LOAD sqlite; ATTACH '${DB}' AS s (TYPE sqlite);"
for t in "${TABLES[@]}"; do
  SQL+=" COPY (SELECT * FROM s.${t}) TO '${OUT}/${t}.parquet' (FORMAT parquet, COMPRESSION zstd);"
done
SQL+=" DETACH s;"

duckdb -c "${SQL}"

echo "---"
echo "exported ${#TABLES[@]} tables to ${OUT}"
ls -la "${OUT}"
