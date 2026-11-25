CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS users (
  id            BIGSERIAL PRIMARY KEY,
  email         TEXT NOT NULL UNIQUE,
  name          TEXT NOT NULL DEFAULT '',
  password_hash TEXT NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sessions (
  token_hash BYTEA PRIMARY KEY,
  user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS sessions_expires_idx ON sessions (expires_at);

CREATE TABLE IF NOT EXISTS documents (
  id         BIGSERIAL PRIMARY KEY,
  user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name       TEXT NOT NULL,
  collection TEXT NOT NULL DEFAULT 'General',
  size_bytes BIGINT NOT NULL,
  status     TEXT NOT NULL DEFAULT 'processing' CHECK (status IN ('processing','indexed','failed')),
  error      TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS documents_user_idx ON documents (user_id, created_at DESC);

-- No ANN index on purpose: every query is filtered by user, and an exact scan
-- stays correct for that. Add HNSW with iterative scans if you outgrow this.
CREATE TABLE IF NOT EXISTS chunks (
  id          BIGSERIAL PRIMARY KEY,
  document_id BIGINT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  user_id     BIGINT NOT NULL,
  page        INT NOT NULL DEFAULT 1,
  idx         INT NOT NULL,
  content     TEXT NOT NULL,
  embedding   vector(768) NOT NULL
);
CREATE INDEX IF NOT EXISTS chunks_doc_idx ON chunks (document_id);
CREATE INDEX IF NOT EXISTS chunks_user_idx ON chunks (user_id);
