-- +goose Up

CREATE TABLE asks (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    question          TEXT        NOT NULL,
    triage_summary    TEXT        NOT NULL DEFAULT '',
    topic             VARCHAR(120) NOT NULL DEFAULT '',
    urgency           VARCHAR(40) NOT NULL DEFAULT 'soon',
    desired_help_type VARCHAR(40) NOT NULL DEFAULT 'advice',
    visibility        VARCHAR(40) NOT NULL DEFAULT 'community',
    embedding         vector(1024),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_asks_user_id ON asks (user_id);
CREATE INDEX idx_asks_created_at ON asks (created_at);

CREATE TABLE trust_profiles (
    user_id           UUID        PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    topics            TEXT        NOT NULL DEFAULT '',
    lived_experience  TEXT        NOT NULL DEFAULT '',
    availability      VARCHAR(40) NOT NULL DEFAULT 'async',
    helped_count      INT         NOT NULL DEFAULT 0,
    response_quality  FLOAT       NOT NULL DEFAULT 0,
    expertise_vector  vector(1024),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_trust_profiles_availability ON trust_profiles (availability);
CREATE INDEX idx_trust_profiles_quality ON trust_profiles (response_quality DESC);

CREATE TABLE bridges (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    ask_id              UUID        NOT NULL REFERENCES asks (id) ON DELETE CASCADE,
    requester_id        UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    recommended_user_id UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    reason              TEXT        NOT NULL,
    bridge_type         VARCHAR(40) NOT NULL,
    confidence          FLOAT       NOT NULL DEFAULT 0,
    status              VARCHAR(40) NOT NULL DEFAULT 'suggested',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (ask_id, recommended_user_id)
);

CREATE INDEX idx_bridges_ask_id ON bridges (ask_id);
CREATE INDEX idx_bridges_requester_id ON bridges (requester_id);
CREATE INDEX idx_bridges_recommended_user_id ON bridges (recommended_user_id);
CREATE INDEX idx_bridges_status ON bridges (status);

CREATE TABLE help_signals (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    bridge_id  UUID        NOT NULL REFERENCES bridges (id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind       VARCHAR(40) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_help_signals_bridge_user_kind ON help_signals (bridge_id, user_id, kind);

CREATE TABLE help_sessions (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    title       VARCHAR(160) NOT NULL,
    intent      VARCHAR(120) NOT NULL DEFAULT '',
    description TEXT        NOT NULL DEFAULT '',
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_help_sessions_expires_at ON help_sessions (expires_at);

CREATE TABLE help_session_members (
    session_id UUID        NOT NULL REFERENCES help_sessions (id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    joined_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (session_id, user_id)
);

ALTER TABLE embeddings ALTER COLUMN vector TYPE vector(1024) USING NULL;

-- +goose Down

ALTER TABLE embeddings ALTER COLUMN vector TYPE vector(256) USING NULL;
DROP TABLE IF EXISTS help_session_members;
DROP TABLE IF EXISTS help_sessions;
DROP TABLE IF EXISTS help_signals;
DROP TABLE IF EXISTS bridges;
DROP TABLE IF EXISTS trust_profiles;
DROP TABLE IF EXISTS asks;
