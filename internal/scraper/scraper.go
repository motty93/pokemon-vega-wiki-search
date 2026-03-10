package scraper

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

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

	for i, link := range links {
		log.Printf("[%d/%d] Scraping %s (No.%03d)...", i+1, len(links), link.Name, link.ID)

		detail, err := ScrapePokemonPage(link)
		if err != nil {
			log.Printf("  WARNING: failed to scrape %s: %v", link.Name, err)
			continue
		}

		if err := SavePokemon(db, detail); err != nil {
			log.Printf("  WARNING: failed to save %s: %v", link.Name, err)
			continue
		}

		// リクエスト間隔: 1〜2秒のランダムスリープ
		sleepDuration := time.Duration(1000+rand.Intn(1000)) * time.Millisecond
		time.Sleep(sleepDuration)
	}

	return nil
}

// FetchPokemonLinks は図鑑一覧ページからポケモンのリンクを取得する
func FetchPokemonLinks() ([]PokemonLink, error) {
	var links []PokemonLink
	c := colly.NewCollector()

	idRegex := regexp.MustCompile(`No\.(\d+)`)

	c.OnHTML("table", func(e *colly.HTMLElement) {
		e.ForEach("tr", func(_ int, row *colly.HTMLElement) {
			cells := childTexts(row, "td")
			if len(cells) < 2 {
				return
			}

			// No.XXX の形式を探す
			match := idRegex.FindStringSubmatch(cells[0])
			if match == nil {
				return
			}

			id, err := strconv.Atoi(match[1])
			if err != nil || id < 1 || id > 386 {
				return
			}

			name := strings.TrimSpace(cells[1])
			href := row.ChildAttr("td:nth-child(2) a", "href")
			if href == "" {
				href = row.ChildAttr("td a", "href")
			}
			if href == "" || name == "" {
				return
			}

			if !strings.HasPrefix(href, "http") {
				href = baseURL + href
			}

			links = append(links, PokemonLink{ID: id, Name: name, URL: href})
		})
	})

	if err := c.Visit(indexURL); err != nil {
		return nil, err
	}

	return links, nil
}

// ScrapePokemonPage は個別ポケモンページをスクレイピングする
func ScrapePokemonPage(link PokemonLink) (*model.PokemonDetail, error) {
	detail := &model.PokemonDetail{}
	detail.Pokemon.ID = link.ID
	detail.Pokemon.Name = link.Name
	detail.Stats.PokemonID = link.ID
	detail.EVYield.PokemonID = link.ID

	c := colly.NewCollector()

	// 画像URL取得
	c.OnHTML("#wikibody img", func(e *colly.HTMLElement) {
		src := e.Attr("src")
		if src != "" && detail.Pokemon.ImageURL == "" {
			if !strings.HasPrefix(src, "http") {
				src = baseURL + src
			}
			detail.Pokemon.ImageURL = src
		}
	})

	// テーブルベースでデータをパース
	c.OnHTML("#wikibody table", func(e *colly.HTMLElement) {
		headerText := e.ChildText("tr:first-child th")
		if headerText == "" {
			headerText = e.ChildText("tr:first-child td")
		}

		switch {
		case containsAny(headerText, "タイプ", "とくせい", "特性"):
			parseBasicInfoTable(e, detail)
		case containsAny(headerText, "種族値", "HP", "こうげき"):
			parseBaseStatsTable(e, detail)
		case containsAny(headerText, "努力値"):
			parseEVYieldTable(e, detail)
		case containsAny(headerText, "進化"):
			parseEvolutionTable(e, detail)
		case containsAny(headerText, "レベルアップ", "Lv."):
			parseLevelMovesTable(e, detail)
		case containsAny(headerText, "技マシン", "わざマシン"):
			parseTMMovesTable(e, detail)
		case containsAny(headerText, "教え技"):
			parseTutorMovesTable(e, detail)
		case containsAny(headerText, "タマゴ技"):
			parseEggMovesTable(e, detail)
		case containsAny(headerText, "入手方法", "入手"):
			parseEncounterTable(e, detail)
		case containsAny(headerText, "タマゴ", "たまご", "孵化", "性別"):
			parseEggDataTable(e, detail)
		}
	})

	if err := c.Visit(link.URL); err != nil {
		return nil, err
	}

	// タイプが空の場合にデフォルト設定
	if detail.Pokemon.Type1 == "" {
		detail.Pokemon.Type1 = "ノーマル"
	}

	return detail, nil
}

