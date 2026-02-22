# Story 5.1: Implement Provider Resolution with BYOK Override

Status: ready-for-dev

## Story

As an end user who has not configured a BYOK key,
I want the AI chat to work out of the box using the deployment's default NeuralGateway,
so that I can use the chat immediately without provider setup.

## Acceptance Criteria

1. **Given** no user config and adapter has `default_provider`, **When** chat sent, **Then** adapter's NG credentials used.
2. **Given** user HAS configured BYOK provider, **When** chat sent, **Then** user's provider used.
3. **Given** neither exists, **When** chat sent, **Then** "Provider not configured" error.
4. **Given** adapter default used, **When** API key needed, **Then** `client_id:client_secret` format constructed.

## Tasks / Subtasks

- [ ] Task 1: Refactor provider resolution (AC: #1-#4)
  - [ ] Extract `resolveProvider(ctx, tenantID)` method returning provider, apiKey, model, endpoint, error
  - [ ] First: try `GetProviderConfig(ctx, tenantID)` — if found, use it (existing logic)
  - [ ] If `sql.ErrNoRows` AND `adapter.HasDefaultProvider()`: construct NG credentials
  - [ ] Format apiKey as `"clientID:clientSecret"` (NeuralGateway format)
  - [ ] Replace lines 113-136 in handleChat() with call to this method
- [ ] Task 2: Update handleConfig GET (AC: #1)
  - [ ] If no user config but adapter has default, return `{"enabled": true, "is_default": true, "provider": "neuralgate"}`

## Dev Notes

- Primary: `backend/internal/api/handler.go`
- Store's `GetProviderConfig` returns `sql.ErrNoRows` when no config for tenant (store.go line 50)
- NeuralGate provider's `parseCredentials()` at neuralgate.go line 74 expects `"client_id:client_secret"`

### References

- [Source: backend/internal/api/handler.go#provider resolution lines 113-136]
- [Source: backend/internal/providers/neuralgate/neuralgate.go#parseCredentials line 74]
- [Source: backend/internal/store/store.go#GetProviderConfig line 43-57]

## Dev Agent Record

### Agent Model Used
### Debug Log References
### Completion Notes List
### File List
