package scraper

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gocolly/colly"
	mydb "github.com/motty93/pokemon-vega-wiki-crawler/internal/db"
	"github.com/motty93/pokemon-vega-wiki-crawler/internal/model"
)

const baseURL = "https://w.atwiki.jp"
const indexURL = baseURL + "/altair1/pages/19.html"

// PokemonLink は図鑑一覧から取得するリンク情報
type PokemonLink struct {
	ID   int
	Name string
	URL  string
}

// Run はスクレイピング全体を実行する
func Run(db *sql.DB) error {
	links, err := FetchPokemonLinks()
	if err != nil {
		return fmt.Errorf("failed to fetch pokemon links: %w", err)
	}
	log.Printf("Found %d pokemon links", len(links))

	// 名前→IDマップを構築（正規化済み）
	nameMap := make(map[string]int, len(links))
	for _, link := range links {
		nameMap[normalizeName(link.Name)] = link.ID
	}

	// 進化データは全ポケモン挿入後にまとめて保存する（FK制約対策）
	var allEvolutions []model.EvolutionDetail
	// 技名→IDのメモリキャッシュ（リモートDBへのクエリを削減）
	moveCache := make(map[string]int)

	for i, link := range links {
		log.Printf("[%d/%d] Scraping %s (No.%03d)...", i+1, len(links), link.Name, link.ID)

		detail, err := ScrapePokemonPage(link)
		if err != nil {
			log.Printf("  WARNING: failed to scrape %s: %v", link.Name, err)
			continue
		}

		// 進化データを退避してからSave
		allEvolutions = append(allEvolutions, detail.Evolutions...)
		detail.Evolutions = nil

		if err := SavePokemon(db, detail, moveCache); err != nil {
			log.Printf("  WARNING: failed to save %s: %v", link.Name, err)
			continue
		}

		// リクエスト間隔: 1〜2秒のランダムスリープ
		sleepDuration := time.Duration(1000+rand.Intn(1000)) * time.Millisecond
		time.Sleep(sleepDuration)
	}

	// 全ポケモン挿入後に進化データを保存
	if len(allEvolutions) > 0 {
		log.Printf("Saving %d evolutions...", len(allEvolutions))
		for _, ev := range allEvolutions {
			fromID, err1 := resolveEvolutionName(ev.FromName, nameMap, db)
			toID, err2 := resolveEvolutionName(ev.ToName, nameMap, db)
			if err1 != nil || err2 != nil {
				log.Printf("  WARNING: could not resolve evolution %q -> %q", ev.FromName, ev.ToName)
				continue
			}
			if err := mydb.InsertEvolution(db, fromID, toID, ev.Condition); err != nil {
				log.Printf("  WARNING: insert evolution: %v", err)
			}
		}
	}

	return nil
}

// FetchPokemonLinks は図鑑一覧ページからポケモンのリンクを取得する
// テーブルは1行に8匹分（No. + 名前 の2列 × 8 = 16列）の構造
func FetchPokemonLinks() ([]PokemonLink, error) {
	var links []PokemonLink
	c := colly.NewCollector()

	idRegex := regexp.MustCompile(`^(\d+)$`)

	c.OnHTML("table", func(e *colly.HTMLElement) {
		e.ForEach("tr", func(_ int, row *colly.HTMLElement) {
			// 各行のtdを取得
			var tds []*colly.HTMLElement
			row.ForEach("td", func(_ int, td *colly.HTMLElement) {
				tds = append(tds, td)
			})

			// 2列ずつペア（No., 名前）で処理
			for i := 0; i+1 < len(tds); i += 2 {
				numText := strings.TrimSpace(tds[i].Text)
				match := idRegex.FindStringSubmatch(numText)
				if match == nil {
					continue
				}

				id, err := strconv.Atoi(match[1])
				if err != nil || id < 1 || id > 386 {
					continue
				}

				name := strings.TrimSpace(tds[i+1].Text)
				href := tds[i+1].ChildAttr("a", "href")
				if href == "" || name == "" {
					continue
				}

				if !strings.HasPrefix(href, "http") {
					href = "https:" + href
				}

				links = append(links, PokemonLink{ID: id, Name: name, URL: href})
			}
		})
	})

	if err := c.Visit(indexURL); err != nil {
		return nil, err
	}

	// ID順にソート
	sort.Slice(links, func(i, j int) bool {
		return links[i].ID < links[j].ID
	})

	return links, nil
}

