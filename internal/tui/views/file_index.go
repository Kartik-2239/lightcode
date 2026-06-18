package views

import (
	"io/fs"
	"os"
	"path/filepath"

	ignore "github.com/sabhiram/go-gitignore"
)

const maxIndexedFiles = 5000

// buildFileIndex walks the current working directory and returns workspace-
// relative paths (forward-slashed) for @file mention completion. Directories are
// included with a trailing "/". It honors the repo-root .gitignore and always
// skips the .git directory, and caps results to keep the picker responsive.
func buildFileIndex() []string {
	workingDir, err := os.Getwd()
	if err != nil || workingDir == "" {
		workingDir = "."
	}

	var gi *ignore.GitIgnore
	if g, gerr := ignore.CompileIgnoreFile(filepath.Join(workingDir, ".gitignore")); gerr == nil {
		gi = g
	}

	var paths []string
	filepath.WalkDir(workingDir, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		rel, relErr := filepath.Rel(workingDir, p)
		if relErr != nil || rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)

		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		// Directory patterns in .gitignore (e.g. "node_modules/") only match a
		// path with a trailing slash, so test directories in that form.
		matchPath := rel
		if d.IsDir() {
			matchPath = rel + "/"
		}
		if gi != nil && gi.MatchesPath(matchPath) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			paths = append(paths, rel+"/")
		} else {
			paths = append(paths, rel)
		}
		if len(paths) >= maxIndexedFiles {
			return filepath.SkipAll
		}
		return nil
	})
	return paths
}
