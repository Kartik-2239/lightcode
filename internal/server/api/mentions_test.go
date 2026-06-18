package api

import (
	"errors"
	"strings"
	"testing"

	"github.com/Kartik-2239/lightcode/internal/server/tools"
)

func TestMentionToolOutput(t *testing.T) {
	cases := []struct {
		name    string
		res     tools.ToolResponse
		err     error
		wantHas string
	}{
		{
			name:    "validation error in content",
			res:     tools.ToolResponse{Content: "Error: Access denied"},
			wantHas: "Error: Access denied",
		},
		{
			name:    "os error with message in content",
			res:     tools.ToolResponse{Content: "Error: open /no/such/file: no such file or directory"},
			err:     errors.New("open /no/such/file: no such file or directory"),
			wantHas: "Error: open /no/such/file",
		},
		{
			name:    "os error empty content",
			res:     tools.ToolResponse{},
			err:     errors.New("permission denied"),
			wantHas: "Error: permission denied",
		},
		{
			name:    "success empty file",
			res:     tools.ToolResponse{Content: ""},
			wantHas: "",
		},
		{
			name:    "success with content",
			res:     tools.ToolResponse{Content: "package main\n"},
			wantHas: "package main",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := mentionToolOutput(c.res, c.err)
			if c.wantHas == "" {
				if got != "" {
					t.Fatalf("got %q, want empty", got)
				}
				return
			}
			if !strings.Contains(got, c.wantHas) {
				t.Fatalf("got %q, want substring %q", got, c.wantHas)
			}
		})
	}
}

func TestExpandMentionsMissingFile(t *testing.T) {
	dir := t.TempDir()
	got := expandMentions("check this file", []string{"missing.go"}, "sess", dir)
	if !strings.Contains(got, "Error:") {
		t.Fatalf("expected error in expanded content, got:\n%s", got)
	}
	if !strings.Contains(got, "missing.go") {
		t.Fatalf("expected path in tool block, got:\n%s", got)
	}
}
