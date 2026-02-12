-- +goose Up

-- Enable pgvector if available (optional, for embeddings table)
CREATE EXTENSION IF NOT EXISTS vector;

-- ──────────────────────────────────────────────
-- users
-- ──────────────────────────────────────────────
CREATE TABLE users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    handle        VARCHAR(30) NOT NULL,
    email         TEXT        NOT NULL,
    password      TEXT        NOT NULL,
    display_name  VARCHAR(100) NOT NULL DEFAULT '',
    bio           VARCHAR(500) NOT NULL DEFAULT '',
    avatar_url    TEXT        NOT NULL DEFAULT '',
    location      VARCHAR(100) NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_users_handle ON users (handle);
CREATE UNIQUE INDEX idx_users_email  ON users (email);

-- ──────────────────────────────────────────────
-- tags
-- ──────────────────────────────────────────────
CREATE TABLE tags (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL,
    usage_count INT          NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_tags_name ON tags (name);

-- ──────────────────────────────────────────────
-- media_assets
-- ──────────────────────────────────────────────
CREATE TABLE media_assets (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id      UUID         NOT NULL REFERENCES users (id),
    content_type  VARCHAR(20)  NOT NULL,
    original_path TEXT         NOT NULL DEFAULT '',
    playback_path TEXT         NOT NULL DEFAULT '',
    filename      VARCHAR(255) NOT NULL DEFAULT '',
    mime_type     VARCHAR(120) NOT NULL DEFAULT '',
    size_bytes    BIGINT       NOT NULL DEFAULT 0,
    status        VARCHAR(20)  NOT NULL,
    error_message VARCHAR(1000) NOT NULL DEFAULT '',
    ready_at      TIMESTAMPTZ,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_media_assets_owner_id ON media_assets (owner_id);
CREATE INDEX idx_media_assets_status   ON media_assets (status);

-- ──────────────────────────────────────────────
-- contents
-- ──────────────────────────────────────────────
CREATE TABLE contents (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_id     UUID        NOT NULL REFERENCES users (id),
    content_type   VARCHAR(20) NOT NULL DEFAULT 'image',
    media_asset_id UUID        REFERENCES media_assets (id),
    media_url      TEXT        NOT NULL DEFAULT '',
    body           TEXT        NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_contents_creator_id     ON contents (creator_id);
CREATE INDEX idx_contents_media_asset_id ON contents (media_asset_id);

-- ──────────────────────────────────────────────
-- content_tags (many-to-many join)
-- ──────────────────────────────────────────────
CREATE TABLE content_tags (
    content_id UUID NOT NULL REFERENCES contents (id) ON DELETE CASCADE,
    tag_id     UUID NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
    PRIMARY KEY (content_id, tag_id)
);

-- ──────────────────────────────────────────────
-- reactions
-- ──────────────────────────────────────────────
CREATE TABLE reactions (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users (id),
    content_id UUID        NOT NULL REFERENCES contents (id),
    kind       VARCHAR(30) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX  idx_reactions_user_id    ON reactions (user_id);
CREATE INDEX  idx_reactions_content_id ON reactions (content_id);
CREATE UNIQUE INDEX idx_reactions_user_content_kind ON reactions (user_id, content_id, kind);

-- ──────────────────────────────────────────────
-- follows
-- ──────────────────────────────────────────────
CREATE TABLE follows (
    follower_id UUID        NOT NULL REFERENCES users (id),
    followee_id UUID        NOT NULL REFERENCES users (id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (follower_id, followee_id)
);

-- ──────────────────────────────────────────────
-- blocks
-- ──────────────────────────────────────────────
CREATE TABLE blocks (
    blocker_id UUID        NOT NULL REFERENCES users (id),
    blocked_id UUID        NOT NULL REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (blocker_id, blocked_id)
);

-- ──────────────────────────────────────────────
-- refresh_tokens
-- ──────────────────────────────────────────────
CREATE TABLE refresh_tokens (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users (id),
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,
    replaced_by UUID,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_tokens_user_id    ON refresh_tokens (user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens (expires_at);
CREATE INDEX idx_refresh_tokens_revoked_at ON refresh_tokens (revoked_at);

-- ──────────────────────────────────────────────
-- rooms
-- ──────────────────────────────────────────────
CREATE TABLE rooms (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_key TEXT        NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_rooms_cluster_key ON rooms (cluster_key);
CREATE INDEX idx_rooms_expires_at  ON rooms (expires_at);

-- ──────────────────────────────────────────────
-- room_tags (many-to-many join)
-- ──────────────────────────────────────────────
CREATE TABLE room_tags (
    room_id UUID NOT NULL REFERENCES rooms (id) ON DELETE CASCADE,
    tag_id  UUID NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
    PRIMARY KEY (room_id, tag_id)
);

-- ──────────────────────────────────────────────
-- room_members
-- ──────────────────────────────────────────────
CREATE TABLE room_members (
    room_id   UUID        NOT NULL REFERENCES rooms (id) ON DELETE CASCADE,
    user_id   UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (room_id, user_id)
);

-- ──────────────────────────────────────────────
-- paths
-- ──────────────────────────────────────────────
CREATE TABLE paths (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_id     UUID         NOT NULL REFERENCES users (id),
    title          VARCHAR(200) NOT NULL,
    description    VARCHAR(1000) NOT NULL DEFAULT '',
    follower_count INT          NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_paths_creator_id ON paths (creator_id);

-- ──────────────────────────────────────────────
-- path_items
-- ──────────────────────────────────────────────
CREATE TABLE path_items (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    path_id    UUID         NOT NULL REFERENCES paths (id) ON DELETE CASCADE,
    content_id UUID         NOT NULL REFERENCES contents (id),
    position   INT          NOT NULL,
    note       VARCHAR(500) NOT NULL DEFAULT ''
);

CREATE INDEX idx_path_items_path_id ON path_items (path_id);

-- ──────────────────────────────────────────────
-- path_follows
-- ──────────────────────────────────────────────
CREATE TABLE path_follows (
    path_id    UUID        NOT NULL REFERENCES paths (id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (path_id, user_id)
);

-- ──────────────────────────────────────────────
-- events
-- ──────────────────────────────────────────────
CREATE TABLE events (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users (id),
    type        VARCHAR(50) NOT NULL,
    target_type VARCHAR(50) NOT NULL DEFAULT '',
    target_id   UUID,
    metadata    JSONB       NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_events_user_id    ON events (user_id);
CREATE INDEX idx_events_type       ON events (type);
CREATE INDEX idx_events_created_at ON events (created_at);

-- ──────────────────────────────────────────────
-- user_affinity_edges
-- ──────────────────────────────────────────────
CREATE TABLE user_affinity_edges (
    user_id        UUID        NOT NULL REFERENCES users (id),
    other_user_id  UUID        NOT NULL REFERENCES users (id),
    score_7d       FLOAT       NOT NULL DEFAULT 0,
    score_30d      FLOAT       NOT NULL DEFAULT 0,
    last_signal_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, other_user_id)
);

CREATE INDEX idx_user_affinity_edges_last_signal_at ON user_affinity_edges (last_signal_at);

-- ──────────────────────────────────────────────
-- embeddings (pgvector)
-- ──────────────────────────────────────────────
CREATE TABLE embeddings (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(20) NOT NULL,
    entity_id   UUID        NOT NULL,
    vector      vector(256),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_embeddings_entity_type ON embeddings (entity_type);
CREATE INDEX idx_embeddings_entity_id   ON embeddings (entity_id);


-- +goose Down

DROP TABLE IF EXISTS embeddings;
DROP TABLE IF EXISTS user_affinity_edges;
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS path_follows;
DROP TABLE IF EXISTS path_items;
DROP TABLE IF EXISTS paths;
DROP TABLE IF EXISTS room_members;
DROP TABLE IF EXISTS room_tags;
DROP TABLE IF EXISTS rooms;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS blocks;
DROP TABLE IF EXISTS follows;
DROP TABLE IF EXISTS reactions;
DROP TABLE IF EXISTS content_tags;
DROP TABLE IF EXISTS contents;
DROP TABLE IF EXISTS media_assets;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS users;
