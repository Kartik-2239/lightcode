package views

import "github.com/Kartik-2239/lightcode/internal/server/db/models"

func shouldRefreshGitAfterToolCall(msg models.StoredMessageData) bool {
	if len(msg.CodeChanges) > 0 {
		return true
	}
	for _, call := range msg.ToolCalls {
		switch call.Name {
		case "bash", "edit", "write_file":
			return true
		}
	}
	return false
}
