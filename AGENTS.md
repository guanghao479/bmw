# Repository Guidelines

## Project Structure & Modules
- `app/`: Static frontend (HTML/CSS/JS) reading data from S3.
- `backend/`: Go services and CLIs under `cmd/` (e.g., `admin_api`, `lambda`, `scraper`, `source_analyzer`, `scraping_orchestrator`), shared code in `internal/`, env template `.env.example`.
- `infrastructure/`: AWS CDK (TypeScript) for deployment.
- `testing/`: Ignored build artifacts, temp data, logs, and Go test helpers.
- `docs/` and `.github/workflows/`: Operational docs and CI workflows.

## Build, Test, and Dev Commands
- Backend build: `cd backend && go build ./...` (binaries from `cmd/...`). Example: `go build -o ../testing/bin/admin_api ./cmd/admin_api`.
- Backend run: `cd backend && go run ./cmd/admin_api` (set env from `.env`).
- Unit tests: `cd backend && go test ./... -v`.
- Integration tests: `cd backend && go test -tags=integration ./internal/services -v` (see `backend/INTEGRATION_TESTS.md`; requires `OPENAI_API_KEY`).
- Lambda build (CI example): `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap cmd/lambda/main.go`.
- Infra (CDK): `cd infrastructure && npm ci && npm run build && npm run synth`.

## Coding Style & Naming
- Go: standard `gofmt`/`go fmt` formatting; packages lower_snake, commands under `cmd/<name>`; exported names use PascalCase.
- TypeScript (infra): `tsc` clean build; prefer camelCase for variables, PascalCase for types.
- Frontend JS/CSS: 2‑space indent; camelCase for variables; keep files small and feature‑oriented (`app/script.js`, `app/admin.js`).

## Testing Guidelines
- Framework: Go `testing` with optional `-tags=integration` for live API tests.
- Artifacts: write binaries/logs under `testing/` (e.g., `testing/bin`, `testing/logs`).
- Naming: `*_test.go`; table‑driven tests preferred; keep tests deterministic unless explicitly marked integration.
- Coverage: prioritize core services in `internal/` and `cmd/lambda`.

## Commit & Pull Requests
- Commits: imperative, concise subject (≤72 chars), scoped when helpful (e.g., "backend: fix OpenAI parsing"). History favors "Add/Implement/Fix/Update" verbs.
- PRs: clear description, linked issues, CI green. Include screenshots for `app/` changes and sample payloads for backend/API changes. Note any env/config changes.

## Security & Config
- Never commit secrets; use `.env` from `.env.example` and local exports. `.envrc` is provided for direnv users. Validate S3 paths and AWS roles via CDK before deploy.