// ScrapePokemonPage は個別ポケモンページをスクレイピングする
// Wiki構造: Table#1に基本情報が全て入っている（td×2列、thなし）
// Table#2以降は習得技（レベル技、技マシン、教え技、タマゴ技）
func ScrapePokemonPage(link PokemonLink) (*model.PokemonDetail, error) {
	detail := &model.PokemonDetail{}
	detail.Pokemon.ID = link.ID
	detail.Pokemon.Name = link.Name
	detail.Stats.PokemonID = link.ID
	detail.EVYield.PokemonID = link.ID

	c := colly.NewCollector()

	// 画像URL取得（bar.gif以外の最初の画像）
	c.OnHTML("#wikibody img", func(e *colly.HTMLElement) {
		src := e.Attr("src")
		if src == "" || detail.Pokemon.ImageURL != "" {
			return
		}
		if strings.Contains(src, "bar.gif") {
			return
		}
		if strings.HasPrefix(src, "//") {
			src = "https:" + src
		} else if !strings.HasPrefix(src, "http") {
			src = baseURL + src
		}
		detail.Pokemon.ImageURL = src
	})

	tableIndex := 0
	c.OnHTML("#wikibody table", func(e *colly.HTMLElement) {
		tableIndex++

		if tableIndex == 1 {
			// Table#1: 基本情報テーブル（全データが1つのテーブルに入っている）
			parseMainTable(e, detail)
		} else {
			// Table#2以降: 習得技テーブル（THヘッダーで判定）
			headerText := e.ChildText("tr:first-child th")
			switch {
			case containsAny(headerText, "Lv"):
				parseLevelMovesTable(e, detail)
			case containsAny(headerText, "No"):
				parseTMMovesTable(e, detail)
			default:
				// TH[技] のみのテーブル → 教え技 or タマゴ技を判定
				// タマゴ技はtdのテキストに遺伝経路（→）が含まれる
				isEggMoves := false
				e.ForEach("tr", func(_ int, row *colly.HTMLElement) {
					cells := childTexts(row, "td")
					if len(cells) >= 1 && strings.Contains(cells[0], "→") {
						isEggMoves = true
					}
				})
				if isEggMoves {
					parseEggMovesTable(e, detail)
				} else {
					parseTutorMovesTable(e, detail)
				}
			}
		}
	})

	if err := c.Visit(link.URL); err != nil {
		return nil, err
	}

	// パース結果のバリデーション
	if err := detail.Validate(); err != nil {
		return nil, err
	}

	return detail, nil
}

// parseMainTable はTable#1（基本情報テーブル）をパースする
// セクションヘッダー（1列のみのtd）でモード切替し、2列のtdからデータを取得
func parseMainTable(e *colly.HTMLElement, detail *model.PokemonDetail) {
	section := ""

	e.ForEach("tr", func(_ int, row *colly.HTMLElement) {
		tds := childTexts(row, "td")

		// 1列のみの行はセクションヘッダー
		if len(tds) == 1 {
			label := tds[0]
			switch {
			case label == "図鑑" || label == "":
				section = "info"
			case label == "種族値":
				section = "stats"
			case label == "前後の進化":
				section = "evolution"
			case label == "入手方法":
				section = "encounter"
			case label == "努力値":
				section = "ev"
			case label == "タマゴデータ":
				section = "egg"
			case label == "隠しデータ":
				section = "hidden"
			case label == "野生で持っている道具":
				section = "item"
			}
			return
		}

		if len(tds) < 2 {
			return
		}

		label := tds[0]
		value := tds[1]

		switch section {
		case "info":
			parseMainInfoRow(label, value, detail)
		case "stats":
			parseStatRow(label, value, &detail.Stats)
		case "evolution":
			parseEvolutionRow(label, value, row, detail)
		case "encounter":
			parseEncounterRow(label, value, detail)
		case "ev":
			parseStatRow(label, value, nil) // EVは別処理
			parseEVRow(label, value, detail)
		case "egg":
			parseEggRow(label, value, detail)
		case "hidden":
			parseHiddenRow(label, value, detail)
		case "item":
			parseItemRow(label, value, detail)
		}
	})
}

