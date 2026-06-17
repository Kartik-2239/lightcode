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
			Models:  []string{"MiniMax-M3", "MiniMax-M2.7", "MiniMax-M2.7-highspeed", "MiniMax-M2.5", "MiniMax-M2.5-highspeed"},
		},
		{
			BaseUrl: "https://api.z.ai/api/paas/v4/",
			ApiKey:  "",
			Models:  []string{"glm-5.1", "glm-5", "glm-4.7", "glm-4.7-flash"},
		},
		{
			BaseUrl: "https://api.moonshot.ai/v1",
			ApiKey:  "",
			Models:  []string{"kimi-k2.7", "kimi-k2.6", "kimi-k2.5", "kimi-k2"},
		},
		{
			BaseUrl: "https://api.xiaomimimo.com/v1",
			ApiKey:  "",
			Models:  []string{"mimo-v2.5-pro", "mimo-v2.5", "mimo-v2-pro"},
		},
		{
			BaseUrl: "https://api.xiaomimimo.com/v1",
			ApiKey:  "",
			Models:  []string{"mimo-v2.5-pro", "mimo-v2.5", "mimo-v2-pro"},
		},
		{
			BaseUrl: "https://token-plan-sgp.xiaomimimo.com/v1",
			ApiKey:  "",
			Models:  []string{"mimo-v2.5-pro", "mimo-v2.5", "mimo-v2-pro"},
		},
		{
			BaseUrl: "https://token-plan-ams.xiaomimimo.com/v1",
			ApiKey:  "",
			Models:  []string{"mimo-v2.5-pro", "mimo-v2.5", "mimo-v2-pro"},
		},
		{
			BaseUrl: "https://token-plan-cn.xiaomimimo.com/v1",
			ApiKey:  "",
			Models:  []string{"mimo-v2.5-pro", "mimo-v2.5", "mimo-v2-pro"},
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
	case "https://api.minimax.io/v1":
		return "minimax"
	case "https://api.z.ai/api/paas/v4/":
		return "zai"
	case "https://api.moonshot.ai/v1":
		return "moonshot"
	case "https://api.xiaomimimo.com/v1":
		return "mimo"
	case "https://token-plan-sgp.xiaomimimo.com/v1":
		return "mimo-sgp"
	case "https://token-plan-ams.xiaomimimo.com/v1":
		return "mimo-ams"
	case "https://token-plan-cn.xiaomimimo.com/v1":
		return "mimo-cn"
	default:
		return p.BaseUrl
	}
}