// childTexts は指定セレクタのテキストをスライスで返す（colly v1互換）
func childTexts(e *colly.HTMLElement, selector string) []string {
	var texts []string
	e.ForEach(selector, func(_ int, el *colly.HTMLElement) {
		texts = append(texts, strings.TrimSpace(el.Text))
	})
	return texts
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func parseBasicInfoTable(e *colly.HTMLElement, detail *model.PokemonDetail) {
	e.ForEach("tr", func(_ int, row *colly.HTMLElement) {
		header := strings.TrimSpace(row.ChildText("th"))
		value := strings.TrimSpace(row.ChildText("td"))

		switch {
		case strings.Contains(header, "タイプ"):
			types := strings.Fields(value)
			if len(types) >= 1 {
				detail.Pokemon.Type1 = types[0]
			}
			if len(types) >= 2 {
				detail.Pokemon.Type2 = types[1]
			}
		case strings.Contains(header, "とくせい") || strings.Contains(header, "特性"):
			abilities := strings.Split(value, "/")
			if len(abilities) >= 1 {
				detail.Pokemon.Ability1 = strings.TrimSpace(abilities[0])
			}
			if len(abilities) >= 2 {
				detail.Pokemon.Ability2 = strings.TrimSpace(abilities[1])
			}
		}
	})
}

func parseBaseStatsTable(e *colly.HTMLElement, detail *model.PokemonDetail) {
	e.ForEach("tr", func(_ int, row *colly.HTMLElement) {
		header := strings.TrimSpace(row.ChildText("th"))
		if header == "" {
			header = strings.TrimSpace(row.ChildText("td:first-child"))
		}
		value := strings.TrimSpace(row.ChildText("td:last-child"))
		if value == "" {
			return
		}
		n, err := strconv.Atoi(value)
		if err != nil {
			return
		}

		switch {
		case header == "HP":
			detail.Stats.HP = n
		case strings.Contains(header, "こうげき") || header == "攻撃":
			detail.Stats.Attack = n
		case strings.Contains(header, "ぼうぎょ") || header == "防御":
			detail.Stats.Defense = n
		case strings.Contains(header, "とくこう") || header == "特攻":
			detail.Stats.SpAttack = n
		case strings.Contains(header, "とくぼう") || header == "特防":
			detail.Stats.SpDefense = n
		case strings.Contains(header, "すばやさ") || header == "素早さ":
			detail.Stats.Speed = n
		}
	})
}

func parseEVYieldTable(e *colly.HTMLElement, detail *model.PokemonDetail) {
	e.ForEach("tr", func(_ int, row *colly.HTMLElement) {
		header := strings.TrimSpace(row.ChildText("th"))
		value := strings.TrimSpace(row.ChildText("td:last-child"))
		n, err := strconv.Atoi(value)
		if err != nil {
			return
		}
		switch {
		case header == "HP":
			detail.EVYield.HP = n
		case strings.Contains(header, "こうげき") || header == "攻撃":
			detail.EVYield.Attack = n
		case strings.Contains(header, "ぼうぎょ") || header == "防御":
			detail.EVYield.Defense = n
		case strings.Contains(header, "とくこう") || header == "特攻":
			detail.EVYield.SpAttack = n
		case strings.Contains(header, "とくぼう") || header == "特防":
			detail.EVYield.SpDefense = n
		case strings.Contains(header, "すばやさ") || header == "素早さ":
			detail.EVYield.Speed = n
		}
	})
}

func parseEvolutionTable(e *colly.HTMLElement, detail *model.PokemonDetail) {
	e.ForEach("tr", func(_ int, row *colly.HTMLElement) {
		cells := childTexts(row, "td")
		if len(cells) < 2 {
			return
		}
		// 進化前 → 条件 → 進化後
		ev := model.EvolutionDetail{
			FromName: strings.TrimSpace(cells[0]),
		}
		if len(cells) >= 3 {
			ev.Condition = strings.TrimSpace(cells[1])
			ev.ToName = strings.TrimSpace(cells[2])
		} else {
			ev.ToName = strings.TrimSpace(cells[1])
		}
		if ev.FromName != "" && ev.ToName != "" {
			detail.Evolutions = append(detail.Evolutions, ev)
		}
	})
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

func parseEggMovesTable(e *colly.HTMLElement, detail *model.PokemonDetail) {
	e.ForEach("tr", func(_ int, row *colly.HTMLElement) {
		cells := childTexts(row, "td")
		if len(cells) < 1 {
			return
		}
		moveName := strings.TrimSpace(cells[0])
		parentChain := ""
		if len(cells) >= 2 {
			parentChain = strings.TrimSpace(cells[1])
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

func parseEncounterTable(e *colly.HTMLElement, detail *model.PokemonDetail) {
	e.ForEach("tr", func(_ int, row *colly.HTMLElement) {
		cells := childTexts(row, "td")
		if len(cells) < 2 {
			return
		}
		location := strings.TrimSpace(cells[0])
		method := strings.TrimSpace(cells[1])
		note := ""
		if len(cells) >= 3 {
			note = strings.TrimSpace(cells[2])
		}
		if location != "" || method != "" {
			detail.Encounters = append(detail.Encounters, model.Encounter{
				PokemonID: detail.Pokemon.ID,
				Location:  location,
				Method:    method,
				Note:      note,
			})
		}
	})
}

func parseEggDataTable(e *colly.HTMLElement, detail *model.PokemonDetail) {
	e.ForEach("tr", func(_ int, row *colly.HTMLElement) {
		header := strings.TrimSpace(row.ChildText("th"))
		value := strings.TrimSpace(row.ChildText("td"))

		switch {
		case strings.Contains(header, "タマゴグループ") || strings.Contains(header, "たまごグループ"):
			groups := strings.Split(value, "/")
			if len(groups) >= 1 {
				detail.Pokemon.EggGroup1 = strings.TrimSpace(groups[0])
			}
			if len(groups) >= 2 {
				detail.Pokemon.EggGroup2 = strings.TrimSpace(groups[1])
			}
		case strings.Contains(header, "孵化歩数"):
			n, err := strconv.Atoi(strings.ReplaceAll(value, ",", ""))
			if err == nil {
				detail.Pokemon.HatchSteps = n
			}
		case strings.Contains(header, "性別"):
			detail.Pokemon.GenderRatio = value
		case strings.Contains(header, "捕獲率") || strings.Contains(header, "被捕獲"):
			n, err := strconv.Atoi(value)
			if err == nil {
				detail.Pokemon.CatchRate = n
			}
		case strings.Contains(header, "なつき") || strings.Contains(header, "初期なつき"):
			n, err := strconv.Atoi(value)
			if err == nil {
				detail.Pokemon.BaseFriendship = n
			}
		case strings.Contains(header, "基礎経験値"):
			n, err := strconv.Atoi(value)
			if err == nil {
				detail.Pokemon.BaseExp = n
			}
		case strings.Contains(header, "経験値タイプ"):
			detail.Pokemon.ExpType = value
		case strings.Contains(header, "持ち物"):
			if strings.Contains(header, "50") {
				detail.Pokemon.Item50pct = value
			} else if strings.Contains(header, "5") {
				detail.Pokemon.Item5pct = value
			}
		}
	})
}

// SavePokemon はパースしたデータをDBに保存する
func SavePokemon(db *sql.DB, detail *model.PokemonDetail) error {
	if err := mydb.InsertPokemon(db, &detail.Pokemon); err != nil {
		return fmt.Errorf("insert pokemon: %w", err)
	}

	if err := mydb.InsertBaseStats(db, &detail.Stats); err != nil {
		return fmt.Errorf("insert base_stats: %w", err)
	}

	if err := mydb.InsertEVYield(db, &detail.EVYield); err != nil {
		return fmt.Errorf("insert ev_yield: %w", err)
	}

	// 入手方法
	for _, enc := range detail.Encounters {
		if err := mydb.InsertEncounter(db, detail.Pokemon.ID, enc.Location, enc.Method, enc.Note); err != nil {
			log.Printf("  WARNING: insert encounter: %v", err)
		}
	}

	// レベルアップ技
	for _, lm := range detail.LevelMoves {
		moveID, err := mydb.GetOrCreateMove(db, lm.MoveName)
		if err != nil {
			log.Printf("  WARNING: get/create move %s: %v", lm.MoveName, err)
			continue
		}
		mydb.InsertLearnsetLevel(db, detail.Pokemon.ID, lm.Level, moveID)
	}

	// 技マシン
	for _, tm := range detail.TMMoves {
		moveID, err := mydb.GetOrCreateMove(db, tm.MoveName)
		if err != nil {
			continue
		}
		mydb.InsertLearnsetTM(db, detail.Pokemon.ID, tm.TMNumber, moveID)
	}

	// 教え技
	for _, t := range detail.TutorMoves {
		moveID, err := mydb.GetOrCreateMove(db, t.MoveName)
		if err != nil {
			continue
		}
		mydb.InsertLearnsetTutor(db, detail.Pokemon.ID, moveID)
	}

	// タマゴ技
	for _, eg := range detail.EggMoves {
		moveID, err := mydb.GetOrCreateMove(db, eg.MoveName)
		if err != nil {
			continue
		}
		mydb.InsertLearnsetEgg(db, detail.Pokemon.ID, moveID, eg.ParentChain)
	}

	return nil
}
