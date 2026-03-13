package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	mydb "github.com/motty93/pokemon-vega-wiki-crawler/internal/db"
	"github.com/motty93/pokemon-vega-wiki-crawler/internal/model"
)

var typeColors = map[string]string{
	"ノーマル": "#A8A878", "ほのお": "#F08030", "みず": "#6890F0", "でんき": "#F8D030",
	"くさ": "#78C850", "こおり": "#98D8D8", "かくとう": "#C03028", "どく": "#A040A0",
	"じめん": "#E0C068", "ひこう": "#A890F0", "エスパー": "#F85888", "むし": "#A8B820",
	"いわ": "#B8A038", "ゴースト": "#705898", "ドラゴン": "#7038F8", "あく": "#705848",
	"はがね": "#B8B8D0", "フェアリー": "#EE99AC",
}

// typeEntry はタイプの正式名と検索用の全表記を保持する
type typeEntry struct {
	Name    string   // DB上の正式名（例: "どく", "ドラゴン"）
	Aliases []string // 検索用の全表記（ひらがな/カタカナ/ローマ字/漢字/英語）
}

var typeEntries = []typeEntry{
	{"ノーマル", []string{"のーまる", "ノーマル", "normal", "no-maru"}},
	{"ほのお", []string{"ほのお", "ホノオ", "honoo", "fire", "炎", "火"}},
	{"みず", []string{"みず", "ミズ", "mizu", "water", "水"}},
	{"でんき", []string{"でんき", "デンキ", "denki", "electric", "電気", "電", "雷"}},
	{"くさ", []string{"くさ", "クサ", "kusa", "grass", "草"}},
	{"こおり", []string{"こおり", "コオリ", "koori", "ice", "氷"}},
	{"かくとう", []string{"かくとう", "カクトウ", "kakutou", "fighting", "格闘"}},
	{"どく", []string{"どく", "ドク", "doku", "poison", "毒"}},
	{"じめん", []string{"じめん", "ジメン", "jimen", "ground", "地面"}},
	{"ひこう", []string{"ひこう", "ヒコウ", "hikou", "flying", "飛行"}},
	{"エスパー", []string{"えすぱー", "エスパー", "esupaa", "esper", "psychic", "超", "超能力"}},
	{"むし", []string{"むし", "ムシ", "mushi", "bug", "虫"}},
	{"いわ", []string{"いわ", "イワ", "iwa", "rock", "岩"}},
	{"ゴースト", []string{"ごーすと", "ゴースト", "goosuto", "ghost", "霊", "幽霊"}},
	{"ドラゴン", []string{"どらごん", "ドラゴン", "doragon", "dragon", "竜", "龍"}},
	{"あく", []string{"あく", "アク", "aku", "dark", "悪"}},
	{"はがね", []string{"はがね", "ハガネ", "hagane", "steel", "鋼"}},
	{"フェアリー", []string{"ふぇありー", "フェアリー", "fearii", "fairy", "妖精"}},
}

// matchTypesByPrefix はクエリ文字列に前方一致するタイプ名を返す
func matchTypesByPrefix(q string) []string {
	if q == "" {
		return nil
	}
	lower := strings.ToLower(q)
	seen := make(map[string]bool)
	var matched []string
	for _, entry := range typeEntries {
		for _, alias := range entry.Aliases {
			if strings.HasPrefix(strings.ToLower(alias), lower) {
				if !seen[entry.Name] {
					seen[entry.Name] = true
					matched = append(matched, entry.Name)
				}
				break
			}
		}
	}
	return matched
}

// normalizeType はタイプ入力を正式名に変換する（完全一致）
func normalizeType(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return q
	}
	lower := strings.ToLower(q)
	for _, entry := range typeEntries {
		for _, alias := range entry.Aliases {
			if strings.ToLower(alias) == lower {
				return entry.Name
			}
		}
	}
	return q
}

// Handler はHTTPハンドラー
type Handler struct {
	DB        *sql.DB
	Templates map[string]*template.Template
	BaseURL   string
}

func getBaseURL() string {
	if u := os.Getenv("BASE_URL"); u != "" {
		return u
	}
	return "https://pokemon-vega.example.com"
}

