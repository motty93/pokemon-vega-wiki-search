package db

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/motty93/pokemon-vega-wiki-crawler/internal/model"
)

// InsertPokemon はポケモン基本情報を挿入する
func InsertPokemon(db *sql.DB, p *model.Pokemon) error {
	_, err := db.Exec(`
		INSERT OR REPLACE INTO pokemon (id, name, type1, type2, ability1, ability2, image_url,
			egg_group1, egg_group2, hatch_steps, gender_ratio, catch_rate, base_friendship,
			base_exp, exp_type, item_50pct, item_5pct)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Name, p.Type1, p.Type2, p.Ability1, p.Ability2, p.ImageURL,
		p.EggGroup1, p.EggGroup2, p.HatchSteps, p.GenderRatio, p.CatchRate, p.BaseFriendship,
		p.BaseExp, p.ExpType, p.Item50pct, p.Item5pct)
	return err
}

// InsertBaseStats は種族値を挿入する
func InsertBaseStats(db *sql.DB, s *model.BaseStats) error {
	_, err := db.Exec(`
		INSERT OR REPLACE INTO base_stats (pokemon_id, hp, attack, defense, sp_attack, sp_defense, speed)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		s.PokemonID, s.HP, s.Attack, s.Defense, s.SpAttack, s.SpDefense, s.Speed)
	return err
}

