<div align="center">
  <h1>🔮 Pulse</h1>
  <p><strong>A person-to-person social platform where content is the medium for discovering real human affinity.</strong></p>
  <p>
    <a href="#features">Features</a> •
    <a href="#quick-start">Quick Start</a> •
    <a href="#architecture">Architecture</a> •
    <a href="#philosophy">Philosophy</a>
  </p>
</div>

---

## What is Pulse?

Pulse is a **social platform** built on a different premise than Instagram or TikTok: instead of optimizing for engagement and likes, it builds a **dynamic affinity graph** between people based on shared aesthetic, mood, and interests.

**Core idea:** "Connect through moments, not likes."

### How it works

1. **Share moments** — post images, videos, or thoughts
2. **React semantically** — instead of likes, you express how content makes you *feel*: `gave_me_energy`, `calmed_me`, `on_repeat`, `surprised_me`, `my_aesthetic`
3. **Discover affinity** — the algorithm finds people who share your taste, not just content that keeps you scrolling
4. **Join mood rooms** — temporary spaces cluster around shared tags in real-time
5. **Follow curated paths** — explore mixed-media journeys created by the community

---

## Features

<table>
  <tr>
    <td width="50%">
      <h4>✨ Semantic Reactions</h4>
      No likes. Instead, express how content makes you feel with 5 reaction types. This creates a richer signal for affinity.
    </td>
    <td width="50%">
      <h4>🌊 Affinity Graph</h4>
      Feed ranked by tag overlap + recency + diversity. Algorithm explains <em>why</em> you're seeing each suggestion.
    </td>
  </tr>
  <tr>
    <td>
      <h4>🎨 Multi-content</h4>
      Post images, videos, short-form video, or text. One platform for all creative media.
    </td>
    <td>
      <h4>🏠 Mood Rooms</h4>
      Real-time temporary spaces clustered by shared tags. See who else is exploring the same aesthetic right now.
    </td>
  </tr>
  <tr>
    <td>
      <h4>🗺️ Paths</h4>
      Curated content journeys (mixed media). Like a playlist, but visual and cross-media.
    </td>
    <td>
      <h4>🏷️ User-defined Tags</h4>
      Anyone creates tags by using them (like Instagram hashtags). The community defines the taxonomy.
    </td>
  </tr>
</table>

---

## Key Differences from Instagram/TikTok

| | Instagram/TikTok | Pulse |
|---|---|---|
| **Feedback** | Likes/engagement | Semantic reactions (5 types) |
| **Feed algorithm** | Maximizes watch time | Finds people with shared taste |
| **Suggestions** | "People you follow" | "You both post about #synthwave and #noir" |
| **Spaces** | Infinite scroll feed | Curated paths + mood rooms |
| **Metrics** | Like counts, follower counts | None (removed vanity metrics) |

---

## Quick Start

### Requirements
- PostgreSQL 14+
- Redis
- Node.js 18+
- Go 1.21+ (or Nix)

### Setup (5 minutes)

```bash
# 1. Clone and enter
git clone https://github.com/yourusername/pulse.git
cd pulse

# 2. Configure environment
cp stone/.env.example stone/.env
# Edit stone/.env with your DATABASE_URL, REDIS_URL, JWT_SECRET

# 3. Prepare demo data and start all servers
make demo
```

**That's it.** Open http://localhost:5173 and log in with:
- **Handle:** `lunanova` (or any of 10 demo users)
- **Password:** `pulse-demo-2024`

For detailed setup: see [SETUP.md](SETUP.md).

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│  stone (Go backend)                   ember (React PWA)    │
│  ├─ Models (User, Content, Tag)      ├─ Feed              │
│  ├─ Services (Auth, Feed, Social)    ├─ Social Discovery  │
│  ├─ WebSocket Hub (Rooms)            ├─ Mood Rooms        │
│  ├─ Embedder (Ollama or Hash)        ├─ Paths             │
│  └─ REST API                         └─ Auth UI           │
│        ↓                                    ↓              │
│  PostgreSQL + Redis                    grove (Landing)    │
│                                           ↓               │
│                                        Static marketing   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Services

| Service | Language | Purpose |
|---------|----------|---------|
| **stone** | Go | REST API, WebSocket hub, auth, feed ranking, vector search |
| **ember** | React (TS) | Main PWA — feed, discovery, social, rooms, paths |
| **grove** | React (TS) | Public landing page |
| **drift** | TypeScript | Shared types and constants |

