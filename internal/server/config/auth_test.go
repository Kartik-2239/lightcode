package config

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReadCodexAuth(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.json")
	token := testJWT(t, 2000000000)
	writeFile(t, path, map[string]any{
		"auth_mode":      "chatgpt",
		"OPENAI_API_KEY": nil,
		"tokens": map[string]any{
			"access_token":  token,
			"refresh_token": "refresh-token",
			"account_id":    "account-id",
		},
	})

	authVal, err := ReadCodexAuth(path)
	if err != nil {
		t.Fatalf("ReadCodexAuth returned error: %v", err)
	}

	if authVal.Type != "oauth" {
		t.Fatalf("expected oauth type, got %q", authVal.Type)
	}
	if authVal.Access != token {
		t.Fatalf("access token was not imported")
	}
	if authVal.Refresh != "refresh-token" {
		t.Fatalf("refresh token was not imported")
	}
	if authVal.AccountId != "account-id" {
		t.Fatalf("account id was not imported")
	}
	if authVal.Expires != 1999999700 {
		t.Fatalf("expected expiry with refresh buffer, got %d", authVal.Expires)
	}
	if len(authVal.Models) == 0 || authVal.Models[0] != "gpt-5.5" {
		t.Fatalf("expected default Codex models, got %#v", authVal.Models)
	}
}

func TestImportCodexAuthFromPathCreatesLightcodeAuth(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	codexPath := filepath.Join(home, ".codex", "auth.json")
	token := testJWT(t, 2000000000)
	writeFile(t, codexPath, map[string]any{
		"auth_mode": "chatgpt",
		"tokens": map[string]any{
			"access_token":  token,
			"refresh_token": "refresh-token",
			"account_id":    "account-id",
		},
	})

	if err := ImportCodexAuthFromPath(codexPath); err != nil {
		t.Fatalf("ImportCodexAuthFromPath returned error: %v", err)
	}

	authVal, err := GetAuthVal(CodexAuthProvider)
	if err != nil {
		t.Fatalf("GetAuthVal returned error: %v", err)
	}
	if authVal.Access != token || authVal.Refresh != "refresh-token" {
		t.Fatalf("imported auth did not round trip")
	}

	info, err := os.Stat(filepath.Join(home, ".lightcode", "auth.json"))
	if err != nil {
		t.Fatalf("expected lightcode auth.json: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("expected auth.json permissions 0600, got %o", got)
	}
}

func TestImportCodexAuthPreservesExistingModels(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := UpdateAuthVal(CodexAuthProvider, AuthVal{
		Type:    "oauth",
		Access:  "old-access",
		Refresh: "old-refresh",
		Models:  []string{"custom-codex-model"},
	}); err != nil {
		t.Fatalf("UpdateAuthVal returned error: %v", err)
	}

	codexPath := filepath.Join(home, ".codex", "auth.json")
	writeFile(t, codexPath, map[string]any{
		"auth_mode": "chatgpt",
		"tokens": map[string]any{
			"access_token":  testJWT(t, 2000000000),
			"refresh_token": "refresh-token",
			"account_id":    "account-id",
		},
	})

	if err := ImportCodexAuthFromPath(codexPath); err != nil {
		t.Fatalf("ImportCodexAuthFromPath returned error: %v", err)
	}
	authVal, err := GetAuthVal(CodexAuthProvider)
	if err != nil {
		t.Fatalf("GetAuthVal returned error: %v", err)
	}
	if len(authVal.Models) != 1 || authVal.Models[0] != "custom-codex-model" {
		t.Fatalf("expected existing models to be preserved, got %#v", authVal.Models)
	}
}

func TestSetCurrentModelClearsAuthBackedApiKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	err := CreateConfig([]string{"codex"}, map[string]string{}, map[string]string{}, map[string][]string{})
	if err != nil {
		t.Fatalf("CreateConfig returned error: %v", err)
	}

	err = SetCurrentModel(ResModel{
		Model:   "gpt-5.5",
		ApiKey:  "not-an-api-key",
		BaseUrl: CodexAuthProvider,
	})
	if err != nil {
		t.Fatalf("SetCurrentModel returned error: %v", err)
	}

	current, err := GetCurrentModel()
	if err != nil {
		t.Fatalf("GetCurrentModel returned error: %v", err)
	}
	if current.ApiKey != "" {
		t.Fatalf("expected auth-backed current model api key to be empty, got %q", current.ApiKey)
	}

	recents := GetRecentModels()
	if len(recents) != 1 || recents[0].ApiKey != "" {
		t.Fatalf("expected auth-backed recent api key to be empty, got %#v", recents)
	}
}

