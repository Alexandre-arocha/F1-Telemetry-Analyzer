YEAR ?= 2024
ROUND ?= 1
GO ?= go

.PHONY: up down ingest ingest-season ingest-local dev-api dev-web test test-api test-ingest

up:
	docker compose up -d --build

down:
	docker compose down

# One race, one command: make ingest YEAR=2024 ROUND=5 (stack must be up, API applies the migrations)
ingest:
	docker compose --profile ingest run --rm ingest --year $(YEAR) --round $(ROUND)

# Same without Docker: needs `pip install -r ingest/requirements.txt` and DATABASE_URL
ingest-local:
	python -m ingest --year $(YEAR) --round $(ROUND)

dev-api:
	cd api && $(GO) run ./cmd/server

dev-web:
	cd web && npm run dev

# Whole season (about 8 minutes per race, OpenF1 throttles): make ingest-season YEAR=2024
ingest-season:
	docker compose --profile ingest run --rm ingest --year $(YEAR) --all

test: test-api test-ingest

test-api:
	cd api && $(GO) test ./...

test-ingest:
	python -m pytest ingest/tests
