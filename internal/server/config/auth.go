package config

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type AuthVal struct {
	Type      string   `json:"type"`
	Access    string   `json:"access_token"`
	Refresh   string   `json:"refresh_token"`
	Expires   int64    `json:"expires"`
	AccountId string   `json:"account_id"`
	Models    []string `json:"models"`
}

type AuthJson map[string]AuthVal

const (
	CodexAuthProvider   = "codex"
	CopilotAuthProvider = "copilot"
)

func DefaultCodexModels() []string {
	return []string{"gpt-5.5", "gpt-5.4-mini", "gpt-5.3-codex-spark"}
}

func GetAuthVal(provider string) (AuthVal, error) {
	authJson, err := readAuthJson()
	if err != nil {
		if provider == CodexAuthProvider {
			if importErr := ImportCodexAuth(); importErr != nil {
				return AuthVal{}, importErr
			}
			authJson, err = readAuthJson()
		}
		if err != nil {
			return AuthVal{}, err
		}
	}

	authVal, ok := authJson[provider]
	if !ok {
		if provider == CodexAuthProvider {
			if importErr := ImportCodexAuth(); importErr != nil {
				return AuthVal{}, importErr
			}
			authJson, err = readAuthJson()
			if err != nil {
				return AuthVal{}, err
			}
			authVal, ok = authJson[provider]
			if ok {
				return authVal, nil
			}
		}
		return AuthVal{}, fmt.Errorf("auth provider %q not found in auth.json", provider)
	}

	return authVal, nil
}

func UpdateAuthVal(provider string, newVal AuthVal) error {
	authJson, err := readAuthJson()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		authJson = AuthJson{}
	}

	authJson[provider] = newVal

	return writeAuthJson(authJson)
}

func DeleteAuthVal(provider string) error {
	authJson, err := readAuthJson()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	delete(authJson, provider)
	return writeAuthJson(authJson)
}

func GetAllAuthModels() ([]ResModel, error) {
	authJson, err := readAuthJson()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		if importErr := ImportCodexAuth(); importErr != nil {
			return nil, nil
		}
		authJson, err = readAuthJson()
		if err != nil {
			return nil, nil
		}
	}
	if _, ok := authJson[CodexAuthProvider]; !ok {
		if err := ImportCodexAuth(); err == nil {
			if refreshed, readErr := readAuthJson(); readErr == nil {
				authJson = refreshed
			}
		}
	}

	keys := make([]string, 0, len(authJson))
	for k := range authJson {
		keys = append(keys, k)
	}
	var models []ResModel

	for _, au := range keys {
		authVal := authJson[au]
		for _, m := range authVal.Models {
			// fmt.Printf("Adding model from auth: %s with provider %s\n", m, au)
			models = append(models, ResModel{Model: m, ApiKey: "", BaseUrl: au})
		}
	}
	return models, nil
}

func HasAnyAuthModel() bool {
	models, err := GetAllAuthModels()
	return err == nil && len(models) > 0
}

func ImportCodexAuth() error {
	return ImportCodexAuthFromPath(defaultCodexAuthPath())
}

func ImportCodexAuthFromPath(path string) error {
	authVal, err := ReadCodexAuth(path)
	if err != nil {
		return err
	}

	existing, err := readAuthJson()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		existing = AuthJson{}
	}
	if current, ok := existing[CodexAuthProvider]; ok && len(current.Models) > 0 {
		authVal.Models = current.Models
	}
	existing[CodexAuthProvider] = authVal

	return writeAuthJson(existing)
}

func ReadCodexAuth(path string) (AuthVal, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return AuthVal{}, err
	}

	var raw struct {
		AuthMode     string  `json:"auth_mode"`
		OpenAIAPIKey *string `json:"OPENAI_API_KEY"`
		Tokens       *struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			AccountID    string `json:"account_id"`
		} `json:"tokens"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return AuthVal{}, err
	}
	if raw.AuthMode != "" && raw.AuthMode != "chatgpt" {
		return AuthVal{}, fmt.Errorf("codex auth cache uses %q auth, expected chatgpt", raw.AuthMode)
	}
	if raw.OpenAIAPIKey != nil && strings.TrimSpace(*raw.OpenAIAPIKey) != "" {
		return AuthVal{}, fmt.Errorf("codex auth cache contains API-key auth, not ChatGPT OAuth tokens")
	}
	if raw.Tokens == nil || raw.Tokens.AccessToken == "" || raw.Tokens.RefreshToken == "" {
		return AuthVal{}, fmt.Errorf("codex auth cache does not contain file-based ChatGPT OAuth tokens")
	}

	return AuthVal{
		Type:      "oauth",
		Access:    raw.Tokens.AccessToken,
		Refresh:   raw.Tokens.RefreshToken,
		Expires:   jwtExpiresAt(raw.Tokens.AccessToken),
		AccountId: raw.Tokens.AccountID,
		Models:    DefaultCodexModels(),
	}, nil
}

func defaultCodexAuthPath() string {
	if codexHome := strings.TrimSpace(os.Getenv("CODEX_HOME")); codexHome != "" {
		return filepath.Join(codexHome, "auth.json")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".codex", "auth.json")
}

func readAuthJson() (AuthJson, error) {
	path, err := GetAuthPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var authJson AuthJson
	if err := json.Unmarshal(data, &authJson); err != nil {
		return nil, err
	}
	return authJson, nil
}

func writeAuthJson(authJson AuthJson) error {
	updatedData, err := json.MarshalIndent(authJson, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(Dir(), "auth.json")
	return os.WriteFile(path, updatedData, 0600)
}

func jwtExpiresAt(token string) int64 {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return 0
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Exp <= 0 {
		return 0
	}
	return claims.Exp - int64((5 * time.Minute).Seconds())
}