func newFuncMap() template.FuncMap {
	return template.FuncMap{
		"seq": func(n int) []int {
			s := make([]int, n)
			for i := range s {
				s[i] = i
			}
			return s
		},
		"add": func(a, b int) int { return a + b },
		"printf": func(format string, args ...interface{}) string {
			return fmt.Sprintf(format, args...)
		},
		"typeColor": func(t string) string {
			if c, ok := typeColors[t]; ok {
				return c
			}
			return "#68A090"
		},
		"statMap": func(s model.BaseStats) [][]interface{} {
			return [][]interface{}{
				{"HP", s.HP},
				{"攻撃", s.Attack},
				{"防御", s.Defense},
				{"特攻", s.SpAttack},
				{"特防", s.SpDefense},
				{"素早さ", s.Speed},
			}
		},
		"statPercent": func(val int) int {
			pct := val * 100 / 255
			if pct > 100 {
				pct = 100
			}
			return pct
		},
		"statColor": func(name string) string {
			colors := map[string]string{
				"HP": "#ef4444", "攻撃": "#f97316", "防御": "#eab308",
				"特攻": "#3b82f6", "特防": "#22c55e", "素早さ": "#ec4899",
			}
			if c, ok := colors[name]; ok {
				return c
			}
			return "#6b7280"
		},
		"evHasAny": func(e model.EVYield) bool {
			return e.HP > 0 || e.Attack > 0 || e.Defense > 0 ||
				e.SpAttack > 0 || e.SpDefense > 0 || e.Speed > 0
		},
		"slice": func(items ...interface{}) []interface{} {
			return items
		},
		"gt": func(a, b int) bool { return a > b },
		"lt": func(a, b int) bool { return a < b },
	}
}

// New はHandlerを作成する
func New(db *sql.DB) (*Handler, error) {
	funcMap := newFuncMap()
	templates := make(map[string]*template.Template)

	// base.htmlをベースとするページテンプレート（Clone + ParseFilesで分離）
	base, err := template.New("").Funcs(funcMap).ParseFiles("templates/base.html")
	if err != nil {
		return nil, fmt.Errorf("parse base.html: %w", err)
	}

	for _, page := range []string{"index.html", "pokemon_detail.html"} {
		t, err := template.Must(base.Clone()).ParseFiles("templates/" + page)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", page, err)
		}
		templates[page] = t
	}

	// パーシャルテンプレート（base不要）
	searchResults, err := template.New("").Funcs(funcMap).ParseFiles("templates/search_results.html")
	if err != nil {
		return nil, fmt.Errorf("parse search_results.html: %w", err)
	}
	templates["search_results.html"] = searchResults

	return &Handler{DB: db, Templates: templates, BaseURL: getBaseURL()}, nil
}

// Index はトップページを表示する（DB不要で高速レスポンス）
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"CanonicalURL": h.BaseURL + "/",
	}
	h.Templates["index.html"].ExecuteTemplate(w, "base", data)
}

// PokemonDetail はポケモン詳細ページを表示する
func (h *Handler) PokemonDetail(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid pokemon ID", http.StatusBadRequest)
		return
	}

	detail, err := mydb.GetPokemonByID(h.DB, id)
	if err != nil {
		http.Error(w, "Pokemon not found", http.StatusNotFound)
		return
	}

	go mydb.LogPageView(h.DB, id)

	p := detail.Pokemon
	typeText := p.Type1
	if p.Type2 != "" {
		typeText += "・" + p.Type2
	}

	pageTitle := fmt.Sprintf("%s (No.%03d) - ポケモンベガ図鑑 | 種族値・技・入手方法", p.Name, p.ID)
	pageDesc := fmt.Sprintf("ポケモンベガの%s（No.%03d）の種族値、習得技、入手方法、進化情報。タイプ: %s。", p.Name, p.ID, typeText)
	canonicalURL := fmt.Sprintf("%s/pokemon/%d", h.BaseURL, p.ID)

	jsonLD := map[string]interface{}{
		"@context":    "https://schema.org",
		"@type":       "WebPage",
		"name":        fmt.Sprintf("%s - ポケモンベガ図鑑", p.Name),
		"description": pageDesc,
		"url":         canonicalURL,
		"breadcrumb": map[string]interface{}{
			"@type": "BreadcrumbList",
			"itemListElement": []map[string]interface{}{
				{
					"@type":    "ListItem",
					"position": 1,
					"name":     "トップ",
					"item":     h.BaseURL + "/",
				},
				{
					"@type":    "ListItem",
					"position": 2,
					"name":     fmt.Sprintf("%s (No.%03d)", p.Name, p.ID),
					"item":     canonicalURL,
				},
			},
		},
	}
	// og:image枠（将来的にOGP画像を設定）
	// ogImage := fmt.Sprintf("%s/og/pokemon/%d.png", h.BaseURL, p.ID)

	jsonLDBytes, _ := json.Marshal(jsonLD)

	data := map[string]interface{}{
		"Pokemon":         detail.Pokemon,
		"Stats":           detail.Stats,
		"EVYield":         detail.EVYield,
		"Evolutions":      detail.Evolutions,
		"Encounters":      detail.Encounters,
		"LevelMoves":      detail.LevelMoves,
		"TMMoves":         detail.TMMoves,
		"TutorMoves":      detail.TutorMoves,
		"EggMoves":        detail.EggMoves,
		"PageTitle":       pageTitle,
		"PageDescription": pageDesc,
		"CanonicalURL":    canonicalURL,
		"OGType":          "article",
		// "OGImage":      ogImage,
		"JSONLD": template.JS(string(jsonLDBytes)),
	}
	h.Templates["pokemon_detail.html"].ExecuteTemplate(w, "base", data)
}

