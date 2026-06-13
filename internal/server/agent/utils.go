package agent

import (
	"os"
	"path"
	"strings"
)

func ReadAgentsMd(dir string) (string, error) {
	data, err := os.ReadFile(path.Join(dir, "AGENTS.md"))

	if err != nil && os.IsNotExist(err) {
		return "", nil
	}

	if err != nil {
		return "", err
	}

	// Treat a whitespace-only AGENTS.md as empty so callers never inject a blank block.
	if strings.TrimSpace(string(data)) == "" {
		return "", nil
	}

	return string(data), nil
}