func TestClearModelProviderClearsCurrentAndRecent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	err := CreateConfig([]string{"codex"}, map[string]string{}, map[string]string{}, map[string][]string{})
	if err != nil {
		t.Fatalf("CreateConfig returned error: %v", err)
	}
	if err := SetCurrentModel(ResModel{Model: "gpt-5.5", BaseUrl: CodexAuthProvider}); err != nil {
		t.Fatalf("SetCurrentModel returned error: %v", err)
	}

	if err := ClearModelProvider(CodexAuthProvider); err != nil {
		t.Fatalf("ClearModelProvider returned error: %v", err)
	}

	customization := GetCustomization()
	if customization.CurrentModel.Model != "" {
		t.Fatalf("expected current model to be cleared, got %#v", customization.CurrentModel)
	}
	if len(customization.RecentModels) != 0 {
		t.Fatalf("expected recent models to be cleared, got %#v", customization.RecentModels)
	}
}

func TestReconcileAuthProviderModelsReplacesUnsupportedCurrentModel(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := CreateConfig([]string{"copilot"}, map[string]string{}, map[string]string{}, map[string][]string{}); err != nil {
		t.Fatalf("CreateConfig returned error: %v", err)
	}
	if err := SetCurrentModel(ResModel{Model: "GPT-4o", BaseUrl: CopilotAuthProvider}); err != nil {
		t.Fatalf("SetCurrentModel returned error: %v", err)
	}
	if err := ReconcileAuthProviderModels(CopilotAuthProvider, []string{"GPT-5.5", "Claude Sonnet 4.6"}); err != nil {
		t.Fatalf("ReconcileAuthProviderModels returned error: %v", err)
	}
	current := GetCustomization().CurrentModel
	if current.Model != "GPT-5.5" || current.BaseUrl != CopilotAuthProvider {
		t.Fatalf("expected current model to move to first supported model, got %#v", current)
	}
	for _, recent := range GetCustomization().RecentModels {
		if recent.BaseUrl == CopilotAuthProvider && recent.Model == "GPT-4o" {
			t.Fatalf("expected unsupported recent model to be removed, got %#v", GetCustomization().RecentModels)
		}
	}
}

func TestSetReasoningEffortForCodexModel(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	err := CreateConfig([]string{"codex"}, map[string]string{}, map[string]string{}, map[string][]string{})
	if err != nil {
		t.Fatalf("CreateConfig returned error: %v", err)
	}
	if err := SetCurrentModel(ResModel{Model: "gpt-5.5", BaseUrl: CodexAuthProvider}); err != nil {
		t.Fatalf("SetCurrentModel returned error: %v", err)
	}

	if err := SetReasoningEffort("xhigh"); err != nil {
		t.Fatalf("SetReasoningEffort returned error: %v", err)
	}

	current := GetCustomization().CurrentModel
	if current.ReasoningEffort != "xhigh" {
		t.Fatalf("expected xhigh effort, got %#v", current)
	}
	recents := GetRecentModels()
	if len(recents) != 1 || recents[0].ReasoningEffort != "xhigh" {
		t.Fatalf("expected recent effort to be updated, got %#v", recents)
	}
}

func TestSetReasoningEffortRejectsUnsupportedModel(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	err := CreateConfig([]string{"openai"}, map[string]string{"openai": "sk-test"}, map[string]string{}, map[string][]string{})
	if err != nil {
		t.Fatalf("CreateConfig returned error: %v", err)
	}
	if err := SetReasoningEffort("high"); err == nil {
		t.Fatalf("expected unsupported model error")
	}
}

func testJWT(t *testing.T, exp int64) string {
	t.Helper()
	encode := func(v any) string {
		data, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("json marshal failed: %v", err)
		}
		return base64.RawURLEncoding.EncodeToString(data)
	}
	return encode(map[string]string{"alg": "none"}) + "." + encode(map[string]int64{"exp": exp}) + ".sig"
}

func writeFile(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("json marshal failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write failed: %v", err)
	}
}