// InsertEVYield は努力値を挿入する
func InsertEVYield(db *sql.DB, e *model.EVYield) error {
	_, err := db.Exec(`
		INSERT OR REPLACE INTO ev_yield (pokemon_id, hp, attack, defense, sp_attack, sp_defense, speed)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		e.PokemonID, e.HP, e.Attack, e.Defense, e.SpAttack, e.SpDefense, e.Speed)
	return err
}

// InsertEvolution は進化チェーンを挿入する
func InsertEvolution(db *sql.DB, fromID, toID int, condition string) error {
	_, err := db.Exec(`
		INSERT OR IGNORE INTO evolution (from_pokemon_id, to_pokemon_id, condition)
		VALUES (?, ?, ?)`, fromID, toID, condition)
	return err
}

// GetOrCreateMove は技名からIDを取得し、なければ作成する
func GetOrCreateMove(db *sql.DB, name string) (int, error) {
	var id int
	err := db.QueryRow("SELECT id FROM move WHERE name = ?", name).Scan(&id)
	if err == sql.ErrNoRows {
		result, err := db.Exec("INSERT INTO move (name) VALUES (?)", name)
		if err != nil {
			return 0, err
		}
		lastID, err := result.LastInsertId()
		if err != nil {
			return 0, err
		}
		return int(lastID), nil
	}
	return id, err
}

// InsertLearnsetLevel はレベルアップ習得技を挿入する
func InsertLearnsetLevel(db *sql.DB, pokemonID, level, moveID int) error {
	_, err := db.Exec(`
		INSERT OR IGNORE INTO learnset_level (pokemon_id, level, move_id)
		VALUES (?, ?, ?)`, pokemonID, level, moveID)
	return err
}

// InsertLearnsetTM は技マシン習得技を挿入する
func InsertLearnsetTM(db *sql.DB, pokemonID int, tmNumber string, moveID int) error {
	_, err := db.Exec(`
		INSERT OR IGNORE INTO learnset_tm (pokemon_id, tm_number, move_id)
		VALUES (?, ?, ?)`, pokemonID, tmNumber, moveID)
	return err
}

// InsertLearnsetTutor は教え技を挿入する
func InsertLearnsetTutor(db *sql.DB, pokemonID, moveID int) error {
	_, err := db.Exec(`
		INSERT OR IGNORE INTO learnset_tutor (pokemon_id, move_id)
		VALUES (?, ?)`, pokemonID, moveID)
	return err
}

// InsertLearnsetEgg はタマゴ技を挿入する
func InsertLearnsetEgg(db *sql.DB, pokemonID, moveID int, parentChain string) error {
	_, err := db.Exec(`
		INSERT OR IGNORE INTO learnset_egg (pokemon_id, move_id, parent_chain)
		VALUES (?, ?, ?)`, pokemonID, moveID, parentChain)
	return err
}

// InsertEncounter は入手方法を挿入する
func InsertEncounter(db *sql.DB, pokemonID int, location, method, note string) error {
	_, err := db.Exec(`
		INSERT INTO encounter (pokemon_id, location, method, note)
		VALUES (?, ?, ?, ?)`, pokemonID, location, method, note)
	return err
}

// UpdateImageURL は画像URLを更新する
func UpdateImageURL(db *sql.DB, pokemonID int, imageURL string) error {
	_, err := db.Exec("UPDATE pokemon SET image_url = ? WHERE id = ?", imageURL, pokemonID)
	return err
}

// GetPokemonByID はIDからポケモン詳細を取得する
func GetPokemonByID(d *sql.DB, id int) (*model.PokemonDetail, error) {
	detail := &model.PokemonDetail{}

	// 基本情報
	err := d.QueryRow(`
		SELECT id, name, type1, COALESCE(type2,''), COALESCE(ability1,''), COALESCE(ability2,''),
			COALESCE(image_url,''), COALESCE(egg_group1,''), COALESCE(egg_group2,''),
			COALESCE(hatch_steps,0), COALESCE(gender_ratio,''), COALESCE(catch_rate,0),
			COALESCE(base_friendship,0), COALESCE(base_exp,0), COALESCE(exp_type,''),
			COALESCE(item_50pct,''), COALESCE(item_5pct,'')
		FROM pokemon WHERE id = ?`, id).Scan(
		&detail.Pokemon.ID, &detail.Pokemon.Name, &detail.Pokemon.Type1, &detail.Pokemon.Type2,
		&detail.Pokemon.Ability1, &detail.Pokemon.Ability2, &detail.Pokemon.ImageURL,
		&detail.Pokemon.EggGroup1, &detail.Pokemon.EggGroup2, &detail.Pokemon.HatchSteps,
		&detail.Pokemon.GenderRatio, &detail.Pokemon.CatchRate, &detail.Pokemon.BaseFriendship,
		&detail.Pokemon.BaseExp, &detail.Pokemon.ExpType, &detail.Pokemon.Item50pct, &detail.Pokemon.Item5pct)
	if err != nil {
		return nil, fmt.Errorf("pokemon not found: %w", err)
	}

	// 種族値
	d.QueryRow(`
		SELECT COALESCE(hp,0), COALESCE(attack,0), COALESCE(defense,0),
			COALESCE(sp_attack,0), COALESCE(sp_defense,0), COALESCE(speed,0)
		FROM base_stats WHERE pokemon_id = ?`, id).Scan(
		&detail.Stats.HP, &detail.Stats.Attack, &detail.Stats.Defense,
		&detail.Stats.SpAttack, &detail.Stats.SpDefense, &detail.Stats.Speed)
	detail.Stats.PokemonID = id

	// 努力値
	d.QueryRow(`
		SELECT COALESCE(hp,0), COALESCE(attack,0), COALESCE(defense,0),
			COALESCE(sp_attack,0), COALESCE(sp_defense,0), COALESCE(speed,0)
		FROM ev_yield WHERE pokemon_id = ?`, id).Scan(
		&detail.EVYield.HP, &detail.EVYield.Attack, &detail.EVYield.Defense,
		&detail.EVYield.SpAttack, &detail.EVYield.SpDefense, &detail.EVYield.Speed)
	detail.EVYield.PokemonID = id

	// 進化チェーン
	rows, err := d.Query(`
		SELECT e.from_pokemon_id, e.to_pokemon_id, COALESCE(e.condition,''),
			COALESCE(pf.name,''), COALESCE(pt.name,'')
		FROM evolution e
		LEFT JOIN pokemon pf ON e.from_pokemon_id = pf.id
		LEFT JOIN pokemon pt ON e.to_pokemon_id = pt.id
		WHERE e.from_pokemon_id = ? OR e.to_pokemon_id = ?`, id, id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var ev model.EvolutionDetail
			rows.Scan(&ev.FromID, &ev.ToID, &ev.Condition, &ev.FromName, &ev.ToName)
			detail.Evolutions = append(detail.Evolutions, ev)
		}
	}

	// 入手方法
	rows, err = d.Query(`
		SELECT COALESCE(location,''), COALESCE(method,''), COALESCE(note,'')
		FROM encounter WHERE pokemon_id = ?`, id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var enc model.Encounter
			enc.PokemonID = id
			rows.Scan(&enc.Location, &enc.Method, &enc.Note)
			detail.Encounters = append(detail.Encounters, enc)
		}
	}

	// レベルアップ技
	rows, err = d.Query(`
		SELECT ll.level, m.name
		FROM learnset_level ll JOIN move m ON ll.move_id = m.id
		WHERE ll.pokemon_id = ? ORDER BY ll.level`, id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var lm model.LearnsetLevel
			lm.PokemonID = id
			rows.Scan(&lm.Level, &lm.MoveName)
			detail.LevelMoves = append(detail.LevelMoves, lm)
		}
	}

	// 技マシン
	rows, err = d.Query(`
		SELECT ll.tm_number, m.name
		FROM learnset_tm ll JOIN move m ON ll.move_id = m.id
		WHERE ll.pokemon_id = ? ORDER BY ll.tm_number`, id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var tm model.LearnsetTM
			tm.PokemonID = id
			rows.Scan(&tm.TMNumber, &tm.MoveName)
			detail.TMMoves = append(detail.TMMoves, tm)
		}
	}

	// 教え技
	rows, err = d.Query(`
		SELECT m.name
		FROM learnset_tutor lt JOIN move m ON lt.move_id = m.id
		WHERE lt.pokemon_id = ?`, id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var t model.LearnsetTutor
			t.PokemonID = id
			rows.Scan(&t.MoveName)
			detail.TutorMoves = append(detail.TutorMoves, t)
		}
	}

	// タマゴ技
	rows, err = d.Query(`
		SELECT m.name, COALESCE(le.parent_chain,'')
		FROM learnset_egg le JOIN move m ON le.move_id = m.id
		WHERE le.pokemon_id = ?`, id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var eg model.LearnsetEgg
			eg.PokemonID = id
			rows.Scan(&eg.MoveName, &eg.ParentChain)
			detail.EggMoves = append(detail.EggMoves, eg)
		}
	}

	return detail, nil
}

// SearchPokemon はポケモンを検索する
func SearchPokemon(d *sql.DB, params model.SearchParams) ([]model.SearchResult, error) {
	var conditions []string
	var args []interface{}

	// 名前検索（FTS5）
	if params.Query != "" {
		conditions = append(conditions, "p.id IN (SELECT rowid FROM pokemon_fts WHERE pokemon_fts MATCH ?)")
		args = append(args, params.Query+"*")
	}

	// タイプフィルター
	if params.Type != "" {
		conditions = append(conditions, "(p.type1 = ? OR p.type2 = ?)")
		args = append(args, params.Type, params.Type)
	}

	// 種族値フィルター
	statFilters := []struct {
		col    string
		minVal int
		maxVal int
	}{
		{"bs.hp", params.HPMin, params.HPMax},
		{"bs.attack", params.AttackMin, params.AttackMax},
		{"bs.defense", params.DefenseMin, params.DefenseMax},
		{"bs.sp_attack", params.SpAtkMin, params.SpAtkMax},
		{"bs.sp_defense", params.SpDefMin, params.SpDefMax},
		{"bs.speed", params.SpeedMin, params.SpeedMax},
	}

	for _, f := range statFilters {
		if f.minVal > 0 {
			conditions = append(conditions, fmt.Sprintf("%s >= ?", f.col))
			args = append(args, f.minVal)
		}
		if f.maxVal > 0 {
			conditions = append(conditions, fmt.Sprintf("%s <= ?", f.col))
			args = append(args, f.maxVal)
		}
	}

	query := `
		SELECT p.id, p.name, p.type1, COALESCE(p.type2,''), COALESCE(p.image_url,''),
			COALESCE(bs.hp,0), COALESCE(bs.attack,0), COALESCE(bs.defense,0),
			COALESCE(bs.sp_attack,0), COALESCE(bs.sp_defense,0), COALESCE(bs.speed,0)
		FROM pokemon p
		LEFT JOIN base_stats bs ON p.id = bs.pokemon_id`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY p.id"

	rows, err := d.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.SearchResult
	for rows.Next() {
		var r model.SearchResult
		err := rows.Scan(&r.Pokemon.ID, &r.Pokemon.Name, &r.Pokemon.Type1, &r.Pokemon.Type2,
			&r.Pokemon.ImageURL, &r.Stats.HP, &r.Stats.Attack, &r.Stats.Defense,
			&r.Stats.SpAttack, &r.Stats.SpDefense, &r.Stats.Speed)
		if err != nil {
			return nil, err
		}
		r.Stats.PokemonID = r.Pokemon.ID
		results = append(results, r)
	}

	return results, nil
}

// GetAllTypes は全タイプの一覧を取得する
func GetAllTypes(d *sql.DB) ([]string, error) {
	rows, err := d.Query(`
		SELECT DISTINCT type1 FROM pokemon
		UNION
		SELECT DISTINCT type2 FROM pokemon WHERE type2 IS NOT NULL AND type2 != ''
		ORDER BY 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var types []string
	for rows.Next() {
		var t string
		rows.Scan(&t)
		if t != "" {
			types = append(types, t)
		}
	}
	return types, nil
}
