package api

import (
	"math/rand"
	"net/url"
	"strings"
)

func randomSessionID() string {
	var chars = "qwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNM1234567890-_"
	length := 10
	var result strings.Builder
	for range length {
		result.WriteString(string(chars[rand.Intn(len(chars))]))
	}
	return result.String()
}

func providerFromBaseUrl(baseUrl string) string {
	trimmed := strings.TrimSpace(baseUrl)
	if trimmed == "" {
		return ""
	}
	if !strings.Contains(trimmed, "://") {
		return baseUrl
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return strings.TrimSpace(baseUrl)
	}

	host := parsed.Hostname()
	if host == "" {
		return strings.TrimSpace(baseUrl)
	}

	parts := strings.Split(host, ".")
	if len(parts) == 1 {
		return parts[0]
	}
	if parts[0] == "api" || parts[0] == "chatgpt" {
		return parts[1]
	}
	return parts[0]
}

func providerLabelFromModel(m ModelInfo) string {
	if strings.TrimSpace(m.Provider) != "" {
		return m.Provider
	}
	return providerLabelFromBaseUrl(m.BaseUrl)
}

func providerLabelFromBaseUrl(baseUrl string) string {
	trimmed := strings.TrimSpace(baseUrl)
	if trimmed != "" && !strings.Contains(trimmed, "://") {
		return trimmed + " auth"
	}
	return providerFromBaseUrl(trimmed)
}
