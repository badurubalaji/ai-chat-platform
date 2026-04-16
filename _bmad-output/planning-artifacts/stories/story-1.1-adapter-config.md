# Story 1.1: Define AdapterConfig Struct and JSON Loader

Status: done

## Audit (2026-04-16)

- AC#1 ✅ `backend/internal/domain/adapter.go` loads JSON into `Adapter` struct.
- AC#2 ✅ Empty path returns nil, no error.
- AC#3 ✅ Invalid JSON → descriptive error wrap; validation errors cover missing domain/system_prompt, tool fields, default_provider fields.
- AC#4 ✅ Only read-only accessors — `Domain()`, `SystemPrompt()`, `Tools()`, `ToolByName()`, `HasTools()`, `HasDefaultProvider()`, `DefaultProvider()`, `ToolsForProvider()`. No setters.
- Task 4 ✅ Unit tests added in `backend/internal/domain/adapter_test.go` covering: empty path, missing file, invalid JSON, valid config, default POST method, 7 validation-failure subtests, default provider fallback defaults, `ToolsForProvider` conversion including nil case.

## Story

As a platform developer,
I want a Go struct that represents the adapter-config.json schema,
so that domain configuration can be loaded and validated at startup.

## Acceptance Criteria

1. **Given** a valid `adapter-config.json` at `ADAPTER_CONFIG_PATH`, **When** the application starts, **Then** the config is loaded into an immutable `Adapter` struct.
2. **Given** `ADAPTER_CONFIG_PATH` is not set, **When** the application starts, **Then** `LoadAdapter("")` returns nil adapter with no error (generic mode).
3. **Given** invalid JSON at the path, **When** `LoadAdapter()` is called, **Then** it returns a descriptive error.
4. **Given** a valid adapter, **When** code accesses it, **Then** only read-only methods are available (no setters).

## Tasks / Subtasks

- [ ] Task 1: Create `backend/internal/domain/adapter.go` (AC: #1, #4)
  - [ ] Define `AdapterConfig` struct: Domain, DisplayName, SystemPrompt, DefaultProvider, Tools
  - [ ] Define `DefaultProviderConfig` sub-struct: Provider, Model, EndpointURL, ClientID, ClientSecret
  - [ ] Define `ToolConfig` struct: Name, Description, Parameters (json.RawMessage), RequiredRole, RequiresConfirmation, Execution
  - [ ] Define `ToolExecution` sub-struct: Type, Method, URL, Headers, TimeoutMs
  - [ ] Define `Adapter` struct wrapping config with read-only methods
  - [ ] Implement `LoadAdapter(path string) (*Adapter, error)`
  - [ ] Implement accessors: `SystemPrompt()`, `Tools()`, `ToolByName()`, `DefaultProvider()`, `HasDefaultProvider()`, `ToolsForProvider() []models.Tool`
- [ ] Task 2: Add `ADAPTER_CONFIG_PATH` to config (AC: #2)
  - [ ] Add field to `Config` struct in `config.go`
  - [ ] Load from env with empty default
- [ ] Task 3: Validate adapter config (AC: #3)
  - [ ] If default_provider present, require client_id + client_secret
  - [ ] Each tool must have name, description, execution block
  - [ ] Return descriptive validation errors
- [ ] Task 4: Write unit tests
  - [ ] Valid config → loads successfully
  - [ ] Empty path → nil adapter, no error
  - [ ] Invalid JSON → error
  - [ ] Missing required fields → validation error

## Dev Notes

- Create: `backend/internal/domain/adapter.go`
- Modify: `backend/internal/config/config.go` (add 1 field + 1 getEnv)
- Reuse: `models.Tool` at `backend/internal/models/models.go:55` — `ToolsForProvider()` converts `[]ToolConfig` to `[]models.Tool`
- Use `json.RawMessage` for Parameters to preserve JSON schema without Go mapping

### Project Structure Notes

- New package `internal/domain/` follows existing pattern of `internal/api/`, `internal/providers/`, etc.
- Package name: `domain` (not `adapter`) to allow future orchestrator/executor in same package

### References

- [Source: backend/internal/models/models.go#Tool struct lines 55-60]
- [Source: backend/internal/config/config.go#Config struct lines 7-11]

## Dev Agent Record

### Agent Model Used
### Debug Log References
### Completion Notes List
### File List
