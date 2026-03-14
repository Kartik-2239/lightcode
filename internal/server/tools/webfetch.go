package tools

import (
	"io"
	"net/http"
	"time"
)

func init() {
	Register("web_fetch", ToolDef{
		Name:        "web_fetch",
		Description: "Fetch the contents of a web page",
		Params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url": map[string]string{
					"type":        "string",
					"description": "The URL to fetch",
				},
			},
			"required": []string{"url"},
		},
	}, func(args map[string]any) (string, error) {
		url, ok := args["url"].(string)
		if !ok {
			return "", nil
		}
		client := &http.Client{
			Timeout: 10 * time.Second,
		}
		resp, err := client.Get(url)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		return string(body), nil
	})
}
