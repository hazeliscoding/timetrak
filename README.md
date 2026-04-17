# ⏱️ TimeTrak

A calm, no-nonsense time tracking tool built for freelancers and small teams. Track billable hours across clients and projects, resolve rates automatically, and generate reports — all from a fast, server-rendered interface.

---

## ✨ Features

- **Workspace → Client → Project → Time Entry** hierarchy
- Automatic rate resolution: project rate → client rate → workspace default
- Live timers and inline edits via HTMX (no SPA overhead)
- Server-rendered HTML — fast, accessible, keyboard-friendly
- WCAG 2.2 AA compliant

---

## 🛠️ Stack

| Layer    | Technology                          |
|----------|-------------------------------------|
| Backend  | Go (modular monolith, stdlib router) |
| UI       | Go `html/template` + HTMX           |
| Database | PostgreSQL                          |
| Auth     | Session-based (Postgres-backed)     |

---

## 🚀 Getting Started

### Prerequisites

- Go 1.22+
- Docker (for local Postgres)

### 1. Clone & configure

```bash
git clone https://github.com/hazeliscoding/timetrak.git
cd timetrak
cp .env.example .env   # fill in SESSION_SECRET (≥32 bytes) and other vars
```

### 2. Start the database

```bash
make db-up
```

### 3. Run migrations & seed demo data

```bash
make migrate-up
make dev-seed
```

### 4. Start the server

```bash
make run
```

Visit `http://localhost:8080` (or whatever `HTTP_ADDR` is set to).

---

## 🧰 Common Commands

| Command             | Description                          |
|---------------------|--------------------------------------|
| `make run`          | Start the web server                 |
| `make build`        | Build `bin/web` and `bin/migrate`    |
| `make test`         | Run all tests                        |
| `make lint`         | Vet the codebase                     |
| `make fmt`          | Format with `gofmt`                  |
| `make migrate-up`   | Apply all pending migrations         |
| `make migrate-down` | Roll back the latest migration       |
| `make dev-seed`     | Seed demo user, workspace, and data  |
| `make db-up`        | Start local Postgres via Docker      |
| `make db-down`      | Stop local Postgres                  |

---

## 🗂️ Project Layout

```
cmd/
  web/       HTTP server entrypoint
  migrate/   Migration runner + dev seeder
internal/
  shared/    DB, sessions, CSRF, middleware, templates
  auth/      Authentication domain
  workspace/ clients/ projects/ tracking/ rates/ reporting/
web/
  templates/ Server-rendered layouts, pages, partials
  static/    CSS tokens, app.js, vendored HTMX
migrations/  Plain SQL up/down files
```

---

## ⚙️ Environment Variables

Copy `.env.example` and set the following:

| Variable        | Description                              |
|-----------------|------------------------------------------|
| `DATABASE_URL`  | Postgres connection string               |
| `SESSION_SECRET`| HMAC signing key (≥ 32 bytes)            |
| `COOKIE_SECURE` | `true` in production, `false` locally    |
| `APP_ENV`       | `development` or `production`            |
| `HTTP_ADDR`     | Listen address (e.g. `:8080`)            |

---

## 📄 License

MIT
