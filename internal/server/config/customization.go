package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ```bash
// OPENAI_API_KEY=sk-...
// OPENAI_BASE_URL=https://...
// SKILL_PATH=path/to/skill/folder
// API_URL=http://localhost:8080
// ```
type Customization struct {
	SkillsPath   string         `json:"skills_path"`
	Port         string         `json:"port"`
	Theme        string         `json:"theme"`
	Providers    []Provider     `json:"providers"`
	CurrentModel ResModel       `json:"current_model"`
	RecentModels []RecentModels `json:"recent_models"`
}

type Provider struct {
	BaseUrl string   `json:"base_url"`
	ApiKey  string   `json:"api_key"`
	Models  []string `json:"models"`
}

type ResModel struct {
	Model           string `json:"model"`
	ApiKey          string `json:"api_key"`
	BaseUrl         string `json:"base_url"`
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
}
type RecentModels struct {
	Model           string `json:"model"`
	ApiKey          string `json:"api_key"`
	BaseUrl         string `json:"base_url"`
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
	LastUsed        int64  `json:"last_used"`
}
type AllModels struct {
	Models       []ResModel
	RecentModels []RecentModels
}

// CreateConfig writes a fresh config.json containing only the chosen providers,
// filling in any API keys collected during onboarding.
// This is the only place that materializes config.json
// reads no longer create it.
func CreateConfig(providerNames []string, keys map[string]string, baseUrls map[string]string, models map[string][]string) error {
	providers := []Provider{}
	for _, name := range providerNames {
		if name == CodexAuthProvider {
			_ = ImportCodexAuth()
			continue
		}
		if p, ok := ProviderByName(name); ok {
			p.ApiKey = keys[name]
			providers = append(providers, p)
			continue
		}
		// custom ("other") provider: user supplied the base URL and models
		if url := baseUrls[name]; url != "" {
			m := models[name]
			if m == nil {
				m = []string{}
			}
			providers = append(providers, Provider{
				BaseUrl: url,
				ApiKey:  keys[name],
				Models:  m,
			})
		}
	}
	if len(providers) == 0 {
		if _, err := GetAuthVal(CodexAuthProvider); err != nil {
			providers = AllProviders()
		}
	}
	// default the current model to the first provider that actually has models
	first := ResModel{}
	for _, p := range providers {
		if len(p.Models) > 0 {
			first = ResModel{Model: p.Models[0], BaseUrl: p.BaseUrl, ApiKey: p.ApiKey}
			break
		}
	}
	if first.Model == "" {
		if authModels, err := GetAllAuthModels(); err == nil && len(authModels) > 0 {
			first = authModels[0]
		}
	}
	bare := Customization{
		Theme:        "light",
		SkillsPath:   filepath.Join(Dir(), "skills"),
		Port:         "8080",
		Providers:    providers,
		CurrentModel: first,
	}
	path := filepath.Join(Dir(), "config.json")
	d, err := json.MarshalIndent(bare, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, d, 0644)
}

func defaultCustomization() Customization {
	providers := getDefaultProviders()
	model := ResModel{}
	if len(providers) > 0 && len(providers[0].Models) > 0 {
		model = ResModel{Model: providers[0].Models[0], BaseUrl: providers[0].BaseUrl}
	}
	return Customization{
		Theme:        "light",
		SkillsPath:   filepath.Join(Dir(), "skills"),
		Port:         "8080",
		Providers:    providers,
		CurrentModel: model,
	}
}

// CustomizationPath returns the config.json path. It no longer creates the file
// — onboarding owns creation (see CreateConfig).
func CustomizationPath() (string, error) {
	return filepath.Join(Dir(), "config.json"), nil
}

func GetCustomization() Customization {
	path, err := CustomizationPath()
	if err != nil {
		return defaultCustomization()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultCustomization()
	}
	var customization Customization
	if err := json.Unmarshal(data, &customization); err != nil {
		return defaultCustomization()
	}
	return customization
}