func parseMainInfoRow(label, value string, detail *model.PokemonDetail) {
	switch label {
	case "タイプ":
		// 「くさ・ひこう」「くさ ひこう」両方に対応
		var types []string
		if strings.Contains(value, "・") {
			types = strings.Split(value, "・")
		} else {
			types = strings.Fields(value)
		}
		if len(types) >= 1 {
			detail.Pokemon.Type1 = strings.TrimSpace(types[0])
		}
		if len(types) >= 2 {
			detail.Pokemon.Type2 = strings.TrimSpace(types[1])
		}
	case "特性":
		// 「・」または「/」区切り
		sep := "・"
		if strings.Contains(value, "/") {
			sep = "/"
		}
		abilities := strings.Split(value, sep)
		if len(abilities) >= 1 {
			detail.Pokemon.Ability1 = strings.TrimSpace(abilities[0])
		}
		if len(abilities) >= 2 {
			detail.Pokemon.Ability2 = strings.TrimSpace(abilities[1])
		}
	}
}

func parseStatRow(label, value string, stats *model.BaseStats) {
	if stats == nil {
		return
	}
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return
	}
	switch label {
	case "HP":
		stats.HP = n
	case "攻撃", "こうげき":
		stats.Attack = n
	case "防御", "ぼうぎょ":
		stats.Defense = n
	case "特攻", "とくこう":
		stats.SpAttack = n
	case "特防", "とくぼう":
		stats.SpDefense = n
	case "素早さ", "すばやさ":
		stats.Speed = n
	}
}

func parseEVRow(label, value string, detail *model.PokemonDetail) {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return
	}
	switch label {
	case "HP":
		detail.EVYield.HP = n
	case "攻撃", "こうげき":
		detail.EVYield.Attack = n
	case "防御", "ぼうぎょ":
		detail.EVYield.Defense = n
	case "特攻", "とくこう":
		detail.EVYield.SpAttack = n
	case "特防", "とくぼう":
		detail.EVYield.SpDefense = n
	case "素早さ", "すばやさ":
		detail.EVYield.Speed = n
	}
}

func parseEvolutionRow(label, value string, row *colly.HTMLElement, detail *model.PokemonDetail) {
	if label != "進化前" && label != "進化後" {
		return
	}
	if value == "‐" || value == "-" || value == "" || value == "進化しない" {
		return
	}

	// aタグが複数ある場合（分岐進化）を処理
	// 各aタグのテキスト（ポケモン名）とその後の括弧内テキスト（条件）を個別に取得
	type evoEntry struct {
		name      string
		condition string
	}
	var entries []evoEntry

	// td:nth-child(2) 内のaタグを個別に取得
	row.ForEach("td:nth-child(2) a", func(_ int, a *colly.HTMLElement) {
		name := strings.TrimSpace(a.Text)
		if name != "" {
			entries = append(entries, evoEntry{name: name})
		}
	})

	if len(entries) > 0 {
		// 条件を抽出: value全体から各ポケモン名+条件を分離
		// 例: "サーナイト(攻撃≠防御 Lv.30)エルレイド(攻撃＝防御 Lv.30)"
		remaining := value
		for i := range entries {
			// ポケモン名の後の括弧内を条件として取得
			idx := strings.Index(remaining, entries[i].name)
			if idx < 0 {
				continue
			}
			remaining = remaining[idx+len(entries[i].name):]
			if len(remaining) > 0 && (remaining[0] == '(' || strings.HasPrefix(remaining, "（")) {
				// 括弧の開始位置を見つける
				openParen := remaining[0]
				var closeParen byte = ')'
				if openParen == 0xef { // 全角括弧の先頭バイト
					closeParen = 0
				}
				endIdx := -1
				if closeParen != 0 {
					endIdx = strings.Index(remaining, string(closeParen))
				}
				if endIdx < 0 {
					// 全角括弧 or 次のポケモン名の手前まで
					if i+1 < len(entries) {
						endIdx = strings.Index(remaining, entries[i+1].name)
					}
				}
				if endIdx > 0 {
					cond := remaining[1:endIdx]
					cond = strings.Trim(cond, "()")
					cond = strings.Trim(cond, "（）")
					entries[i].condition = strings.TrimSpace(cond)
					remaining = remaining[endIdx:]
				}
			}
		}
	} else {
		// aタグがない場合はテキストをそのまま解析
		name := value
		condition := ""
		if idx := strings.Index(value, "("); idx > 0 {
			name = strings.TrimSpace(value[:idx])
			condition = strings.TrimSpace(value[idx:])
			condition = strings.Trim(condition, "()")
		}
		entries = append(entries, evoEntry{name: name, condition: condition})
	}

	for _, entry := range entries {
		ev := model.EvolutionDetail{Condition: entry.condition}
		if label == "進化前" {
			ev.FromName = entry.name
			ev.ToName = detail.Pokemon.Name
		} else {
			ev.FromName = detail.Pokemon.Name
			ev.ToName = entry.name
		}
		detail.Evolutions = append(detail.Evolutions, ev)
	}
}

