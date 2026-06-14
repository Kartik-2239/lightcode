package config

import (
	"path/filepath"
	"testing"
)

func TestSkillsPathUsesEnvOverride(t *testing.T) {
	home := t.TempDir()
	skills := filepath.Join(home, "skills")
	t.Setenv("HOME", home)
	t.Setenv("SKILL_PATH", "  "+skills+"  ")

	if got := SkillsPath(); got != skills {
		t.Fatalf("expected SKILL_PATH override %q, got %q", skills, got)
	}
}
