package handler

import (
	"net/http"
	"strings"
)

var blockedAIBots = []string{
	"GPTBot",
	"ChatGPT-User",
	"OAI-SearchBot",
	"ClaudeBot",
	"Claude-Web",
	"anthropic-ai",
	"Google-Extended",
	"CCBot",
	"PerplexityBot",
	"Meta-ExternalAgent",
	"Meta-ExternalFetcher",
	"Bytespider",
	"Amazonbot",
	"Applebot-Extended",
	"cohere-ai",
	"Diffbot",
	"ImagesiftBot",
	"Omgilibot",
	"Timpibot",
}

// BlockAIBotsMiddleware は User-Agent に AI crawler の名前を含むリクエストを 403 で打ち返す。
// robots.txt を無視するbotへの強制ブロック用。
func BlockAIBotsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		for _, bot := range blockedAIBots {
			if strings.Contains(ua, bot) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