func GetTheme() string {
	return GetCustomization().Theme
}

func SetTheme(theme string) error {
	customization := GetCustomization()
	customization.Theme = theme
	d, err := json.MarshalIndent(customization, "", " ")
	if err != nil {
		return errors.New("Error Setting theme")
	}
	path, err := CustomizationPath()
	if err != nil {
		return errors.New("Error Setting theme")
	}
	err = os.WriteFile(path, d, 0644)
	if err != nil {
		return errors.New("Error Setting theme")
	}
	return nil
}

func GetModels() ([]ResModel, []RecentModels, error) {
	providers := GetCustomization().Providers
	models := []ResModel{}
	for _, provider := range providers {
		for _, model := range provider.Models {
			models = append(models, ResModel{
				ApiKey:  provider.ApiKey,
				BaseUrl: provider.BaseUrl,
				Model:   model,
			})
		}
	}
	return models, GetCustomization().RecentModels, nil
}

func SetApiKey(m ResModel, apikey string) error {
	customization := GetCustomization()
	for i := range customization.Providers {
		if customization.Providers[i].BaseUrl == m.BaseUrl {
			customization.Providers[i].ApiKey = apikey
		}
	}
	if customization.CurrentModel.BaseUrl == m.BaseUrl {
		customization.CurrentModel.ApiKey = apikey
	}
	for i := range customization.RecentModels {
		if customization.RecentModels[i].BaseUrl == m.BaseUrl {
			customization.RecentModels[i].ApiKey = apikey
		}
	}
	d, err := json.MarshalIndent(customization, "", " ")
	if err != nil {
		return errors.New("Error Setting api key")
	}
	path, err := CustomizationPath()
	if err != nil {
		return errors.New("Error Setting current model")
	}
	err = os.WriteFile(path, d, 0644)
	if err != nil {
		return errors.New("Error Setting current model")
	}
	return nil
}

func GetRecentModels() []RecentModels {
	customization := GetCustomization()
	return customization.RecentModels
}

func SetCurrentModel(model ResModel) error {
	customization := GetCustomization()
	if isAuthBackedBaseURL(model.BaseUrl) {
		model.ApiKey = ""
	}
	customization.CurrentModel = model
	r := customization.RecentModels
	changed := false
	if len(r) != 0 {
		for i, m := range r {
			if m.BaseUrl == model.BaseUrl && m.Model == model.Model {
				r[i].LastUsed = time.Now().Unix()
				r[i].ApiKey = model.ApiKey
				r[i].ReasoningEffort = model.ReasoningEffort
				changed = true
			}
		}
	}

	if !changed {
		r = append(r, RecentModels{
			Model:           model.Model,
			ApiKey:          model.ApiKey,
			BaseUrl:         model.BaseUrl,
			ReasoningEffort: model.ReasoningEffort,
			LastUsed:        time.Now().Unix(),
		})
	}
	customization.RecentModels = r
	d, err := json.MarshalIndent(customization, "", " ")
	if err != nil {
		return errors.New("Error Setting current model")
	}
	path, err := CustomizationPath()
	if err != nil {
		return errors.New("Error Setting current model")
	}
	err = os.WriteFile(path, d, 0644)
	if err != nil {
		return errors.New("Error Setting current model")
	}
	return nil
}

func SetReasoningEffort(effort string) error {
	effort = strings.TrimSpace(effort)
	if !IsReasoningEffort(effort) {
		return errors.New("unsupported reasoning effort")
	}

	customization := GetCustomization()
	if customization.CurrentModel.Model == "" {
		return errors.New("no current model selected")
	}
	if !SupportsReasoningEffort(customization.CurrentModel) {
		return errors.New("current model does not support reasoning effort")
	}

	customization.CurrentModel.ReasoningEffort = effort
	for i := range customization.RecentModels {
		if customization.RecentModels[i].BaseUrl == customization.CurrentModel.BaseUrl &&
			customization.RecentModels[i].Model == customization.CurrentModel.Model {
			customization.RecentModels[i].ReasoningEffort = effort
		}
	}

	d, err := json.MarshalIndent(customization, "", " ")
	if err != nil {
		return errors.New("Error setting reasoning effort")
	}
	path, err := CustomizationPath()
	if err != nil {
		return errors.New("Error setting reasoning effort")
	}
	if err := os.WriteFile(path, d, 0644); err != nil {
		return errors.New("Error setting reasoning effort")
	}
	return nil
}

