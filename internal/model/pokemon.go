package model

// Pokemon はポケモンの基本情報
type Pokemon struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Type1          string `json:"type1"`
	Type2          string `json:"type2"`
	Ability1       string `json:"ability1"`
	Ability2       string `json:"ability2"`
	ImageURL       string `json:"image_url"`
	EggGroup1      string `json:"egg_group1"`
	EggGroup2      string `json:"egg_group2"`
	HatchSteps     int    `json:"hatch_steps"`
	GenderRatio    string `json:"gender_ratio"`
	CatchRate      int    `json:"catch_rate"`
	BaseFriendship int    `json:"base_friendship"`
	BaseExp        int    `json:"base_exp"`
	ExpType        string `json:"exp_type"`
	Item50pct      string `json:"item_50pct"`
	Item5pct       string `json:"item_5pct"`
}

// BaseStats は種族値
type BaseStats struct {
	PokemonID int `json:"pokemon_id"`
	HP        int `json:"hp"`
	Attack    int `json:"attack"`
	Defense   int `json:"defense"`
	SpAttack  int `json:"sp_attack"`
	SpDefense int `json:"sp_defense"`
	Speed     int `json:"speed"`
}

// Total は種族値合計を返す
func (b BaseStats) Total() int {
	return b.HP + b.Attack + b.Defense + b.SpAttack + b.SpDefense + b.Speed
}

// EVYield は努力値
type EVYield struct {
	PokemonID int `json:"pokemon_id"`
	HP        int `json:"hp"`
	Attack    int `json:"attack"`
	Defense   int `json:"defense"`
	SpAttack  int `json:"sp_attack"`
	SpDefense int `json:"sp_defense"`
	Speed     int `json:"speed"`
}

// Evolution は進化チェーン
type Evolution struct {
	ID            int    `json:"id"`
	FromPokemonID int    `json:"from_pokemon_id"`
	ToPokemonID   int    `json:"to_pokemon_id"`
	Condition     string `json:"condition"`
}

// Move は技
type Move struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// LearnsetLevel はレベルアップ習得技
type LearnsetLevel struct {
	PokemonID int    `json:"pokemon_id"`
	Level     int    `json:"level"`
	MoveName  string `json:"move_name"`
}

// LearnsetTM は技マシン習得技
type LearnsetTM struct {
	PokemonID int    `json:"pokemon_id"`
	TMNumber  string `json:"tm_number"`
	MoveName  string `json:"move_name"`
}

// LearnsetTutor は教え技
type LearnsetTutor struct {
	PokemonID int    `json:"pokemon_id"`
	MoveName  string `json:"move_name"`
}

// LearnsetEgg はタマゴ技
type LearnsetEgg struct {
	PokemonID   int    `json:"pokemon_id"`
	MoveName    string `json:"move_name"`
	ParentChain string `json:"parent_chain"`
}

// Encounter は入手方法
type Encounter struct {
	ID        int    `json:"id"`
	PokemonID int    `json:"pokemon_id"`
	Location  string `json:"location"`
	Method    string `json:"method"`
	Note      string `json:"note"`
}

// PokemonDetail は詳細ページ用の集約型
type PokemonDetail struct {
	Pokemon    Pokemon
	Stats      BaseStats
	EVYield    EVYield
	Evolutions []EvolutionDetail
	Encounters []Encounter
	LevelMoves []LearnsetLevel
	TMMoves    []LearnsetTM
	TutorMoves []LearnsetTutor
	EggMoves   []LearnsetEgg
}

// EvolutionDetail は進化チェーン表示用
type EvolutionDetail struct {
	FromName  string `json:"from_name"`
	ToName    string `json:"to_name"`
	FromID    int    `json:"from_id"`
	ToID      int    `json:"to_id"`
	Condition string `json:"condition"`
}

// SearchResult は検索結果
type SearchResult struct {
	Pokemon Pokemon
	Stats   BaseStats
}

// SearchParams は検索パラメータ
type SearchParams struct {
	Query      string
	Type       string
	HPMin      int
	HPMax      int
	AttackMin  int
	AttackMax  int
	DefenseMin int
	DefenseMax int
	SpAtkMin   int
	SpAtkMax   int
	SpDefMin   int
	SpDefMax   int
	SpeedMin   int
	SpeedMax   int
}
