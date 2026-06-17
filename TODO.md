# Hackathon TODO

## Phase 1: Design

- [x] UI/UX: define the main screens and user flow.
- [x] DB: define tables, columns, relations, and indexes.
- [x] API: define endpoints, request JSON, and response JSON.

## Phase 2: Foundation

- [x] Backend: create a Go API server.
- [x] Frontend: create a React app.
- [x] DB: prepare local MySQL schema.
- [x] Infra: prepare Cloud Run, Vercel, and Cloud SQL deployment notes.

## Phase 3: Core Features

- [x] User registration and login.
- [x] Item listing and item detail.
- [x] Item creation.
- [x] Purchase flow without payment integration.
- [x] Direct messages between users.
- [x] Likes.
- [x] OpenAI API integration for AI item description and Q&A.

## Phase 4: Polish

- [x] Improve UI/UX for demo flow.
- [ ] Add basic tests.
- [x] Add README startup steps.
- [ ] Prepare demo script.

## Work Log

### 2026-06-17

- Created design documents in `docs/`.
- Added `docs/ui-flow.md` for main screens, user flow, and demo flow.
- Added `docs/db-design.md` for tables, relations, and indexes.
- Added `docs/api-spec.md` for auth, items, purchases, messages, and AI endpoints.
- Created Go backend in `backend/`.
- Added MySQL migration at `backend/migrations/001_init.sql`.
- Implemented REST APIs for user registration, login, items, likes, purchases, conversations, messages, and OpenAI calls.
- Added `backend/Dockerfile` for Cloud Run deployment.
- Added `backend/.env.example` for local and Cloud Run environment variables.
- Created React frontend in `frontend/`.
- Implemented login/register, item list, item detail, item creation, AI description generation, AI Q&A, purchase, like, and DM UI.
- Added local MySQL setup in `docker-compose.yaml`.
- Updated `README.md` with local startup steps and deployment direction for Cloud Run, Vercel, and Cloud SQL.
- Verified backend with `GOMODCACHE=/private/tmp/gomodcache GOCACHE=/private/tmp/gocache go test ./...`.
- Verified frontend with `npm run build`.
- Verified production dependency audit with `npm audit --omit=dev`.
- Started local MySQL, backend, and frontend dev server.
- Confirmed `GET /api/healthz` returns `{"status":"ok"}`.
- Confirmed `POST /api/auth/register` creates a user in MySQL.

## Remaining Notes

- `OPENAI_API_KEY` must be set before AI generation works in the running backend.
- Basic tests are still TODO.
- Demo script is still TODO.