func parseEncounterRow(label, value string, detail *model.PokemonDetail) {
	if label == "入手方法" {
		return // セクションヘッダーの重複スキップ
	}
	detail.Encounters = append(detail.Encounters, model.Encounter{
		PokemonID: detail.Pokemon.ID,
		Location:  label,
		Method:    value,
	})
}

func parseEggRow(label, value string, detail *model.PokemonDetail) {
	switch {
	case label == "タマゴグループ":
		groups := strings.Split(value, "・")
		if len(groups) == 1 {
			groups = strings.Split(value, "/")
		}
		if len(groups) >= 1 {
			detail.Pokemon.EggGroup1 = strings.TrimSpace(groups[0])
		}
		if len(groups) >= 2 {
			detail.Pokemon.EggGroup2 = strings.TrimSpace(groups[1])
		}
	case label == "孵化歩数":
		cleaned := strings.ReplaceAll(value, ",", "")
		cleaned = strings.ReplaceAll(cleaned, "歩", "")
		cleaned = strings.TrimSpace(cleaned)
		n, err := strconv.Atoi(cleaned)
		if err == nil {
			detail.Pokemon.HatchSteps = n
		}
	}
}

func parseHiddenRow(label, value string, detail *model.PokemonDetail) {
	switch {
	case label == "性別比率":
		detail.Pokemon.GenderRatio = value
	case label == "被捕獲率":
		n, err := strconv.Atoi(value)
		if err == nil {
			detail.Pokemon.CatchRate = n
		}
	case strings.Contains(label, "なつき"):
		n, err := strconv.Atoi(value)
		if err == nil {
			detail.Pokemon.BaseFriendship = n
		}
	case label == "基礎経験値":
		n, err := strconv.Atoi(value)
		if err == nil {
			detail.Pokemon.BaseExp = n
		}
	case label == "経験値タイプ":
		detail.Pokemon.ExpType = value
	}
}

func parseItemRow(label, value string, detail *model.PokemonDetail) {
	switch {
	case strings.Contains(label, "50%"):
		detail.Pokemon.Item50pct = value
	case strings.Contains(label, "5%"):
		detail.Pokemon.Item5pct = value
	}
}

// childTexts は指定セレクタのテキストをスライスで返す
func childTexts(e *colly.HTMLElement, selector string) []string {
	var texts []string
	e.ForEach(selector, func(_ int, el *colly.HTMLElement) {
		texts = append(texts, strings.TrimSpace(el.Text))
	})
	return texts
}

