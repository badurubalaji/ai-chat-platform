# Story 2.3: Build HTTP-Based Tool Executor

Status: ready-for-dev

## Story

As the platform,
I want to execute tools by making HTTP calls to internal service endpoints,
so that AI tool calls translate into real actions in the domain application.

## Acceptance Criteria

1. **Given** a tool call matching an adapter tool, **When** `ExecuteTool()` is called, **Then** it makes an HTTP request with correct method, headers, and body.
2. **Given** a URL with `{param}` placeholders, **When** arguments include that param, **Then** the URL is interpolated.
3. **Given** a successful HTTP response, **When** received, **Then** the response body is returned as `json.RawMessage`.
4. **Given** a timeout or error status, **When** execution completes, **Then** a structured error is returned.
5. **Given** an unknown tool name, **When** `ExecuteTool()` is called, **Then** it returns a "tool not found" error.

## Tasks / Subtasks

- [ ] Task 1: Create `backend/internal/domain/executor.go` (AC: #1-#5)
  - [ ] Define `Executor` struct with `*Adapter` reference
  - [ ] Implement `NewExecutor(adapter *Adapter) *Executor`
  - [ ] Implement `ExecuteTool(ctx context.Context, toolName string, arguments json.RawMessage, tenantID, userID string) (json.RawMessage, error)`
    - Look up tool by name via adapter
    - Parse arguments into `map[string]interface{}`
    - Interpolate URL: `strings.ReplaceAll(url, "{"+key+"}", value)`
    - POST/PUT: marshal arguments as body; GET: append as query params
    - Set configured headers + `X-Tenant-ID` + `X-User-ID`
    - Create request with `context.WithTimeout` using tool's TimeoutMs
    - Return response body or structured error
- [ ] Task 2: Write unit tests with `httptest.Server`
  - [ ] Successful POST execution
  - [ ] GET with path param interpolation
  - [ ] Timeout handling
  - [ ] Error status (4xx, 5xx)
  - [ ] Unknown tool name

## Dev Notes

- Create: `backend/internal/domain/executor.go`
- Use `context.WithTimeout` for per-tool timeouts rather than `http.Client.Timeout`
- Default timeout: 5000ms if not specified in config
- For GET, move arguments to query string; for POST/PUT/DELETE, use JSON body

### References

- [Source: backend/internal/domain/adapter.go#ToolConfig.Execution struct — from Story 1.1]

## Dev Agent Record

### Agent Model Used
### Debug Log References
### Completion Notes List
### File List
