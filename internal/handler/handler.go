package handler

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

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

// Handler はHTTPハンドラー
type Handler struct {
	DB        *sql.DB
	Templates *template.Template
}

// New はHandlerを作成する
func New(db *sql.DB) (*Handler, error) {
	funcMap := template.FuncMap{
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
			// 最大値255で正規化
			pct := val * 100 / 255
			if pct > 100 {
				pct = 100
			}
			return pct
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

	tmpl, err := template.New("").Funcs(funcMap).ParseGlob("templates/*.html")
	if err != nil {
		return nil, err
	}
	return &Handler{DB: db, Templates: tmpl}, nil
}

// Index はトップページを表示する
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	types, _ := mydb.GetAllTypes(h.DB)
	data := map[string]interface{}{
		"Types": types,
	}
	h.Templates.ExecuteTemplate(w, "index.html", data)
}

// PokemonDetail はポケモン詳細ページを表示する
func (h *Handler) PokemonDetail(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
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

	h.Templates.ExecuteTemplate(w, "pokemon_detail.html", detail)
}

// Search はhtmx部分レスポンスで検索結果を返す
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	params := model.SearchParams{
		Query: r.URL.Query().Get("q"),
		Type:  r.URL.Query().Get("type"),
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

	data := map[string]interface{}{
		"Results": results,
		"Query":   params.Query,
	}
	h.Templates.ExecuteTemplate(w, "search_results.html", data)
}
