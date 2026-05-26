# Pulse — Project Guide

## What is Pulse?

A **person-to-person social platform** where content is the *medium* for discovering real human affinity — not the end goal. Unlike Instagram/TikTok where algorithms optimize for engagement, Pulse builds a **dynamic affinity graph** between people based on shared aesthetic, mood, and interests.

**Core philosophy**: "Connect through moments, not likes."

### Key differentiators from Instagram/TikTok

- **No likes** — replaced by **semantic reactions** (gave_me_energy, calmed_me, on_repeat, surprised_me, my_aesthetic)
- **No algorithmic feed manipulation** — feed ranked by affinity + recency + diversity
- **Bridges** — every suggestion explains *why* ("You both post about #synthwave and #noir")
- **Mood Rooms** — temporary spaces clustered by shared tags, real-time co-consumption
- **Paths** — curated content journeys (mixed media), not infinite scroll
- **Tags are user-defined hashtags** — anyone creates tags by using them (like Instagram hashtags, NOT a fixed curated set)
- **Multi-content** — images, videos, short videos, AND text posts

### Who is it for?

**Everyone** — photography is the beachhead niche for go-to-market, but the platform is general-purpose social media. The plan expands niche-by-niche: photographers → videographers → illustrators → musicians → general creative community.

---

## Architecture Overview

| Codename | What | Tech | Port |
|----------|------|------|------|
| **stone** | Go API backend | Go + Gin + GORM + PostgreSQL + Redis + WebSocket | :8080 |
| **ember** | React PWA (main app) | React + Vite + TypeScript + Tailwind v4 + Zustand | :5173 |
| **grove** | Landing page | React + Vite + TypeScript + Tailwind v4 | :5174 |
| **drift** | Shared TS types | TypeScript package consumed by ember & grove | — |

### Running locally

```bash
# Backend (requires Postgres + Redis running)
make dev-stone    # go run ./cmd/server on :8080

# Frontend
make dev-ember    # vite dev on :5173 (proxies /api + /ws to :8080)
make dev-grove    # vite dev on :5174

# Database
make migrate      # run versioned SQL migrations
make seed         # seed 10 users, 100 mixed content, tags, follows, rooms, paths
```

### Environment

Copy `stone/.env.example` → `stone/.env` and configure:
- `DATABASE_URL` — Postgres connection string
- `REDIS_URL` — Redis connection string
- `JWT_SECRET` — change for production

---

## Data Model (Key Concepts)

