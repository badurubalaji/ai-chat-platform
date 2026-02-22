package claude

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (p *ClaudeProvider) parseError(resp *http.Response) error {
	var errorResp anthropicErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil && errorResp.Error.Message != "" {
		return fmt.Errorf("Claude API error (status %d): %s", resp.StatusCode, errorResp.Error.Message)
	}
	// Fallback if parsing fails
	return fmt.Errorf("Claude API error (status %d)", resp.StatusCode)
}
