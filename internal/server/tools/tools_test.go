package tools

import (
	"strings"
	"testing"
)

func TestTools(t *testing.T) {
	ctx := ToolContext{WorkingDirectory: "/Users/kartikkannan/Desktop/lightcode/internal/server/tools"}
	// testFile := "test_unit.txt"

	// Test WriteFile
	// t.Run("WriteFile", func(t *testing.T) {
	// 	res, err := WriteFile(ctx, map[string]any{"path": testFile, "content": "hello world"})
	// 	if err != nil {
	// 		t.Errorf("WriteFile failed: %v", err)
	// 	}
	// 	if res != "File written successfully" {
	// 		t.Errorf("WriteFile unexpected result: %s", res)
	// 	}
	// })

	// // Test ReadFile
	// t.Run("ReadFile", func(t *testing.T) {
	// 	res, err := ReadFile(ctx, map[string]any{"path": testFile})
	// 	if err != nil {
	// 		t.Errorf("ReadFile failed: %v", err)
	// 	}
	// 	if res != "hello world" {
	// 		t.Errorf("ReadFile got %q, want %q", res, "hello world")
	// 	}
	// })

	// // Test Edit
	// t.Run("Edit", func(t *testing.T) {
	// 	_, err := Edit(ctx, map[string]any{
	// 		"filePath":   testFile,
	// 		"oldString":  "world",
	// 		"newString":  "universe",
	// 		"replaceAll": 1.0,
	// 	})
	// 	if err != nil {
	// 		t.Errorf("Edit failed: %v", err)
	// 	}

	// 	res, _ := ReadFile(ctx, map[string]any{"path": testFile})
	// 	if res != "hello universe" {
	// 		t.Errorf("After Edit, ReadFile got %q, want %q", res, "hello universe")
	// 	}
	// })

	// // Test ListDir
	// t.Run("ListDir", func(t *testing.T) {
	// 	res, err := ListDir(ctx, map[string]any{"path": "."})
	// 	if err != nil {
	// 		t.Errorf("ListDir failed: %v", err)
	// 	}
	// 	if !strings.Contains(res, testFile) {
	// 		t.Errorf("ListDir output does not contain %s", testFile)
	// 	}
	// })

	// // Test Bash
	t.Run("Bash", func(t *testing.T) {
		res, err := Bash(ctx, map[string]any{"command": "ffmpeg -i luf.jpg out.jpg"})
		if err != nil {
			t.Errorf("Bash failed: %v", err)
		}
		if !strings.Contains(res, "out.jpg") {
			t.Errorf("Bash output got %q, want %q", res, "out.jpg")
		}
	})

	// // Test Glob
	// t.Run("Glob", func(t *testing.T) {
	// 	res, err := Glob(ctx, map[string]any{"path": ".", "pattern": testFile})
	// 	if err != nil {
	// 		t.Errorf("Glob failed: %v", err)
	// 	}
	// 	if !strings.Contains(res, testFile) {
	// 		t.Errorf("Glob output does not contain %s", testFile)
	// 	}
	// })

	// // Test Grep
	// t.Run("Grep", func(t *testing.T) {
	// 	res, err := Grep(ctx, map[string]any{"path": ".", "pattern": "universe", "include": testFile})
	// 	if err != nil {
	// 		t.Errorf("Grep failed: %v", err)
	// 	}
	// 	if !strings.Contains(res, testFile) {
	// 		t.Errorf("Grep output does not contain %s", testFile)
	// 	}
	// })

	// // Clean up
	// os.Remove(testFile)
}
