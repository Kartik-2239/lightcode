package tools

import (
	"io"
	"net/http"
	"time"
)

func WebFetch(ctx ToolContext, args map[string]any) (ToolResponse, error) {
	url, ok := args["url"].(string)
	if !ok {
		return ToolResponse{Content: "Error: url is required and must be a string", Printable: "Error: url is required and must be a string"}, nil
	}
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil {
		return ToolResponse{Content: "Error: " + err.Error(), Printable: "Error: " + err.Error()}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ToolResponse{Content: "Error: " + err.Error(), Printable: "Error: " + err.Error()}, err
	}
	return ToolResponse{Content: string(body), Printable: string(body)}, nil
}

// func init() {
// 	Register("web_fetch", ToolDef{
// 		Name:        "web_fetch",
// 		Description: "Fetch the contents of a web page",
// 		Params: map[string]any{
// 			"type": "object",
// 			"properties": map[string]any{
// 				"url": map[string]string{
// 					"type":        "string",
// 					"description": "The URL to fetch",
// 				},
// 			},
// 			"required": []string{"url"},
// 		},
// 	}, WebFetch)
// }
