package tools

import (
	"encoding/json"
	"sync"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
)

func formatSize(size int64) string {
	if size < 1024 {
		return "<1KB"
	}
	if size < 1024*1024 {
		return "<1MB"
	}
	return "<10MB"
}

type ToolFunc func(args map[string]any) (string, error)

type ToolDef struct {
	Name        string
	Description string
	Params      map[string]any
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

func Execute(name string, args map[string]any) (string, error) {
	mu.RLock()
	fn := funcs[name]
	mu.RUnlock()

	if fn == nil {
		return "", nil
	}

	return fn(args)
}

func GetAllTools() []responses.ToolUnionParam {
	mu.RLock()
	defer mu.RUnlock()

	var result []responses.ToolUnionParam
	for name, def := range defs {
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

func GetToolsForChat() []openai.ChatCompletionToolUnionParam {
	mu.RLock()
	defer mu.RUnlock()

	var result []openai.ChatCompletionToolUnionParam
	for name, def := range defs {
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

func ParseArgs(raw string) (map[string]any, error) {
	var args map[string]any
	err := json.Unmarshal([]byte(raw), &args)
	return args, err
}
