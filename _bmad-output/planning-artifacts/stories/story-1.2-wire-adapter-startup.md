# Story 1.2: Wire Adapter Loading into main.go and Handler

Status: ready-for-dev

## Story

As a platform developer,
I want the adapter loaded at server startup and injected into the handler,
so that handleChat() has access to domain tools and system prompt.

## Acceptance Criteria

1. **Given** a valid `ADAPTER_CONFIG_PATH`, **When** `main.go` runs, **Then** the adapter is loaded before handler creation and passed to `NewHandler()`.
2. **Given** `ADAPTER_CONFIG_PATH` is empty, **When** `main.go` runs, **Then** `NewHandler()` receives nil adapter and server starts normally.
3. **Given** the adapter is loaded, **When** `NewHandler()` is called, **Then** the handler stores the adapter reference.

## Tasks / Subtasks

- [ ] Task 1: Modify `main.go` (AC: #1, #2)
  - [ ] After `config.Load()`, check `cfg.AdapterConfigPath`
  - [ ] If non-empty: call `domain.LoadAdapter(cfg.AdapterConfigPath)`, log success
  - [ ] If load fails: `log.Fatalf` with error
  - [ ] Pass adapter (possibly nil) to `api.NewHandler()`
- [ ] Task 2: Modify `Handler` struct and `NewHandler()` (AC: #3)
  - [ ] Add `adapter *domain.Adapter` field to Handler struct
  - [ ] Change signature: `NewHandler(s store.Store, cfg *config.Config, adapter *domain.Adapter) *Handler`
  - [ ] Store adapter reference

## Dev Notes

- Modify: `backend/cmd/server/main.go` — add ~5 lines after config load
- Modify: `backend/internal/api/handler.go` — Handler struct (line 26-30), NewHandler (line 32-58)
- Import: `"ai-chat-platform/backend/internal/domain"` in both files

### References

- [Source: backend/cmd/server/main.go#startup flow lines 1-31]
- [Source: backend/internal/api/handler.go#Handler struct lines 26-30]
- [Source: backend/internal/api/handler.go#NewHandler lines 32-58]

## Dev Agent Record

### Agent Model Used
### Debug Log References
### Completion Notes List
### File List