### Database

- **PostgreSQL** — relational data (users, posts, tags, reactions, follows, rooms, paths)
- **Redis** — session cache, tag autocomplete, real-time room presence
- **pgvector** — vector embeddings for semantic search (future enhancement)

### Stack

- **Backend:** Go + Gin + GORM + PostgreSQL + Redis + WebSocket
- **Frontend:** React + Vite + TypeScript + Tailwind v4 + Zustand
- **Embedder:** Ollama (qwen3-embedding) with local fallback

---

## Data Model

### Core entities

- **Users** — profiles, bios, follows/followers, trust ratings
- **Content** — posts (image/video/short-video/text), creator, timestamp
- **Tags** — user-defined hashtags with usage counts
- **Reactions** — semantic feedback (5 kinds, not likes)
- **Follows** — follower/following relationships
- **Rooms** — temporary mood clusters (24h TTL)
- **Paths** — curated journeys with ordered content items

### Affinity calculation

Feed is ranked by:
1. **Tag overlap** (SQL: shared tags between your content and others)
2. **Recency** (prefer newer content)
3. **Diversity** (balance different creators/topics)

Each suggestion includes a **bridge** — a human-readable explanation of why you're seeing it.

---

## Philosophy

Pulse is built on a belief: **social media should connect people, not optimize for addiction.**

- **No like counts** → removes vanity metrics and comparison culture
- **No algorithmic feed manipulation** → you see what's relevant, not what's rage-inducing
- **Semantic reactions** → richer signal for affinity (mood + aesthetic, not just "engagement")
- **Bridges** → transparency about why suggestions happen
- **User-defined tags** → community owns the taxonomy
- **Mood rooms** → discover people exploring the same aesthetic *right now*

The goal: **meaningful connections through shared taste, not infinite scroll.**

---

## Development

### Local development

```bash
# All-in-one TUI (starts infrastructure, runs migrations, and launches API + PWA + landing)
make local

# Reset demo data, then launch everything
make demo

# Or separate terminals
make dev-stone      # Backend on :8080
make dev-ember      # PWA on :5173
make dev-grove      # Landing on :5174
```

### Testing

```bash
# Backend
cd stone && go test ./...

# Frontend
cd ember && npm run test
cd grove && npm run test
```

### Building for production

```bash
make build-stone
cd ember && npm run build
cd grove && npm run build
```

---

## Environment Variables

**Backend (stone/.env):**

```env
# Required
DATABASE_URL=postgres://pulse@127.0.0.1:5433/pulse_dev?sslmode=disable
REDIS_URL=redis://127.0.0.1:6379/0
JWT_SECRET=your-secret-key-min-32-chars

# Optional
CORS_ORIGINS=http://localhost:5173,http://localhost:5174
STORAGE_PATH=./uploads
OLLAMA_BASE_URL=http://localhost:11434
OLLAMA_MODEL=qwen3-embedding
```

See [stone/internal/config/config.go](stone/internal/config/config.go) for all options.

---

## Project Status

✅ **Done**
- Core backend (auth, content CRUD, reactions, follows)
- Feed ranking (tag-based affinity)
- WebSocket rooms (real-time mood clusters)
- React PWA (feed, social, discovery, rooms, paths)
- Landing page
- TypeScript types & shared constants
- Vector embeddings (Ollama + fallback)

🚀 **Next**
- Vitest coverage
- Path builder UI refinements
- Mobile optimization
- Trust system (help/ask features)

---

## Docs

For detailed context, see:

- **[SETUP.md](SETUP.md)** — Step-by-step local setup guide
- **[CLAUDE.md](CLAUDE.md)** — Architecture & codebase guide
- **docs/ideas.md** — Vision document (Spanish) — the north star for product & design
- **docs/business.md** — Go-to-market, unit economics, pitch
- **docs/tech-stack.md** — Full technical spec, schema, affinity query

---

## License

[MIT License](LICENSE)

---

<div align="center">
  <p><strong>Built by passionate people who believe social platforms should connect, not addict.</strong></p>
  <p>
    <a href="https://github.com/yourusername/pulse">GitHub</a> •
    <a href="SETUP.md">Get Started</a> •
    <a href="docs/ideas.md">Vision</a>
  </p>
</div>
