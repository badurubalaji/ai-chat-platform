package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	_ "github.com/lib/pq" // generic driver
	"github.com/mdp/ai-chat-platform/backend/internal/models"
)

type Store interface {
	GetProviderConfig(ctx context.Context, tenantID string) (*models.ProviderConfig, error)
	SaveProviderConfig(ctx context.Context, config *models.ProviderConfig) error
	DeleteProviderConfig(ctx context.Context, tenantID string) error
	CreateConversation(ctx context.Context, convo *models.Conversation) error
	GetConversation(ctx context.Context, id uuid.UUID) (*models.Conversation, error)
	ListConversations(ctx context.Context, tenantID string, userID uuid.UUID) ([]*models.Conversation, error)
	DeleteConversation(ctx context.Context, id uuid.UUID) error
	CreateMessage(ctx context.Context, msg *models.Message) error
	ListMessages(ctx context.Context, conversationID uuid.UUID) ([]*models.Message, error)
	LogUsage(ctx context.Context, usage *models.UsageLog) error
	GetUsageStats(ctx context.Context, tenantID string, days int) (*models.UsageStats, error)
	LogToolExecution(ctx context.Context, exec *models.ToolExecution) error
	// Tool Registry
	RegisterTool(ctx context.Context, tool *models.RegisteredTool) error
	GetTool(ctx context.Context, id uuid.UUID) (*models.RegisteredTool, error)
	ListTools(ctx context.Context, tenantID string, appName string) ([]*models.RegisteredTool, error)
	UpdateTool(ctx context.Context, tool *models.RegisteredTool) error
	DeleteTool(ctx context.Context, id uuid.UUID) error
}

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(dbURL string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) GetProviderConfig(ctx context.Context, tenantID string) (*models.ProviderConfig, error) {
	query := `SELECT id, tenant_id, provider, model, api_key_encrypted, endpoint_url, settings, enabled, created_at FROM ai_provider_configs WHERE tenant_id = $1`
	row := s.db.QueryRowContext(ctx, query, tenantID)

	var c models.ProviderConfig
	var settingsJSON []byte
	err := row.Scan(&c.ID, &c.TenantID, &c.Provider, &c.Model, &c.APIKeyEncrypted, &c.EndpointURL, &settingsJSON, &c.Enabled, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	if len(settingsJSON) > 0 {
		_ = json.Unmarshal(settingsJSON, &c.Settings)
	}
	return &c, nil
}

func (s *PostgresStore) SaveProviderConfig(ctx context.Context, c *models.ProviderConfig) error {
	settingsJSON, _ := json.Marshal(c.Settings)
	query := `
		INSERT INTO ai_provider_configs (tenant_id, provider, model, api_key_encrypted, endpoint_url, settings, enabled, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (tenant_id) DO UPDATE SET
			provider = EXCLUDED.provider,
			model = EXCLUDED.model,
			api_key_encrypted = EXCLUDED.api_key_encrypted,
			endpoint_url = EXCLUDED.endpoint_url,
			settings = EXCLUDED.settings,
			enabled = EXCLUDED.enabled,
			updated_at = NOW()
		RETURNING id, created_at`
	return s.db.QueryRowContext(ctx, query, c.TenantID, c.Provider, c.Model, c.APIKeyEncrypted, c.EndpointURL, settingsJSON, c.Enabled).Scan(&c.ID, &c.CreatedAt)
}

func (s *PostgresStore) CreateConversation(ctx context.Context, c *models.Conversation) error {
	query := `INSERT INTO ai_conversations (id, tenant_id, user_id, title) VALUES ($1, $2, $3, $4) RETURNING created_at, updated_at`
	return s.db.QueryRowContext(ctx, query, c.ID, c.TenantID, c.UserID, c.Title).Scan(&c.CreatedAt, &c.UpdatedAt)
}

func (s *PostgresStore) GetConversation(ctx context.Context, id uuid.UUID) (*models.Conversation, error) {
	query := `SELECT id, tenant_id, user_id, title, created_at, updated_at FROM ai_conversations WHERE id = $1`
	var c models.Conversation
	err := s.db.QueryRowContext(ctx, query, id).Scan(&c.ID, &c.TenantID, &c.UserID, &c.Title, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *PostgresStore) ListConversations(ctx context.Context, tenantID string, userID uuid.UUID) ([]*models.Conversation, error) {
	query := `SELECT id, tenant_id, user_id, title, created_at, updated_at FROM ai_conversations WHERE tenant_id = $1 AND user_id = $2 ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(ctx, query, tenantID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var convos []*models.Conversation
	for rows.Next() {
		var c models.Conversation
		if err := rows.Scan(&c.ID, &c.TenantID, &c.UserID, &c.Title, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		convos = append(convos, &c)
	}
	return convos, nil
}

func (s *PostgresStore) CreateMessage(ctx context.Context, msg *models.Message) error {
	metaJSON, _ := json.Marshal(msg.Metadata)
	query := `INSERT INTO ai_messages (id, conversation_id, role, content, metadata) VALUES ($1, $2, $3, $4, $5) RETURNING created_at`
	return s.db.QueryRowContext(ctx, query, msg.ID, msg.ConversationID, msg.Role, msg.Content, metaJSON).Scan(&msg.CreatedAt)
}

func (s *PostgresStore) ListMessages(ctx context.Context, conversationID uuid.UUID) ([]*models.Message, error) {
	query := `SELECT id, conversation_id, role, content, metadata, created_at FROM ai_messages WHERE conversation_id = $1 ORDER BY created_at ASC`
	rows, err := s.db.QueryContext(ctx, query, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []*models.Message
	for rows.Next() {
		var m models.Message
		var metaJSON []byte
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &metaJSON, &m.CreatedAt); err != nil {
			return nil, err
		}
		if len(metaJSON) > 0 {
			_ = json.Unmarshal(metaJSON, &m.Metadata)
		}
		msgs = append(msgs, &m)
	}
	return msgs, nil
}

func (s *PostgresStore) LogUsage(ctx context.Context, u *models.UsageLog) error {
	query := `INSERT INTO ai_usage_logs (id, tenant_id, user_id, conversation_id, provider, model, input_tokens, output_tokens) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := s.db.ExecContext(ctx, query, u.ID, u.TenantID, u.UserID, u.ConversationID, u.Provider, u.Model, u.InputTokens, u.OutputTokens)
	return err
}

func (s *PostgresStore) GetUsageStats(ctx context.Context, tenantID string, days int) (*models.UsageStats, error) {
	// 1. Get daily stats
	interval := fmt.Sprintf("%d days", days)
	query := `
		SELECT 
			TO_CHAR(created_at, 'YYYY-MM-DD') as date, 
			COALESCE(SUM(input_tokens), 0) as input, 
			COALESCE(SUM(output_tokens), 0) as output, 
			COUNT(*) as requests 
		FROM ai_usage_logs 
		WHERE tenant_id = $1 AND created_at >= NOW() - $2::INTERVAL
		GROUP BY date 
		ORDER BY date ASC`

	rows, err := s.db.QueryContext(ctx, query, tenantID, interval)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := &models.UsageStats{
		Daily: []models.UsagePeriod{},
	}

	for rows.Next() {
		var p models.UsagePeriod
		if err := rows.Scan(&p.Date, &p.InputTokens, &p.OutputTokens, &p.RequestCount); err != nil {
			return nil, err
		}
		stats.Daily = append(stats.Daily, p)
		stats.TotalInputTokens += p.InputTokens
		stats.TotalOutputTokens += p.OutputTokens
	}

	// 2. Get conversation count (total or in period? usually total active conversations for dashboard)
	// But let's stick to period for now + total all time if needed.
	// The interface implies specific structure.
	// Let's count conversations created in period.

	convoQuery := `SELECT COUNT(*) FROM ai_conversations WHERE tenant_id = $1 AND created_at >= NOW() - $2::INTERVAL`
	if err := s.db.QueryRowContext(ctx, convoQuery, tenantID, interval).Scan(&stats.ConversationCount); err != nil {
		return nil, err
	}

	return stats, nil
}

func (s *PostgresStore) LogToolExecution(ctx context.Context, exec *models.ToolExecution) error {
	argsJSON, _ := json.Marshal(exec.Arguments)
	resultJSON, _ := json.Marshal(exec.Result)
	query := `INSERT INTO ai_tool_executions (id, tenant_id, user_id, conversation_id, tool_name, arguments, result, status, confirmed_by_user, execution_duration_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := s.db.ExecContext(ctx, query, exec.ID, exec.TenantID, exec.UserID, exec.ConversationID,
		exec.ToolName, argsJSON, resultJSON, exec.Status, exec.ConfirmedByUser, exec.ExecutionDuration)
	return err
}

func (s *PostgresStore) DeleteProviderConfig(ctx context.Context, tenantID string) error {
	query := `DELETE FROM ai_provider_configs WHERE tenant_id = $1`
	_, err := s.db.ExecContext(ctx, query, tenantID)
	return err
}

func (s *PostgresStore) DeleteConversation(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM ai_conversations WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// -- Tool Registry --

func (s *PostgresStore) RegisterTool(ctx context.Context, tool *models.RegisteredTool) error {
	query := `
		INSERT INTO ai_tool_registry (id, tenant_id, app_name, tool_name, description, parameters, execution_config, requires_confirmation, enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (tenant_id, app_name, tool_name) DO UPDATE SET
			description = EXCLUDED.description,
			parameters = EXCLUDED.parameters,
			execution_config = EXCLUDED.execution_config,
			requires_confirmation = EXCLUDED.requires_confirmation,
			enabled = EXCLUDED.enabled,
			updated_at = NOW()
		RETURNING created_at, updated_at`
	return s.db.QueryRowContext(ctx, query,
		tool.ID, tool.TenantID, tool.AppName, tool.ToolName, tool.Description,
		tool.Parameters, tool.ExecutionConfig, tool.RequiresConfirmation, tool.Enabled,
	).Scan(&tool.CreatedAt, &tool.UpdatedAt)
}

func (s *PostgresStore) GetTool(ctx context.Context, id uuid.UUID) (*models.RegisteredTool, error) {
	query := `SELECT id, tenant_id, app_name, tool_name, description, parameters, execution_config, requires_confirmation, enabled, created_at, updated_at
		FROM ai_tool_registry WHERE id = $1`
	var t models.RegisteredTool
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&t.ID, &t.TenantID, &t.AppName, &t.ToolName, &t.Description,
		&t.Parameters, &t.ExecutionConfig, &t.RequiresConfirmation, &t.Enabled,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *PostgresStore) ListTools(ctx context.Context, tenantID string, appName string) ([]*models.RegisteredTool, error) {
	var rows *sql.Rows
	var err error

	if appName != "" {
		query := `SELECT id, tenant_id, app_name, tool_name, description, parameters, execution_config, requires_confirmation, enabled, created_at, updated_at
			FROM ai_tool_registry WHERE tenant_id = $1 AND app_name = $2 AND enabled = true ORDER BY app_name, tool_name`
		rows, err = s.db.QueryContext(ctx, query, tenantID, appName)
	} else {
		query := `SELECT id, tenant_id, app_name, tool_name, description, parameters, execution_config, requires_confirmation, enabled, created_at, updated_at
			FROM ai_tool_registry WHERE tenant_id = $1 AND enabled = true ORDER BY app_name, tool_name`
		rows, err = s.db.QueryContext(ctx, query, tenantID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tools []*models.RegisteredTool
	for rows.Next() {
		var t models.RegisteredTool
		if err := rows.Scan(
			&t.ID, &t.TenantID, &t.AppName, &t.ToolName, &t.Description,
			&t.Parameters, &t.ExecutionConfig, &t.RequiresConfirmation, &t.Enabled,
			&t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		tools = append(tools, &t)
	}
	return tools, nil
}

func (s *PostgresStore) UpdateTool(ctx context.Context, tool *models.RegisteredTool) error {
	query := `UPDATE ai_tool_registry SET
		description = $2, parameters = $3, execution_config = $4,
		requires_confirmation = $5, enabled = $6, updated_at = NOW()
		WHERE id = $1 RETURNING updated_at`
	return s.db.QueryRowContext(ctx, query,
		tool.ID, tool.Description, tool.Parameters, tool.ExecutionConfig,
		tool.RequiresConfirmation, tool.Enabled,
	).Scan(&tool.UpdatedAt)
}

func (s *PostgresStore) DeleteTool(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM ai_tool_registry WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}
