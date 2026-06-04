-- +goose Up

CREATE TABLE bridge_responses (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    bridge_id    UUID        NOT NULL REFERENCES bridges (id) ON DELETE CASCADE,
    responder_id UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    body         TEXT        NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (bridge_id, responder_id)
);

CREATE INDEX idx_bridge_responses_bridge_id ON bridge_responses (bridge_id);
CREATE INDEX idx_bridge_responses_responder_id ON bridge_responses (responder_id);
CREATE INDEX idx_bridge_responses_created_at ON bridge_responses (created_at DESC);

-- +goose Down

DROP TABLE IF EXISTS bridge_responses;
