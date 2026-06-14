package config

func getDefaultProviders() []Provider {
	return AllProviders()
}

// Providers displayed in onboarding
func AllProviders() []Provider {
	return []Provider{
		{
			BaseUrl: "https://openrouter.ai/api/v1",
			ApiKey:  "",
			Models: []string{
				"minimax/minimax-m3",
				"moonshotai/kimi-k2.7-code",
				"deepseek/deepseek-v4-pro",
			},
		},
		{
			BaseUrl: "https://api.openai.com/v1",
			ApiKey:  "",
			Models: []string{
				"gpt-5.5",
				"gpt-5.4",
				"gpt-5.3-codex",
				"gpt-5.3",
			},
		},
		{
			BaseUrl: "https://api.anthropic.com/v1",
			ApiKey:  "",
			Models: []string{
				"claude-opus-4-8",
				"claude-sonnet-4-8",
				"claude-haiku-4-5",
			},
		},
		{
			BaseUrl: "http://127.0.0.1:11434/v1",
			ApiKey:  "",
			Models:  []string{},
		},
		{
			BaseUrl: "https://api.minimax.io/v1",
			ApiKey:  "",
			Models:  []string{},
		},
	}
}

func ProviderByName(name string) (Provider, bool) {
	for _, p := range AllProviders() {
		if p.Name() == name {
			return p, true
		}
	}
	return Provider{}, false
}

func (p Provider) Name() string {
	switch p.BaseUrl {
	case "https://openrouter.ai/api/v1":
		return "openrouter"
	case "https://api.openai.com/v1":
		return "openai"
	case "https://api.anthropic.com/v1":
		return "anthropic"
	case "http://127.0.0.1:11434/v1":
		return "ollama"
	default:
		return p.BaseUrl
	}
}
