include .env
export

.PHONY: build dev test clean deploy setup-gcp

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

# GCP初期セットアップ（初回のみ）
setup-gcp:
	./scripts/setup-gcp.sh

# Cloud Runへデプロイ（Cloud Build経由）
deploy:
	gcloud builds submit --config=cloudbuild.yml \
		--substitutions=SHORT_SHA=$(shell git rev-parse --short HEAD)