// Search はhtmx部分レスポンスで検索結果を返す
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	rawQuery := normalizeFullWidth(strings.TrimSpace(r.URL.Query().Get("q")))
	params := model.SearchParams{
		Query:         rawQuery,
		RomajiQuery:   romajiToKatakana(rawQuery),
		KatakanaQuery: hiraganaToKatakana(rawQuery),
		Type:          normalizeType(normalizeFullWidth(r.URL.Query().Get("type"))),
		MatchedTypes:  matchTypesByPrefix(rawQuery),
	}

	params.HPMin, _ = strconv.Atoi(r.URL.Query().Get("hp_min"))
	params.HPMax, _ = strconv.Atoi(r.URL.Query().Get("hp_max"))
	params.AttackMin, _ = strconv.Atoi(r.URL.Query().Get("attack_min"))
	params.AttackMax, _ = strconv.Atoi(r.URL.Query().Get("attack_max"))
	params.DefenseMin, _ = strconv.Atoi(r.URL.Query().Get("defense_min"))
	params.DefenseMax, _ = strconv.Atoi(r.URL.Query().Get("defense_max"))
	params.SpAtkMin, _ = strconv.Atoi(r.URL.Query().Get("sp_atk_min"))
	params.SpAtkMax, _ = strconv.Atoi(r.URL.Query().Get("sp_atk_max"))
	params.SpDefMin, _ = strconv.Atoi(r.URL.Query().Get("sp_def_min"))
	params.SpDefMax, _ = strconv.Atoi(r.URL.Query().Get("sp_def_max"))
	params.SpeedMin, _ = strconv.Atoi(r.URL.Query().Get("speed_min"))
	params.SpeedMax, _ = strconv.Atoi(r.URL.Query().Get("speed_max"))

	results, err := mydb.SearchPokemon(h.DB, params)
	if err != nil {
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	if params.Query != "" {
		go mydb.LogSearch(h.DB, params.Query, len(results))
	}

	// 最も多いタイプを集計してカラーを決定
	dominantColor := ""
	if len(results) > 0 {
		typeCounts := make(map[string]int)
		for _, r := range results {
			typeCounts[r.Pokemon.Type1]++
			if r.Pokemon.Type2 != "" {
				typeCounts[r.Pokemon.Type2]++
			}
		}
		maxCount := 0
		dominantType := ""
		for t, c := range typeCounts {
			if c > maxCount {
				maxCount = c
				dominantType = t
			}
		}
		if c, ok := typeColors[dominantType]; ok {
			dominantColor = c
		}
	}

	data := map[string]interface{}{
		"Results":       results,
		"Query":         params.Query,
		"DominantColor": dominantColor,
	}
	h.Templates["search_results.html"].ExecuteTemplate(w, "search_results", data)
}

// Sitemap は /sitemap.xml を返す
func (h *Handler) Sitemap(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")

	now := time.Now().Format("2006-01-02")
	fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>`)
	fmt.Fprint(w, `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)

	fmt.Fprintf(w, `<url><loc>%s/</loc><lastmod>%s</lastmod><changefreq>weekly</changefreq><priority>1.0</priority></url>`, h.BaseURL, now)

	for i := 1; i <= 386; i++ {
		fmt.Fprintf(w, `<url><loc>%s/pokemon/%d</loc><lastmod>%s</lastmod><changefreq>monthly</changefreq><priority>0.8</priority></url>`, h.BaseURL, i, now)
	}

	fmt.Fprint(w, `</urlset>`)
}

// RobotsTxt は /robots.txt を返す
func (h *Handler) RobotsTxt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	fmt.Fprintf(w, "User-agent: *\nAllow: /\nDisallow: /search\n\nSitemap: %s/sitemap.xml\n", h.BaseURL)
}
