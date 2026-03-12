include .env
export

.PHONY: build-server build-scraper dev scraper test clean

# ビルド
build-server:
	CGO_ENABLED=0 go build -o bin/server ./cmd/server

build-scraper:
	CGO_ENABLED=0 go build -o bin/scraper ./cmd/scraper

build: build-server build-scraper

# ローカル実行（Air: ホットリロード）
dev:
	air

scraper:
	go run ./cmd/scraper

# テスト
test:
	go test ./...

# クリーン
clean:
	rm -rf bin/
