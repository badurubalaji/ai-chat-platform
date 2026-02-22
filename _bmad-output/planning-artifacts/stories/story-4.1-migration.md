# Story 4.1: Create Database Migration for ai_tool_executions

Status: ready-for-dev

## Story

As a platform operator,
I want tool executions recorded in the database,
so that I have an audit trail of all AI-triggered actions.

## Acceptance Criteria

1. **Given** migration applied, **When** querying DB, **Then** `ai_tool_executions` exists with all columns.
2. **Given** migration applied, **When** querying DB, **Then** indexes on `(tenant_id, created_at)` and `(conversation_id)` exist.

## Tasks / Subtasks

- [ ] Task 1: Create migration files (AC: #1, #2)
  - [ ] Create `backend/db/migrations/000002_tool_executions.up.sql`
  - [ ] Create `backend/db/migrations/000002_tool_executions.down.sql`
- [ ] Task 2: Define `ToolExecution` model (AC: #1)
  - [ ] Add struct to `backend/internal/models/models.go`

## Dev Notes

- Migration dir: `backend/db/migrations/`
- Existing: `000001_init_schema`. New: `000002_tool_executions`
- UP SQL:
  ```sql
  CREATE TABLE ai_tool_executions (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      tenant_id VARCHAR(100) NOT NULL,
      user_id UUID NOT NULL,
      conversation_id UUID NOT NULL REFERENCES ai_conversations(id) ON DELETE CASCADE,
      tool_name VARCHAR(100) NOT NULL,
      arguments JSONB DEFAULT '{}',
      result JSONB,
      status VARCHAR(20) NOT NULL DEFAULT 'pending',
      confirmed_by_user BOOLEAN DEFAULT false,
      execution_duration_ms INT,
      created_at TIMESTAMPTZ DEFAULT NOW()
  );
  CREATE INDEX idx_ai_tool_executions_tenant ON ai_tool_executions(tenant_id, created_at);
  CREATE INDEX idx_ai_tool_executions_conversation ON ai_tool_executions(conversation_id);
  ```
- DOWN SQL: `DROP TABLE IF EXISTS ai_tool_executions;`

### References

- [Source: backend/db/migrations/000001_init_schema.up.sql]
- [Source: backend/internal/models/models.go#existing model patterns]

## Dev Agent Record

### Agent Model Used
### Debug Log References
### Completion Notes List
### File List