### Tags = User-defined hashtags
Tags are **NOT** a fixed curated set. Users create them by typing hashtags when posting. The `TagService.FindOrCreateByNames()` method normalizes (lowercase, strip #) and creates on first use. Tags have a `usage_count` for trending/autocomplete.

### Content types
The `contents` table has a `content_type` field: `image`, `video`, `short_video`, `text`. Text posts have no `media_url`. The `body` field serves as caption (for media) or main text (for text posts).

### Semantic reactions (NOT likes)
The `reactions` table replaces likes. Each reaction has a `kind`: `gave_me_energy`, `calmed_me`, `on_repeat`, `surprised_me`, `my_aesthetic`. Users can add one reaction per kind per content. These map to mood vectors for affinity calculation.

### Affinity = tag overlap SQL
MVP uses raw SQL: count shared tags between users' content, exclude followed/blocked, order by overlap count DESC. Each suggestion includes a **Bridge** — a human-readable sentence explaining why.

### Rooms = mood clusters
Rooms are created by hashing sorted tag IDs + date bucket (YYYY-MM-DD) into a `cluster_key`. Same tags + same day = same room. Rooms expire after 24h. WebSocket broadcasts presence changes.

### Paths = curated content journeys
Ordered sequences of mixed-type content items with creator notes. Users follow paths (not necessarily the creator). Following 2-3 paths from the same person → suggest connection.

---

## API Endpoints (stone)

### Public
- `POST /api/v1/auth/register` — register (handle, email, password, display_name)
- `POST /api/v1/auth/login` — login → JWT access + refresh tokens
- `POST /api/v1/auth/refresh` — refresh tokens
- `GET /api/v1/tags` — list popular tags (for autocomplete)
- `GET /api/v1/tags/search?q=` — search tags by prefix
- `GET /api/v1/health` — health check (postgres + redis status)

### Protected (JWT Bearer)
- `GET /api/v1/me`, `PUT /api/v1/me` — current user profile
- `GET /api/v1/users/:id` — user profile with follower/following counts + is_following/is_blocked
- `POST /api/v1/content` — create content (multipart: file + content_type + body + tags)
- `GET /api/v1/content/:id`, `DELETE /api/v1/content/:id`
- `POST /api/v1/content/:id/react` — add semantic reaction `{kind}`
- `DELETE /api/v1/content/:id/react?kind=` — remove reaction
- `GET /api/v1/feed?cursor=&limit=` — cursor-paginated feed
- `GET /api/v1/suggestions` — affinity-based user suggestions with bridges
- `POST /api/v1/follow/:id`, `DELETE /api/v1/follow/:id`
- `POST /api/v1/block/:id`, `DELETE /api/v1/block/:id`
- `GET /api/v1/rooms` — list active mood rooms
- `POST /api/v1/rooms/:id/enter`, `POST /api/v1/rooms/:id/leave`
- `POST /api/v1/paths`, `GET /api/v1/paths`, `GET /api/v1/paths/:id`
- `POST /api/v1/paths/:id/follow`
- `POST /api/v1/events` — batch event recording

### WebSocket
- `GET /ws` + subprotocol auth (`["bearer", "<access_jwt>"]`) — real-time connection, client sends `join_room`/`leave_room`, server broadcasts `user_joined`/`user_left`/`room_presence`

---

## Current Progress

### Done
- [x] Repo skeleton: .gitignore, .editorconfig, Makefile
- [x] **stone**: Full Go backend compiles cleanly — all models, services, handlers, middleware, WebSocket hub/client, migrations, seed command
- [x] **ember**: Full React app scaffold — 11 pages, layout/auth/feed/social/room/path components, Zustand stores, API clients, WebSocket hook
- [x] **grove**: Landing page with Hero, Features, CTA, Navbar, Footer
- [x] **drift**: Shared types and constants
- [x] Data model corrected: user-defined tags (hashtags), multi-content types, semantic reactions, bridges in suggestions

### Needs work
- [x] **FeedCard component** supports multi-content types (image/video/short video/text)
- [x] **PhotoModal** was replaced by **ContentModal** with video playback and text display
- [x] **SuggestionCard** displays the `bridge` field
- [x] Removed stale **TagSelector** component
- [x] **Frontend TypeScript/build check** passes via `npm run build`
- [x] **Grove landing page** messaging updated to "social platform" (not photographer-only)
- [x] **Content grid/gallery** added on profile pages
- [x] **Reaction UI** added to feed cards and modal
- [x] **Trending tags** displayed in the app feed
- [x] **Path builder** now includes content picker + ordering + notes
- [ ] **Frontend Vitest coverage** is still missing (basic Go unit tests were added)

---

## Key Design Decisions

| Decision | Choice | Why |
|----------|--------|-----|
| Tags | User-defined hashtags (find-or-create) | Like Instagram — users define the taxonomy, not us |
| Content types | image, video, short_video, text | General social platform, not photo-only |
| Feedback | Semantic reactions (5 kinds) | No likes/dislikes — richer signal for affinity, no vanity metrics |
| Suggestions | Tag-overlap SQL + Bridge explanation | MVP-simple, no ML needed; bridges reduce creepiness |
| Backend | Single Go binary, internal pkg boundaries | MVP speed; split to microservices later |
| ORM | GORM + raw SQL for affinity query | Fast CRUD; complex queries stay readable |
| Config | envconfig (env-var only) | Simpler than viper |
| File storage | Local filesystem behind `Storage` interface | Swap to S3 by implementing `S3Storage` |
| WS auth | JWT in query param | Browser WS can't set headers; mitigated by TLS + short-lived tokens |
| Pagination | Cursor-based | Consistent performance regardless of dataset size |
| Frontend state | Zustand | Lightweight, no boilerplate vs Redux |

---

## Docs Reference

Read these for full context:
- `docs/ideas.md` — **THE vision document** (Spanish). Affinity graph, mood rooms, paths, bridges, semantic reactions, session twins, decay, co-curation. This is the north star.
- `docs/business.md` — Go-to-market, revenue model (ethical ads), unit economics, seed pitch
- `docs/tech-stack.md` — Full technical spec, MVP scope, schema, affinity SQL query, WebSocket protocol
- `docs/technical.md` — Enterprise-scale architecture (post-MVP reference, Spanish)
- `docs/srs/` — Detailed SRS documents for each sprint

---

## Common Gotchas

- **Tag model has NO `namespace` field** — it was removed. Tags are flat user-defined hashtags with `name` + `usage_count`. If you see old code referencing `tag.namespace`, it's stale.
- **Content model has NO `like_count`** — replaced by the `reactions` table. If you see `like_count` anywhere, it's stale.
- **Content model has `body` not `caption`** — `body` serves dual purpose: text content for text posts, caption for media posts.
- **ContentService.Create** takes `tagNames []string` (hashtag strings), NOT tag UUIDs. The service calls `TagService.FindOrCreateByNames()` internally.
- **Ember imports**: stores are `useAuthStore`, `useUiStore`, `useWsStore` (camelCase, not ALLCAPS).
- **Drift types use `import type`** — the ember project has `verbatimModuleSyntax: true`, so type-only imports MUST use `import type`.
- **Tailwind v4** — uses `@import "tailwindcss"` in CSS, NOT `@tailwind` directives. Config via `@tailwindcss/vite` plugin, no `tailwind.config.ts` needed.
