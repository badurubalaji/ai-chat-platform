#!/bin/bash

# Configuration (defaults match setup_db.sh)
export DATABASE_URL=${DATABASE_URL:-"postgres://ai_user:ai_password@localhost:5432/ai_chat_db?sslmode=disable"}
export AI_ENCRYPTION_KEY=${AI_ENCRYPTION_KEY:-"0123456789abcdef0123456789abcdef"} # 32-byte key for dev

echo "Using Database: $DATABASE_URL"

# 1. Run Migrations
if [ -f "./bin/migrate" ]; then
    echo "Running migrations..."
    ./bin/migrate up
else
    echo "Error: ./bin/migrate not found. Run 'make setup' first."
    exit 1
fi

# 2. Build Backend Binary
echo "Building backend..."
cd backend && go build -o ../bin/server cmd/server/main.go 2>&1
if [ $? -ne 0 ]; then
    echo "Error: Backend build failed."
    exit 1
fi
cd ..
echo "Backend built successfully."

# 3. Build Frontend Library (required before serving demo app)
echo "Building frontend library (mdp-ai-chat)..."
cd frontend && npx ng build mdp-ai-chat 2>&1
if [ $? -ne 0 ]; then
    echo "Error: Frontend library build failed."
    exit 1
fi
echo "Frontend library built successfully."

# 4. Start Backend (background)
echo "Starting backend on port ${PORT:-8080}..."
cd .. && ./bin/server &
BACKEND_PID=$!

# 5. Start Library Watch Rebuild (background) — keeps dist updated on source changes
echo "Starting library watch build..."
cd frontend && npx ng build mdp-ai-chat --watch &
LIB_WATCH_PID=$!

# 6. Wait a moment for first watch build, then start demo app serve
sleep 2
echo "Starting frontend dev server on port 4200..."
npx ng serve demo-app --open &
FRONTEND_PID=$!

# 7. Handle cleanup on exit
cleanup() {
    echo ""
    echo "Shutting down..."
    kill $BACKEND_PID 2>/dev/null
    kill $LIB_WATCH_PID 2>/dev/null
    kill $FRONTEND_PID 2>/dev/null
    exit 0
}

trap cleanup SIGINT SIGTERM

echo ""
echo "=========================================="
echo "  AI Chat Platform Running"
echo "  Backend:  http://localhost:${PORT:-8080}"
echo "  Frontend: http://localhost:4200"
echo "  Press Ctrl+C to stop"
echo "=========================================="
echo ""

# Wait for any process to exit
wait
