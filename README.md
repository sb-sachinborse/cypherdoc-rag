# CypherDoc

Chat with your own documents. Upload PDF, DOCX, TXT or Markdown files and ask questions; answers are generated locally by Ollama from your documents only, with page-level citations and, for numeric questions, stat cards, tables and charts.

**Stack:** Go (backend) · Vue 3 (frontend, served by nginx) · Postgres + pgvector · Ollama · Docker Compose.

## Run it

```bash
docker compose up --build
```

Open http://localhost:8080, create an account, upload a document, ask a question.

The first start downloads the AI models (`llama3.2:3b` about 2 GB and `nomic-embed-text` about 270 MB). The sidebar shows "Downloading AI models…" until they are ready; uploads wait for the models automatically.

Optional: `make up` does the same but first creates `.env` with a random database password. `make logs` follows progress.

## Commands

| Command | What it does |
|---|---|
| `make up` | create `.env`, build, start in the background |
| `make test` | `go vet` and Go unit tests (inside Docker, no local Go needed) |
| `make smoke` | end-to-end checks against the running stack (auth, upload, RAG answer with citation, user isolation) |
| `make down` / `make clean` | stop / stop and delete all data and models |

## How it works

```
Upload -> extract text (PDF/DOCX/TXT/MD) -> ~200-word chunks -> nomic-embed-text -> pgvector
Question -> embed -> top-5 chunks for THIS user -> llama3.2 streams an answer using only those chunks
         -> (numeric questions) JSON extraction of stats/table/chart, kept only if every number
            literally appears in the retrieved text
```

If the best match is too distant (`MAX_DISTANCE`, default 0.75) the app answers that it could not find the topic instead of guessing.

## Security

- Passwords hashed with bcrypt (cost 12); 10 to 72 characters; constant-time-ish login for unknown emails.
- Server-side sessions: a random 256-bit token in an `HttpOnly`, `SameSite=Strict` cookie; only its SHA-256 is stored, so logout really revokes it.
- CSRF: state-changing requests need a custom header, which browsers cannot send cross-site, plus SameSite=Strict.
- Every query is scoped to the logged-in user (documents, chunks, retrieval). `make smoke` checks that a second user cannot see or retrieve the first user's files.
- Uploads: size limit (10 MB), extension allow-list, magic-byte check, streamed (no temp files), parser panics recovered, DOCX decompression capped, filenames sanitised.
- Prompt-injection hardening: fixed system prompt, document text delimited and treated as untrusted data, client-sent `system` messages dropped, AI-generated numbers validated against the source text.
- Rate limits on login/register, chat and upload.
- The model never renders HTML: all content is displayed as text (no `v-html`). nginx sends a strict CSP, `nosniff`, `frame-ancestors 'none'` and similar headers.
- Network: only nginx is published, and only on `127.0.0.1`. Postgres, Ollama and the backend sit on an internal Docker network with no route to the host or the internet (Ollama alone also joins an outbound network to download models).
- Containers: backend runs as non-root on a distroless image with a read-only filesystem, all capabilities dropped and `no-new-privileges`; the frontend uses unprivileged nginx with the same hardening.

### Before exposing it beyond your machine

1. Put it behind HTTPS (Caddy, Traefik, or a cloud load balancer), then set `COOKIE_SECURE=true` and add an HSTS header at that proxy.
2. Run `make setup` (random DB password) and consider `ALLOW_SIGNUP=false` once your users exist.
3. Change `APP_PORT` binding in `docker-compose.yml` only if you really intend to publish it.
4. Behind a proxy, the backend trusts `X-Real-IP` from nginx for rate limiting; keep the backend unpublished.
5. Pin image versions (`ollama/ollama:latest`) and commit a `go.sum` and `package-lock.json`.

## Configuration (`.env`)

`POSTGRES_PASSWORD`, `APP_PORT`, `CHAT_MODEL`, `EMBED_MODEL`, `COOKIE_SECURE`, `ALLOW_SIGNUP`, `RICH_BLOCKS`. Backend-only knobs: `MAX_UPLOAD_MB`, `MAX_DISTANCE`, `TOP_K`, `MAX_DOCS_PER_USER`.

- **Better answers:** set `CHAT_MODEL=qwen2.5:7b` (needs about 8 GB RAM, much faster with a GPU).
- **GPU:** `docker compose -f docker-compose.yml -f docker-compose.gpu.yml up --build`.
- The embedding column is `vector(768)`, matching `nomic-embed-text`. A different embedding model needs a schema change.

## Known limits

- Scanned PDFs (images of text) are not supported; there is no OCR.
- Small local models can still get things wrong, so check the cited sources.
- Retrieval is an exact scan per user (no ANN index), which is accurate and fine for thousands of chunks per user.
- Chat history lives in the browser tab only.

## Layout

```
backend/    Go API: auth, upload + ingestion workers, retrieval, SSE chat, tests
frontend/   Vue 3 app (landing, login/register, workspace) + nginx config
scripts/    setup.sh, smoke.sh
docker-compose.yml  db, ollama, ollama-pull (model download), backend, frontend
```
