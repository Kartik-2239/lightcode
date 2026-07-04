package tools

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
)

type ToolContext struct {
	WorkingDirectory string
	SessionID        string
}

type ToolFunc func(ctx ToolContext, args map[string]any) (ToolResponse, error)

type ToolDef struct {
	Name        string
	Description string
	Params      map[string]any
}

type ToolResponse struct {
	Content     string
	Printable   string
	CodeChanges []string
}

var (
	mu    sync.RWMutex
	funcs = make(map[string]ToolFunc)
	defs  = make(map[string]ToolDef)
)

func Register(name string, def ToolDef, fn ToolFunc) {
	mu.Lock()
	defer mu.Unlock()
	funcs[name] = fn
	defs[name] = def
}

func Execute(name string, ctx ToolContext, args map[string]any) (ToolResponse, error) {
	mu.RLock()
	fn := funcs[name]
	mu.RUnlock()

	if fn == nil {
		return ToolResponse{Content: "Error: tool not found", Printable: "Error: tool not found"}, nil
	}

	return fn(ctx, args)

}

func GetAllTools() []responses.ToolUnionParam {
	mu.RLock()
	defer mu.RUnlock()

	var result []responses.ToolUnionParam
	// for name, def := range defs {
	for _, name := range sortedToolNamesLocked() {
		def := defs[name]
		result = append(result, responses.ToolUnionParam{
			OfFunction: &responses.FunctionToolParam{
				Name:        name,
				Description: openai.String(def.Description),
				Parameters:  def.Params,
			},
		})
	}
	return result
}

func GetToolsForPlan() []openai.ChatCompletionToolUnionParam {
	mu.RLock()
	defer mu.RUnlock()

	var result []openai.ChatCompletionToolUnionParam
	// for name, def := range defs {
	for _, name := range sortedToolNamesLocked() {
		def := defs[name]
		if name == "write_file" || name == "edit" {
			continue
		}
		result = append(result, openai.ChatCompletionToolUnionParam{
			OfFunction: &openai.ChatCompletionFunctionToolParam{
				Type: "function",
				Function: shared.FunctionDefinitionParam{
					Name:        name,
					Description: openai.String(def.Description),
					Parameters:  def.Params,
				},
			},
		})
	}
	return result
}

func GetToolsForChat() []openai.ChatCompletionToolUnionParam {
	mu.RLock()
	defer mu.RUnlock()

	var result []openai.ChatCompletionToolUnionParam
	// for name, def := range defs {
	for _, name := range sortedToolNamesLocked() {
		def := defs[name]
		result = append(result, openai.ChatCompletionToolUnionParam{
			OfFunction: &openai.ChatCompletionFunctionToolParam{
				Type: "function",
				Function: shared.FunctionDefinitionParam{
					Name:        name,
					Description: openai.String(def.Description),
					Parameters:  def.Params,
				},
			},
		})
	}
	return result
}

func ValidatePath(ctx ToolContext, path string) (string, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(ctx.WorkingDirectory, path)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		resolved = filepath.Clean(path)
	}
	allowedDir, err := filepath.EvalSymlinks(ctx.WorkingDirectory)
	if err != nil {
		allowedDir = filepath.Clean(ctx.WorkingDirectory)
	}
	if !strings.HasPrefix(resolved, allowedDir+string(filepath.Separator)) && resolved != allowedDir {
		return "", fmt.Errorf("Access denied: path %q is outside the allowed working directory %q", path, ctx.WorkingDirectory)
	}
	return resolved, nil
}

func ParseArgs(raw string) (map[string]any, error) {
	var args map[string]any
	err := json.Unmarshal([]byte(raw), &args)
	return args, err
}

func sortedToolNamesLocked() []string {
	names := make([]string, 0, len(defs))
	for name := range defs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
