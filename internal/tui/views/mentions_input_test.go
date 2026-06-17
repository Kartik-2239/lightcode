package views

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestActiveMention(t *testing.T) {
	cases := []struct {
		name      string
		value     string
		line, col int
		wantTok   string
		wantAt    int
		wantCur   int
		wantOK    bool
	}{
		{"end of line", "hello @foo", 0, 10, "foo", 6, 10, true},
		{"at start", "@foo", 0, 4, "foo", 0, 4, true},
		{"path with slashes", "see @a/b.go", 0, 11, "a/b.go", 4, 11, true},
		{"email is not a mention", "email@x.com", 0, 11, "", 0, 0, false},
		{"space breaks token", "@foo bar", 0, 8, "", 0, 0, false},
		{"no at sign", "just text", 0, 9, "", 0, 0, false},
		{"second line", "line1\n@foo", 1, 4, "foo", 6, 10, true},
		{"cursor mid token", "@foobar", 0, 4, "foo", 0, 4, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tok, at, cur, ok := activeMention(c.value, c.line, c.col)
			if ok != c.wantOK {
				t.Fatalf("ok = %v, want %v", ok, c.wantOK)
			}
			if !ok {
				return
			}
			if tok != c.wantTok || at != c.wantAt || cur != c.wantCur {
				t.Fatalf("got (%q, %d, %d), want (%q, %d, %d)", tok, at, cur, c.wantTok, c.wantAt, c.wantCur)
			}
		})
	}
}

func TestBuildFileIndexRespectsGitignore(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "a.go"), "package a")
	mustWrite(t, filepath.Join(dir, ".gitignore"), "node_modules/\nignored.txt\n")
	mustWrite(t, filepath.Join(dir, "ignored.txt"), "secret")
	mustWrite(t, filepath.Join(dir, "node_modules", "x.js"), "junk")
	mustWrite(t, filepath.Join(dir, "src", "main.go"), "package main")

	t.Chdir(dir)
	idx := buildFileIndex()

	if !slices.Contains(idx, "a.go") {
		t.Errorf("expected a.go in index, got %v", idx)
	}
	if !slices.Contains(idx, "src/main.go") {
		t.Errorf("expected src/main.go in index, got %v", idx)
	}
	if !slices.Contains(idx, "src/") {
		t.Errorf("expected src/ directory in index, got %v", idx)
	}
	if slices.Contains(idx, "node_modules/x.js") || slices.Contains(idx, "node_modules/") {
		t.Errorf("node_modules should be gitignored, got %v", idx)
	}
	if slices.Contains(idx, "ignored.txt") {
		t.Errorf("ignored.txt should be gitignored, got %v", idx)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
