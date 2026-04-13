include .env
export

.PHONY: build dev test clean deploy

# ビルド
build:
	CGO_ENABLED=0 go build -o bin/server ./cmd/server

# ローカル実行（Air: ホットリロード）
dev:
	air

# テスト
test:
	go test ./...

# クリーン
clean:
	rm -rf bin/

# Cloud Runへデプロイ（Cloud Build経由）
deploy:
	gcloud builds submit --config=cloudbuild.yml
