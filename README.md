# AI Chat Platform

A complete AI Chat platform with an Angular frontend library and a configurable multi-provider backend (Go).

## 🚀 Features

- **Multi-Provider Support**: Claude, OpenAI, and more.
- **Frontend Library**: Reusable `@mdp/ai-chat` Angular library.
- **Streaming**: Server-Sent Events (SSE) for real-time responses.
- **Secure**: AES-256-GCM encryption for API keys.
- **Analytics**: Token usage tracking and visualization.

## 🛠️ Prerequisites

- PostgreSQL (Local or Podman/Docker)
- Go 1.22+
- Node.js 18+

## 🏁 Getting Started

### 1. Configure Database
Ensure your local PostgreSQL is running.
Set the `DATABASE_URL` environment variable if your credentials differ from default:

```bash
export DATABASE_URL="postgres://user:pass@localhost:5432/dbname?sslmode=disable"
```

### 2. Build & Run Migrations
Build the tools and apply database schema:

```bash
# Build backend and migration tool
make build-backend
cd backend && go build -o ../bin/migrate cmd/migrate/main.go && cd ..

# Run migrations
./bin/migrate up
```

### 3. Start Backend
Run the server (using the helper script or directly):
```bash
./start.sh
# or
# DATABASE_URL=... ./bin/server
```

### 4. Start Frontend
Run the Angular Demo App:
```bash
cd frontend
npm start
# Host: http://localhost:4200
```

## 📚 Documentation

- [Implementation Plan](implementation_plan.md)
- [Task List](task.md)
- [Walkthrough](walkthrough.md)

## 🔧 Configuration

Set the following environment variables (or use `.env`):
- `DATABASE_URL`: Postgres connection string (Default: `postgres://ai_user:ai_password@localhost:5432/ai_chat_db?sslmode=disable`)
- `AI_ENCRYPTION_KEY`: 32-byte hex string for key encryption
- `PORT`: Backend port (default 8080)