// normalizeName は名前を正規化する（全角→半角、スペース除去）
func normalizeName(s string) string {
	s = strings.TrimSpace(s)
	// 全角英数字・記号を半角に変換
	var b strings.Builder
	for _, r := range s {
		// 全角スペース→除去
		if r == '　' || r == ' ' {
			continue
		}
		// 全角英数字 (Ａ-Ｚ, ａ-ｚ, ０-９) → 半角
		if r >= 'Ａ' && r <= 'Ｚ' {
			b.WriteRune(r - 'Ａ' + 'A')
		} else if r >= 'ａ' && r <= 'ｚ' {
			b.WriteRune(r - 'ａ' + 'a')
		} else if r >= '０' && r <= '９' {
			b.WriteRune(r - '０' + '0')
		} else if unicode.IsSpace(r) {
			continue
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// resolveEvolutionName は進化テーブルのポケモン名からIDを解決する
// 1. nameMapから正規化マッチ → 2. DBから完全一致 → 3. DBから部分一致
func resolveEvolutionName(name string, nameMap map[string]int, db *sql.DB) (int, error) {
	normalized := normalizeName(name)

	// nameMapから正規化マッチ
	if id, ok := nameMap[normalized]; ok {
		return id, nil
	}

	// DBから完全一致
	id, err := mydb.GetPokemonIDByName(db, name)
	if err == nil {
		return id, nil
	}

	// DBから部分一致（LIKE）
	var partialID int
	err = db.QueryRow("SELECT id FROM pokemon WHERE name LIKE ? LIMIT 1", "%"+name+"%").Scan(&partialID)
	if err == nil {
		return partialID, nil
	}

	return 0, fmt.Errorf("pokemon %q not found", name)
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}


func parseLevelMovesTable(e *colly.HTMLElement, detail *model.PokemonDetail) {
	e.ForEach("tr", func(_ int, row *colly.HTMLElement) {
		cells := childTexts(row, "td")
		if len(cells) < 2 {
			return
		}
		levelStr := strings.TrimSpace(cells[0])
		levelStr = strings.Replace(levelStr, "Lv.", "", 1)
		levelStr = strings.TrimSpace(levelStr)
		level, err := strconv.Atoi(levelStr)
		if err != nil {
			return
		}
		moveName := strings.TrimSpace(cells[1])
		if moveName != "" {
			detail.LevelMoves = append(detail.LevelMoves, model.LearnsetLevel{
				PokemonID: detail.Pokemon.ID,
				Level:     level,
				MoveName:  moveName,
			})
		}
	})
}

func parseTMMovesTable(e *colly.HTMLElement, detail *model.PokemonDetail) {
	e.ForEach("tr", func(_ int, row *colly.HTMLElement) {
		cells := childTexts(row, "td")
		if len(cells) < 2 {
			return
		}
		tmNumber := strings.TrimSpace(cells[0])
		moveName := strings.TrimSpace(cells[1])
		if tmNumber != "" && moveName != "" {
			detail.TMMoves = append(detail.TMMoves, model.LearnsetTM{
				PokemonID: detail.Pokemon.ID,
				TMNumber:  tmNumber,
				MoveName:  moveName,
			})
		}
	})
}

func parseTutorMovesTable(e *colly.HTMLElement, detail *model.PokemonDetail) {
	e.ForEach("tr", func(_ int, row *colly.HTMLElement) {
		cells := childTexts(row, "td")
		if len(cells) < 1 {
			return
		}
		moveName := strings.TrimSpace(cells[0])
		if moveName != "" {
			detail.TutorMoves = append(detail.TutorMoves, model.LearnsetTutor{
				PokemonID: detail.Pokemon.ID,
				MoveName:  moveName,
			})
		}
	})
}

// parseEggMovesTable はタマゴ技テーブルをパースする
// 各tdに「技名\n遺伝経路」がまとめて入っている
func parseEggMovesTable(e *colly.HTMLElement, detail *model.PokemonDetail) {
	e.ForEach("tr", func(_ int, row *colly.HTMLElement) {
		cells := childTexts(row, "td")
		if len(cells) < 1 {
			return
		}
		// 技名と遺伝経路を分離
		text := cells[0]
		lines := strings.SplitN(text, "\n", 2)
		moveName := strings.TrimSpace(lines[0])
		parentChain := ""
		if len(lines) >= 2 {
			parentChain = strings.TrimSpace(lines[1])
		}
		if moveName != "" {
			detail.EggMoves = append(detail.EggMoves, model.LearnsetEgg{
				PokemonID:   detail.Pokemon.ID,
				MoveName:    moveName,
				ParentChain: parentChain,
			})
		}
	})
}

// SavePokemon はパースしたデータをDBに保存する
// moveCacheは技名→IDのメモリキャッシュ（リモートDBクエリ削減用）
// 未登録の技だけDBに登録し、残りはトランザクションでバッチINSERTする
func SavePokemon(db *sql.DB, detail *model.PokemonDetail, moveCache map[string]int) error {
	// 未登録の技名を収集してまとめてDB登録
	allMoveNames := collectMoveNames(detail)
	for _, name := range allMoveNames {
		if _, ok := moveCache[name]; !ok {
			id, err := mydb.GetOrCreateMove(db, name)
			if err != nil {
				log.Printf("  WARNING: get/create move %s: %v", name, err)
				continue
			}
			moveCache[name] = id
		}
	}

	// トランザクションで全データをバッチINSERT
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 基本情報
	if _, err := tx.Exec(`
		INSERT OR REPLACE INTO pokemon (id, name, type1, type2, ability1, ability2, image_url,
			egg_group1, egg_group2, hatch_steps, gender_ratio, catch_rate, base_friendship,
			base_exp, exp_type, item_50pct, item_5pct)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		detail.Pokemon.ID, detail.Pokemon.Name, detail.Pokemon.Type1, detail.Pokemon.Type2,
		detail.Pokemon.Ability1, detail.Pokemon.Ability2, detail.Pokemon.ImageURL,
		detail.Pokemon.EggGroup1, detail.Pokemon.EggGroup2, detail.Pokemon.HatchSteps,
		detail.Pokemon.GenderRatio, detail.Pokemon.CatchRate, detail.Pokemon.BaseFriendship,
		detail.Pokemon.BaseExp, detail.Pokemon.ExpType, detail.Pokemon.Item50pct, detail.Pokemon.Item5pct,
	); err != nil {
		return fmt.Errorf("insert pokemon: %w", err)
	}

	// 種族値
	if _, err := tx.Exec(`
		INSERT OR REPLACE INTO base_stats (pokemon_id, hp, attack, defense, sp_attack, sp_defense, speed)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		detail.Stats.PokemonID, detail.Stats.HP, detail.Stats.Attack, detail.Stats.Defense,
		detail.Stats.SpAttack, detail.Stats.SpDefense, detail.Stats.Speed,
	); err != nil {
		return fmt.Errorf("insert base_stats: %w", err)
	}

	// 努力値
	if _, err := tx.Exec(`
		INSERT OR REPLACE INTO ev_yield (pokemon_id, hp, attack, defense, sp_attack, sp_defense, speed)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		detail.EVYield.PokemonID, detail.EVYield.HP, detail.EVYield.Attack, detail.EVYield.Defense,
		detail.EVYield.SpAttack, detail.EVYield.SpDefense, detail.EVYield.Speed,
	); err != nil {
		return fmt.Errorf("insert ev_yield: %w", err)
	}

	// 入手方法
	for _, enc := range detail.Encounters {
		tx.Exec(`INSERT OR IGNORE INTO encounter (pokemon_id, location, method, note) VALUES (?, ?, ?, ?)`,
			detail.Pokemon.ID, enc.Location, enc.Method, enc.Note)
	}

	// レベルアップ技
	for _, lm := range detail.LevelMoves {
		if moveID, ok := moveCache[lm.MoveName]; ok {
			tx.Exec(`INSERT OR IGNORE INTO learnset_level (pokemon_id, level, move_id) VALUES (?, ?, ?)`,
				detail.Pokemon.ID, lm.Level, moveID)
		}
	}

	// 技マシン
	for _, tm := range detail.TMMoves {
		if moveID, ok := moveCache[tm.MoveName]; ok {
			tx.Exec(`INSERT OR IGNORE INTO learnset_tm (pokemon_id, tm_number, move_id) VALUES (?, ?, ?)`,
				detail.Pokemon.ID, tm.TMNumber, moveID)
		}
	}

	// 教え技
	for _, t := range detail.TutorMoves {
		if moveID, ok := moveCache[t.MoveName]; ok {
			tx.Exec(`INSERT OR IGNORE INTO learnset_tutor (pokemon_id, move_id) VALUES (?, ?)`,
				detail.Pokemon.ID, moveID)
		}
	}

	// タマゴ技
	for _, eg := range detail.EggMoves {
		if moveID, ok := moveCache[eg.MoveName]; ok {
			tx.Exec(`INSERT OR IGNORE INTO learnset_egg (pokemon_id, move_id, parent_chain) VALUES (?, ?, ?)`,
				detail.Pokemon.ID, moveID, eg.ParentChain)
		}
	}

	return tx.Commit()
}

// collectMoveNames は全技名をユニークに収集する
func collectMoveNames(detail *model.PokemonDetail) []string {
	seen := make(map[string]bool)
	var names []string
	add := func(name string) {
		if name != "" && !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	for _, lm := range detail.LevelMoves {
		add(lm.MoveName)
	}
	for _, tm := range detail.TMMoves {
		add(tm.MoveName)
	}
	for _, t := range detail.TutorMoves {
		add(t.MoveName)
	}
	for _, eg := range detail.EggMoves {
		add(eg.MoveName)
	}
	return names
}
