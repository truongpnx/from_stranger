# From Stranger

From Stranger is a small anonymous social app where people publish short sentences and react to random sentences from others.

## App Idea

- Users can publish a sentence (up to 100 words).
- Published sentences are visible to random users for 24 hours.
- Other users can react with:
  - ❤️ `heart`
  - 💔 `hate`
  - 🙈 `ignore`
- After 24 hours, the author can view final reaction results.
- If no usable random sentence exists, the app shows a fallback sentence.

## Current Flow (MVP)

1. Open home page.
2. Random sentence auto-loads.
3. Reacting disables buttons and automatically loads the next sentence.
4. Publish form lets a user submit new sentences (limited per user).
5. "My published" lists user posts.
6. "Ready results" lists expired sentences; click one to see final stats.

## Tech Stack

- Go (HTTP server)
- HTMX (dynamic UI updates)
- Redis (data storage)

## Redis Data Model (Current)

- `sentence:{id}` (HASH)
- `sentences:active` (SET)
- `user:{id}:seen` (SET)
- `user:{id}:publish_count` (STRING counter)
- `user:{id}:published` (ZSET)
- `user:{id}:reacted:{sentence_id}` (STRING lock)

## Run Locally

### 1) Start Redis

Example with Docker:

```bash
docker run --name from-stranger-redis -p 6379:6379 -d redis:7
```

### 2) Set env (optional)

Defaults are already:
- `REDIS_HOST=127.0.0.1`
- `REDIS_PORT=6379`
- `REDIS_DB=0`

If needed:

```bash
export REDIS_HOST=127.0.0.1
export REDIS_PORT=6379
export REDIS_USER=
export REDIS_PASSWORD=
```

### 3) Build and run

```bash
make build
make run
```

Open: `http://localhost:8080`

## Docker

```bash
docker build -t from-stranger .
docker run --rm -p 8080:8080 \
  -e REDIS_HOST=host.docker.internal \
  -e REDIS_PORT=6379 \
  from-stranger
```

## Notes

- This project is currently an MVP focused on core product behavior.
- Next steps can include moderation, weighted randomization, and better anti-abuse controls.