func IsReasoningEffort(effort string) bool {
	switch effort {
	case "", "low", "medium", "high", "xhigh":
		return true
	default:
		return false
	}
}

func SupportsReasoningEffort(model ResModel) bool {
	return model.BaseUrl == CodexAuthProvider
}

func ClearModelProvider(baseURL string) error {
	customization := GetCustomization()
	if customization.CurrentModel.BaseUrl == baseURL {
		customization.CurrentModel = ResModel{}
	}
	recents := make([]RecentModels, 0, len(customization.RecentModels))
	for _, model := range customization.RecentModels {
		if model.BaseUrl != baseURL {
			recents = append(recents, model)
		}
	}
	customization.RecentModels = recents

	d, err := json.MarshalIndent(customization, "", " ")
	if err != nil {
		return errors.New("Error clearing model provider")
	}
	path, err := CustomizationPath()
	if err != nil {
		return errors.New("Error clearing model provider")
	}
	if err := os.WriteFile(path, d, 0644); err != nil {
		return errors.New("Error clearing model provider")
	}
	return nil
}

func ReconcileAuthProviderModels(baseURL string, models []string) error {
	allowed := make(map[string]struct{}, len(models))
	for _, model := range models {
		trimmed := strings.TrimSpace(model)
		if trimmed != "" {
			allowed[trimmed] = struct{}{}
		}
	}

	customization := GetCustomization()
	if customization.CurrentModel.BaseUrl == baseURL {
		if _, ok := allowed[customization.CurrentModel.Model]; !ok {
			customization.CurrentModel = ResModel{}
			if len(models) > 0 {
				customization.CurrentModel = ResModel{Model: models[0], BaseUrl: baseURL}
			}
		}
	}

	recents := make([]RecentModels, 0, len(customization.RecentModels))
	for _, model := range customization.RecentModels {
		if model.BaseUrl == baseURL {
			if _, ok := allowed[model.Model]; !ok {
				continue
			}
		}
		recents = append(recents, model)
	}
	customization.RecentModels = recents

	d, err := json.MarshalIndent(customization, "", " ")
	if err != nil {
		return errors.New("Error reconciling model provider")
	}
	path, err := CustomizationPath()
	if err != nil {
		return errors.New("Error reconciling model provider")
	}
	if err := os.WriteFile(path, d, 0644); err != nil {
		return errors.New("Error reconciling model provider")
	}
	return nil
}

func isAuthBackedBaseURL(baseURL string) bool {
	trimmed := strings.TrimSpace(baseURL)
	return trimmed != "" && !strings.HasPrefix(trimmed, "http")
}

func HasAnyApiKey() bool {
	for _, p := range GetCustomization().Providers {
		if p.ApiKey != "" {
			return true
		}
	}
	return false
}

func GetCurrentModel() (ResModel, error) {
	customization := GetCustomization()
	if customization.CurrentModel.Model != "" {
		return customization.CurrentModel, nil
	}
	models, recent_models, err := GetModels()
	if err != nil {
		return ResModel{}, err
	}
	sort.Slice(recent_models, func(i, j int) bool {
		return recent_models[i].LastUsed > recent_models[j].LastUsed
	})
	if len(recent_models) > 0 {
		r := recent_models[0]
		return ResModel{Model: r.Model, ApiKey: r.ApiKey, BaseUrl: r.BaseUrl, ReasoningEffort: r.ReasoningEffort}, nil
	} else if len(models) > 0 {
		return models[0], nil
	}
	return ResModel{}, errors.New("no models configured")
}
